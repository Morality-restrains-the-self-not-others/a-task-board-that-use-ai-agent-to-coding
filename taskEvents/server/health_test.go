package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskEvents/broker"
	"taskEvents/domain"
	"taskEvents/server"
)

func TestReadinessHandlerDegradedWhenDisconnected(t *testing.T) {
	conn := broker.NewConnectionState(false)
	handler := server.NewReadinessHandler("test", domain.TransportRedis, []string{"USER_CREATED"}, conn)

	req := httptest.NewRequest(http.MethodGet, "/api/health/ready", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "degraded" {
		t.Fatalf("status field = %v", body["status"])
	}
}

func TestReadinessHandlerOkWhenConnected(t *testing.T) {
	conn := broker.NewConnectionState(true)
	handler := server.NewReadinessHandler("test", domain.TransportRedis, []string{"USER_CREATED"}, conn)

	req := httptest.NewRequest(http.MethodGet, "/api/health/ready", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestLivenessHandlerAlwaysOk(t *testing.T) {
	handler := server.NewHealthHandler("test", domain.TransportRedis, []string{"USER_CREATED"})
	req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
