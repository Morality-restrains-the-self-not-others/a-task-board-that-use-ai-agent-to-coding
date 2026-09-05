package wechatmpegress_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskAuth/wechatmpegress"
)

func TestForwardRejectsMissingSecret(t *testing.T) {
	s := &wechatmpegress.Server{Secret: "s3cret"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/wechat-mp/forward", strings.NewReader(`{"method":"GET","url":"https://api.weixin.qq.com/cgi-bin/token"}`))
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestForwardRejectsNonWeChatHost(t *testing.T) {
	s := &wechatmpegress.Server{Secret: "s3cret"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/wechat-mp/forward", strings.NewReader(`{"method":"GET","url":"https://evil.example/x"}`))
	req.Header.Set("X-Internal-Secret", "s3cret")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestForwardProxiesAllowlistedHost(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cgi-bin/token" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":7200}`))
	}))
	t.Cleanup(upstream.Close)

	// Rewrite allowlist check by dialing our test server via custom client + URL host spoof:
	// Server validates Hostname()==api.weixin.qq.com, so we pass that host and use Transport rewrite.
	s := &wechatmpegress.Server{
		Secret: "s3cret",
		Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			r.URL.Scheme = "http"
			r.URL.Host = strings.TrimPrefix(upstream.URL, "http://")
			r.Host = r.URL.Host
			return http.DefaultTransport.RoundTrip(r)
		})},
	}
	rec := httptest.NewRecorder()
	body := `{"method":"GET","url":"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/wechat-mp/forward", strings.NewReader(body))
	req.Header.Set("X-Internal-Secret", "s3cret")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	var parsed struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Status != 200 || !strings.Contains(parsed.Body, "access_token") {
		t.Fatalf("unexpected %#v", parsed)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestHealthz(t *testing.T) {
	s := &wechatmpegress.Server{Secret: "x"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("got %d", rec.Code)
	}
	b, _ := io.ReadAll(rec.Body)
	if string(b) != "ok" {
		t.Fatalf("body %q", b)
	}
}
