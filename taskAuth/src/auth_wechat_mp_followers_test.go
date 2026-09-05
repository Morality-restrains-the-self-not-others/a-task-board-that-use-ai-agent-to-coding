package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskAuth/domain"
)

func TestMatchFollowerQRScene(t *testing.T) {
	openID, unionID, ok := matchFollowerQRScene([]wechatMPFollowerInfo{
		{OpenID: "oOther", Subscribe: 1, QRSceneStr: ""},
		{OpenID: "oHit", UnionID: "u1", Subscribe: 1, QRSceneStr: "880376598551883776"},
	}, "880376598551883776")
	if !ok || openID != "oHit" || unionID != "u1" {
		t.Fatalf("got ok=%v open=%s union=%s", ok, openID, unionID)
	}
	if _, _, ok := matchFollowerQRScene([]wechatMPFollowerInfo{
		{OpenID: "oUnsub", Subscribe: 0, QRSceneStr: "880376598551883776"},
	}, "880376598551883776"); ok {
		t.Fatal("unsubscribed follower must not match")
	}
	if _, _, ok := matchFollowerQRScene(nil, "880376598551883776"); ok {
		t.Fatal("empty list")
	}
}

func stubWechatMPFollowers(t *testing.T, listJSON, batchJSON string) {
	t.Helper()
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
		if strings.Contains(urlStr, "/token") {
			return []byte(`{"access_token":"at-recon","expires_in":7200}`), nil
		}
		if strings.Contains(urlStr, "/user/get") {
			return []byte(listJSON), nil
		}
		t.Fatalf("unexpected get %s", urlStr)
		return nil, nil
	}
	wechatHTTPPostJSON = func(urlStr string, payload []byte) ([]byte, error) {
		if !strings.Contains(urlStr, "user/info/batchget") {
			t.Fatalf("unexpected post %s", urlStr)
		}
		var req struct {
			UserList []struct {
				OpenID string `json:"openid"`
			} `json:"user_list"`
		}
		if err := json.Unmarshal(payload, &req); err != nil || len(req.UserList) == 0 {
			t.Fatalf("batch payload %s", payload)
		}
		return []byte(batchJSON), nil
	}
}

func TestWeChatMPFollowStatusBindsFromQRSceneStr(t *testing.T) {
	setupAuthTestDB(t)
	withWechatMPApp(t, "tok")
	unionID := "union-recon-1"
	uid := mustCreateWechatTestUser(t, unionID)
	upsertWechatIdentity(uid, "web", "wxweb", "oWebRecon", unionID, "n", "", timeNowUTC())
	tempID := "880376598551883776"
	insertMPFollowTicket(t, tempID, uid, "pending", "wx-t", time.Now().UTC().Add(time.Hour))
	stubWechatMPFollowers(t,
		`{"total":2,"count":2,"data":{"openid":["oSceneHit","oSearch"]},"next_openid":""}`,
		`{"user_info_list":[
			{"openid":"oSceneHit","subscribe":1,"qr_scene_str":"880376598551883776"},
			{"openid":"oSearch","subscribe":1,"qr_scene_str":""}
		]}`,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/mp/follow-status/", nil)
	req.Header.Set("X-User-Id", uid)
	rec := httptest.NewRecorder()
	handleWeChatMPFollowStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"bound":true`) {
		t.Fatalf("bound: %s", body)
	}
	if !strings.Contains(body, `"ticket_status":"bound"`) {
		t.Fatalf("ticket_status: %s", body)
	}
	owner, ok := findWechatUserByAppOpenID(domain.WechatMPAppKey, "oSceneHit")
	if !ok || owner != uid {
		t.Fatalf("mp owner=%s ok=%v want %s", owner, ok, uid)
	}
	if _, ok := findWechatUserByAppOpenID(domain.WechatMPAppKey, "oSearch"); ok {
		t.Fatal("must not bind unrelated follower")
	}
}

func TestWeChatMPFollowStatusQRSceneNoMatchStaysPending(t *testing.T) {
	setupAuthTestDB(t)
	withWechatMPApp(t, "tok")
	uid := mustCreateWechatTestUser(t, "recon-nomatch")
	tempID := "880399999999999999"
	insertMPFollowTicket(t, tempID, uid, "pending", "wx-t", time.Now().UTC().Add(time.Hour))
	stubWechatMPFollowers(t,
		`{"total":1,"count":1,"data":{"openid":["oOther"]},"next_openid":""}`,
		`{"user_info_list":[{"openid":"oOther","subscribe":1,"qr_scene_str":""}]}`,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/mp/follow-status/", nil)
	req.Header.Set("X-User-Id", uid)
	rec := httptest.NewRecorder()
	handleWeChatMPFollowStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"bound":false`) {
		t.Fatalf("bound: %s", body)
	}
	if !strings.Contains(body, `"ticket_status":"pending"`) {
		t.Fatalf("ticket_status: %s", body)
	}
	if _, ok := findWechatUserByAppOpenID(domain.WechatMPAppKey, "oOther"); ok {
		t.Fatal("must not bind unmatched follower")
	}
}

