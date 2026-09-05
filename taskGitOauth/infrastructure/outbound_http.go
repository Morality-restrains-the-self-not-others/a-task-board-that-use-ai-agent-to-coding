package infrastructure

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// Optional GitHub egress (socks5/http) when direct dial is flaky.
// Must come from conf/env — never from shell HTTP(S)_PROXY.
var (
	proxyClientMu    sync.Mutex
	proxyClientCache = map[string]*http.Client{}
	// Shorter direct dial when outbound_proxy is set, so fallback is not blocked ~12s.
	httpClientFastFail = newOAuthHTTPClientWithDial(5 * time.Second)
	// GitLab Doorkeeper /oauth/token 在 2GiB 区域实例上常 >4s 才回响应头；
	// 与 GitHub 共用 4s ResponseHeaderTimeout 会在服务端已旋转 refresh 后客户端超时，
	// 新 refresh 无法落库 → 后续全部 invalid_grant。
	gitlabHTTPClient = newGitLabOAuthHTTPClient()
)

const gitlabResponseHeaderTimeout = 25 * time.Second

func newOAuthTransport(dialContext func(ctx context.Context, network, addr string) (net.Conn, error)) *http.Transport {
	return &http.Transport{
		Proxy:               nil,
		DialContext:         dialContext,
		TLSHandshakeTimeout: 12 * time.Second,
		// 4s：正常换票 <1s；挂起边缘快速失败，给 Refresh/Exchange 留出同 client 重试预算
		ResponseHeaderTimeout: 4 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// DialContext 非空时默认关闭 HTTP/2；显式 false + 仅协商 http/1.1，避免 http2 响应头超时
		ForceAttemptHTTP2: false,
		TLSClientConfig:   &tls.Config{NextProtos: []string{"http/1.1"}},
		MaxIdleConns:      32,
		IdleConnTimeout:   90 * time.Second,
	}
}

func newOAuthHTTPClientWithDial(dialTimeout time.Duration) *http.Client {
	// github.com DNS 边缘可能本机不可达；DialContext 并行回退到可达 IP（见 outbound_github_dial.go）
	tr := newOAuthTransport(githubAwareDialContext(dialTimeout))
	return &http.Client{Timeout: oauthHTTPTimeout, Transport: tr}
}

func newOAuthHTTPClient() *http.Client {
	return newOAuthHTTPClientWithDial(12 * time.Second)
}

func newGitLabOAuthHTTPClient() *http.Client {
	dial := (&net.Dialer{Timeout: 12 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	tr := newOAuthTransport(dial)
	tr.ResponseHeaderTimeout = gitlabResponseHeaderTimeout
	return &http.Client{Timeout: oauthHTTPTimeout, Transport: tr}
}

func githubOutboundProxyURL(cfg *Config) string {
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.GithubOutboundProxy)
}

// oauthHTTPClients returns direct first, then optional configured outbound proxy.
func oauthHTTPClients(cfg *Config) []*http.Client {
	proxyURL := githubOutboundProxyURL(cfg)
	direct := httpClient
	if proxyURL != "" {
		direct = httpClientFastFail
	}
	out := []*http.Client{direct}
	if proxyURL == "" {
		return out
	}
	pc, err := proxiedOAuthHTTPClient(proxyURL)
	if err != nil || pc == nil {
		log.Printf("[taskGitOauth] GithubOutboundProxy invalid %q: %v", proxyURL, err)
		return out
	}
	return append(out, pc)
}

func proxiedOAuthHTTPClient(proxyURL string) (*http.Client, error) {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return nil, fmt.Errorf("empty outbound proxy")
	}
	proxyClientMu.Lock()
	defer proxyClientMu.Unlock()
	if c, ok := proxyClientCache[proxyURL]; ok {
		return c, nil
	}
	c, err := newProxiedOAuthHTTPClient(proxyURL)
	if err != nil {
		return nil, err
	}
	proxyClientCache[proxyURL] = c
	return c, nil
}

func newProxiedOAuthHTTPClient(proxyURL string) (*http.Client, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "socks5", "socks5h":
		host := u.Host
		if host == "" {
			return nil, fmt.Errorf("socks proxy missing host")
		}
		var auth *proxy.Auth
		if u.User != nil {
			pass, _ := u.User.Password()
			auth = &proxy.Auth{User: u.User.Username(), Password: pass}
		}
		base := &net.Dialer{Timeout: 12 * time.Second, KeepAlive: 30 * time.Second}
		d, err := proxy.SOCKS5("tcp", host, auth, base)
		if err != nil {
			return nil, err
		}
		var dialContext func(ctx context.Context, network, addr string) (net.Conn, error)
		if cd, ok := d.(proxy.ContextDialer); ok {
			dialContext = cd.DialContext
		} else {
			dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return d.Dial(network, addr)
			}
		}
		return &http.Client{Timeout: oauthHTTPTimeout, Transport: newOAuthTransport(dialContext)}, nil
	case "http", "https":
		tr := newOAuthTransport(githubAwareDialContext(12 * time.Second))
		tr.Proxy = http.ProxyURL(u)
		return &http.Client{Timeout: oauthHTTPTimeout, Transport: tr}, nil
	default:
		return nil, fmt.Errorf("unsupported outbound proxy scheme %q", u.Scheme)
	}
}
