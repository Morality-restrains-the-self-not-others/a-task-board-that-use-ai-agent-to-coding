package aiassistant

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"taskEvents/domain"
	"taskEvents/internal/repository/saas"
)

func TestDispatchPersistsOnStreamOK(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method %s", r.Method)
		}
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	t.Setenv("TASK_AI_COMMENT_URL", srv.URL)

	repo := &saas.Repository{}
	h := &Handler{Repo: repo}
	data, _ := json.Marshal(map[string]interface{}{
		"stream_ok":      true,
		"ai_comment_id":  900,
		"assistant_text": "hello from ai",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "AI_ASSISTANT_REPLY_COMPLETED",
		Envelope:  domain.EventEnvelope{EventType: "AI_ASSISTANT_REPLY_COMPLETED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if gotBody == "" {
		t.Fatal("expected PATCH body")
	}
}

func TestDispatchSkipsWhenNotPersistable(t *testing.T) {
	os.Unsetenv("TASK_AI_COMMENT_URL")
	repo := &saas.Repository{}
	h := &Handler{Repo: repo}
	data, _ := json.Marshal(map[string]interface{}{
		"stream_ok":      false,
		"ai_comment_id":  900,
		"assistant_text": "",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "AI_ASSISTANT_REPLY_COMPLETED",
		Envelope:  domain.EventEnvelope{EventType: "AI_ASSISTANT_REPLY_COMPLETED", Data: data},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("outcome %v err=%v", out, err)
	}
}
