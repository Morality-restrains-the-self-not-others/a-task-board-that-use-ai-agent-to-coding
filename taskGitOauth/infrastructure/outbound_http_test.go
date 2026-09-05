package infrastructure

import (
	"strings"
	"testing"
	"time"

	"net/http"
)

func TestNewProxiedOAuthHTTPClientRejectsBadScheme(t *testing.T) {
	_, err := newProxiedOAuthHTTPClient("ftp://127.0.0.1:1080")
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("err=%v", err)
	}
}

func TestNewProxiedOAuthHTTPClientSOCKS(t *testing.T) {
	c, err := newProxiedOAuthHTTPClient("socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil || c.Timeout != oauthHTTPTimeout {
		t.Fatalf("client=%v", c)
	}
}

func TestOauthHTTPClientsIncludesProxy(t *testing.T) {
	cfg := &Config{GithubOutboundProxy: "socks5://127.0.0.1:1080"}
	clients := oauthHTTPClients(cfg)
	if len(clients) != 2 {
		t.Fatalf("len=%d want 2", len(clients))
	}
	if clients[0] != httpClientFastFail {
		t.Fatal("with outbound_proxy, first client must be fast-fail direct")
	}
	if clients[1] == httpClient || clients[1] == httpClientFastFail {
		t.Fatal("second client must be proxied")
	}
}

func TestOauthHTTPClientsDirectOnly(t *testing.T) {
	clients := oauthHTTPClients(&Config{})
	if len(clients) != 1 || clients[0] != httpClient {
		t.Fatalf("clients=%v", clients)
	}
}

func TestGitLabHTTPClientHeaderTimeoutExceedsGitHubFailFast(t *testing.T) {
	gl, ok := gitlabHTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("gitlab transport type %T", gitlabHTTPClient.Transport)
	}
	gh, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("github transport type %T", httpClient.Transport)
	}
	if gh.ResponseHeaderTimeout != 4*time.Second {
		t.Fatalf("github header timeout=%s want 4s", gh.ResponseHeaderTimeout)
	}
	if gl.ResponseHeaderTimeout < 20*time.Second {
		t.Fatalf("gitlab header timeout=%s want ≥20s so slow Doorkeeper does not rotate-and-drop", gl.ResponseHeaderTimeout)
	}
}
