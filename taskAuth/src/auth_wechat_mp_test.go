package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskAuth/domain"
)

func TestParseWechatMPXMLSubscribe(t *testing.T) {
	body := []byte(`<xml>
<ToUserName><![CDATA[gh_test]]></ToUserName>
<FromUserName><![CDATA[oMpOpenid1]]></FromUserName>
<CreateTime>1409659813</CreateTime>
<MsgType><![CDATA[event]]></MsgType>
<Event><![CDATA[subscribe]]></Event>
<UnionID><![CDATA[union-hit-1]]></UnionID>
</xml>`)
	msg, err := parseWechatMPXML(body)
	if err != nil {
		t.Fatal(err)
	}
	if msg.FromUserName != "oMpOpenid1" || msg.UnionID != "union-hit-1" {
		t.Fatalf("parsed %+v", msg)
	}
	if !domain.WechatMPIsSubscribeEvent(msg.MsgType, msg.Event) {
		t.Fatal("not subscribe")
	}
}

func TestDecryptWechatMPAESRoundtrip(t *testing.T) {
	keyB64 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("k"), 32))
	encodingKey := strings.TrimRight(keyB64, "=")
	inner := []byte(`<xml><ToUserName><![CDATA[gh]]></ToUserName><FromUserName><![CDATA[oid]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[subscribe]]></Event><UnionID><![CDATA[u1]]></UnionID></xml>`)
	cipherB64 := mustEncryptWechatMPAES(t, encodingKey, inner, "wx31273ca77c89dffe")
	plain, err := decryptWechatMPAES(encodingKey, cipherB64)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := parseWechatMPXML(plain)
	if err != nil {
		t.Fatal(err)
	}
	if msg.UnionID != "u1" || msg.FromUserName != "oid" {
		t.Fatalf("got %+v", msg)
	}
}

func mustEncryptWechatMPAES(t *testing.T, encodingAESKey string, xmlMsg []byte, appID string) string {
	t.Helper()
	key, err := wechatMPAESKey(encodingAESKey)
	if err != nil {
		t.Fatal(err)
	}
	rand16 := make([]byte, 16)
	_, _ = rand.Read(rand16)
	buf := bytes.NewBuffer(rand16)
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(xmlMsg)))
	buf.Write(lenBuf[:])
	buf.Write(xmlMsg)
	buf.WriteString(appID)
	padded := pkcs7Pad(buf.Bytes(), aes.BlockSize)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(out, padded)
	return base64.StdEncoding.EncodeToString(out)
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - (len(data) % blockSize)
	return append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
}

func withWechatMPApp(t *testing.T, token string) {
	t.Helper()
	prev := wechatApps
	t.Cleanup(func() { wechatApps = prev })
	wechatApps = map[string]*WeChatAppConfig{
		"mp": {
			Key:       "mp",
			AppID:     "wx31273ca77c89dffe",
			AppSecret: "test-secret",
			Token:     token,
			Type:      "mp",
		},
	}
}

func mpCallbackURL(token, timestamp, nonce, extra string) string {
	sig := domain.WechatMPSignature(token, timestamp, nonce)
	u := "/api/auth/wechat/mp/callback/?signature=" + sig + "&timestamp=" + timestamp + "&nonce=" + nonce
	if extra != "" {
		u += "&" + extra
	}
	return u
}

