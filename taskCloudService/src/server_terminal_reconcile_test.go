package main

import (
	"errors"
	"testing"
	"time"
)

func TestReconcileTerminalTaskCSCsReleasesCancelled(t *testing.T) {
	setupCloudTestDB(t)
	prevKafka := cfg.KafkaBootstrapServers
	cfg.KafkaBootstrapServers = ""
	t.Cleanup(func() { cfg.KafkaBootstrapServers = prevKafka })

	seedInboundCSC(t, "cfg-term-rec", "task-term-rec", "cmt-term-rec", "i-term-rec", "203.0.113.91")
	seedInboundCSC(t, "cfg-open-rec", "task-open-rec", "cmt-open-rec", "i-open-rec", "203.0.113.92")

	prevFn := lookupTaskTerminalKindsFn
	lookupTaskTerminalKindsFn = func(ids []string) (map[string]string, error) {
		out := map[string]string{}
		for _, id := range ids {
			if id == "task-term-rec" {
				out[id] = "cancelled"
			} else {
				out[id] = ""
			}
		}
		return out, nil
	}
	t.Cleanup(func() { lookupTaskTerminalKindsFn = prevFn })

	cleaned, err := reconcileTerminalTaskCSCs(time.Now().UTC())
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if cleaned != 1 {
		t.Fatalf("cleaned=%d want 1", cleaned)
	}

	var released int
	if err := db.QueryRow(`SELECT COALESCE(terminal_released,0) FROM cloud_server_configs WHERE id='cfg-term-rec'`).Scan(&released); err != nil {
		t.Fatal(err)
	}
	if released != 1 {
		t.Fatalf("cancelled CSC terminal_released=%d want 1", released)
	}
	var instance string
	if err := db.QueryRow(`SELECT COALESCE(instance_id,'') FROM cloud_server_configs WHERE id='cfg-term-rec'`).Scan(&instance); err != nil {
		t.Fatal(err)
	}
	if instance != "" {
		t.Fatalf("cancelled CSC instance_id=%q want empty", instance)
	}

	var openReleased int
	if err := db.QueryRow(`SELECT COALESCE(terminal_released,0) FROM cloud_server_configs WHERE id='cfg-open-rec'`).Scan(&openReleased); err != nil {
		t.Fatal(err)
	}
	if openReleased != 0 {
		t.Fatalf("open CSC must not be released, got %d", openReleased)
	}
}

func TestReconcileTerminalTaskCSCsLookupErrorNoOp(t *testing.T) {
	setupCloudTestDB(t)
	seedInboundCSC(t, "cfg-err", "task-err", "cmt-err", "i-err", "203.0.113.93")

	prevFn := lookupTaskTerminalKindsFn
	lookupTaskTerminalKindsFn = func(ids []string) (map[string]string, error) {
		return nil, errors.New("task service down")
	}
	t.Cleanup(func() { lookupTaskTerminalKindsFn = prevFn })

	_, err := reconcileTerminalTaskCSCs(time.Now().UTC())
	if err == nil {
		t.Fatal("want lookup error")
	}
	var released int
	if err := db.QueryRow(`SELECT COALESCE(terminal_released,0) FROM cloud_server_configs WHERE id='cfg-err'`).Scan(&released); err != nil {
		t.Fatal(err)
	}
	if released != 0 {
		t.Fatalf("must not release on lookup error, terminal_released=%d", released)
	}
}
