package cloudconfig

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// OPT-20260821-020: 跨进程连接默认走 INFRA_HOST（同节点），禁止只解析成 loopback-only。
func TestServiceBaseNotLoopbackOnly(t *testing.T) {
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", "")
	t.Setenv("INFRA_HOST", "")
	base := serviceBase()
	if base == "http://127.0.0.1:8018" || base == "http://localhost:8018" {
		t.Fatalf("serviceBase=%q is loopback-only; want INFRA_HOST-derived default", base)
	}
	if !strings.HasSuffix(base, ":8018") {
		t.Fatalf("serviceBase=%q should keep cloud port 8018", base)
	}
}

func TestServiceBaseEnvOverrideWins(t *testing.T) {
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", "http://cloud.example.com")
	if got := serviceBase(); got != "http://cloud.example.com" {
		t.Fatalf("serviceBase=%q, want env override", got)
	}
}

func TestStopRequestProcessed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/cloud/stops/processed" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("stop_request_id") != "sr-x" {
			t.Fatalf("stop_request_id query=%q", r.URL.Query().Get("stop_request_id"))
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"processed": true})
	}))
	defer srv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)

	processed, err := StopRequestProcessed("sr-x")
	if err != nil {
		t.Fatal(err)
	}
	if !processed {
		t.Fatal("processed=false want true")
	}
}

func TestClearAfterStopSendsStopRequestID(t *testing.T) {
	var bodyMap map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/internal/cloud-server-config/clear-after-stop/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &bodyMap)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "cleared": true})
	}))
	defer srv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)

	if err := ClearAfterStop("t1", "w1", "task1", "stop_vm", "i-1", "sr-9"); err != nil {
		t.Fatal(err)
	}
	if bodyMap == nil || bodyMap["stop_request_id"] != "sr-9" {
		t.Fatalf("clear-after-stop body missing stop_request_id: %v", bodyMap)
	}
}

func TestUpsertAfterStartLooksUpAndImportsCommentCSC(t *testing.T) {
	var lookupURL string
	var imported map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup/"):
			lookupURL = r.URL.String()
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "csc-cmt-1", "comment_id": "cmt-1", "instance_id": "",
			})
		case r.URL.Path == "/api/internal/cloud-server-config/import/":
			raw, _ := io.ReadAll(r.Body)
			var body map[string]interface{}
			_ = json.Unmarshal(raw, &body)
			rows, _ := body["configs"].([]interface{})
			if len(rows) > 0 {
				imported, _ = rows[0].(map[string]interface{})
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "imported": 1})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)

	err := UpsertAfterStart(100, "ws1", "task-1", "cpa_1", "", StartVMResult{
		Platform:   "aliyun",
		InstanceID: "i-new",
		Region:     "cn-hongkong",
		CommentID:  "cmt-1",
		CSCID:      "csc-cmt-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lookupURL, "comment_id=cmt-1") {
		t.Fatalf("lookup must include comment_id: %s", lookupURL)
	}
	if imported == nil {
		t.Fatal("import body missing")
	}
	if imported["comment_id"] != "cmt-1" {
		t.Fatalf("import comment_id=%v", imported["comment_id"])
	}
	if imported["instance_id"] != "i-new" {
		t.Fatalf("import instance_id=%v", imported["instance_id"])
	}
	if imported["id"] != "csc-cmt-1" {
		t.Fatalf("import must reuse comment CSC id, got %v", imported["id"])
	}
}
