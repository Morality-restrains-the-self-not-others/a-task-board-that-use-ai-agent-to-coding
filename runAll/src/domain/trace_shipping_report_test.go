package domain

import "testing"

func TestNewProbeTraceID(t *testing.T) {
	got := NewProbeTraceID(1738000000123)
	want := "runall-ship-1738000000123"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if len(got) < 8 {
		t.Fatalf("probe id too short: %q", got)
	}
}
