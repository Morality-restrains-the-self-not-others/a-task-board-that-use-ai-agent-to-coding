package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestValidateProxyTokenScopeMatch(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/token/validate" {
			http.NotFound(w, r)
			return
		}
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"valid": true, "company_id": "t1", "workspace_id": "w1", "task_id": "k1",
		})
	}))
	t.Cleanup(srv.Close)
	prev := cfg.CredentialServiceURL
	cfg.CredentialServiceURL = srv.URL
	t.Cleanup(func() { cfg.CredentialServiceURL = prev })

	ok, detail := validateProxyToken(nil, "t1", "w1", "k1", "tok")
	if !ok || detail != "" {
		t.Fatalf("ok=%v detail=%q", ok, detail)
	}
	ok, detail = validateProxyToken(nil, "t1", "w1", "other", "tok")
	if ok || detail != "token scope mismatch" {
		t.Fatalf("mismatch ok=%v detail=%q", ok, detail)
	}
	if hits.Load() != 2 {
		t.Fatalf("hits=%d", hits.Load())
	}
}

func TestCloudResolveRouteAndUpstream(t *testing.T) {
	var routeHits, credHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/internal/ai-endpoint/resolve-route/":
			routeHits.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"provider": "deepseek", "upstream_base_url": "https://api.deepseek.com/v1",
				"use_sub_token": true, "budget_enabled": true,
			})
		case "/api/internal/ai-endpoint/upstream-credentials/":
			credHits.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"provider": "deepseek", "base_url": "https://api.deepseek.com/v1",
				"api_key": "sk", "derive_needed": true,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	prev := cfg.CloudServiceURL
	cfg.CloudServiceURL = srv.URL
	t.Cleanup(func() { cfg.CloudServiceURL = prev })

	route, status, _ := cloudResolveRoute(nil, "t", "w", "k", "deepseek")
	if status != http.StatusOK || route == nil || route.UpstreamBase == "" {
		t.Fatalf("route status=%d %+v", status, route)
	}
	cred, cStatus, _ := cloudUpstreamCredential(nil, "t", "deepseek", route.UpstreamBase)
	if cStatus != http.StatusOK || cred == nil || cred.APIKey != "sk" {
		t.Fatalf("cred status=%d %+v", cStatus, cred)
	}
	if routeHits.Load() != 1 || credHits.Load() != 1 {
		t.Fatalf("hits route=%d cred=%d", routeHits.Load(), credHits.Load())
	}
}

func TestCloudBudgetGateAndCommit(t *testing.T) {
	var reserveHits, commitHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/internal/budget/reserve-or-deny/":
			reserveHits.Add(1)
			if got := r.Header.Get("X-Internal-Secret"); got != "cloud-sec" {
				t.Errorf("secret=%q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"allowed": false, "code": "budget_exhausted", "message": "exhausted",
			})
		case "/api/internal/budget/record-usage/":
			commitHits.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true,"usage_id":"1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	prevURL := cfg.CloudServiceURL
	prevSec := cfg.CloudInternalSecret
	cfg.CloudServiceURL = srv.URL
	cfg.CloudInternalSecret = "cloud-sec"
	t.Cleanup(func() {
		cfg.CloudServiceURL = prevURL
		cfg.CloudInternalSecret = prevSec
	})

	dec, status, _ := cloudBudgetGate(nil, "t", "w", "task", "openai", "https://api.openai.com/v1", "gpt")
	if status != http.StatusPaymentRequired {
		t.Fatalf("status=%d", status)
	}
	if dec == nil || dec.Code != "budget_exhausted" {
		t.Fatalf("dec=%+v", dec)
	}
	cloudCommitUsage(nil, commitUsageRequest{
		TenantID: "t", WorkspaceID: "w", TaskID: "task",
		Provider: "openai", BaseURL: "https://api.openai.com/v1", ModelName: "gpt",
		InputTokens: 1, OutputTokens: 2, IdempotencyKey: "k1", RequestID: "k1",
	})
	if reserveHits.Load() != 1 || commitHits.Load() != 1 {
		t.Fatalf("hits reserve=%d commit=%d", reserveHits.Load(), commitHits.Load())
	}
}
