package main

import (
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	backendProxyModeNoProxy = "no_proxy"
	backendProxyModeSystem  = "system"
)

func backendProxyMode() string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_BACKEND_PROXY_MODE")))
	if mode == backendProxyModeSystem {
		return backendProxyModeSystem
	}
	return backendProxyModeNoProxy
}

func newBackendHTTPClient(timeout time.Duration) *http.Client {
	proxyFn := http.ProxyFromEnvironment
	if backendProxyMode() == backendProxyModeNoProxy {
		proxyFn = nil
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 proxyFn,
			DialContext:           (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
			MaxIdleConns:          32,
			MaxIdleConnsPerHost:   8,
			IdleConnTimeout:       30 * time.Second,
			ResponseHeaderTimeout: timeout,
		},
		Timeout: timeout,
	}
}
