package main

import (
	"io"
	"log"
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

// detectPlatformEgressCIDR returns this process's public egress IP as host/32
// so auto-created security groups allow SaaS → container probes (port 8765).
func detectPlatformEgressCIDR() string {
	return detectPlatformEgressCIDRFn()
}

func detectPlatformEgressCIDRCached() string {
	platformEgressOnce.Do(func() {
		// Explicit direct client: never inherit HTTP(S)_PROXY / ALL_PROXY.
		client := tracelog.DirectClient(3 * time.Second)
		for _, url := range []string{
			"https://ifconfig.me/ip",
			"https://icanhazip.com",
		} {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				continue
			}
			req.Header.Set("User-Agent", "taskCloudService-sg-whitelist/1.0")
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("[taskCloudService] platform_egress_probe_failed url=%s err=%v", url, err)
				continue
			}
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Printf("[taskCloudService] platform_egress_probe_bad_status url=%s status=%d", url, resp.StatusCode)
				continue
			}
			ip := strings.TrimSpace(string(body))
			parsed := net.ParseIP(ip)
			if parsed == nil {
				continue
			}
			if parsed.To4() != nil {
				platformEgressCIDR = parsed.String() + "/32"
				return
			}
			platformEgressCIDR = parsed.String() + "/128"
			return
		}
	})
	return platformEgressCIDR
}
