package tracelog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetDaydaymoneyMeta_DefaultTagsFromServiceID(t *testing.T) {
	SetDaydaymoneyMeta("taskProjectService", nil)
	if got := AidevServiceID(); got != "taskProjectService" {
		t.Fatalf("AidevServiceID = %q want taskProjectService", got)
	}
	if got := DaydaymoneyTags(); len(got) != 1 || got[0] != "svc:taskProjectService" {
		t.Fatalf("DaydaymoneyTags = %v want [svc:taskProjectService]", got)
	}
}

func TestSetDaydaymoneyMeta_TrimsAndPreservesExplicitTags(t *testing.T) {
	SetDaydaymoneyMeta("  task2app  ", []string{" svc:task2app ", "", "domain:saas"})
	if got := AidevServiceID(); got != "task2app" {
		t.Fatalf("AidevServiceID = %q want task2app", got)
	}
	want := []string{"svc:task2app", "domain:saas"}
	got := DaydaymoneyTags()
	if len(got) != len(want) {
		t.Fatalf("DaydaymoneyTags = %v want %v", got, want)
	}
	for i, tag := range want {
		if got[i] != tag {
			t.Fatalf("DaydaymoneyTags[%d] = %q want %q", i, got[i], tag)
		}
	}
}

func TestInit_DefaultsDaydaymoneyMetaToService(t *testing.T) {
	SetDaydaymoneyMeta("", nil)
	out := captureStdout(t, func() {
		Init("my-service-test")
	})
	_ = out
	if got := AidevServiceID(); got != "my-service-test" {
		t.Fatalf("AidevServiceID = %q want my-service-test", got)
	}
	if got := DaydaymoneyTags(); len(got) != 1 || got[0] != "svc:my-service-test" {
		t.Fatalf("DaydaymoneyTags = %v want [svc:my-service-test]", got)
	}
}

func TestEmit_IncludesAidevFields(t *testing.T) {
	out := captureStdout(t, func() {
		Init("emit-service-test")
		SetDaydaymoneyMeta("emit-service-test", []string{"svc:emit-service-test", "team:core"})
		Emit("info", "startup complete", "main", nil)
	})
	line := strings.TrimSpace(out)
	var payload map[string]string
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["daydaymoney_service_id"] != "emit-service-test" {
		t.Fatalf("daydaymoney_service_id = %q", payload["daydaymoney_service_id"])
	}
	if payload["daydaymoney_tags"] != "svc:emit-service-test,team:core" {
		t.Fatalf("daydaymoney_tags = %q", payload["daydaymoney_tags"])
	}
}

func TestMiddleware_EmitsAidevFieldsViaSlog(t *testing.T) {
	out := captureStdout(t, func() {
		Init("slog-daydaymoney-test")
		SetDaydaymoneyMeta("slog-daydaymoney-test", []string{"svc:slog-daydaymoney-test"})
		h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	})
	line := strings.TrimSpace(out)
	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["daydaymoney_service_id"] != "slog-daydaymoney-test" {
		t.Fatalf("daydaymoney_service_id = %v", payload["daydaymoney_service_id"])
	}
	if payload["daydaymoney_tags"] != "svc:slog-daydaymoney-test" {
		t.Fatalf("daydaymoney_tags = %v", payload["daydaymoney_tags"])
	}
}
