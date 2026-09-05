package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"confload"
)

const paypalPointsSourceType = "user_recharge_paypal"

type paypalConfig struct {
	ClientID         string `yaml:"client_id"`
	ClientSecret     string `yaml:"client_secret"`
	Mode             string `yaml:"mode"`
	Currency         string `yaml:"currency"`
	WebhookIDSandbox string `yaml:"webhook_id_sandbox"`
	WebhookIDLive    string `yaml:"webhook_id_live"`
	FrontendBaseURL  string `yaml:"frontend_base_url"`
}

type paypalPendingOrder struct {
	OrderID     string
	TenantID    int64
	UserID      string
	AmountYuan  string // decimal string e.g. "10.00"
	Currency    string
	Description string
	CreatedAt   time.Time
}

var (
	paypalCfg     paypalConfig
	paypalMu      sync.Mutex
	paypalPending = map[string]paypalPendingOrder{}
	paypalHTTP    = &http.Client{Timeout: 30 * time.Second}
	paypalTokenMu sync.Mutex
	paypalToken   string
	paypalTokenAt time.Time
)

func paypalConfigured() bool {
	return strings.TrimSpace(paypalCfg.ClientID) != "" && strings.TrimSpace(paypalCfg.ClientSecret) != ""
}

func loadPaypalConfig(repoRoot string) {
	// OPT-20260806-057: django config fallback 删除（Django 退役）；
	// conf/billing/paypal/config.yaml 为唯一 SSOT（含 client/secret/mode/currency/webhook_id×2）。
	// ADR-0054：必须叠 conf-local，否则 tracked 空 client_secret 会让充值 503。
	if err := confload.UnmarshalYAMLMerged(repoRoot, "billing/paypal/config.yaml", &paypalCfg); err != nil {
		log.Printf("[taskBill] paypal config missing: %v", err)
	}
	if v := strings.TrimSpace(os.Getenv("PAYPAL_CLIENT_ID")); v != "" {
		paypalCfg.ClientID = v
	}
	if v := strings.TrimSpace(os.Getenv("PAYPAL_CLIENT_SECRET")); v != "" {
		paypalCfg.ClientSecret = v
	}
	if v := strings.TrimSpace(os.Getenv("PAYPAL_MODE")); v != "" {
		paypalCfg.Mode = v
	}
	if v := strings.TrimSpace(os.Getenv("PAYPAL_CURRENCY")); v != "" {
		paypalCfg.Currency = v
	}
	if v := strings.TrimSpace(os.Getenv("FRONTEND_BASE_URL")); v != "" {
		paypalCfg.FrontendBaseURL = v
	}
	paypalCfg.Mode = strings.ToLower(strings.TrimSpace(paypalCfg.Mode))
	if paypalCfg.Mode != "live" {
		paypalCfg.Mode = "sandbox"
	}
	paypalCfg.Currency = strings.ToUpper(strings.TrimSpace(paypalCfg.Currency))
	if paypalCfg.Currency == "" {
		paypalCfg.Currency = "USD"
	}
	if paypalCfg.FrontendBaseURL == "" {
		var vc struct {
			PublicBaseURL string `yaml:"publicBaseUrl"`
		}
		if err := confload.UnmarshalYAMLMerged(repoRoot, "frontend/vue/config.yaml", &vc); err == nil {
			paypalCfg.FrontendBaseURL = strings.TrimSpace(vc.PublicBaseURL)
		}
	}
	if paypalConfigured() {
		log.Printf("[taskBill] PayPal configured mode=%s currency=%s", paypalCfg.Mode, paypalCfg.Currency)
	} else {
		log.Printf("[taskBill] PayPal not configured — recharge_paypal_* will return 503")
	}
}

func paypalAPIBase(mode string) string {
	if strings.ToLower(mode) == "live" {
		return "https://api-m.paypal.com"
	}
	return "https://api-m.sandbox.paypal.com"
}

