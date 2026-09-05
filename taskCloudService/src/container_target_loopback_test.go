package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPreferLoopbackContainerBaseURLWhenLocalHealthy(t *testing.T) {
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health/" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer local.Close()

	// Force preferLoopback to probe the test server via rewriting port of a fake public origin.
	// preferLoopback always probes 127.0.0.1:<port from input URL>.
	u := local.URL // e.g. http://127.0.0.1:12345
	port := u[len("http://127.0.0.1:"):]
	publicOrigin := "http://203.0.113.9:" + port

	got := preferLoopbackContainerBaseURL(publicOrigin)
	want := "http://127.0.0.1:" + port
	if got != want {
		t.Fatalf("preferLoopback=%q want=%q", got, want)
	}
}

func TestPreferLoopbackContainerBaseURLKeepsRemoteWhenLocalDown(t *testing.T) {
	publicOrigin := "http://203.0.113.9:1"
	got := preferLoopbackContainerBaseURL(publicOrigin)
	if got != publicOrigin {
		t.Fatalf("preferLoopback=%q want unchanged %q", got, publicOrigin)
	}
}

func TestPreferLoopbackContainerBaseURLKeepsLoopback(t *testing.T) {
	in := "http://127.0.0.1:8765"
	if got := preferLoopbackContainerBaseURL(in); got != in {
		t.Fatalf("preferLoopback=%q want=%q", got, in)
	}
}
