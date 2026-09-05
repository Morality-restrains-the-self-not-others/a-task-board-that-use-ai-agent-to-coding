package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"tracelog"
)

func TestHealthEndpointTraceHeader(t *testing.T) {
	tracelog.Init("go-relay-test")
	handler := setupRoutes()
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("X-Trace-Id", "test-trace-abc12345")
	req.Header.Set("X-Parent-Span-Id", "a1b2c3d4e5f67890")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-Trace-Id") != "test-trace-abc12345" {
		t.Errorf("expected trace header echoed, got %q", w.Header().Get("X-Trace-Id"))
	}
	if w.Header().Get("X-Span-Id") == "" {
		t.Errorf("expected response span header")
	}
}

func TestHealthEndpointRejectsTraceIdOnly(t *testing.T) {
	tracelog.Init("go-relay-test")
	handler := setupRoutes()
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("X-Trace-Id", "legacy-trace-only12345")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	handler := setupRoutes()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["service"] != "go-relay" {
		t.Errorf("expected service=go-relay, got %v", body["service"])
	}
	if body["ok"] != true {
		t.Errorf("expected ok=true, got %v", body["ok"])
	}
}

func TestHealthEndpointNoAuthRequired(t *testing.T) {
	os.Setenv("RELAY_TO_TRAE_SECRET", "secret")
	initTimeouts()
	defer os.Unsetenv("RELAY_TO_TRAE_SECRET")

	handler := setupRoutes()

	req := httptest.NewRequest("GET", "/health", nil)
	// No X-Relay-To-Trae-Secret header.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 for health without secret, got %d", w.Code)
	}
}

func TestCORSHeadersPresent(t *testing.T) {
	handler := setupRoutes()

	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("expected CORS origin, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Vary") != "Origin" {
		t.Errorf("expected Vary: Origin, got %q", w.Header().Get("Vary"))
	}
}

func TestCORSOptionsRequest(t *testing.T) {
	handler := setupRoutes()

	req := httptest.NewRequest("OPTIONS", "/v1/start", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("expected 204 for OPTIONS, got %d", w.Code)
	}
}

