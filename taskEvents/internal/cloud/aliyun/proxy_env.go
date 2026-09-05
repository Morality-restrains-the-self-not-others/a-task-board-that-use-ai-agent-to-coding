package aliyun

import (
	"os"
	"strings"
)

var proxyEnvVarNames = []string{
	"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY",
	"http_proxy", "https_proxy", "all_proxy",
}

func init() {
	stripOutboundProxyEnv()
}

// stripOutboundProxyEnv removes process-wide HTTP/SOCKS proxy variables.
// Cloud event workers call Aliyun public APIs and must not inherit broken dev proxies.
func stripOutboundProxyEnv() {
	for _, key := range proxyEnvVarNames {
		_ = os.Unsetenv(key)
	}
	mergeNoProxyForAliyun()
}

func mergeNoProxyForAliyun() {
	const aliyunHosts = ".aliyuncs.com"
	existing := strings.TrimSpace(os.Getenv("NO_PROXY"))
	if existing == "" {
		existing = strings.TrimSpace(os.Getenv("no_proxy"))
	}
	parts := []string{}
	if existing != "" {
		parts = append(parts, strings.Split(existing, ",")...)
	}
	parts = append(parts, aliyunHosts)
	seen := map[string]struct{}{}
	merged := make([]string, 0, len(parts))
	for _, raw := range parts {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, item)
	}
	joined := strings.Join(merged, ",")
	_ = os.Setenv("NO_PROXY", joined)
	_ = os.Setenv("no_proxy", joined)
}
