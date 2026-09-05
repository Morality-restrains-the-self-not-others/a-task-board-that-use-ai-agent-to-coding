// Package registryhost detects cloud-vendor VPC/intranet docker registry hosts
// that the platform cannot reach, and produces the vendor-facing hint.
//
// Single source of truth shared by taskAiProvider (image manifest / vendor portal)
// and taskCloudService (resolve-container-image). Keep the frontend helper in sync:
// taskAiProvider/frontend/src/utils/privateRegistryHint.js. Both are driven by the
// shared case table testdata/registry_cases.json to prevent drift.
package registryhost

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// PrivateRegistryHintText is shown when the platform cannot reach a VPC/intranet registry.
// Keep in sync with taskFE/app/src/utils/privateRegistryHint.js.
const PrivateRegistryHintText = "该地址为云厂商内网/VPC 私有地址，本平台无法触及。请改用公网 Registry 地址。"

var privateDNSLabels = map[string]struct{}{
	"vpc":      {},
	"vpce":     {},
	"internal": {},
	"intranet": {},
	"inner":    {},
	"private":  {},
}

// RegistryHostname extracts the host from a docker registry prefix (host[:port]).
func RegistryHostname(registry string) string {
	host := strings.TrimSpace(registry)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if i := strings.IndexAny(host, "/\\"); i >= 0 {
		host = host[:i]
	}
	if host == "" {
		return ""
	}
	if strings.HasPrefix(host, "[") {
		if end := strings.Index(host, "]"); end > 0 {
			return strings.ToLower(host[1:end])
		}
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return strings.ToLower(h)
	}
	if u, err := url.Parse("dummy://" + host); err == nil && u.Hostname() != "" {
		return strings.ToLower(u.Hostname())
	}
	return strings.ToLower(host)
}

// IsCloudVendorPrivateRegistryHost reports VPC / intranet / RFC1918 registry hosts.
func IsCloudVendorPrivateRegistryHost(registry string) bool {
	host := RegistryHostname(registry)
	if host == "" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		// RFC 1918 / RFC 4193 only. Loopback is used by httptest registries and is not a
		// cloud-vendor VPC endpoint.
		return ip.IsPrivate()
	}
	for _, lab := range strings.Split(host, ".") {
		if _, ok := privateDNSLabels[lab]; ok {
			return true
		}
		if strings.HasPrefix(lab, "registry-vpc") || strings.HasPrefix(lab, "registry-internal") {
			return true
		}
	}
	return false
}

func suggestPublicAliyunACR(host string) string {
	host = RegistryHostname(host)
	if !strings.HasSuffix(host, ".aliyuncs.com") {
		return ""
	}
	if strings.HasPrefix(host, "registry-vpc.") {
		return "registry." + strings.TrimPrefix(host, "registry-vpc.")
	}
	if strings.HasPrefix(host, "registry-internal.") {
		return "registry." + strings.TrimPrefix(host, "registry-internal.")
	}
	return ""
}

// PrivateRegistryUserMessage returns the vendor-facing hint, with Aliyun public mapping when known.
func PrivateRegistryUserMessage(registry string) string {
	host := RegistryHostname(registry)
	msg := PrivateRegistryHintText
	if pub := suggestPublicAliyunACR(host); pub != "" {
		msg += "（可将 " + host + " 改为 " + pub + "）"
	}
	return msg
}

// ErrPrivateRegistry is returned instead of dialing a VPC/intranet registry.
type ErrPrivateRegistry struct {
	Host string
}

func (e *ErrPrivateRegistry) Error() string {
	return PrivateRegistryUserMessage(e.Host)
}

// RejectPrivateRegistry returns ErrPrivateRegistry when host is not reachable from the platform.
func RejectPrivateRegistry(registry string) error {
	if !IsCloudVendorPrivateRegistryHost(registry) {
		return nil
	}
	return &ErrPrivateRegistry{Host: RegistryHostname(registry)}
}

// AnnotateRegistryFetchError appends the intranet hint when a fetch failed against a private host.
func AnnotateRegistryFetchError(err error, registry string) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*ErrPrivateRegistry); ok {
		return err
	}
	if !IsCloudVendorPrivateRegistryHost(registry) {
		return err
	}
	return fmt.Errorf("%s（仓库 %s：%s）", PrivateRegistryUserMessage(registry), RegistryHostname(registry), sanitizeFetchErr(err))
}

func sanitizeFetchErr(err error) string {
	msg := err.Error()
	if i := strings.Index(msg, "context deadline exceeded"); i >= 0 {
		return "context deadline exceeded"
	}
	if i := strings.Index(msg, "i/o timeout"); i >= 0 {
		return "i/o timeout"
	}
	if len(msg) > 80 {
		return msg[:80]
	}
	return msg
}