func paypalAccessToken(ctx context.Context, mode string) (string, error) {
	if !paypalConfigured() {
		return "", fmt.Errorf("paypal not configured")
	}
	paypalTokenMu.Lock()
	defer paypalTokenMu.Unlock()
	if paypalToken != "" && time.Since(paypalTokenAt) < 50*time.Minute && mode == paypalCfg.Mode {
		return paypalToken, nil
	}
	base := paypalAPIBase(mode)
	pair := base64.StdEncoding.EncodeToString([]byte(paypalCfg.ClientID + ":" + paypalCfg.ClientSecret))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/oauth2/token", strings.NewReader("grant_type=client_credentials"))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+pair)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := paypalHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("paypal oauth %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var data map[string]interface{}
	_ = json.Unmarshal(raw, &data)
	tok, _ := data["access_token"].(string)
	if tok == "" {
		return "", fmt.Errorf("paypal oauth missing access_token")
	}
	if mode == paypalCfg.Mode {
		paypalToken = tok
		paypalTokenAt = time.Now()
	}
	return tok, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func storePaypalPending(o paypalPendingOrder) {
	paypalMu.Lock()
	defer paypalMu.Unlock()
	paypalPending[o.OrderID] = o
}

func getPaypalPending(orderID string) (paypalPendingOrder, bool) {
	paypalMu.Lock()
	defer paypalMu.Unlock()
	o, ok := paypalPending[orderID]
	return o, ok
}

func deletePaypalPending(orderID string) {
	paypalMu.Lock()
	defer paypalMu.Unlock()
	delete(paypalPending, orderID)
}

func paypalTxnID(orderID string) string {
	id := "paypal:" + orderID
	if len(id) > 100 {
		return id[:100]
	}
	return id
}

func paypalCreateOrder(ctx context.Context, amountCents int64, currency, returnURL, cancelURL, customID, description string) (orderID, approvalURL string, err error) {
	token, err := paypalAccessToken(ctx, paypalCfg.Mode)
	if err != nil {
		return "", "", err
	}
	// OPT-20260819-003: 金额一律按分入参并转十进制字符串，避免不足 1 元时被
	// fmt.Sprintf("%d.00", ...) 取整成 "0.00" 或调用方先取整。
	value := centsToYuanStr(amountCents)
	payload := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"amount": map[string]string{
					"currency_code": currency,
					"value":         value,
				},
				"description": truncate(description, 127),
				"custom_id":   truncate(customID, 127),
			},
		},
		"application_context": map[string]string{
			"return_url":  returnURL,
			"cancel_url":  cancelURL,
			"user_action": "PAY_NOW",
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, paypalAPIBase(paypalCfg.Mode)+"/v2/checkout/orders", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	resp, err := paypalHTTP.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("paypal create order %d: %s", resp.StatusCode, truncate(string(raw), 400))
	}
	var data map[string]interface{}
	_ = json.Unmarshal(raw, &data)
	orderID, _ = data["id"].(string)
	if orderID == "" {
		return "", "", fmt.Errorf("paypal missing order id")
	}
	if links, ok := data["links"].([]interface{}); ok {
		for _, l := range links {
			m, _ := l.(map[string]interface{})
			if fmt.Sprint(m["rel"]) == "approve" {
				approvalURL = fmt.Sprint(m["href"])
				break
			}
		}
	}
	if approvalURL == "" {
		return "", "", fmt.Errorf("paypal missing approve url")
	}
	return orderID, approvalURL, nil
}

func paypalCaptureOrder(ctx context.Context, orderID string) (map[string]interface{}, int, error) {
	token, err := paypalAccessToken(ctx, paypalCfg.Mode)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, paypalAPIBase(paypalCfg.Mode)+"/v2/checkout/orders/"+orderID+"/capture", bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	resp, err := paypalHTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	_ = json.Unmarshal(raw, &data)
	return data, resp.StatusCode, nil
}

func paypalGetOrder(ctx context.Context, orderID string) (map[string]interface{}, int, error) {
	token, err := paypalAccessToken(ctx, paypalCfg.Mode)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, paypalAPIBase(paypalCfg.Mode)+"/v2/checkout/orders/"+orderID, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := paypalHTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	_ = json.Unmarshal(raw, &data)
	return data, resp.StatusCode, nil
}

