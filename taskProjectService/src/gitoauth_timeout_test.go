package main

import (
	"testing"
	"time"
)

// TestApplyGitoauthTimeoutConfOverride verifies conf/taskProjectService/config.yaml 的
// gitoauth_timeout_seconds 能覆盖 gitHTTPClient.Timeout，且带 ≥10s 下限保护（OPT-20260817-036）。
func TestApplyGitoauthTimeoutConfOverride(t *testing.T) {
	old := gitHTTPClient.Timeout
	defer func() { gitHTTPClient.Timeout = old }()

	// 0 = 未配置 → 保持默认，不覆盖。
	applyGitoauthTimeout(0)
	if gitHTTPClient.Timeout != old {
		t.Fatalf("unconfigured should keep default %v, got %v", old, gitHTTPClient.Timeout)
	}

	// conf 覆盖生效：25s 直接应用。
	applyGitoauthTimeout(25)
	if gitHTTPClient.Timeout != 25*time.Second {
		t.Fatalf("expected 25s from conf, got %v", gitHTTPClient.Timeout)
	}

	// 低于下限 → 钳到 10s，防止误配过短超时。
	applyGitoauthTimeout(3)
	if gitHTTPClient.Timeout != 10*time.Second {
		t.Fatalf("expected lower bound 10s, got %v", gitHTTPClient.Timeout)
	}

	// 正好下限 10s 允许。
	applyGitoauthTimeout(10)
	if gitHTTPClient.Timeout != 10*time.Second {
		t.Fatalf("expected 10s at lower bound, got %v", gitHTTPClient.Timeout)
	}
}
