package tracelog

import (
	"net/http"
	"sync"
	"time"
)

var disableEnvProxyOnce sync.Once

// disableDefaultEnvProxy makes the process-wide DefaultTransport ignore
// HTTP(S)_PROXY / ALL_PROXY. Dev shells often export a local SOCKS for
// package downloads; business outbound must not inherit a dead 127.0.0.1 proxy.
// See .ai/01_project_constraints/23_app_startup_no_env_proxy.md.
func disableDefaultEnvProxy() {
	disableEnvProxyOnce.Do(func() {
		base, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			http.DefaultTransport = &http.Transport{Proxy: nil}
			return
		}
		tr := base.Clone()
		tr.Proxy = nil
		http.DefaultTransport = tr
	})
}

// DirectClient returns an http.Client that never uses environment proxies.
func DirectClient(timeout time.Duration) *http.Client {
	disableDefaultEnvProxy()
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{Proxy: nil},
		}
	}
	tr := base.Clone()
	tr.Proxy = nil
	return &http.Client{Timeout: timeout, Transport: tr}
}
