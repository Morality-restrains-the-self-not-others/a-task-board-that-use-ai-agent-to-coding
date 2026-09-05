package tracelog

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

func TestDisableDefaultEnvProxyIgnoresDeadProxy(t *testing.T) {
	// Reset once so this test can exercise disable even if Init ran earlier.
	disableEnvProxyOnce = sync.Once{}
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("http_proxy", "http://127.0.0.1:1")
	t.Setenv("https_proxy", "http://127.0.0.1:1")
	t.Setenv("ALL_PROXY", "socks5h://127.0.0.1:1")
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")

	disableDefaultEnvProxy()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("DefaultTransport must ignore dead env proxy: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestDirectClientProxyIsNil(t *testing.T) {
	c := DirectClient(2 * time.Second)
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type %T", c.Transport)
	}
	if tr.Proxy != nil {
		// Proxy may be a non-nil func that returns nil; evaluate against a URL.
		u, _ := url.Parse("https://example.com")
		p, err := tr.Proxy(&http.Request{URL: u})
		if err != nil {
			t.Fatal(err)
		}
		if p != nil {
			t.Fatalf("expected nil proxy URL, got %v", p)
		}
	}
}
