package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"confload"
	"tracelog"
)

func main() {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		log.Fatalf("[taskAIEndPoint] %v", err)
	}
	initConfig(repoRoot)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	mux := http.NewServeMux()
	mountRoutes(mux)

	tracelog.Init("task-ai-endpoint")
	slog.Info("ai endpoint listening", "addr", addr, "cloud", cfg.CloudServiceURL, "credential", cfg.CredentialServiceURL)
	handler := tracelog.Middleware(corsMiddleware(mux))
	if err := tracelog.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("[taskAIEndPoint] server error: %v", err)
	}
}

func findMonorepoRoot() (string, error) {
	return confload.FindMonorepoRoot()
}
