package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"tracelog"
)

const (
	reasonKycTierBlocked        = "KYC_TIER_BLOCKED"
	reasonKycAmountExceeds      = "KYC_AMOUNT_EXCEEDS_SINGLE"
	reasonKycDailyExceeds       = "KYC_AMOUNT_EXCEEDS_DAILY"
	reasonKycAMLReview          = "KYC_AML_REVIEW"
	reasonKycServiceUnavailable = "KYC_SERVICE_UNAVAILABLE"
	reasonKycOK                 = "OK"

	// Align with taskAuth T2 single-cap; per-tier enforcement is via Auth + daily sum.
	kycPaymentAmountMaxYuan = 50000
)

var (
	taskAuthHTTP = tracelog.DirectClient(10 * time.Second)

	kycGateMockMu sync.Mutex
	kycGateMockFn func(ctx context.Context, userID string, amountYuan int64) (kycGateAuthResult, error)
)

type kycGateAuthResult struct {
	Allowed           bool   `json:"allowed"`
	ReasonCode        string `json:"reason_code"`
	Tier              string `json:"tier"`
	Status            string `json:"status"`
	MaxSingleYuan     int64  `json:"max_single_yuan"`
	MaxDailyYuan      int64  `json:"max_daily_yuan"`
	DailyUsedYuanHint int64  `json:"daily_used_yuan_hint"`
}

type kycGateError struct {
	Message    string
	ReasonCode string
	Tier       string
}

func (e *kycGateError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.ReasonCode
}

func kycGateDenied(reasonCode, tier, message string) *kycGateError {
	if message == "" {
		message = kycGateUserMessage(reasonCode)
	}
	return &kycGateError{Message: message, ReasonCode: reasonCode, Tier: tier}
}

func kycGateUserMessage(reasonCode string) string {
	switch reasonCode {
	case reasonKycTierBlocked:
		return "当前身份等级不允许充值，请联系管理员升级身份等级"
	case reasonKycAmountExceeds:
		return "超过单笔支付限额"
	case reasonKycDailyExceeds:
		return "超过当日累计支付限额"
	case reasonKycAMLReview:
		return "账户充值暂不可用，请联系客服"
	case reasonKycServiceUnavailable:
		return "支付校验服务暂不可用，请稍后重试"
	default:
		return "支付校验未通过"
	}
}

func setKycGateMock(fn func(ctx context.Context, userID string, amountYuan int64) (kycGateAuthResult, error)) {
	kycGateMockMu.Lock()
	defer kycGateMockMu.Unlock()
	kycGateMockFn = fn
}

func clearKycGateMock() {
	setKycGateMock(nil)
}

func writeKycGateDenied(w http.ResponseWriter, err error) {
	ke, ok := err.(*kycGateError)
	if !ok || ke == nil {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"error":       kycGateUserMessage(reasonKycServiceUnavailable),
			"reason_code": reasonKycServiceUnavailable,
			"tier":        "",
		})
		return
	}
	writeJSON(w, http.StatusForbidden, map[string]interface{}{
		"error":       ke.Message,
		"reason_code": ke.ReasonCode,
		"tier":        ke.Tier,
	})
}

// checkKycPaymentGate asks taskAuth for tier/AML/single limits, then enforces
// Bill-side daily recharge sum against max_daily_yuan.
func checkKycPaymentGate(ctx context.Context, userID string, amountYuan int64) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return kycGateDenied(reasonKycTierBlocked, "", "未登录")
	}
	if amountYuan < 1 {
		return kycGateDenied(reasonKycAmountExceeds, "", "请输入有效的支付金额")
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.KycGateMode))
	if mode == "" {
		mode = "live"
	}
	if mode == "off" {
		return nil
	}

	// OPT-20260726-015: Second layer — query Django auth_system_feature_policy.
	// If enable_recharge_phone_verification is disabled, skip KYC gate
	// (tenant-level grayscale control for the entire recharge gate system).
	if enabled, err := fetchSmsFeaturePolicyCached(); err == nil && enabled != nil && !enabled.EnableRechargePhoneVerification {
		return nil
	}

	gate, err := fetchKycPaymentGate(ctx, mode, userID, amountYuan)
	if err != nil {
		return err
	}
	if !gate.Allowed {
		code := strings.TrimSpace(gate.ReasonCode)
		if code == "" || code == reasonKycOK {
			code = reasonKycTierBlocked
		}
		return kycGateDenied(code, gate.Tier, "")
	}

	if gate.MaxDailyYuan > 0 {
		usedYuan, err := todayUserRechargeYuan(userID)
		if err != nil {
			return kycGateDenied(reasonKycServiceUnavailable, gate.Tier, "")
		}
		if usedYuan+amountYuan > gate.MaxDailyYuan {
			return kycGateDenied(reasonKycDailyExceeds, gate.Tier, "")
		}
	}
	return nil
}

func fetchKycPaymentGate(ctx context.Context, mode, userID string, amountYuan int64) (kycGateAuthResult, error) {
	if mode == "mock" {
		kycGateMockMu.Lock()
		fn := kycGateMockFn
		kycGateMockMu.Unlock()
		if fn != nil {
			return fn(ctx, userID, amountYuan)
		}
		// Default mock: allow with T1 policy so existing create-route tests keep working.
		return kycGateAuthResult{
			Allowed:       true,
			ReasonCode:    reasonKycOK,
			Tier:          "T1_basic",
			Status:        "approved",
			MaxSingleYuan: 4999,
			MaxDailyYuan:  20000,
		}, nil
	}

	base := strings.TrimRight(strings.TrimSpace(cfg.TaskAuthBaseURL), "/")
	if base == "" {
		return kycGateAuthResult{}, kycGateDenied(reasonKycServiceUnavailable, "", "")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	q := url.Values{}
	q.Set("user_id", userID)
	q.Set("amount_yuan", fmt.Sprintf("%d", amountYuan))
	reqURL := base + "/api/internal/kyc/recharge-gate/?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return kycGateAuthResult{}, kycGateDenied(reasonKycServiceUnavailable, "", "")
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	if secret := strings.TrimSpace(cfg.TaskAuthInternalSecret); secret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", secret)
	}
	resp, err := taskAuthHTTP.Do(req)
	if err != nil {
		return kycGateAuthResult{}, kycGateDenied(reasonKycServiceUnavailable, "", "")
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return kycGateAuthResult{}, kycGateDenied(reasonKycServiceUnavailable, "", "")
	}
	var gate kycGateAuthResult
	if err := json.Unmarshal(raw, &gate); err != nil {
		return kycGateAuthResult{}, kycGateDenied(reasonKycServiceUnavailable, "", "")
	}
	return gate, nil
}

// todayUserRechargeYuan sums today's user-facing recharge points for userID and
// converts to yuan (points/100). Admin grants are excluded from the daily cap.
func todayUserRechargeYuan(userID string) (int64, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return 0, nil
	}
	var points sql.NullInt64
	err := db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) FROM billing_transaction
		WHERE user_id = ?
		  AND transaction_type = 'recharge'
		  AND date(created_at) = date('now')
		  AND points_source_type IN (
			'user_recharge', 'user_recharge_paypal', 'user_recharge_wechat'
		  )`, userID).Scan(&points)
	if err != nil {
		return 0, err
	}
	p := int64(0)
	if points.Valid {
		p = points.Int64
	}
	return p / pointsPerYuan, nil
}
