package broker_test

import (
	"context"
	"testing"
	"time"

	"taskEvents/broker"
)

func TestDelayForAttemptGrowsUntilMax(t *testing.T) {
	initial := time.Second
	max := 30 * time.Second
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{1, time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{5, 16 * time.Second},
		{6, 30 * time.Second},
		{11, 30 * time.Second},
		{0, time.Second},
	}
	for _, tc := range cases {
		got := broker.DelayForAttempt(initial, max, tc.attempt)
		if got != tc.want {
			t.Fatalf("attempt=%d got %v want %v", tc.attempt, got, tc.want)
		}
	}
}

func TestBackoffWaitAttemptUsesDurableAttempt(t *testing.T) {
	b := broker.Backoff{Initial: 10 * time.Millisecond, Max: 40 * time.Millisecond}
	ctx := context.Background()
	start := time.Now()
	if err := b.WaitAttempt(ctx, 2); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 15*time.Millisecond {
		t.Fatalf("attempt 2 wait too short: %v", elapsed)
	}
}

func TestBackoffWaitIncreasesUntilMax(t *testing.T) {
	b := broker.Backoff{Initial: 10 * time.Millisecond, Max: 40 * time.Millisecond}
	ctx := context.Background()

	start := time.Now()
	if err := b.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	first := time.Since(start)
	if first < 8*time.Millisecond {
		t.Fatalf("first wait too short: %v", first)
	}

	start = time.Now()
	if err := b.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	second := time.Since(start)
	if second < 15*time.Millisecond {
		t.Fatalf("second wait too short: %v", second)
	}
}

func TestBackoffReset(t *testing.T) {
	b := broker.Backoff{Initial: 10 * time.Millisecond, Max: 40 * time.Millisecond}
	ctx := context.Background()
	_ = b.Wait(ctx)
	_ = b.Wait(ctx)
	b.Reset()
	start := time.Now()
	if err := b.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Fatalf("expected reset to initial wait, got %v", elapsed)
	}
}

func TestBackoffRespectsContextCancel(t *testing.T) {
	b := broker.Backoff{Initial: time.Second, Max: time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := b.Wait(ctx); err == nil {
		t.Fatal("expected context error")
	}
}

func TestConnectionState(t *testing.T) {
	state := broker.NewConnectionState(false)
	if state.Connected() {
		t.Fatal("expected disconnected")
	}
	state.SetConnected(true)
	if !state.Connected() {
		t.Fatal("expected connected")
	}
}
