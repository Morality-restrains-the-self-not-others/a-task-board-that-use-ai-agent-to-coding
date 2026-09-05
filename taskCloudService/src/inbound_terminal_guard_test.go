package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRefuseInboundIfTaskTerminalCancelledReleasesMachine(t *testing.T) {
	setupCloudTestDB(t)
	seedInboundCSC(t, "cfg-term", "task-term", "cmt-term", "i-term", "203.0.113.77")

	prevFn := lookupTaskTerminalKindsFn
	lookupTaskTerminalKindsFn = func(ids []string) (map[string]string, error) {
		return map[string]string{"task-term": "cancelled"}, nil
	}
	t.Cleanup(func() { lookupTaskTerminalKindsFn = prevFn })

	prevCred := cfg.CredentialServiceURL
	prevKafka := cfg.KafkaBootstrapServers
	cred := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "validate") {
			_, _ = io.WriteString(w, `{"valid":true,"company_id":"t1","workspace_id":"ws1","task_id":"task-term","expires_at":"2099-01-01"}`)
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"tok"}`)
	}))
	defer cred.Close()
	cfg.CredentialServiceURL = cred.URL
	cfg.KafkaBootstrapServers = ""
	t.Cleanup(func() {
		cfg.CredentialServiceURL = prevCred
		cfg.KafkaBootstrapServers = prevKafka
	})

	body, _ := json.Marshal(map[string]any{
		"access_token": "tok",
		"comment_id":   "cmt-term",
		"seq":          1,
	})
	req := httptest.NewRequest(http.MethodPost, "/heartbeat", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleContainerInboundToken(rec, req, "t1", "ws1", "task-term", "heartbeat")
	if rec.Code != http.StatusGone {
		t.Fatalf("status=%d body=%s want 410", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["code"] != "TASK_TERMINAL" {
		t.Fatalf("code=%v", out["code"])
	}
	var released int
	if err := db.QueryRow(`SELECT COALESCE(terminal_released,0) FROM cloud_server_configs WHERE id='cfg-term'`).Scan(&released); err != nil {
		t.Fatal(err)
	}
	if released != 1 {
		t.Fatalf("terminal_released=%d want 1", released)
	}
	var instance string
	if err := db.QueryRow(`SELECT COALESCE(instance_id,'') FROM cloud_server_configs WHERE id='cfg-term'`).Scan(&instance); err != nil {
		t.Fatal(err)
	}
	if instance != "" {
		t.Fatalf("instance_id=%q want empty after terminal release", instance)
	}
}

func TestRefuseInboundIfTaskTerminalSkipsRequestMachineRelease(t *testing.T) {
	setupCloudTestDB(t)
	seedInboundCSC(t, "cfg-rel", "task-rel", "cmt-rel", "i-rel", "203.0.113.78")

	called := false
	prevFn := lookupTaskTerminalKindsFn
	lookupTaskTerminalKindsFn = func(ids []string) (map[string]string, error) {
		called = true
		return map[string]string{"task-rel": "cancelled"}, nil
	}
	t.Cleanup(func() { lookupTaskTerminalKindsFn = prevFn })

	prevCred := cfg.CredentialServiceURL
	prevKafka := cfg.KafkaBootstrapServers
	cred := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "validate") {
			_, _ = io.WriteString(w, `{"valid":true,"company_id":"t1","workspace_id":"ws1","task_id":"task-rel","expires_at":"2099-01-01"}`)
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"tok"}`)
	}))
	defer cred.Close()
	cfg.CredentialServiceURL = cred.URL
	cfg.KafkaBootstrapServers = ""
	t.Cleanup(func() {
		cfg.CredentialServiceURL = prevCred
		cfg.KafkaBootstrapServers = prevKafka
	})

	body, _ := json.Marshal(map[string]any{
		"access_token":  "tok",
		"comment_id":    "cmt-rel",
		"terminal_kind": "cancelled",
	})
	req := httptest.NewRequest(http.MethodPost, "/release", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleContainerInboundToken(rec, req, "t1", "ws1", "task-rel", "request-machine-release")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s want 200", rec.Code, rec.Body.String())
	}
	if called {
		t.Fatal("request-machine-release must not lookup terminal kind")
	}
}

