package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStartVMInjectsImageInvokerUserID(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	prev := cfg.CloudServiceURL
	cfg.CloudServiceURL = srv.URL
	t.Cleanup(func() { cfg.CloudServiceURL = prev })

	err := startVM(context.Background(), "t1", "ws1", "start-vm-auto", "commenter-1", map[string]interface{}{
		"task_id": "task1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["image_invoker_user_id"] != "commenter-1" {
		t.Fatalf("body=%v", gotBody)
	}
}
