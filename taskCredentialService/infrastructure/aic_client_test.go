package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAICommentClientFetchActiveContextPack(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/task-ai-comment/container-agent-comments/active-by-task" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("X-TaskAIComment-Internal-Secret") != "sec" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.URL.Query().Get("task_id") != "task-1" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "agent-1",
			"context_pack": map[string]interface{}{
				"at_mention_run": map[string]interface{}{
					"run_id": "agent-1",
					"installed_image": map[string]string{
						"id": "img-1", "name": "N",
					},
				},
				"comment_thread": []map[string]interface{}{
					{"kind": "human", "id": "c1", "content": "hi"},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := NewAICommentClient(srv.URL, "sec")
	pack, err := client.FetchActiveContextPack("task-1")
	if err != nil {
		t.Fatalf("FetchActiveContextPack: %v", err)
	}
	if pack == nil {
		t.Fatal("expected pack")
	}
	at, _ := pack["at_mention_run"].(map[string]interface{})
	if at == nil || at["run_id"] != "agent-1" {
		t.Fatalf("at_mention_run=%#v", pack["at_mention_run"])
	}
	thread, _ := pack["comment_thread"].([]interface{})
	if len(thread) != 1 {
		t.Fatalf("comment_thread=%#v", pack["comment_thread"])
	}
}

func TestAICommentClientFetchActiveContextPackNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	client := NewAICommentClient(srv.URL, "sec")
	pack, err := client.FetchActiveContextPack("missing")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if pack != nil {
		t.Fatalf("expected nil pack, got %#v", pack)
	}
}

func TestSoftFetchActiveContextPackOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	t.Cleanup(srv.Close)
	client := NewAICommentClient(srv.URL, "sec")
	pack := SoftFetchActiveContextPack(client, "task-1")
	if pack != nil {
		t.Fatalf("soft-fail should return nil, got %#v", pack)
	}
}