func TestRefuseInboundIfTaskTerminalOpenTaskAllowsHeartbeat(t *testing.T) {
	setupCloudTestDB(t)

	prevFn := lookupTaskTerminalKindsFn
	lookupTaskTerminalKindsFn = func(ids []string) (map[string]string, error) {
		return map[string]string{"task1": ""}, nil
	}
	t.Cleanup(func() { lookupTaskTerminalKindsFn = prevFn })

	if refuseInboundIfTaskTerminal(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/heartbeat", nil), nil, &CloudServerConfig{
		CompanyID: "t1", WorkspaceID: "w1", TaskID: "task1",
	}, "t1", "w1", "task1", "heartbeat") {
		t.Fatal("open task must not be refused")
	}
}

func TestTerminalReleasedFlagIsCommentScoped(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-22 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, terminal_released, last_runtime_status, created_at, updated_at)
		VALUES
		('cfg-tpl-flag', 't1', 'ws1', 'task-flag', '', 'aliyun', '', 'cn-qingdao', 'cn-qingdao-b', 'auth-1', 1, 'Released', ?, ?),
		('cfg-cmt-flag', 't1', 'ws1', 'task-flag', 'cmt-open', 'aliyun', 'i-open', 'cn-qingdao', 'cn-qingdao-b', 'auth-1', 0, 'Running', ?, ?)`,
		now, now, now, now)
	if err != nil {
		t.Fatal(err)
	}
	cmt := &CloudServerConfig{ID: "cfg-cmt-flag", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-flag", CommentID: "cmt-open"}
	if got := cmt.TerminalReleasedFlag(); got != 0 {
		t.Fatalf("comment CSC flag=%d want 0 (must not inherit task-level released row)", got)
	}
	tpl := &CloudServerConfig{ID: "cfg-tpl-flag", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-flag"}
	if got := tpl.TerminalReleasedFlag(); got != 1 {
		t.Fatalf("task-level CSC flag=%d want 1", got)
	}
}

func TestRefuseInboundOpenTaskIgnoresLeftoverFlagOnLiveInstance(t *testing.T) {
	setupCloudTestDB(t)
	now := "2026-08-22 00:00:00"
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, terminal_released, last_runtime_status, created_at, updated_at)
		VALUES ('cfg-tpl-rebind', 't1', 'ws1', 'task-rebind', '', 'aliyun', '', 'cn-qingdao', 'cn-qingdao-b', 'auth-1', 1, 'Released', ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	seedInboundCSC(t, "cfg-live-rebind", "task-rebind", "cmt-rebind", "i-new", "203.0.113.90")
	if _, err := db.Exec(`UPDATE cloud_server_configs SET terminal_released=1 WHERE id='cfg-live-rebind'`); err != nil {
		t.Fatal(err)
	}

	prevFn := lookupTaskTerminalKindsFn
	lookupTaskTerminalKindsFn = func(ids []string) (map[string]string, error) {
		return map[string]string{"task-rebind": ""}, nil
	}
	t.Cleanup(func() { lookupTaskTerminalKindsFn = prevFn })

	cfg := &CloudServerConfig{
		ID: "cfg-live-rebind", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-rebind",
		CommentID: "cmt-rebind", InstanceID: "i-new", ServerURL: "http://203.0.113.90:8765/",
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/boot-progress", nil)
	if refuseInboundIfTaskTerminal(rec, req, nil, cfg, "t1", "ws1", "task-rebind", "boot-progress") {
		t.Fatalf("open task leftover flag must allow inbound, status=%d body=%s", rec.Code, rec.Body.String())
	}
	var instance string
	if err := db.QueryRow(`SELECT COALESCE(instance_id,'') FROM cloud_server_configs WHERE id='cfg-live-rebind'`).Scan(&instance); err != nil {
		t.Fatal(err)
	}
	if instance != "i-new" {
		t.Fatalf("instance_id=%q must not be released", instance)
	}
}

func TestLookupTaskTerminalKindsHTTPFailOpenEmptyURL(t *testing.T) {
	prev := cfg.TaskServiceURL
	cfg.TaskServiceURL = ""
	t.Cleanup(func() { cfg.TaskServiceURL = prev })
	got, err := lookupTaskTerminalKindsHTTP([]string{"t1"})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got=%v", got)
	}
}

func TestRefuseInboundIfTaskTerminalLookupErrorFailOpen(t *testing.T) {
	prevFn := lookupTaskTerminalKindsFn
	lookupTaskTerminalKindsFn = func(ids []string) (map[string]string, error) {
		return nil, errors.New("task service down")
	}
	t.Cleanup(func() { lookupTaskTerminalKindsFn = prevFn })

	if refuseInboundIfTaskTerminal(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/heartbeat", nil), nil, &CloudServerConfig{
		CompanyID: "t1", WorkspaceID: "w1", TaskID: "task-open-err",
	}, "t1", "w1", "task-open-err", "heartbeat") {
		t.Fatal("lookup error must fail-open")
	}
}

// OPT-20260821-019: 终态查找短 TTL 缓存 — 同 task 第二次查找不发 HTTP，终态缓存更久，错误不长期缓存。
func TestTerminalKindCacheSkipsSecondLookup(t *testing.T) {
	resetTerminalKindCache()
	calls := 0
	prevFn := lookupTaskTerminalKindsFn
	lookupTaskTerminalKindsFn = func(ids []string) (map[string]string, error) {
		calls++
		return map[string]string{"task-cached": "cancelled"}, nil
	}
	t.Cleanup(func() { lookupTaskTerminalKindsFn = prevFn })

	got, err := lookupTaskTerminalKindsCached([]string{"task-cached"})
	if err != nil || got["task-cached"] != "cancelled" {
		t.Fatalf("first lookup got=%v err=%v", got, err)
	}
	got, err = lookupTaskTerminalKindsCached([]string{"task-cached"})
	if err != nil || got["task-cached"] != "cancelled" {
		t.Fatalf("second lookup got=%v err=%v", got, err)
	}
	if calls != 1 {
		t.Fatalf("lookup calls=%d want 1 (second served from cache)", calls)
	}
}

func TestTerminalKindCacheExpiresAndRequeries(t *testing.T) {
	resetTerminalKindCache()
	calls := 0
	prevFn := lookupTaskTerminalKindsFn
	lookupTaskTerminalKindsFn = func(ids []string) (map[string]string, error) {
		calls++
		return map[string]string{"task-exp": "running"}, nil
	}
	t.Cleanup(func() { lookupTaskTerminalKindsFn = prevFn })

	if _, err := lookupTaskTerminalKindsCached([]string{"task-exp"}); err != nil {
		t.Fatal(err)
	}
	// 强制使缓存过期，验证 TTL 过期后重查
	terminalKindCacheMu.Lock()
	terminalKindCache["task-exp"] = terminalKindCacheEntry{kind: "running", expiresAt: time.Now().Add(-time.Second)}
	terminalKindCacheMu.Unlock()

	if _, err := lookupTaskTerminalKindsCached([]string{"task-exp"}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("lookup calls=%d want 2 after TTL expiry", calls)
	}
}

func TestTerminalKindCacheErrorNotLongTerm(t *testing.T) {
	resetTerminalKindCache()
	prevFn := lookupTaskTerminalKindsFn
	lookupTaskTerminalKindsFn = func(ids []string) (map[string]string, error) {
		return nil, errors.New("task service down")
	}
	t.Cleanup(func() { lookupTaskTerminalKindsFn = prevFn })

	if _, err := lookupTaskTerminalKindsCached([]string{"task-err"}); err == nil {
		t.Fatal("expected error on first lookup")
	}
	terminalKindCacheMu.Lock()
	e, ok := terminalKindCache["task-err"]
	terminalKindCacheMu.Unlock()
	if !ok {
		t.Fatal("error miss should be short-cached")
	}
	if e.expiresAt.Sub(time.Now()) > terminalKindCacheTTLNormal+time.Second {
		t.Fatalf("error cache TTL too long: %v", e.expiresAt.Sub(time.Now()))
	}
}

func TestLookupTaskTerminalKindsHTTPParsesKinds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/tasks/terminal-kinds/" {
			t.Errorf("path=%s", r.URL.Path)
		}
		if r.Header.Get("X-Internal-Secret") != "sec" {
			t.Errorf("missing secret")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"kinds":{"task-a":"cancelled","task-b":""}}`)
	}))
	defer srv.Close()
	prevURL := cfg.TaskServiceURL
	prevSec := cfg.InternalSecret
	cfg.TaskServiceURL = srv.URL
	cfg.InternalSecret = "sec"
	t.Cleanup(func() {
		cfg.TaskServiceURL = prevURL
		cfg.InternalSecret = prevSec
	})
	got, err := lookupTaskTerminalKindsHTTP([]string{"task-a", "task-b"})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got["task-a"] != "cancelled" || got["task-b"] != "" {
		t.Fatalf("got=%v", got)
	}
}
