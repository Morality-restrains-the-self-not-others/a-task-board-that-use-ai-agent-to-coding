package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckExec_UsesWorkDir(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "probe-ok")
	script := filepath.Join(dir, "probe.sh")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\ntest -f probe-ok\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if err := os.WriteFile(marker, []byte("1"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := checkExec(ctx, "bash probe.sh", dir); err != nil {
		t.Fatalf("checkExec with WorkDir: %v", err)
	}
	if err := checkExec(ctx, "bash probe.sh", ""); err == nil {
		t.Fatal("expected checkExec without WorkDir to fail finding probe.sh")
	}
}

func TestCheckTCP_Healthy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := checkTCP(ctx, addr); err != nil {
		t.Fatalf("checkTCP(%q): %v", addr, err)
	}
}

func TestCheckTCP_Unreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := checkTCP(ctx, "127.0.0.1:1"); err == nil {
		t.Fatal("expected error for closed port")
	}
}

func TestWaitHealthy_TCPProbe(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	cfg := HealthCheck{
		TCP:     ln.Addr().String(),
		Timeout: 5,
		Retries: 5,
		Backoff: Backoff{Initial: 0.05, Max: 0.1, Multiplier: 1.5},
	}

	if err := waitHealthy(context.Background(), cfg); err != nil {
		t.Fatalf("waitHealthy tcp: %v", err)
	}
}
