package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// ---- system feature policy — DB-backed feature toggles ---- //
// OPT-049: Migrated from Django saas-backend to taskAuth 2026-07-30.
// Replaces the hardcoded nil-returning stubs in django_client.go and sms_gate.go.

type systemFeaturePolicy struct {
	EnablePhoneLogin                 bool     `json:"enable_phone_login"`
	EnableRechargePhoneVerification  bool     `json:"enable_recharge_phone_verification"`
	EnableEmailRegister              bool     `json:"enable_email_register"`
	EnableWechatLogin                bool     `json:"enable_wechat_login"`
	AllowedPhoneCountryCodes         []string `json:"allowed_phone_country_codes"`
}

type featurePolicyRow struct {
	EnablePhoneLogin                bool
	EnableRechargePhoneVerification bool
	EnableEmailRegister             bool
	EnableWechatLogin               bool
	AllowedPhoneCountryCodesJSON    string
}

// featurePolicyQueryHook 测试接缝（与 rbac_pdp 的 membershipRevFn 同模式）：
// 非 nil 时每次 loadFeaturePolicy 查询前调用，用于断言 TTL 缓存命中的查询次数。
var featurePolicyQueryHook func()

func loadFeaturePolicy() (*featurePolicyRow, error) {
	if featurePolicyQueryHook != nil {
		featurePolicyQueryHook()
	}
	var r featurePolicyRow
	err := db.QueryRow(`SELECT
		COALESCE(enable_phone_login, 1),
		COALESCE(enable_recharge_phone_verification, 0),
		COALESCE(enable_email_register, 0),
		COALESCE(enable_wechat_login, 1),
		COALESCE(allowed_phone_country_codes, '["+86"]')
		FROM auth_system_feature_policy WHERE id = 1`).Scan(
		&r.EnablePhoneLogin,
		&r.EnableRechargePhoneVerification,
		&r.EnableEmailRegister,
		&r.EnableWechatLogin,
		&r.AllowedPhoneCountryCodesJSON,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func saveFeaturePolicy(p *systemFeaturePolicy) error {
	codesJSON, err := json.Marshal(p.AllowedPhoneCountryCodes)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE auth_system_feature_policy SET
		enable_phone_login = ?,
		enable_recharge_phone_verification = ?,
		enable_email_register = ?,
		enable_wechat_login = ?,
		allowed_phone_country_codes = ?
		WHERE id = 1`,
		p.EnablePhoneLogin,
		p.EnableRechargePhoneVerification,
		p.EnableEmailRegister,
		p.EnableWechatLogin,
		string(codesJSON),
	)
	if err == nil {
		// OPT-20260806-028: 保存成功即失效 wechatLoginPolicyEnabled 的 TTL 缓存，
		// 保证管理员开关修改立即生效（无需等 30s 缓存过期）。
		invalidateWechatPolicyCache()
	}
	return err
}

func featurePolicyToJSON(row *featurePolicyRow) map[string]interface{} {
	var codes []string
	if row.AllowedPhoneCountryCodesJSON != "" && row.AllowedPhoneCountryCodesJSON != "[]" {
		if err := json.Unmarshal([]byte(row.AllowedPhoneCountryCodesJSON), &codes); err != nil {
			codes = []string{}
		}
	} else {
		codes = []string{}
	}
	return map[string]interface{}{
		"enable_phone_login":                 row.EnablePhoneLogin,
		"enable_recharge_phone_verification":  row.EnableRechargePhoneVerification,
		"enable_email_register":              row.EnableEmailRegister,
		"enable_wechat_login":                row.EnableWechatLogin,
		"allowed_phone_country_codes":         codes,
	}
}

// ---- public feature policy endpoint ---- //

func handlePublicSystemFeaturePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	row, err := loadFeaturePolicy()
	if err != nil {
		log.Printf("[taskAuth] auth_system_feature_policy load: %v", err)
		// Return safe defaults on DB error（错误路径保持 fail-closed：微信登录按关闭处理，
		// 与产品默认「开启」无关 — DB 故障时保守拒绝扫码入口，避免登录链路半可用）
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enable_phone_login":     true,
			"enable_email_register":  true,
			"enable_wechat_login":    false,
			"wechat_login_available": false,
			"phone_country_options":  []interface{}{},
		})
		return
	}
	payload := featurePolicyToJSON(row)
	// phone_country_options: derived from allowed_phone_country_codes.
	// 非空白名单 → 只返回白名单内的区号选项（登录页下拉只显示允许的区域）；
	// 空数组 = 允许所有 (frontend falls back to full COUNTRY_DIAL_OPTIONS)。
	codes, _ := payload["allowed_phone_country_codes"].([]string)
	payload["phone_country_options"] = derivePhoneCountryOptions(codes)
	// wechat_login_available: 策略开启且微信应用已配置时才可用（前端据此展示扫码入口）。
	payload["wechat_login_available"] = row.EnableWechatLogin && wechatEnabled()
	writeJSON(w, http.StatusOK, payload)
}

// ---- admin feature policy CRUD ---- //

func handleAdminSystemFeaturePolicy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleAdminGetFeaturePolicy(w, r)
	case http.MethodPost:
		handleAdminSaveFeaturePolicy(w, r)
	default:
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleAdminGetFeaturePolicy(w http.ResponseWriter, r *http.Request) {
	row, err := loadFeaturePolicy()
	if err != nil {
		log.Printf("[taskAuth] auth_system_feature_policy load: %v", err)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enable_phone_login":                 true,
			"enable_recharge_phone_verification":  false,
			"enable_email_register":              true,
			"enable_wechat_login":                false,
			"allowed_phone_country_codes":         []string{},
		})
		return
	}
	payload := featurePolicyToJSON(row)
	// OPT-20260806-029: 管理后台在开关开启但微信应用未配置时提示管理员
	// （公共端点 wechat_login_available 无法区分「开关未开」与「应用未配置」）。
	payload["wechat_app_configured"] = wechatEnabled()
	writeJSON(w, http.StatusOK, payload)
}

func handleAdminSaveFeaturePolicy(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	policy := systemFeaturePolicy{
		EnablePhoneLogin:                boolField(body, "enable_phone_login"),
		EnableRechargePhoneVerification: boolField(body, "enable_recharge_phone_verification"),
		EnableEmailRegister:             boolField(body, "enable_email_register"),
		EnableWechatLogin:               boolField(body, "enable_wechat_login"),
		AllowedPhoneCountryCodes:        stringSliceField(body, "allowed_phone_country_codes"),
	}
	if err := saveFeaturePolicy(&policy); err != nil {
		log.Printf("[taskAuth] auth_system_feature_policy save: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "save failed")
		return
	}
	// Reload and return the saved policy
	row, err := loadFeaturePolicy()
	if err != nil {
		writeErrorDetail(w, r, http.StatusOK, "saved")
		return
	}
	payload := featurePolicyToJSON(row)
	// OPT-20260806-029: 与 GET 一致，返回应用配置状态供管理页提示
	payload["wechat_app_configured"] = wechatEnabled()
	writeJSON(w, http.StatusOK, payload)
}

func boolField(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func stringSliceField(m map[string]interface{}, key string) []string {
	v, ok := m[key]
	if !ok {
		return []string{}
	}
	arr, ok := v.([]interface{})
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if ok {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}
