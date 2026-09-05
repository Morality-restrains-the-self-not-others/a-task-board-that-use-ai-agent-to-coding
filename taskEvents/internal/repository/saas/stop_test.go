package saas

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestClearCloudServerConfigAfterStopForwards：deprecated 包装器仍须转发到
// taskCloudService clear-after-stop（OPT-20260818-015 改签名后保持行为不变）。
func TestClearCloudServerConfigAfterStopForwards(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/internal/cloud-server-config/clear-after-stop/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		called = true
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "cleared": true})
	}))
	defer srv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)

	if err := (&Repository{}).ClearCloudServerConfigAfterStop(100, "ws1", "task1", "stop_vm"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected clear-after-stop call")
	}
	if err := RequestClearReachabilityOnTaskCloudService("t1", "w1", "task2", "relay_stop"); err != nil {
		t.Fatal(err)
	}
}
