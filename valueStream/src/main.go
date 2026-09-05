package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	domainservices "valueStream/domain/services"
)

// startupTraceID returns a non-empty id for process-level errors (no HTTP request context).
func startupTraceID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("startup-%d", time.Now().UnixNano())
	}
	return "startup-" + hex.EncodeToString(b[:])
}

// emitStructuredError writes one JSON log line for Promtail/Loki level=error filters.
func emitStructuredError(msg string, err error) {
	payload := map[string]string{
		"ts":       time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"level":    "error",
		"msg":      fmt.Sprintf("%s: %v", msg, err),
		"service":  "value-stream",
		"trace_id": startupTraceID(),
	}
	b, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		fmt.Fprintf(os.Stderr, "{\"level\":\"error\",\"service\":\"value-stream\",\"msg\":%q}\n", fmt.Sprintf("%s: %v", msg, err))
		return
	}
	fmt.Fprintln(os.Stderr, string(b))
}

func main() {
	configPath := flag.String("config", "", "Path to YAML configuration file (required)")
	uiPort := flag.String("ui-port", ":9998", "Web UI listen address")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "error: --config is required")
		flag.Usage()
		os.Exit(2)
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		emitStructuredError("Config error", err)
		os.Exit(1)
	}

	impactRepo := NewGitCommitChangeRepository(cfg.ConfigDir, cfg.Runner.WorkingDir)
	impactService := domainservices.NewHeadCommitImpactService(impactRepo)
	store := NewStatusStore(cfg, impactService)
	runner := NewRunner(cfg, store)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	runner.SetTestContext(ctx)

	srv := startUIServerWithImpact(store, runner, *uiPort, impactService)
	log.Printf("valueStream UI: http://localhost%s", *uiPort)
	log.Printf("[valueStream] Config: %s", *configPath)

	<-ctx.Done()
	log.Println("[valueStream] Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[valueStream] HTTP shutdown: %v", err)
	}
	if err := runner.WaitIdle(shutdownCtx); err != nil {
		log.Printf("[valueStream] waiting for in-flight tests: %v", err)
	}
}
