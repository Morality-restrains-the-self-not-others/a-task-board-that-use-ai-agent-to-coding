package relaylifecycle

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskEvents/domain"
)

func TestClearReachabilityHandlerUsesTaskCloudService(t *testing.T) {
	var called bool
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/cloud-server-config/clear-after-stop/" {
			called = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "cleared": true})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer cloudSrv.Close()

	h := &ClearReachabilityHandler{
		TaskCloud: &TaskCloudInternalClient{BaseURL: cloudSrv.URL},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t1", "tenant_id": "100", "workspace_id": "ws1", "reason": "relay_stop",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "RELAY_STOP_SUCCEEDED",
		Envelope:  domain.EventEnvelope{EventType: "RELAY_STOP_SUCCEEDED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if !called {
		t.Fatal("expected taskCloudService clear-after-stop call")
	}
}
