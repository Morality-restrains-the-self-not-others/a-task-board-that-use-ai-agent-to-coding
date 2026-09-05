package relaylifecycle

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskEvents/domain"
)

func TestOpenRuntimeSessionHandlerUsesTaskCloudService(t *testing.T) {
	var called bool
	var gotPath string
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.URL.Path == "/api/internal/runtime-session/open/" ||
			r.URL.Path == "/api/internal/runtime-session/open" {
			called = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer cloudSrv.Close()

	h := &OpenRuntimeSessionHandler{
		TaskCloud: &TaskCloudInternalClient{BaseURL: cloudSrv.URL},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t1", "tenant_id": "100", "workspace_id": "ws1",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "RELAY_START_ACCEPTED",
		Envelope:  domain.EventEnvelope{EventType: "RELAY_START_ACCEPTED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if !called {
		t.Fatalf("expected taskCloudService runtime-session/open call, path=%q", gotPath)
	}
}
