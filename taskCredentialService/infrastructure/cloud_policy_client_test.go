package infrastructure

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCloudPolicyHTTPClientParsesMinutesAndOmitsEmptySTS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/cloud/workspace-machine-policy/" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.URL.Query().Get("company_id") != "t1" || r.URL.Query().Get("workspace_id") != "w1" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"success","idle_recycle_minutes":12,"enabled_authorization_ids":[]}`)
	}))
	t.Cleanup(srv.Close)
	c := NewCloudPolicyHTTPClient(srv.URL, "", 2)
	p, err := c.FetchWorkspaceMachinePolicy("t1", "w1", "task", "cmt")
	if err != nil {
		t.Fatal(err)
	}
	if p.IdleRecycleMinutes != 12 {
		t.Fatalf("minutes=%d", p.IdleRecycleMinutes)
	}
	if p.MachineReleaseSTS != nil {
		t.Fatalf("sts=%v", p.MachineReleaseSTS)
	}
}
