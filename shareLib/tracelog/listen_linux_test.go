//go:build linux

package tracelog

import (
	"context"
	"testing"
)

func TestReusePortListenConfig_TwoListenersSamePort(t *testing.T) {
	lc := ReusePortListenConfig()
	ln1, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln1.Close()
	addr := ln1.Addr().String()
	ln2, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		t.Fatalf("second listen on %s: %v", addr, err)
	}
	defer ln2.Close()
}
