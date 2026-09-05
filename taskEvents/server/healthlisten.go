package server

import (
	"context"
	"net/http"
	"time"

	"tracelog"
)

// ListenHealthReusePort serves handler with SO_REUSEPORT until ctx is cancelled,
// then http.Server.Shutdown (ADR-0058 canary overlap).
func ListenHealthReusePort(ctx context.Context, serviceName, addr string, handler http.Handler) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := tracelog.ListenAndServeContext(ctx, addr, handler); err != nil {
			tracelog.Fatal(serviceName, "http", err)
		}
	}()
	return done
}

// WaitHealthShutdown waits for ListenHealthReusePort to finish draining, or DrainTimeout.
func WaitHealthShutdown(done <-chan struct{}) {
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(tracelog.DrainTimeout + time.Second):
	}
}
