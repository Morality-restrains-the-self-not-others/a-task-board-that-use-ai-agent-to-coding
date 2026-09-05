package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// roundTripFunc 允许在单测中拦截任意主机请求（paypalAPIBase 固定指向 api-m.*.paypal.com）。
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResp(code int, body interface{}) *http.Response {
	raw, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: code,
		Status:     http.StatusText(code),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(raw))),
	}
}

// OPT-20260819-003: 订单金额按分入参，55 分 → "0.55"，不得再被取整成 "0.00"。
func TestPaypalCreateOrderValueCents(t *testing.T) {
	var mu sync.Mutex
	var lastOrderBody string

	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/v1/oauth2/token") {
			return jsonResp(200, map[string]interface{}{"access_token": "test-token"}), nil
		}
		if strings.HasSuffix(r.URL.Path, "/v2/checkout/orders") {
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			lastOrderBody = string(body)
			mu.Unlock()
			return jsonResp(200, map[string]interface{}{
				"id": "PAY-123",
				"links": []map[string]interface{}{
					{"rel": "approve", "href": "https://example.com/approve"},
				},
			}), nil
		}
		return jsonResp(404, map[string]interface{}{"message": "unexpected path"}), nil
	})

	origHTTP, origCfg := paypalHTTP, paypalCfg
	paypalHTTP = &http.Client{Transport: rt}
	paypalCfg = paypalConfig{ClientID: "cid", ClientSecret: "csecret", Mode: "sandbox"}
	defer func() { paypalHTTP, paypalCfg = origHTTP, origCfg }()

	paypalTokenMu.Lock()
	paypalToken, paypalTokenAt = "", time.Time{}
	paypalTokenMu.Unlock()

	orderID, approvalURL, err := paypalCreateOrder(context.Background(), 55, "USD",
		"https://return.example", "https://cancel.example", "tenant:t1:user:u1", "desc")
	if err != nil {
		t.Fatalf("paypalCreateOrder: %v", err)
	}
	if orderID != "PAY-123" || approvalURL != "https://example.com/approve" {
		t.Fatalf("orderID=%q approvalURL=%q", orderID, approvalURL)
	}

	mu.Lock()
	defer mu.Unlock()
	var payload struct {
		PurchaseUnits []struct {
			Amount struct {
				Value string `json:"value"`
			} `json:"amount"`
		} `json:"purchase_units"`
	}
	if err := json.Unmarshal([]byte(lastOrderBody), &payload); err != nil {
		t.Fatalf("unmarshal order body: %v", err)
	}
	if len(payload.PurchaseUnits) == 0 {
		t.Fatalf("order body missing purchase_units: %s", lastOrderBody)
	}
	if got := payload.PurchaseUnits[0].Amount.Value; got != "0.55" {
		t.Fatalf("order amount value = %q, want 0.55 (body=%s)", got, lastOrderBody)
	}
}

func TestLoadPaypalConfigOverlaysConfLocal(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "conf", "billing", "paypal")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "conf", "base.yaml"), []byte("scheme: https\nbaseDomain: example.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "config.yaml"), []byte("client_id: pub-id\nclient_secret: \"\"\nmode: sandbox\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	locDir := filepath.Join(root, "conf-local", "billing", "paypal")
	if err := os.MkdirAll(locDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locDir, "config.yaml"), []byte("client_secret: from-conf-local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prev := paypalCfg
	t.Cleanup(func() { paypalCfg = prev })
	t.Setenv("PAYPAL_CLIENT_ID", "")
	t.Setenv("PAYPAL_CLIENT_SECRET", "")
	loadPaypalConfig(root)
	if paypalCfg.ClientID != "pub-id" {
		t.Fatalf("client_id=%q", paypalCfg.ClientID)
	}
	if paypalCfg.ClientSecret != "from-conf-local" {
		t.Fatalf("client_secret=%q, want conf-local overlay", paypalCfg.ClientSecret)
	}
}
