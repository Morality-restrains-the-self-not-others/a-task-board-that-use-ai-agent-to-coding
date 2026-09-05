package main

import (
	"embed"
	"encoding/json"
	"log"
	"net/http"
)

//go:embed index.html
var indexHTML embed.FS

func registerUIHandlers(mux *http.ServeMux, store *StatusStore, runner *Runner) {
	registerUIHandlersWithImpact(mux, store, runner)
}

func registerUIHandlersWithImpact(mux *http.ServeMux, store *StatusStore, runner *Runner, impactEvaluator ...streamImpactEvaluator) {
	if len(impactEvaluator) > 0 {
		store.SetImpactEvaluator(impactEvaluator[0])
	}
	mux.HandleFunc("/api/streams", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(store.AllStreams()); err != nil {
			log.Printf("[ui] encode streams: %v", err)
		}
	})

	mux.HandleFunc("/api/test/step", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleTestRequest(w, r, store, runner, true)
	})

	mux.HandleFunc("/api/test/stream", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleTestRequest(w, r, store, runner, false)
	})

	mux.HandleFunc("/api/domain-order", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleDomainOrderRequest(w, r, store)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := indexHTML.ReadFile("index.html")
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})
}

type testRequestBody struct {
	Stream string `json:"stream"`
	Step   string `json:"step"`
}

type domainOrderRequestBody struct {
	DomainOrder []string `json:"domain_order"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[ui] encode json: %v", err)
	}
}

func handleTestRequest(w http.ResponseWriter, r *http.Request, store *StatusStore, runner *Runner, isStep bool) {
	var body testRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Stream == "" {
		http.Error(w, "stream is required", http.StatusBadRequest)
		return
	}
	if isStep && body.Step == "" {
		http.Error(w, "step is required", http.StatusBadRequest)
		return
	}
	if err := runner.validateTestTarget(body.Stream, body.Step, isStep); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !store.TryAcquire() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": errBusy.Error()})
		return
	}

	runner.runAsync(func() {
		defer store.Release()
		ctx := runner.testContext()
		var err error
		if isStep {
			err = runner.runStep(ctx, body.Stream, body.Step)
		} else {
			err = runner.runStream(ctx, body.Stream)
		}
		if err != nil {
			log.Printf("[runner] %v", err)
		}
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func handleDomainOrderRequest(w http.ResponseWriter, r *http.Request, store *StatusStore) {
	var body domainOrderRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if len(body.DomainOrder) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "domain_order is required"})
		return
	}
	if err := store.ReorderDomainsAndPersist(body.DomainOrder); err != nil {
		if err == errBusy {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, store.AllStreams())
}

func startUIServer(store *StatusStore, runner *Runner, port string) *http.Server {
	return startUIServerWithImpact(store, runner, port)
}

func startUIServerWithImpact(store *StatusStore, runner *Runner, port string, impactEvaluator ...streamImpactEvaluator) *http.Server {
	mux := http.NewServeMux()
	registerUIHandlersWithImpact(mux, store, runner, impactEvaluator...)
	srv := &http.Server{Addr: port, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[ui] server error: %v", err)
		}
	}()
	return srv
}
