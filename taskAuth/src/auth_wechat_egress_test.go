package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUseWechatMPEgressOnlyForWeixinHost(t *testing.T) {
	wechatMPEgressBaseURL = "http://egress.test"
	wechatMPEgressSecret = "sec"
	t.Cleanup(func() {
		wechatMPEgressBaseURL = ""
		wechatMPEgressSecret = ""
	})
	if !useWechatMPEgress("https://api.weixin.qq.com/cgi-bin/token") {
		t.Fatal("expected true")
	}
	if useWechatMPEgress("https://open.weixin.qq.com/connect/qrconnect") {
		t.Fatal("oauth open host must stay direct")
	}
}

func TestWechatMPEgressDoDecodesBody(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Secret") != "sec" {
			t.Fatalf("secret %q", r.Header.Get("X-Internal-Secret"))
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": 200,
			"body":   `{"access_token":"abc","expires_in":7200}`,
		})
	}))
	t.Cleanup(upstream.Close)
	wechatMPEgressBaseURL = upstream.URL
	wechatMPEgressSecret = "sec"
	t.Cleanup(func() {
		wechatMPEgressBaseURL = ""
		wechatMPEgressSecret = ""
	})
	got, err := wechatMPEgressDo(http.MethodGet, "https://api.weixin.qq.com/cgi-bin/token", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "access_token") {
		t.Fatalf("got %s", got)
	}
}
