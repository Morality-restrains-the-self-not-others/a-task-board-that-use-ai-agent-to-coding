package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPLokiQueryClient_CountLogsByJob(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ready":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ready"))
			return
		case "/loki/api/v1/query":
			if !contains(r.URL.RawQuery, "runall-ship-999") {
				t.Fatalf("query missing trace: %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data": map[string]any{
					"resultType": "vector",
					"result": []map[string]any{
						{
							"metric": map[string]string{"job": "task-cloud-service"},
							"value":  []any{float64(time.Now().Unix()), "2"},
						},
						{
							"metric": map[string]string{"job": "saas-backend"},
							"value":  []any{float64(time.Now().Unix()), "1"},
						},
					},
				},
			})
			return
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewHTTPLokiQueryClient(srv.URL)
	if err := client.Ready(context.Background()); err != nil {
		t.Fatalf("Ready: %v", err)
	}
	counts, err := client.CountLogsByJob(context.Background(), "runall-ship-999", 10*time.Minute)
	if err != nil {
		t.Fatalf("CountLogsByJob: %v", err)
	}
	if counts["task-cloud-service"] != 2 {
		t.Fatalf("task-cloud-service count = %d", counts["task-cloud-service"])
	}
	if counts["saas-backend"] != 1 {
		t.Fatalf("saas-backend count = %d", counts["saas-backend"])
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || len(s) > 0 && containsLoop(s, sub)))
}

func containsLoop(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
