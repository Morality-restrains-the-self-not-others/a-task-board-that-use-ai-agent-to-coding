package main

import (
	"os"
	"strings"
	"testing"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
)

func TestStripOutboundProxyEnv(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy:8080")
	t.Setenv("HTTPS_PROXY", "socks5h://127.0.0.1:1234")
	t.Setenv("ALL_PROXY", "socks5h://127.0.0.1:1234")
	t.Setenv("NO_PROXY", "127.0.0.1,localhost")

	stripOutboundProxyEnv()

	for _, key := range proxyEnvVarNames {
		if v, ok := os.LookupEnv(key); ok {
			t.Fatalf("expected %s to be unset, got %q", key, v)
		}
	}
	noProxy := os.Getenv("NO_PROXY")
	if !strings.Contains(noProxy, ".aliyuncs.com") {
		t.Fatalf("expected NO_PROXY to include .aliyuncs.com, got %q", noProxy)
	}
	if !strings.Contains(noProxy, "127.0.0.1") {
		t.Fatalf("expected existing NO_PROXY entries preserved, got %q", noProxy)
	}
}

func TestApplyAliyunECSNetworkSetsEndpointAndNoProxy(t *testing.T) {
	cfg := &openapiutil.Config{}
	applyAliyunECSNetwork(cfg, "cn-qingdao")
	if cfg.Endpoint == nil || *cfg.Endpoint != "ecs.cn-qingdao.aliyuncs.com" {
		t.Fatalf("endpoint=%v", cfg.Endpoint)
	}
	if cfg.NoProxy == nil || !strings.Contains(*cfg.NoProxy, "ecs.cn-qingdao.aliyuncs.com") {
		t.Fatalf("noProxy=%v", cfg.NoProxy)
	}
}
