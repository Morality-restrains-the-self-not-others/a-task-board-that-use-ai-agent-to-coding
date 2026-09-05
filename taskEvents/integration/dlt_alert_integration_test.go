//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"taskEvents/broker"
	"taskEvents/integration"
)

// TestCrossProcessDLTCooldown verifies that two consumer processes sharing the
// same Redis cooldown store emit at most one DLT report per window, and that
// the window reopens after the TTL elapses.
func TestCrossProcessDLTCooldown(t *testing.T) {
	host, port, db := integration.RedisAvailable(t)
	key := fmt.Sprintf("dlt-alert:test:%d", time.Now().UnixNano())

	storeA := broker.NewRedisDLTCooldownStore(host, port, db, key)
	storeB := broker.NewRedisDLTCooldownStore(host, port, db, key)
	defer func() {
		// Clean up the test key so it never lingers as a real alert key.
		_ = storeA.Del(context.Background())
	}()

	ttl := 2 * time.Second

	ok, err := storeA.Acquire(t.Context(), ttl)
	if err != nil {
		t.Fatalf("storeA acquire: %v", err)
	}
	if !ok {
		t.Fatal("storeA should win the first window")
	}

	ok, err = storeB.Acquire(t.Context(), ttl)
	if err != nil {
		t.Fatalf("storeB acquire: %v", err)
	}
	if ok {
		t.Fatal("storeB should be suppressed while storeA holds the window")
	}

	time.Sleep(ttl + 500*time.Millisecond)

	ok, err = storeB.Acquire(t.Context(), ttl)
	if err != nil {
		t.Fatalf("storeB acquire after ttl: %v", err)
	}
	if !ok {
		t.Fatal("storeB should win once the previous window expired")
	}
}
