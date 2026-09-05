package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseWechatMPAccessTokenBodyRejectsErrcode(t *testing.T) {
	_, _, err := parseWechatMPAccessTokenBody([]byte(`{"errcode":40164,"errmsg":"invalid ip 222.76.156.241 ipv6 ::ffff:222.76.156.241, not in whitelist"}`))
	var apiErr wechatMPAPIError
	if err == nil || !errors.As(err, &apiErr) || apiErr.Code != 40164 {
		t.Fatalf("err=%v apiErr=%+v", err, apiErr)
	}
	if wechatMPInvalidIPFromMsg(apiErr.Msg) != "222.76.156.241" {
		t.Fatalf("ip from msg %q", apiErr.Msg)
	}
}

func TestWechatMPAccessTokenErrcodeIsNotCached(t *testing.T) {
	calls := 0
	prevGet := wechatHTTPGet
	prevPost := wechatHTTPPostJSON
	wechatMPTokenMu.Lock()
	wechatMPAccessToken = ""
	wechatMPTokenExpires = time.Time{}
	wechatMPTokenMu.Unlock()
	t.Cleanup(func() {
		wechatHTTPGet = prevGet
		wechatHTTPPostJSON = prevPost
		wechatMPTokenMu.Lock()
		wechatMPAccessToken = ""
		wechatMPTokenExpires = time.Time{}
		wechatMPTokenMu.Unlock()
	})
	wechatHTTPGet = func(urlStr string) ([]byte, error) {
		calls++
		if calls == 1 {
			return []byte(`{"errcode":40164,"errmsg":"invalid ip 222.76.156.241 ipv6 ::ffff:222.76.156.241, not in whitelist"}`), nil
		}
		return []byte(`{"access_token":"at-ok","expires_in":7200}`), nil
	}
	wechatHTTPPostJSON = func(urlStr string, payload []byte) ([]byte, error) {
		return []byte(`{"ticket":"TICKET-OK","expire_seconds":86400}`), nil
	}
	app := &WeChatAppConfig{AppID: "wx-test", AppSecret: "secret"}
	_, err := createWechatMPSceneQR(app, "scene-1")
	if err == nil || !strings.Contains(err.Error(), "40164") {
		t.Fatalf("first err %v", err)
	}
	ticket, err := createWechatMPSceneQR(app, "scene-1")
	if err != nil || ticket != "TICKET-OK" {
		t.Fatalf("second ticket=%s err=%v", ticket, err)
	}
	if calls != 2 {
		t.Fatalf("token fetches %d want 2 (failure must not be cached)", calls)
	}
}

func TestWeChatMPFollowQRIPWhitelistMessage(t *testing.T) {
	setupAuthTestDB(t)
	withWechatMPApp(t, "tok")
	uid := mustCreateWechatTestUser(t, "qr-ip-user")
	prevGet := wechatHTTPGet
	wechatMPTokenMu.Lock()
	wechatMPAccessToken = ""
	wechatMPTokenExpires = time.Time{}
	wechatMPTokenMu.Unlock()
	t.Cleanup(func() {
		wechatHTTPGet = prevGet
		wechatMPTokenMu.Lock()
		wechatMPAccessToken = ""
		wechatMPTokenExpires = time.Time{}
		wechatMPTokenMu.Unlock()
	})
	wechatHTTPGet = func(urlStr string) ([]byte, error) {
		return []byte(`{"errcode":40164,"errmsg":"invalid ip 222.76.156.241 ipv6 ::ffff:222.76.156.241, not in whitelist"}`), nil
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/wechat/mp/follow-qr/", nil)
	req.Header.Set("X-User-Id", uid)
	req.Header.Set("Idempotency-Key", "ik-40164")
	rec := httptest.NewRecorder()
	handleWeChatMPFollowQR(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "222.76.156.241") || !strings.Contains(body, "IP 白名单") {
		t.Fatalf("user message: %s", body)
	}
}
