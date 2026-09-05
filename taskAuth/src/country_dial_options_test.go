package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestDerivePhoneCountryOptions_ChinaOnly(t *testing.T) {
	got := derivePhoneCountryOptions([]string{"+86"})
	want := []map[string]string{{"code": "+86", "name": "中国"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("China-only: got %#v, want %#v", got, want)
	}
}

func TestDerivePhoneCountryOptions_Multi(t *testing.T) {
	got := derivePhoneCountryOptions([]string{"+852", "+86"})
	want := []map[string]string{
		{"code": "+86", "name": "中国"},
		{"code": "+852", "name": "中国香港"},
	}
	// 输出顺序与静态表一致，与配置顺序无关
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("multi: got %#v, want %#v", got, want)
	}
}

func TestDerivePhoneCountryOptions_Empty(t *testing.T) {
	got := derivePhoneCountryOptions(nil)
	if len(got) != 0 {
		t.Fatalf("empty: got %#v, want empty slice", got)
	}
}

func TestDerivePhoneCountryOptions_UnknownCodeKept(t *testing.T) {
	// 白名单含静态表未收录的区号（手工写库场景）：保留原样，杜绝静默退化为"允许所有"
	got := derivePhoneCountryOptions([]string{"+999", "+86"})
	want := []map[string]string{
		{"code": "+86", "name": "中国"},
		{"code": "+999", "name": "+999"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unknown: got %#v, want %#v", got, want)
	}
}

func TestDerivePhoneCountryOptions_DupCodes(t *testing.T) {
	got := derivePhoneCountryOptions([]string{"+86", "+86", "+1"})
	want := []map[string]string{
		{"code": "+86", "name": "中国"},
		{"code": "+1", "name": "美国/加拿大"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dup: got %#v, want %#v", got, want)
	}
}

// 端到端：DB 默认策略（2026-08-09 产品调整：邮箱注册关、微信登录开、区域仅 +86）
// 时公开端点 phone_country_options 必须只含 +86，wechat_login_available 需同时满足
// 策略开启与应用已配置（测试库未配置微信应用 → false，fail-closed 与默认值解耦）。
func TestHandlePublicSystemFeaturePolicy_DefaultChinaOnly(t *testing.T) {
	setupAuthTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/public/system-feature-policy/", nil)
	rec := httptest.NewRecorder()
	handlePublicSystemFeaturePolicy(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if payload["enable_email_register"] != false {
		t.Fatalf("default enable_email_register = %#v, want false", payload["enable_email_register"])
	}
	if payload["enable_wechat_login"] != true {
		t.Fatalf("default enable_wechat_login = %#v, want true", payload["enable_wechat_login"])
	}
	opts, ok := payload["phone_country_options"].([]interface{})
	if !ok {
		t.Fatalf("phone_country_options missing or wrong type: %#v", payload["phone_country_options"])
	}
	if len(opts) != 1 {
		t.Fatalf("default policy must return exactly the +86 option, got %#v", opts)
	}
	first, ok := opts[0].(map[string]interface{})
	if !ok || first["code"] != "+86" {
		t.Fatalf("default policy first option must be +86, got %#v", opts)
	}
}

// 端到端：管理员保存白名单 ['+86'] 后，公开接口只返回该区域的选项
func TestHandlePublicSystemFeaturePolicy_ChinaOnly(t *testing.T) {
	setupAuthTestDB(t)
	if err := saveFeaturePolicy(&systemFeaturePolicy{
		EnablePhoneLogin:         true,
		EnableEmailRegister:      true,
		AllowedPhoneCountryCodes: []string{"+86"},
	}); err != nil {
		t.Fatalf("saveFeaturePolicy: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/public/system-feature-policy/", nil)
	rec := httptest.NewRecorder()
	handlePublicSystemFeaturePolicy(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload struct {
		PhoneCountryOptions []map[string]string `json:"phone_country_options"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	want := []map[string]string{{"code": "+86", "name": "中国"}}
	if !reflect.DeepEqual(payload.PhoneCountryOptions, want) {
		t.Fatalf("china-only: got %#v, want %#v", payload.PhoneCountryOptions, want)
	}
}

// 端到端：管理端默认策略 = 邮箱注册关闭、微信登录开启、手机号区域仅 ['+86']。
// 即系统管理「登录与支付策略」页全新部署时的展示值（2026-08-09 产品调整回归）。
func TestHandleAdminGetFeaturePolicy_DefaultValues(t *testing.T) {
	setupAuthTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/system-feature-policy/", nil)
	rec := httptest.NewRecorder()
	handleAdminSystemFeaturePolicy(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload struct {
		EnableEmailRegister     bool     `json:"enable_email_register"`
		EnableWechatLogin       bool     `json:"enable_wechat_login"`
		AllowedPhoneCountryCode []string `json:"allowed_phone_country_codes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if payload.EnableEmailRegister {
		t.Fatal("default enable_email_register = true, want false")
	}
	if !payload.EnableWechatLogin {
		t.Fatal("default enable_wechat_login = false, want true")
	}
	want := []string{"+86"}
	if !reflect.DeepEqual(payload.AllowedPhoneCountryCode, want) {
		t.Fatalf("default allowed_phone_country_codes = %#v, want %#v", payload.AllowedPhoneCountryCode, want)
	}
}
