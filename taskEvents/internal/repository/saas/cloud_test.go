package saas

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCloudAuthorizationByIDStringID 回归（OPT-20260809-026）：平台 SSOT 授权 ID 为
// 字符串 cpa_<snowflake>（cloud_platform_authorizations.id），CloudAuthorizationByID
// 必须按字符串透传查询 lookup 端点，而非 int64 解析（历史代码 strconv.ParseInt
// 对 "cpa_..." 失败 → 误报"缺少 authorization_id"）。
func TestCloudAuthorizationByIDStringID(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "cpa_-3077015452416368750", "platform_type": "aliyun",
			"secret_id": "ak", "secret_key": "sk",
		})
	}))
	defer srv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)

	auth, err := (&Repository{}).CloudAuthorizationByID("cpa_-3077015452416368750")
	if err != nil {
		t.Fatal(err)
	}
	if auth == nil {
		t.Fatal("expected authorization")
	}
	if auth.ID != "cpa_-3077015452416368750" {
		t.Fatalf("ID = %q, want cpa_ string ID", auth.ID)
	}
	if !strings.Contains(gotQuery, "id=cpa_-3077015452416368750") {
		t.Fatalf("lookup query %q missing string authorization id", gotQuery)
	}
	if auth.SecretID != "ak" || auth.SecretKey != "sk" {
		t.Fatalf("secret fields not parsed: %+v", auth)
	}
}

// TestCloudAuthorizationByIDEmptyString：空 ID 直接报错，不发起 lookup 请求。
func TestCloudAuthorizationByIDEmptyString(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)

	_, err := (&Repository{}).CloudAuthorizationByID("   ")
	if err == nil {
		t.Fatal("expected error for empty id")
	}
	if called {
		t.Fatal("lookup must not be called for empty id")
	}
}

// TestCloudServerEventStatusByID（OPT-20260818-015）：按 event_id 查 DB 启动事件状态，
// 供消费者判断「已成功处理」以幂等跳过重放。
func TestCloudServerEventStatusByID(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/cloud-server-events/status" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"event_id": "evt-1", "status": "success"})
	}))
	defer srv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)

	st, err := (&Repository{}).CloudServerEventStatusByID("evt-1")
	if err != nil {
		t.Fatal(err)
	}
	if st != "success" {
		t.Fatalf("status=%q want success", st)
	}
	if !strings.Contains(gotQuery, "event_id=evt-1") {
		t.Fatalf("query %q missing event_id", gotQuery)
	}
}

// TestClaimCloudServerStartEvent（OPT-20260818-015 剩余 gap）：带 from_status 守卫的
// 原子 claim。200 → claimed=true；409（事件已被并发消费者抢先 claim）→ claimed=false、err=nil；
// 其它非 2xx → claimed=false、err != nil。
func TestClaimCloudServerStartEvent(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		wantClaim bool
		wantErr   bool
	}{
		{"claim success", http.StatusOK, true, false},
		{"claim lost conflict", http.StatusConflict, false, false},
		{"claim transient error", http.StatusInternalServerError, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotFromStatus, gotStatus string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/internal/cloud-server-events/update-status" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)
				gotFromStatus, _ = body["from_status"].(string)
				gotStatus, _ = body["status"].(string)
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": tc.status < 400})
			}))
			defer srv.Close()
			t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)

			claimed, err := (&Repository{}).ClaimCloudServerStartEvent(123)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got claimed=%v", claimed)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if claimed != tc.wantClaim {
				t.Fatalf("claimed=%v want %v", claimed, tc.wantClaim)
			}
			if gotFromStatus != "pending" || gotStatus != "processing" {
				t.Fatalf("claim body from_status=%q status=%q want pending/processing", gotFromStatus, gotStatus)
			}
		})
	}
}
