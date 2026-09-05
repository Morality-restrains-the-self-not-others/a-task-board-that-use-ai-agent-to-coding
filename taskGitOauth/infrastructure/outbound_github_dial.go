package infrastructure

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// defaultGithubDialFallbackIPs — DNS 可能返回本机不可达的 github.com 边缘
// （2026-08-11：20.205.243.166 TCP 超时）。下列 IP 经实测可完成 OAuth 换票。
// 顺序即拨号优先级；运维可用 GITOAUTH_GITHUB_DIAL_FALLBACK_IPS=ip1,ip2 覆盖。
// 注意：部分 IP 会 TCP/TLS 成功但 HTTP 挂起；HTTP 失败后由 ReportGithubDialHTTPFailure
// 将该 endpoint 临时降级（见 githubDialDemoteTTL），避免轮转反复命中挂起节点。
var defaultGithubDialFallbackIPs = []string{
	"20.27.177.113",
	"140.82.114.3",
	"140.82.112.3",
}

const githubDialDemoteTTL = 2 * time.Minute

var (
	lookupIPAddr = func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return net.DefaultResolver.LookupIPAddr(ctx, host)
	}
	githubDialFallbackIPs  = defaultGithubDialFallbackIPs
	githubDialPerIPTimeout = 3 * time.Second

	githubDialRotateMu sync.Mutex
	githubDialRotate   int // 失败后轮转 fallback 起点，配合 demote 避开挂起节点

	githubLastDialMu   sync.Mutex
	githubLastDialEP   string
	githubDemotedUntil = map[string]time.Time{} // host:port → demotedUntil
)

func isGitHubWebHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	return h == "github.com" || h == "www.github.com"
}

func githubAwareDialContext(dialTimeout time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	base := &net.Dialer{Timeout: dialTimeout, KeepAlive: 30 * time.Second}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil || !isGitHubWebHost(host) {
			return base.DialContext(ctx, network, addr)
		}
		ips, lookupErr := lookupIPAddr(ctx, host)
		endpoints := buildGithubDialEndpoints(ips, port)
		if len(endpoints) == 0 {
			if lookupErr != nil {
				return nil, fmt.Errorf("github.com lookup: %w", lookupErr)
			}
			return base.DialContext(ctx, network, addr)
		}
		conn, used, err := dialGithubEndpointsSequential(ctx, network, endpoints, githubDialPerIPTimeout)
		if err != nil {
			rotateGithubDialPreference()
			if lookupErr != nil {
				return nil, fmt.Errorf("github.com dial (lookup=%v): %w", lookupErr, err)
			}
			return nil, err
		}
		noteGithubDialUsed(used)
		dnsPrimary := ""
		if len(ips) > 0 && ips[0].IP != nil {
			dnsPrimary = net.JoinHostPort(ips[0].IP.String(), port)
		}
		if used != "" && used != dnsPrimary {
			log.Printf("[taskGitOauth] github.com dial via fallback endpoint %s (DNS may be unreachable)", used)
		}
		return conn, nil
	}
}

// buildGithubDialEndpoints — fallback IP 优先于 DNS（本地区 DNS 常指向不可达边缘），
// 再追加 DNS IP；失败轮转改变 fallback 起点；HTTP 降级中的 endpoint 排到队尾。
func buildGithubDialEndpoints(dnsIPs []net.IPAddr, port string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(ip string) {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			return
		}
		ep := net.JoinHostPort(ip, port)
		if _, ok := seen[ep]; ok {
			return
		}
		seen[ep] = struct{}{}
		out = append(out, ep)
	}

	fallbacks := append([]string(nil), githubDialFallbackIPs...)
	if n := len(fallbacks); n > 0 {
		githubDialRotateMu.Lock()
		rot := githubDialRotate % n
		githubDialRotateMu.Unlock()
		for i := 0; i < n; i++ {
			add(fallbacks[(rot+i)%n])
		}
	}
	for _, a := range dnsIPs {
		if a.IP == nil {
			continue
		}
		add(a.IP.String())
	}
	return orderGithubDialEndpoints(out)
}

// orderGithubDialEndpoints — 将仍在 demote TTL 内的 endpoint 排到队尾（软降级，不剔除），
// 保证「TCP 通但 HTTP 挂起」的节点不会在重试中反复抢到首拨号位。
func orderGithubDialEndpoints(eps []string) []string {
	if len(eps) == 0 {
		return eps
	}
	now := time.Now()
	githubLastDialMu.Lock()
	defer githubLastDialMu.Unlock()
	var active, demoted []string
	for _, ep := range eps {
		until, ok := githubDemotedUntil[ep]
		if !ok {
			active = append(active, ep)
			continue
		}
		if now.Before(until) {
			demoted = append(demoted, ep)
			continue
		}
		delete(githubDemotedUntil, ep)
		active = append(active, ep)
	}
	return append(active, demoted...)
}

func noteGithubDialUsed(ep string) {
	ep = strings.TrimSpace(ep)
	if ep == "" {
		return
	}
	githubLastDialMu.Lock()
	githubLastDialEP = ep
	githubLastDialMu.Unlock()
}

func rotateGithubDialPreference() {
	githubDialRotateMu.Lock()
	githubDialRotate++
	githubDialRotateMu.Unlock()
}

// ReportGithubDialHTTPFailure — HTTP 层超时/网络失败时：
// 1) 将最近一次拨通的 endpoint 临时降级（避开 TCP 通但 HTTP 挂起的边缘）；
// 2) 轮转 fallback 起点，配合重试换票/refresh。
func ReportGithubDialHTTPFailure() {
	githubLastDialMu.Lock()
	ep := githubLastDialEP
	if ep != "" {
		githubDemotedUntil[ep] = time.Now().Add(githubDialDemoteTTL)
		log.Printf("[taskGitOauth] github.com dial endpoint demoted for %s after HTTP failure: %s", githubDialDemoteTTL, ep)
	}
	githubLastDialMu.Unlock()
	rotateGithubDialPreference()
}

// resetGithubDialDemotionState — 测试辅助：清空 last-used / demote 表（不改轮转游标）。
func resetGithubDialDemotionState() {
	githubLastDialMu.Lock()
	githubLastDialEP = ""
	githubDemotedUntil = map[string]time.Time{}
	githubLastDialMu.Unlock()
}

func dialGithubEndpointsSequential(ctx context.Context, network string, endpoints []string, perIP time.Duration) (net.Conn, string, error) {
	if len(endpoints) == 0 {
		return nil, "", fmt.Errorf("no dial endpoints")
	}
	var firstErr error
	d := net.Dialer{Timeout: perIP, KeepAlive: 30 * time.Second}
	for _, ep := range endpoints {
		if err := ctx.Err(); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			break
		}
		dctx, cancel := context.WithTimeout(ctx, perIP)
		c, err := d.DialContext(dctx, network, ep)
		cancel()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		return c, ep, nil
	}
	if firstErr == nil {
		firstErr = fmt.Errorf("all github.com dial endpoints failed")
	}
	return nil, "", firstErr
}