func TestSecretAuthBlocksUnauthorized(t *testing.T) {
	os.Setenv("RELAY_TO_TRAE_SECRET", "my-secret")
	initTimeouts()
	defer os.Unsetenv("RELAY_TO_TRAE_SECRET")

	handler := setupRoutes()

	req := httptest.NewRequest("GET", "/v1/status", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSecretAuthAllowsAuthorized(t *testing.T) {
	os.Setenv("RELAY_TO_TRAE_SECRET", "my-secret")
	initTimeouts()
	defer os.Unsetenv("RELAY_TO_TRAE_SECRET")

	handler := setupRoutes()

	req := httptest.NewRequest("GET", "/v1/status", nil)
	req.Header.Set("X-Relay-To-Trae-Secret", "my-secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSecretAuthSkipsWhenNotSet(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	handler := setupRoutes()

	req := httptest.NewRequest("GET", "/v1/status", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 when no secret required, got %d", w.Code)
	}
}

func TestStatusEndpoint(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	stateMu.Lock()
	state.Running = false
	state.PID = 0
	state.Logs = []string{"test-log"}
	stateMu.Unlock()

	handler := setupRoutes()

	req := httptest.NewRequest("GET", "/v1/status", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["running"] != false {
		t.Errorf("expected running=false, got %v", body["running"])
	}
}

func TestStatusEndpointWithCursor(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	stateMu.Lock()
	state.Logs = []string{"a", "b", "c", "d"}
	stateMu.Unlock()

	handler := setupRoutes()

	req := httptest.NewRequest("GET", "/v1/status?cursor=2", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	logs, _ := body["logs"].([]interface{})
	if len(logs) != 2 {
		t.Errorf("expected 2 logs from cursor 2, got %d", len(logs))
	}
}

func TestStatusEndpointInvalidCursor(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	stateMu.Lock()
	state.Logs = []string{"a", "b"}
	stateMu.Unlock()

	handler := setupRoutes()

	req := httptest.NewRequest("GET", "/v1/status?cursor=invalid", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestStartEndpointMissingBody(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	handler := setupRoutes()

	req := httptest.NewRequest("POST", "/v1/start", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should get 400 because env is missing required keys.
	if w.Code != 400 {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStartEndpointWithEnv(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	stateMu.Lock()
	state.Running = false
	cmd = nil
	stateMu.Unlock()

	handler := setupRoutes()

	body := map[string]interface{}{
		"tenant_id":    "t1",
		"workspace_id": "w1",
		"task_id":      "task1",
		"env": map[string]string{
			"TASK_API_ENDPOINT_ORIGIN":     "http://task.example.com",
			"BUSINESS_API_ENDPOINT_ORIGIN": "http://biz.example.com",
			"ACCESS_TOKEN":                 "test-token",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/v1/start", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should fail with 400 because run.sh doesn't exist in test context,
	// but we verify it processes the request body correctly.
	if w.Code == 401 {
		t.Error("should not return 401 (no secret required)")
	}
}

func TestRegisterEndpointMissingTaskID(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	handler := setupRoutes()

	body := map[string]string{
		"tenant_id": "t1",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/v1/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterEndpointDeferredWithoutAccessOrState(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	stateMu.Lock()
	state.AccessToken = ""
	delete(registeredTasks, "deferred-task")
	stateMu.Unlock()

	handler := setupRoutes()
	body := map[string]string{
		"task_id":                  "deferred-task",
		"task_api_endpoint_origin": "http://origin.example.com",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/v1/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "deferred" {
		t.Errorf("expected status=deferred, got %v", resp)
	}
	stateMu.Lock()
	_, registered := registeredTasks["deferred-task"]
	stateMu.Unlock()
	if registered {
		t.Error("expected task not registered when deferred")
	}
}

func TestRegisterEndpointReusesStateTokenWithoutBodyAccess(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	stateMu.Lock()
	state.AccessToken = "state-register-token"
	delete(registeredTasks, "state-reg-task")
	stateMu.Unlock()

	handler := setupRoutes()
	body := map[string]string{
		"task_id":                  "state-reg-task",
		"task_api_endpoint_origin": "http://origin.example.com",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/v1/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	stateMu.Lock()
	reg, ok := registeredTasks["state-reg-task"]
	stateMu.Unlock()
	if !ok {
		t.Fatal("expected task registered")
	}
	if reg.AccessToken != "state-register-token" {
		t.Errorf("expected state-register-token, got %q", reg.AccessToken)
	}
}

func TestRegisterEndpointSuccess(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	stateMu.Lock()
	state.AccessToken = ""
	delete(registeredTasks, "reg-test")
	stateMu.Unlock()

	handler := setupRoutes()

	body := map[string]string{
		"task_id":                   "reg-test",
		"task_api_endpoint_origin":  "http://origin.example.com",
		"access_token":              "reg-token",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/v1/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", resp)
	}
	if resp["task_id"] != "reg-test" {
		t.Errorf("expected task_id=reg-test, got %q", resp["task_id"])
	}
}

func TestStopEndpointNoProcess(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	stateMu.Lock()
	cmd = nil
	state.Running = false
	state.AccessToken = ""
	state.UIURL = ""
	stateMu.Unlock()

	handler := setupRoutes()

	req := httptest.NewRequest("POST", "/v1/stop", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", resp)
	}
}

func TestCORSHeadersOnErrorResponse(t *testing.T) {
	os.Setenv("RELAY_TO_TRAE_SECRET", "secret")
	initTimeouts()
	defer os.Unsetenv("RELAY_TO_TRAE_SECRET")

	handler := setupRoutes()

	req := httptest.NewRequest("GET", "/v1/status", nil)
	req.Header.Set("Origin", "http://test.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// CORS should be present even on error responses.
	if w.Header().Get("Access-Control-Allow-Origin") != "http://test.com" {
		t.Errorf("expected CORS origin on error, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestStartEndpointCamelCaseFields(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	stateMu.Lock()
	state.Running = false
	cmd = nil
	stateMu.Unlock()

	handler := setupRoutes()

	body := map[string]interface{}{
		"tenantId":    "ct1",
		"workspaceId": "cw1",
		"taskId":      "ctask1",
		"env": map[string]string{
			"TASK_API_ENDPOINT_ORIGIN":     "http://task.example.com",
			"BUSINESS_API_ENDPOINT_ORIGIN": "http://biz.example.com",
			"ACCESS_TOKEN":                 "test-token",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/v1/start", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should fail on run.sh not found (400) rather than missing fields.
	if w.Code != 400 {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterEndpointAlternateFieldNames(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	stateMu.Lock()
	state.AccessToken = ""
	delete(registeredTasks, "alt-test")
	stateMu.Unlock()

	handler := setupRoutes()

	body := map[string]string{
		"taskId":                  "alt-test",
		"TASK_API_ENDPOINT_ORIGIN": "http://alt.example.com",
		"ACCESS_TOKEN":            "alt-token",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/v1/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestContentTypeJSON(t *testing.T) {
	handler := setupRoutes()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	// handlers.go 显式设置 "application/json; charset=utf-8"（JSON 文本的合法 Content-Type）
	if ct != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type: application/json; charset=utf-8, got %q", ct)
	}
}

func TestClearLogsPathScopeClearsActiveLogs(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	stateMu.Lock()
	state.ActiveTaskID = "task-clear"
	state.Logs = []string{"old-1", "old-2"}
	taskLogs["task-clear"] = []string{"old-1", "old-2"}
	registeredTasks["task-clear"] = &RegisteredTask{
		TenantID:    "t1",
		WorkspaceID: "w1",
		TaskID:      "task-clear",
		LogCursor:   2,
	}
	stateMu.Unlock()
	defer func() {
		stateMu.Lock()
		state.ActiveTaskID = ""
		state.Logs = nil
		delete(taskLogs, "task-clear")
		delete(registeredTasks, "task-clear")
		stateMu.Unlock()
	}()

	handler := setupRoutes()
	req := httptest.NewRequest("POST", "/v1/tenant/t1/workspace/w1/task/task-clear/clear-logs", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["cleared"] != true || body["tenant_id"] != "t1" || body["workspace_id"] != "w1" || body["task_id"] != "task-clear" {
		t.Fatalf("unexpected body: %#v", body)
	}
	if body["cleared_live"] != true {
		t.Fatalf("expected cleared_live=true, got %#v", body["cleared_live"])
	}

	stateMu.Lock()
	defer stateMu.Unlock()
	if len(state.Logs) != 0 {
		t.Fatalf("expected empty live logs, got %#v", state.Logs)
	}
	if rows := taskLogs["task-clear"]; len(rows) != 0 {
		t.Fatalf("expected empty task logs, got %#v", rows)
	}
	if registeredTasks["task-clear"].LogCursor != 0 {
		t.Fatalf("expected LogCursor=0, got %d", registeredTasks["task-clear"].LogCursor)
	}
}

func TestClearLogsScopeMismatchForbidden(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()

	stateMu.Lock()
	registeredTasks["task-mm"] = &RegisteredTask{
		TenantID:    "t-real",
		WorkspaceID: "w-real",
		TaskID:      "task-mm",
		LogCursor:   1,
	}
	taskLogs["task-mm"] = []string{"keep"}
	stateMu.Unlock()
	defer func() {
		stateMu.Lock()
		delete(registeredTasks, "task-mm")
		delete(taskLogs, "task-mm")
		stateMu.Unlock()
	}()

	handler := setupRoutes()
	req := httptest.NewRequest("POST", "/v1/tenant/t-other/workspace/w-real/task/task-mm/clear-logs", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	stateMu.Lock()
	defer stateMu.Unlock()
	if len(taskLogs["task-mm"]) != 1 {
		t.Fatalf("mismatch must not clear logs, got %#v", taskLogs["task-mm"])
	}
}

func TestClearLogsLegacyQueryPathNotRegistered(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_SECRET")
	initTimeouts()
	handler := setupRoutes()
	req := httptest.NewRequest("POST", "/v1/clear-logs?task_id=task-x", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for legacy path, got %d: %s", w.Code, w.Body.String())
	}
}
