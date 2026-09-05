package tracelog

import (
	"encoding/json"
	"testing"
)

func TestTraceIDFromEnvelopeData(t *testing.T) {
	raw := json.RawMessage(`{"trace_id":"1cd1a1cc-e64d-4325-8b31-caabdd8aa74d","task_id":"1"}`)
	got := TraceIDFromEnvelopeData(raw)
	if got != "1cd1a1cc-e64d-4325-8b31-caabdd8aa74d" {
		t.Fatalf("got %q", got)
	}
}

func TestOtelTraceIDHex_uuid(t *testing.T) {
	got := OtelTraceIDHex("1cd1a1cc-e64d-4325-8b31-caabdd8aa74d")
	want := "1cd1a1cce64d43258b31caabdd8aa74d"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