func TestWeChatMPCallbackGETEchostr(t *testing.T) {
	withWechatMPApp(t, "tok")
	req := httptest.NewRequest(http.MethodGet, mpCallbackURL("tok", "111", "n1", "echostr=hello-echo"), nil)
	rec := httptest.NewRecorder()
	handleWeChatMPCallback(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "hello-echo" {
		t.Fatalf("body %q", rec.Body.String())
	}
}

func TestWeChatMPCallbackGETBadSignature(t *testing.T) {
	withWechatMPApp(t, "tok")
	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/mp/callback/?signature=dead&timestamp=1&nonce=n&echostr=x", nil)
	rec := httptest.NewRecorder()
	handleWeChatMPCallback(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestWeChatMPCallbackUnconfigured(t *testing.T) {
	prev := wechatApps
	t.Cleanup(func() { wechatApps = prev })
	wechatApps = map[string]*WeChatAppConfig{}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/mp/callback/?signature=x&timestamp=1&nonce=n&echostr=x", nil)
	rec := httptest.NewRecorder()
	handleWeChatMPCallback(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestWeChatMPSubscribeBindsExistingUnionID(t *testing.T) {
	setupAuthTestDB(t)
	withWechatMPApp(t, "tok")
	unionID := "union-mp-bind-1"
	uid := mustCreateWechatTestUser(t, unionID)
	upsertWechatIdentity(uid, "web", "wx625802b55b33608f", "oWeb1", unionID, "n", "", timeNowUTC())
	usersBefore := countAuthUsers(t)

	xmlBody := `<xml><ToUserName><![CDATA[gh]]></ToUserName><FromUserName><![CDATA[oMpBind1]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[subscribe]]></Event><UnionID><![CDATA[` + unionID + `]]></UnionID></xml>`
	req := httptest.NewRequest(http.MethodPost, mpCallbackURL("tok", "111", "n1", ""), strings.NewReader(xmlBody))
	rec := httptest.NewRecorder()
	handleWeChatMPCallback(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "success" {
		t.Fatalf("resp %d %s", rec.Code, rec.Body.String())
	}
	owner, ok := findWechatUserByAppOpenID(domain.WechatMPAppKey, "oMpBind1")
	if !ok || owner != uid {
		t.Fatalf("mp alias owner=%s ok=%v want %s", owner, ok, uid)
	}
	if countAuthUsers(t) != usersBefore {
		t.Fatal("subscribe must not create a user")
	}
}

func TestWeChatMPSubscribeUnknownUnionIDPending(t *testing.T) {
	setupAuthTestDB(t)
	withWechatMPApp(t, "tok")
	usersBefore := countAuthUsers(t)
	xmlBody := `<xml><ToUserName><![CDATA[gh]]></ToUserName><FromUserName><![CDATA[oMpPend1]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[subscribe]]></Event><UnionID><![CDATA[union-unknown-1]]></UnionID></xml>`
	req := httptest.NewRequest(http.MethodPost, mpCallbackURL("tok", "111", "n1", ""), strings.NewReader(xmlBody))
	rec := httptest.NewRecorder()
	handleWeChatMPCallback(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d %s", rec.Code, rec.Body.String())
	}
	if countAuthUsers(t) != usersBefore {
		t.Fatal("unknown unionid must not create a user")
	}
	if _, ok := findWechatUserByAppOpenID(domain.WechatMPAppKey, "oMpPend1"); ok {
		t.Fatal("must not bind mp alias without user")
	}
	var openID string
	if err := db.QueryRow(`SELECT openid FROM auth_wechat_mp_subscribe_pending WHERE unionid = ?`, "union-unknown-1").Scan(&openID); err != nil {
		t.Fatalf("pending: %v", err)
	}
	if openID != "oMpPend1" {
		t.Fatalf("pending openid %s", openID)
	}
}

func TestWeChatMPFollowStatusClaimsPending(t *testing.T) {
	setupAuthTestDB(t)
	withWechatMPApp(t, "tok")
	unionID := "union-claim-1"
	uid := mustCreateWechatTestUser(t, unionID)
	upsertWechatIdentity(uid, "web", "wxweb", "oWebClaim", unionID, "n", "", timeNowUTC())
	saveMpSubscribePending(unionID, "oMpClaim1", "wx31273ca77c89dffe")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/mp/follow-status/", nil)
	req.Header.Set("X-User-Id", uid)
	rec := httptest.NewRecorder()
	handleWeChatMPFollowStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"bound":true`) {
		t.Fatalf("body %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"has_unionid":true`) {
		t.Fatalf("has_unionid missing: %s", rec.Body.String())
	}
	owner, ok := findWechatUserByAppOpenID(domain.WechatMPAppKey, "oMpClaim1")
	if !ok || owner != uid {
		t.Fatalf("claimed owner=%s ok=%v", owner, ok)
	}
}

func TestWeChatLoginUpsertClaimsMpPending(t *testing.T) {
	setupAuthTestDB(t)
	unionID := "union-login-claim-1"
	uid := mustCreateWechatTestUser(t, unionID)
	saveMpSubscribePending(unionID, "oMpLoginClaim", "wx31273ca77c89dffe")
	upsertWechatIdentity(uid, "web", "wxweb", "oWebLoginClaim", unionID, "n", "", timeNowUTC())
	owner, ok := findWechatUserByAppOpenID(domain.WechatMPAppKey, "oMpLoginClaim")
	if !ok || owner != uid {
		t.Fatalf("login upsert should claim pending mp alias owner=%s ok=%v", owner, ok)
	}
}

func TestWeChatMPFollowStatusNoUnionID(t *testing.T) {
	setupAuthTestDB(t)
	uid := mustCreateWechatTestUser(t, "phone-only-no-union")
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
	if !strings.Contains(body, `"has_unionid":false`) {
		t.Fatalf("has_unionid: %s", body)
	}
}

func TestWeChatMPFollowStatusUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/mp/follow-status/", nil)
	rec := httptest.NewRecorder()
	handleWeChatMPFollowStatus(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func insertMPFollowTicket(t *testing.T, id, userID, status, wxTicket string, expireAt time.Time) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO auth_wechat_mp_follow_ticket
			(id, user_id, status, wechat_ticket, expire_at, conflict_code, mp_openid, unionid, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, '', '', '', ?, ?)`,
		id, userID, status, wxTicket, expireAt, timeNowUTC(), timeNowUTC())
	if err != nil {
		t.Fatalf("insert ticket: %v", err)
	}
}

func stubWechatMPQRCreate(t *testing.T) *int {
	t.Helper()
	postCalls := 0
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
			return []byte(`{"access_token":"at-test","expires_in":7200}`), nil
		}
		return []byte(`{}`), nil
	}
	wechatHTTPPostJSON = func(urlStr string, payload []byte) ([]byte, error) {
		postCalls++
		if !strings.Contains(urlStr, "qrcode/create") {
			t.Fatalf("unexpected post %s", urlStr)
		}
		return []byte(`{"ticket":"TICKET-TEST","expire_seconds":86400}`), nil
	}
	return &postCalls
}

func TestWeChatMPScanBindsTicketUser(t *testing.T) {
	setupAuthTestDB(t)
	withWechatMPApp(t, "tok")
	uid := mustCreateWechatTestUser(t, "scan-ticket-user")
	tempID := "880311111111111111"
	insertMPFollowTicket(t, tempID, uid, "pending", "wx-t", time.Now().UTC().Add(time.Hour))
	usersBefore := countAuthUsers(t)

	xmlBody := `<xml><ToUserName><![CDATA[gh]]></ToUserName><FromUserName><![CDATA[oMpScan1]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[SCAN]]></Event><EventKey><![CDATA[` + tempID + `]]></EventKey><UnionID><![CDATA[union-scan-free]]></UnionID></xml>`
	req := httptest.NewRequest(http.MethodPost, mpCallbackURL("tok", "111", "n1", ""), strings.NewReader(xmlBody))
	rec := httptest.NewRecorder()
	handleWeChatMPCallback(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "success" {
		t.Fatalf("resp %d %s", rec.Code, rec.Body.String())
	}
	owner, ok := findWechatUserByAppOpenID(domain.WechatMPAppKey, "oMpScan1")
	if !ok || owner != uid {
		t.Fatalf("mp owner=%s ok=%v want %s", owner, ok, uid)
	}
	if countAuthUsers(t) != usersBefore {
		t.Fatal("SCAN must not create a user")
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM auth_wechat_mp_follow_ticket WHERE id = ?`, tempID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "bound" {
		t.Fatalf("ticket status %s", status)
	}
}

func TestWeChatMPSceneConflictDoesNotRebind(t *testing.T) {
	setupAuthTestDB(t)
	withWechatMPApp(t, "tok")
	unionID := "union-conflict-other"
	ownerID := mustCreateWechatTestUser(t, unionID)
	upsertWechatIdentity(ownerID, "web", "wxweb", "oWebOwner", unionID, "n", "", timeNowUTC())
	scannerID := mustCreateWechatTestUser(t, "scanner-no-union")
	tempID := "880322222222222222"
	insertMPFollowTicket(t, tempID, scannerID, "pending", "wx-t", time.Now().UTC().Add(time.Hour))

	xmlBody := `<xml><ToUserName><![CDATA[gh]]></ToUserName><FromUserName><![CDATA[oMpConflict]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[subscribe]]></Event><EventKey><![CDATA[qrscene_` + tempID + `]]></EventKey><UnionID><![CDATA[` + unionID + `]]></UnionID></xml>`
	req := httptest.NewRequest(http.MethodPost, mpCallbackURL("tok", "111", "n1", ""), strings.NewReader(xmlBody))
	rec := httptest.NewRecorder()
	handleWeChatMPCallback(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d %s", rec.Code, rec.Body.String())
	}
	if _, ok := findWechatUserByAppOpenID(domain.WechatMPAppKey, "oMpConflict"); ok {
		t.Fatal("must not bind mp to scanner when unionid belongs to another user")
	}
	stReq := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/mp/follow-status/", nil)
	stReq.Header.Set("X-User-Id", scannerID)
	stRec := httptest.NewRecorder()
	handleWeChatMPFollowStatus(stRec, stReq)
	body := stRec.Body.String()
	if stRec.Code != http.StatusOK {
		t.Fatalf("status %d %s", stRec.Code, body)
	}
	if !strings.Contains(body, `"ticket_status":"conflict"`) {
		t.Fatalf("ticket_status: %s", body)
	}
	if !strings.Contains(body, wechatMPConflictUnionOther) {
		t.Fatalf("conflict_code: %s", body)
	}
	if strings.Contains(body, ownerID) || strings.Contains(body, unionID) {
		t.Fatalf("leaked other identity: %s", body)
	}
}

func TestWeChatMPFollowQRRequiresAuthAndIdempotency(t *testing.T) {
	withWechatMPApp(t, "tok")
	req := httptest.NewRequest(http.MethodPost, "/api/auth/wechat/mp/follow-qr/", nil)
	rec := httptest.NewRecorder()
	handleWeChatMPFollowQR(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing key code %d", rec.Code)
	}
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/wechat/mp/follow-qr/", nil)
	req2.Header.Set("Idempotency-Key", "ik-1")
	rec2 := httptest.NewRecorder()
	handleWeChatMPFollowQR(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("unauth code %d", rec2.Code)
	}
}

// TestWeChatMPFollowQRRejectsNonNumericUserID 回归 OPT-20260826-007：
// 确定性非数字 user_id（如 bootstrap-admin）写入 BIGINT 票表前应被拒绝，
// 避免 MySQL 1366。
func TestWeChatMPFollowQRRejectsNonNumericUserID(t *testing.T) {
	withWechatMPApp(t, "tok")
	req := httptest.NewRequest(http.MethodPost, "/api/auth/wechat/mp/follow-qr/", nil)
	req.Header.Set("Idempotency-Key", "ik-non-numeric")
	req.Header.Set("X-User-Id", "bootstrap-admin")
	rec := httptest.NewRecorder()
	handleWeChatMPFollowQR(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("non-numeric user_id code %d, want 400", rec.Code)
	}
}

func TestWeChatMPFollowQRReusesPendingTicket(t *testing.T) {
	setupAuthTestDB(t)
	withWechatMPApp(t, "tok")
	calls := stubWechatMPQRCreate(t)
	uid := mustCreateWechatTestUser(t, "qr-issuer")
	issue := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/wechat/mp/follow-qr/", nil)
		req.Header.Set("X-User-Id", uid)
		req.Header.Set("Idempotency-Key", "ik-reuse")
		rec := httptest.NewRecorder()
		handleWeChatMPFollowQR(rec, req)
		return rec
	}
	rec1 := issue()
	if rec1.Code != http.StatusOK {
		t.Fatalf("first %d %s", rec1.Code, rec1.Body.String())
	}
	if !strings.Contains(rec1.Body.String(), "showqrcode") {
		t.Fatalf("qr_src: %s", rec1.Body.String())
	}
	rec2 := issue()
	if rec2.Code != http.StatusOK {
		t.Fatalf("second %d %s", rec2.Code, rec2.Body.String())
	}
	if *calls != 1 {
		t.Fatalf("wechat create calls %d want 1", *calls)
	}
	got1 := parseFollowQRBody(t, rec1.Body.Bytes())
	got2 := parseFollowQRBody(t, rec2.Body.Bytes())
	if got1["temp_id"] != got2["temp_id"] || got1["qr_src"] != got2["qr_src"] {
		t.Fatalf("reuse mismatch %+v vs %+v", got1, got2)
	}
}

func TestWeChatMPEncryptedSubscribeBindsExistingUnionID(t *testing.T) {
	setupAuthTestDB(t)
	withWechatMPApp(t, "tok")
	aesKey := strings.TrimRight(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("k"), 32)), "=")
	wechatApps["mp"].EncodingAESKey = aesKey
	unionID := "union-mp-enc-1"
	uid := mustCreateWechatTestUser(t, unionID)
	upsertWechatIdentity(uid, "web", "wx625802b55b33608f", "oWebEnc", unionID, "n", "", timeNowUTC())

	inner := []byte(`<xml><ToUserName><![CDATA[gh]]></ToUserName><FromUserName><![CDATA[oMpEnc1]]></FromUserName><CreateTime>1</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[subscribe]]></Event><UnionID><![CDATA[` + unionID + `]]></UnionID></xml>`)
	cipherB64 := mustEncryptWechatMPAES(t, aesKey, inner, "wx31273ca77c89dffe")
	xmlBody := `<xml><Encrypt><![CDATA[` + cipherB64 + `]]></Encrypt></xml>`
	msgSig := domain.WechatMPSignature("tok", "111", "n1", cipherB64)
	req := httptest.NewRequest(http.MethodPost, mpCallbackURL("tok", "111", "n1", "encrypt_type=aes&msg_signature="+msgSig), strings.NewReader(xmlBody))
	rec := httptest.NewRecorder()
	handleWeChatMPCallback(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "success" {
		t.Fatalf("resp %d %s", rec.Code, rec.Body.String())
	}
	owner, ok := findWechatUserByAppOpenID(domain.WechatMPAppKey, "oMpEnc1")
	if !ok || owner != uid {
		t.Fatalf("encrypted mp owner=%s ok=%v want %s", owner, ok, uid)
	}
}

func parseFollowQRBody(t *testing.T, raw []byte) map[string]string {
	t.Helper()
	var parsed map[string]string
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("json: %v %s", err, raw)
	}
	return parsed
}

func countAuthUsers(t *testing.T) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_user`).Scan(&n); err != nil {
		t.Fatalf("count users: %v", err)
	}
	return n
}
