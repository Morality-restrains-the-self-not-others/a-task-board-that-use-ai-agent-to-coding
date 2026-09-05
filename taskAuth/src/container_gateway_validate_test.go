package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContainerGatewayValidateSession_TokenOK(t *testing.T) {
	userID, tokenKey := setupForwardAuthTest(t)
	body, _ := json.Marshal(map[string]string{
		"authorization": "Token " + tokenKey,
		"tenant_id":     "t1",
		"task_id":       "task1",
		"path":          "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/container-layer-command/",
		"method":        "POST",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/internal/container-gateway/validate-session/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleContainerGatewayValidateSession(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["user_id"] != userID {
		t.Fatalf("user_id=%v want %s", out["user_id"], userID)
	}
	if out["auth_method"] != "token" {
		t.Fatalf("auth_method=%v", out["auth_method"])
	}
	if out["scope_ok"] != true {
		t.Fatalf("scope_ok=%v", out["scope_ok"])
	}
}

func TestContainerGatewayValidateSession_Unauthorized(t *testing.T) {
	setupForwardAuthTest(t)
	body, _ := json.Marshal(map[string]string{"authorization": "Token bad-token"})
	req := httptest.NewRequest(http.MethodPost, "/api/internal/container-gateway/validate-session/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleContainerGatewayValidateSession(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestContainerGatewayValidateSession_ForbiddenSecret(t *testing.T) {
	setupForwardAuthTest(t)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/container-gateway/validate-session/", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAuth-Internal-Secret", "wrong")
	rec := httptest.NewRecorder()
	handleContainerGatewayValidateSession(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rec.Code)
	}
}
