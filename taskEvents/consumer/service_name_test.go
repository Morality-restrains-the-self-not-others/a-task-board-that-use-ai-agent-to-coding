package consumer

import (
	"testing"
)

func TestResolveObservabilityServiceName_NoDoublePrefix(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "")
	got := resolveObservabilityServiceName("task-events-cloud-server-stopped-1-process-server-stop")
	want := "task-events-cloud-server-stopped-1-process-server-stop"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveObservabilityServiceName_PrefixesShortName(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "")
	got := resolveObservabilityServiceName("cloud")
	want := "task-events-cloud"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveObservabilityServiceName_PrefersOTELEnv(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "task-events-sse-message-1-send-sse-message")
	got := resolveObservabilityServiceName("task-events-task-events-sse-message-1-send-sse-message")
	want := "task-events-sse-message-1-send-sse-message"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveObservabilityServiceName_EmptyFallsBack(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "")
	got := resolveObservabilityServiceName("  ")
	if got != "task-events" {
		t.Fatalf("got %q", got)
	}
}
