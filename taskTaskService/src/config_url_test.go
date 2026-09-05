package main

import "testing"

func TestResolveLoopbackServiceURLDefaultsEmptyHostAndPort(t *testing.T) {
	got := resolveLoopbackServiceURL("", 0, 8019)
	if got != "http://127.0.0.1:8019" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveLoopbackServiceURLRewritesListenAnyHost(t *testing.T) {
	got := resolveLoopbackServiceURL("0.0.0.0", 8019, 8019)
	if got != "http://127.0.0.1:8019" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveLoopbackServiceURLKeepsExplicitLoopback(t *testing.T) {
	got := resolveLoopbackServiceURL("127.0.0.1", 8019, 0)
	if got != "http://127.0.0.1:8019" {
		t.Fatalf("got %q", got)
	}
}
