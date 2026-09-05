package aliyun

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"tracelog"
)

var (
	platformEgressOnce sync.Once
	platformEgressCIDR string
	// detectPlatformEgressCIDRFn is overridable in tests.
	detectPlatformEgressCIDRFn = detectPlatformEgressCIDRCached
)

// detectPlatformEgressCIDR returns this process's public egress IP as /32|/128
// (for SaaS → container probes). Cached for process lifetime; empty on failure.
// Prefers IPv4: Aliyun SourceCidrIp rejects IPv6, and most probe paths are IPv4.
func detectPlatformEgressCIDR() string {
	return detectPlatformEgressCIDRFn()
}

func detectPlatformEgressCIDRCached() string {
	platformEgressOnce.Do(func() {
		// Explicit direct client: never inherit HTTP(S)_PROXY / ALL_PROXY.
		client := tracelog.DirectClient(3 * time.Second)
		var ipv6Fallback string
		for _, url := range []string{
			"https://ipv4.icanhazip.com",
			"https://api.ipify.org",
			"https://ifconfig.me/ip",
			"https://icanhazip.com",
		} {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				continue
			}
			req.Header.Set("User-Agent", "taskEvents-sg-whitelist/1.0")
			resp, err := client.Do(req)
			if err != nil {
				slog.Warn("platform_egress_probe_failed",
					"level", "warn",
					"url", url,
					"error", err.Error(),
				)
				continue
			}
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				slog.Warn("platform_egress_probe_bad_status",
					"level", "warn",
					"url", url,
					"status", resp.StatusCode,
				)
				continue
			}
			ip := strings.TrimSpace(string(body))
			parsed := net.ParseIP(ip)
			if parsed == nil {
				continue
			}
			if parsed.To4() != nil {
				if cidr := normalizePublicIPCIDR(parsed.String()); cidr != "" {
					platformEgressCIDR = cidr
					return
				}
			}
			if ipv6Fallback == "" {
				if cidr := normalizePublicIPCIDR(parsed.String()); cidr != "" {
					ipv6Fallback = cidr
				}
			}
		}
		platformEgressCIDR = ipv6Fallback
	})
	return platformEgressCIDR
}

// platformExtraIngressCIDRs merges env extras with auto-detected SaaS egress.
func platformExtraIngressCIDRs() []string {
	return mergeIngressCIDRs(extraIngressCIDRsFromEnv(), []string{detectPlatformEgressCIDR()})
}
