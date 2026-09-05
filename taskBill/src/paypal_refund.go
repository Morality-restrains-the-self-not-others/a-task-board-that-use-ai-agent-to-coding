package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

func extractPaypalCaptureID(orderData map[string]interface{}) string {
	units, _ := orderData["purchase_units"].([]interface{})
	for _, u := range units {
		um, _ := u.(map[string]interface{})
		payments, _ := um["payments"].(map[string]interface{})
		caps, _ := payments["captures"].([]interface{})
		for _, c := range caps {
			cm, _ := c.(map[string]interface{})
			id := strings.TrimSpace(fmt.Sprint(cm["id"]))
			if id != "" && id != "<nil>" {
				status := strings.ToUpper(strings.TrimSpace(fmt.Sprint(cm["status"])))
				if status == "" || status == "COMPLETED" || status == "PENDING" {
					return id
				}
			}
		}
	}
	return ""
}

func paypalResolveCaptureID(ctx context.Context, orderID, storedCaptureID string) (string, error) {
	if id := strings.TrimSpace(storedCaptureID); id != "" {
		return id, nil
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return "", fmt.Errorf("paypal order id empty")
	}
	data, status, err := paypalGetOrder(ctx, orderID)
	if err != nil {
		return "", err
	}
	if status >= 400 {
		return "", fmt.Errorf("paypal get order status=%d", status)
	}
	id := extractPaypalCaptureID(data)
	if id == "" {
		return "", fmt.Errorf("paypal capture id not found for order %s", orderID)
	}
	return id, nil
}

func paypalRefundCapture(ctx context.Context, captureID string, amountMinor int64, currency, idempotencyKey string) (refundID string, err error) {
	captureID = strings.TrimSpace(captureID)
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if captureID == "" || amountMinor <= 0 || currency == "" {
		return "", fmt.Errorf("invalid paypal refund params")
	}
	token, err := paypalAccessToken(ctx, paypalCfg.Mode)
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(map[string]interface{}{
		"amount": map[string]string{
			"value":         formatMoneyMinor(amountMinor),
			"currency_code": currency,
		},
	})
	url := paypalAPIBase(paypalCfg.Mode) + "/v2/payments/captures/" + captureID + "/refund"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	if key := strings.TrimSpace(idempotencyKey); key != "" {
		req.Header.Set("PayPal-Request-Id", key)
	}
	resp, err := paypalHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	_ = json.Unmarshal(raw, &data)
	if resp.StatusCode >= 400 {
		slog.WarnContext(ctx, "paypal_refund_http_error",
			"level", "warn",
			"status", resp.StatusCode,
			"capture_id", captureID,
			"body", truncateForLog(string(raw), 400),
		)
		return "", fmt.Errorf("paypal refund status=%d", resp.StatusCode)
	}
	id := strings.TrimSpace(fmt.Sprint(data["id"]))
	if id == "" || id == "<nil>" {
		return "", fmt.Errorf("paypal refund missing id")
	}
	return id, nil
}

func truncateForLog(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