func extractPaypalPaidAmount(data map[string]interface{}) (currency, value string) {
	units, _ := data["purchase_units"].([]interface{})
	for _, u := range units {
		um, _ := u.(map[string]interface{})
		payments, _ := um["payments"].(map[string]interface{})
		caps, _ := payments["captures"].([]interface{})
		for _, c := range caps {
			cm, _ := c.(map[string]interface{})
			amt, _ := cm["amount"].(map[string]interface{})
			if amt["value"] != nil {
				return fmt.Sprint(amt["currency_code"]), fmt.Sprint(amt["value"])
			}
		}
	}
	for _, u := range units {
		um, _ := u.(map[string]interface{})
		amt, _ := um["amount"].(map[string]interface{})
		if amt["value"] != nil {
			return fmt.Sprint(amt["currency_code"]), fmt.Sprint(amt["value"])
		}
	}
	return "", ""
}

var customIDRe = regexp.MustCompile(`^tenant:([^:]+):user:(.+)$`)

func parsePaypalCustomID(customID string) (tenantID, userID string, ok bool) {
	m := customIDRe.FindStringSubmatch(strings.TrimSpace(customID))
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

func verifyPaypalWebhook(ctx context.Context, apiMode, webhookID string, headers http.Header, body []byte) bool {
	if strings.TrimSpace(webhookID) == "" {
		return false
	}
	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		return false
	}
	transmissionID := headers.Get("Paypal-Transmission-Id")
	transmissionTime := headers.Get("Paypal-Transmission-Time")
	transmissionSig := headers.Get("Paypal-Transmission-Sig")
	certURL := headers.Get("Paypal-Cert-Url")
	authAlgo := headers.Get("Paypal-Auth-Algo")
	if authAlgo == "" {
		authAlgo = "SHA256withRSA"
	}
	if transmissionID == "" || transmissionTime == "" || transmissionSig == "" || certURL == "" {
		return false
	}
	token, err := paypalAccessToken(ctx, apiMode)
	if err != nil {
		return false
	}
	payload := map[string]interface{}{
		"transmission_id":   transmissionID,
		"transmission_time": transmissionTime,
		"cert_url":          certURL,
		"auth_algo":         authAlgo,
		"transmission_sig":  transmissionSig,
		"webhook_id":        webhookID,
		"webhook_event":     event,
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, paypalAPIBase(apiMode)+"/v1/notifications/verify-webhook-signature", bytes.NewReader(raw))
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := paypalHTTP.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	respRaw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return false
	}
	var out map[string]interface{}
	_ = json.Unmarshal(respRaw, &out)
	return fmt.Sprint(out["verification_status"]) == "SUCCESS"
}

// handleRechargePaypalCreate 处理 PayPal 支付创建（余额充值路径）。
// 前端已无独立充值页面，但 PayPal 回调、管理员授权等仍通过本路径为账户充值余额。
// 用户下单购买资源走 handlers_orders.go handlePayOrder 路径。

var paypalCaptureLocks sync.Map

func handlePaypalWebhook(w http.ResponseWriter, r *http.Request, apiMode string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !paypalConfigured() {
		http.Error(w, "PayPal 未配置", http.StatusServiceUnavailable)
		return
	}
	apiMode = strings.ToLower(apiMode)
	if apiMode != "sandbox" && apiMode != "live" {
		http.Error(w, "bad path", http.StatusNotFound)
		return
	}
	wid := paypalCfg.WebhookIDSandbox
	if apiMode == "live" {
		wid = paypalCfg.WebhookIDLive
	}
	if strings.TrimSpace(wid) == "" {
		log.Printf("[taskBill] paypal webhook id missing for mode=%s", apiMode)
		http.Error(w, "webhook 未配置", http.StatusServiceUnavailable)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	if !verifyPaypalWebhook(r.Context(), apiMode, wid, r.Header, body) {
		log.Printf("[taskBill] paypal webhook signature invalid mode=%s", apiMode)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	eventType := fmt.Sprint(event["event_type"])
	// 余额充值路径已移除（2026-07-26）；PayPal webhook 仅记录日志
	log.Printf("[taskBill] paypal webhook event_type=%s (balance recharge removed)", eventType)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
