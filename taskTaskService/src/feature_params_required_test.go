package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateFeatureParamsSourceRequired(t *testing.T) {
	if err := validateFeatureParamsSourceRequired("", ""); err == nil {
		t.Fatal("expected error for empty source")
	}
	if err := validateFeatureParamsSourceRequired("none", ""); err == nil {
		t.Fatal("expected error for none")
	}
	if err := validateFeatureParamsSourceRequired("company", ""); err != nil {
		t.Fatalf("company should pass: %v", err)
	}
	if err := validateFeatureParamsSourceRequired("workspace", ""); err != nil {
		t.Fatalf("workspace should pass: %v", err)
	}
	if err := validateFeatureParamsSourceRequired("personal", ""); err == nil {
		t.Fatal("personal without config should fail")
	}
	if err := validateFeatureParamsSourceRequired("personal", "cfg-1"); err != nil {
		t.Fatalf("personal with config should pass: %v", err)
	}
}

func TestWriteFeatureParamsRequiredErrorPayload(t *testing.T) {
	rec := httptest.NewRecorder()
	writeFeatureParamsRequiredError(rec, featureParamsSourceRequiredMsg)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["code"] != "FEATURE_PARAMS_SOURCE_REQUIRED" {
		t.Fatalf("expected FEATURE_PARAMS_SOURCE_REQUIRED code, got %#v", payload)
	}
	msg, _ := payload["message"].(string)
	if !strings.Contains(msg, "智能体资源配置为必填项") {
		t.Fatalf("expected required message, got %#v", payload)
	}
}

func TestWriteFeatureParamsPersonalConfigRequiredErrorPayload(t *testing.T) {
	rec := httptest.NewRecorder()
	writeFeatureParamsRequiredError(rec, featureParamsPersonalConfigRequiredMsg)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "personal_feature_params_config_id") {
		t.Fatalf("expected personal config id message, got: %s", rec.Body.String())
	}
}
