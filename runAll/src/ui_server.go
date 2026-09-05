package main

import (
	"context"
	"log"
	"net/http"
)

func registerIndexHandler(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := statusHTML.ReadFile("status.html")
		if err != nil {
			log.Printf("[ui] read embedded html error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write(data); err != nil {
			log.Printf("[ui] write status html error: %v", err)
		}
	})
}

func uiListenerLost(runner *Runner, shutdownFunc context.CancelFunc, cause error) {
	log.Printf("[ui] UI listener lost: %v — exiting runAll without touching managed services", cause)
	if runner != nil {
		runner.SetSkipShutdownServices(true)
	}
	if shutdownFunc != nil {
		shutdownFunc()
	}
}

func startUIServer(store *StatusStore, runner *Runner, port string, shutdownFunc context.CancelFunc, exitReason *runAllExitReason) *http.Server {
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner, shutdownFunc, exitReason)

	// Request-level trace logging: every UI request carrying X-Trace-Id is
	// emitted as a structured log entry with trace_id, so Status-UI error
	// banners' data-traceId are queryable in Loki (OPT-20260820-009).
	handler := WithPidHeader(uiRequestTraceLogging(runner)(mux))
	srv := &http.Server{Addr: port, Handler: handler}
	lc := ReusePortListenConfig()
	ln, err := lc.Listen(context.Background(), "tcp", port)
	if err != nil {
		uiListenerLost(runner, shutdownFunc, err)
		return srv
	}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			// http.ErrServerClosed means srv.Close() ran during normal process
			// teardown; any other return means the UI port was lost (bind failure,
			// listener died) — assert process exit via the run loop's cancel func.
			uiListenerLost(runner, shutdownFunc, err)
		}
	}()
	return srv
}
