package main

import (
	"context"
	"strings"
	"testing"
)

func TestProgressPhaseForContextErr(t *testing.T) {
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if got := progressPhaseForContextErr(ctx); got != "cancelled" {
			t.Fatalf("phase = %q, want cancelled", got)
		}
	})

	t.Run("deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()
		<-ctx.Done()
		if got := progressPhaseForContextErr(ctx); got != "error" {
			t.Fatalf("phase = %q, want error", got)
		}
	})
}

func TestProgressBroadcaster_PublishStampsRunID(t *testing.T) {
	b := NewProgressBroadcaster()
	runID := "run-stamp-1"
	b.Publish(runID, StartAllProgressEvent{
		Total: 3, Started: 1, Remaining: 2, Phase: "progress", Operation: "build", Current: "svc-a",
	})

	// Latest snapshot carries the run_id for late subscribers / page refresh.
	ev, ok := b.Latest(runID)
	if !ok {
		t.Fatal("expected latest snapshot")
	}
	if ev.RunID != runID {
		t.Fatalf("latest RunID = %q, want %q", ev.RunID, runID)
	}
	if sse := ev.ToSSE(); !strings.Contains(sse, `"run_id":"run-stamp-1"`) {
		t.Fatalf("ToSSE = %q, want run_id in payload", sse)
	}

	// Live subscriber sees the stamped run_id too.
	ch := b.Subscribe(runID)
	select {
	case got := <-ch:
		if got.RunID != runID {
			t.Fatalf("subscriber RunID = %q, want %q", got.RunID, runID)
		}
	default:
		t.Fatal("expected immediate snapshot delivery to late subscriber")
	}

	// An explicit RunID on the event is preserved, not overwritten.
	b.Publish(runID, StartAllProgressEvent{Phase: "done", RunID: "explicit-run"})
	if ev, ok := b.Latest(runID); !ok || ev.RunID != "explicit-run" {
		t.Fatalf("explicit RunID not preserved: %#v", ev)
	}

	b.CloseRun(runID)
}

func TestProgressBroadcaster_LatestForLateSubscriber(t *testing.T) {
	b := NewProgressBroadcaster()
	runID := "run-1"
	b.Publish(runID, StartAllProgressEvent{
		Total: 3, Started: 1, Remaining: 2, Phase: "progress", Operation: "build", Current: "svc-a",
	})

	ev, ok := b.Latest(runID)
	if !ok {
		t.Fatal("expected latest snapshot")
	}
	if ev.Started != 1 || ev.Current != "svc-a" || ev.Operation != "build" {
		t.Fatalf("latest = %#v", ev)
	}

	ch := b.Subscribe(runID)
	select {
	case got := <-ch:
		if got.Started != 1 || got.Current != "svc-a" {
			t.Fatalf("late subscriber got %#v", got)
		}
	default:
		t.Fatal("expected immediate snapshot delivery to late subscriber")
	}

	b.CloseRun(runID)
	if _, ok := b.Latest(runID); ok {
		t.Fatal("latest should be cleared after CloseRun")
	}
}
