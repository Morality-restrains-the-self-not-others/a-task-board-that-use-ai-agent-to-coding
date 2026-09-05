package main

import (
	"net"
	"net/http"
	"os"
	"strconv"
)

// PidHeader is set on every response so runAll canary restart can tell which
// process answered a health probe while two listeners share a port (ADR-0058).
// Inlined here (rather than reusing shareLib/tracelog) so runAll — whose only
// need from tracelog is this canary listener — does not pull the OTel indirect
// dependency into its module graph (OPT-20260903-003). tracelog keeps its own
// copy for managed services that serve health endpoints through ListenAndServe.
const PidHeader = "X-RunAll-Pid"

// WithPidHeader injects X-RunAll-Pid on every response.
func WithPidHeader(next http.Handler) http.Handler {
	pid := strconv.Itoa(os.Getpid())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(PidHeader, pid)
		next.ServeHTTP(w, r)
	})
}

// ReusePortListenConfig returns a ListenConfig that sets SO_REUSEADDR and,
// on Linux, SO_REUSEPORT so a canary peer can bind the same port.
func ReusePortListenConfig() net.ListenConfig {
	return net.ListenConfig{Control: reusePortControl}
}
