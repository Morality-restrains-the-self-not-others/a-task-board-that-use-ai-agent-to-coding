package syncprofile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"taskEvents/domain"
	"taskEvents/internal/repository/saas"
)

const testUserID = "849095291104686080"

// TestDispatchSkipsEmptyUsername: 空 username 事件必须跳过（不调用 taskAuth
// upsert API）并返回 DispatchSuccess — 手机号注册/微信登录事件不带昵称
// （个人昵称由 taskAuth 登录链路同步直写），无条件 upsert 空值会抹掉昵称。
func TestDispatchSkipsEmptyUsername(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "upsert should not be called for empty username", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("TASK_AUTH_INTERNAL_URL", srv.URL)

	repo := saas.New("")
	h := &Handler{Repo: repo}
	data, _ := json.Marshal(map[string]interface{}{"user_id": testUserID, "username": ""})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "USER_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: data},
	})
	if err != nil {
		t.Fatalf("dispatch empty username: %v", err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome = %v, want DispatchSuccess", out)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("expected no upsert API call, got %d", got)
	}
}

// TestDispatchUpsertsUsername: 非空 username → 调用 taskAuth 内部 upsert API
// 并返回 DispatchSuccess（邮箱注册路径：个人昵称 = 注册用户名）。
func TestDispatchUpsertsUsername(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer srv.Close()
	t.Setenv("TASK_AUTH_INTERNAL_URL", srv.URL)

	repo := saas.New("")
	h := &Handler{Repo: repo}
	data, _ := json.Marshal(map[string]interface{}{"user_id": testUserID, "username": "evtuser"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "USER_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: data},
	})
	if err != nil {
		t.Fatalf("dispatch username: %v", err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome = %v, want DispatchSuccess", out)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 upsert API call, got %d", got)
	}
}
