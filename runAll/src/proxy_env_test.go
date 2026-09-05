package main

import (
	"os"
	"strings"
	"testing"
)

func TestStripProxyEnvVars_RemovesAllProxyKeys(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"HOME=/home/user",
		"HTTP_PROXY=http://proxy:8080",
		"HTTPS_PROXY=http://proxy:8080",
		"ALL_PROXY=socks5://127.0.0.1:7890",
		"NO_PROXY=localhost,127.0.0.1",
		"http_proxy=http://proxy:8080",
		"https_proxy=http://proxy:8080",
		"all_proxy=socks5://127.0.0.1:7890",
		"no_proxy=localhost,127.0.0.1",
		"MY_APP_KEY=secret",
	}

	result := stripProxyEnvVars(env)

	// Should keep PATH, HOME, MY_APP_KEY
	for _, want := range []string{"PATH=", "HOME=", "MY_APP_KEY="} {
		found := false
		for _, r := range result {
			if strings.HasPrefix(r, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %s to be preserved", want)
		}
	}

	// Should remove all proxy vars
	for _, r := range result {
		key := ""
		if idx := strings.Index(r, "="); idx >= 0 {
			key = r[:idx]
		}
		for _, p := range proxyEnvVarNames {
			if strings.EqualFold(key, p) {
				t.Errorf("expected %s to be stripped, but it was present", key)
			}
		}
	}

	// Count: 11 original - 8 proxy vars = 3 remaining
	if len(result) != 3 {
		t.Errorf("expected 3 remaining vars, got %d: %v", len(result), result)
	}
}

func TestStripProxyEnvVars_EmptyEnv(t *testing.T) {
	result := stripProxyEnvVars(nil)
	if result != nil && len(result) != 0 {
		t.Errorf("expected empty result for nil input, got %v", result)
	}

	result = stripProxyEnvVars([]string{})
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got %v", result)
	}
}

func TestStripProxyEnvVars_NoProxyVars(t *testing.T) {
	env := []string{"PATH=/usr/bin", "HOME=/home/user", "USER=test"}
	result := stripProxyEnvVars(env)
	if len(result) != 3 {
		t.Errorf("expected 3 vars, got %d: %v", len(result), result)
	}
}

func TestShouldUseProxy_GlobalDefaultsFalse(t *testing.T) {
	svc := Service{UseProxy: nil}
	if svc.ShouldUseProxy(false) {
		t.Error("expected false when global is false and per-service is nil")
	}
	if !svc.ShouldUseProxy(true) {
		t.Error("expected true when global is true and per-service is nil")
	}
}

func TestShouldUseProxy_PerServiceOverride(t *testing.T) {
	trueVal := true
	falseVal := false

	// Per-service true overrides global false
	svc := Service{UseProxy: &trueVal}
	if !svc.ShouldUseProxy(false) {
		t.Error("per-service true should override global false")
	}

	// Per-service false overrides global true
	svc2 := Service{UseProxy: &falseVal}
	if svc2.ShouldUseProxy(true) {
		t.Error("per-service false should override global true")
	}
}

func TestBuildServiceEnv_StripsProxyWhenDisabled(t *testing.T) {
	// Set proxy env vars in the process environment for this test
	os.Setenv("HTTP_PROXY", "http://test-proxy:8080")
	os.Setenv("ALL_PROXY", "socks5://test-proxy:7890")
	defer func() {
		os.Unsetenv("HTTP_PROXY")
		os.Unsetenv("ALL_PROXY")
	}()

	r := &Runner{cfg: &Config{UseProxy: false}}

	svc := Service{
		Name: "test-service",
		Env:  map[string]string{"MY_VAR": "my_value"},
	}

	env := r.buildServiceEnv(svc)

	// Proxy vars must be stripped
	for _, e := range env {
		key := ""
		if idx := strings.Index(e, "="); idx >= 0 {
			key = e[:idx]
		}
		for _, p := range proxyEnvVarNames {
			if strings.EqualFold(key, p) {
				t.Errorf("proxy var %s should have been stripped, got: %s", key, e)
			}
		}
	}

	// Per-service env vars must be present
	found := false
	for _, e := range env {
		if strings.HasPrefix(e, "MY_VAR=my_value") {
			found = true
			break
		}
	}
	if !found {
		t.Error("per-service env var MY_VAR not found")
	}
}

func TestBuildServiceEnv_KeepsProxyWhenEnabled(t *testing.T) {
	os.Setenv("HTTP_PROXY", "http://test-proxy:8080")
	os.Setenv("ALL_PROXY", "socks5://test-proxy:7890")
	defer func() {
		os.Unsetenv("HTTP_PROXY")
		os.Unsetenv("ALL_PROXY")
	}()

	r := &Runner{cfg: &Config{UseProxy: true}}

	svc := Service{
		Name: "test-service",
	}

	env := r.buildServiceEnv(svc)

	// Proxy vars must be present
	foundHTTP := false
	foundALL := false
	for _, e := range env {
		if strings.HasPrefix(e, "HTTP_PROXY=") {
			foundHTTP = true
		}
		if strings.HasPrefix(e, "ALL_PROXY=") {
			foundALL = true
		}
	}
	if !foundHTTP {
		t.Error("HTTP_PROXY should be preserved when use_proxy is true")
	}
	if !foundALL {
		t.Error("ALL_PROXY should be preserved when use_proxy is true")
	}
}

func TestBuildServiceEnv_AlsoStripsProxyWhenNoPerServiceEnv(t *testing.T) {
	// When svc.Env is empty, buildServiceEnv must still strip proxy vars.
	// This covers the runBuild path which previously had a bug where
	// cmd.Env was only set when len(svc.Env) > 0, skipping proxy stripping.
	os.Setenv("HTTP_PROXY", "http://test-proxy:8080")
	os.Setenv("ALL_PROXY", "socks5://test-proxy:7890")
	defer func() {
		os.Unsetenv("HTTP_PROXY")
		os.Unsetenv("ALL_PROXY")
	}()

	r := &Runner{cfg: &Config{UseProxy: false}}
	svc := Service{Name: "build-service"} // no Env set

	env := r.buildServiceEnv(svc)

	for _, e := range env {
		key := ""
		if idx := strings.Index(e, "="); idx >= 0 {
			key = e[:idx]
		}
		for _, p := range proxyEnvVarNames {
			if strings.EqualFold(key, p) {
				t.Errorf("proxy var %s should have been stripped in build path, got: %s", key, e)
			}
		}
	}
}

func TestBuildServiceEnv_PerServiceProxyOverride(t *testing.T) {
	os.Setenv("HTTP_PROXY", "http://test-proxy:8080")
	defer func() {
		os.Unsetenv("HTTP_PROXY")
	}()

	trueVal := true
	r := &Runner{cfg: &Config{UseProxy: false}} // global: no proxy

	svc := Service{
		Name:     "proxy-service",
		UseProxy: &trueVal, // per-service: use proxy
	}

	env := r.buildServiceEnv(svc)

	// HTTP_PROXY must be present because per-service override is true
	found := false
	for _, e := range env {
		if strings.HasPrefix(e, "HTTP_PROXY=") {
			found = true
			break
		}
	}
	if !found {
		t.Error("HTTP_PROXY should be present when per-service use_proxy overrides global")
	}
}
