package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInstructWorkerLegacyStream(t *testing.T) {
	container := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/ai/instructs/") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer container.Close()

	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/cloud-server-config/lookup/" {
			_, _ = w.Write([]byte(`{"company_id":"t1","workspace_id":"ws1","task_id":"task1","server_url":"` + container.URL + `/"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer cloud.Close()
	cfg.TaskCloudServiceURL = cloud.URL
	cfg.TaskSseURL = ""
	cfg.KafkaBootstrapServers = ""

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	runInstructStreamForComment(ctx, instructStreamParams{
		CommentID:   "c1",
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TaskID:      "task1",
		UserContent: "ping",
	})
}

func TestStreamLegacyInstruct(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "stream-data")
	}))
	defer srv.Close()

	result := streamLegacyInstruct(context.Background(), srv.URL, "t1", "ws1", "task1", "c1", "hi", "")
	if result.StatusCode != 200 {
		t.Fatalf("status=%d", result.StatusCode)
	}
	text := readStreamChunks(result.Chunks)
	if text != "stream-data" {
		t.Fatalf("text=%q", text)
	}
}