func TestWeChatMPFollowStatusQRSceneDoesNotStealOpenID(t *testing.T) {
	setupAuthTestDB(t)
	withWechatMPApp(t, "tok")
	ownerID := mustCreateWechatTestUser(t, "recon-owner")
	upsertWechatIdentity(ownerID, domain.WechatMPAppKey, "wxmp", "oOwned", "", "n", "", timeNowUTC())
	scannerID := mustCreateWechatTestUser(t, "recon-scanner")
	tempID := "880388888888888888"
	insertMPFollowTicket(t, tempID, scannerID, "pending", "wx-t", time.Now().UTC().Add(time.Hour))
	stubWechatMPFollowers(t,
		`{"total":1,"count":1,"data":{"openid":["oOwned"]},"next_openid":""}`,
		`{"user_info_list":[{"openid":"oOwned","subscribe":1,"qr_scene_str":"880388888888888888"}]}`,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/mp/follow-status/", nil)
	req.Header.Set("X-User-Id", scannerID)
	rec := httptest.NewRecorder()
	handleWeChatMPFollowStatus(rec, req)
	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d %s", rec.Code, body)
	}
	if strings.Contains(body, `"bound":true`) {
		t.Fatalf("must not bind scanner: %s", body)
	}
	if !strings.Contains(body, `"ticket_status":"conflict"`) {
		t.Fatalf("ticket_status: %s", body)
	}
	if strings.Contains(body, ownerID) || strings.Contains(body, scannerID) {
		t.Fatalf("leaked user id: %s", body)
	}
	owner, ok := findWechatUserByAppOpenID(domain.WechatMPAppKey, "oOwned")
	if !ok || owner != ownerID {
		t.Fatalf("openid stolen owner=%s ok=%v", owner, ok)
	}
}

func TestWeChatMPFollowStatusSkipsQRSceneWhenAlreadyBound(t *testing.T) {
	setupAuthTestDB(t)
	withWechatMPApp(t, "tok")
	uid := mustCreateWechatTestUser(t, "recon-bound")
	upsertWechatIdentity(uid, domain.WechatMPAppKey, "wxmp", "oAlready", "", "n", "", timeNowUTC())
	insertMPFollowTicket(t, "880377777777777777", uid, "pending", "wx-t", time.Now().UTC().Add(time.Hour))
	userGets := 0
	prevGet := wechatHTTPGet
	t.Cleanup(func() { wechatHTTPGet = prevGet })
	wechatHTTPGet = func(urlStr string) ([]byte, error) {
		if strings.Contains(urlStr, "/user/get") {
			userGets++
		}
		t.Fatalf("already bound must not call wechat %s", urlStr)
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/mp/follow-status/", nil)
	req.Header.Set("X-User-Id", uid)
	rec := httptest.NewRecorder()
	handleWeChatMPFollowStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d %s", rec.Code, rec.Body.String())
	}
	if userGets != 0 {
		t.Fatalf("user/get calls %d", userGets)
	}
	if !strings.Contains(rec.Body.String(), `"bound":true`) {
		t.Fatalf("body %s", rec.Body.String())
	}
}
