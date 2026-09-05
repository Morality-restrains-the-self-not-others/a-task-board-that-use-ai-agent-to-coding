package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"tracelog"
)

func main() {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		log.Fatalf("[taskAgentSupport] %v", err)
	}
	initConfig(repoRoot)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	mux := http.NewServeMux()
	mountRoutes(mux)

	tracelog.Init("task-agent-support")
	slog.Info("agent support listening", "addr", addr)
	handler := tracelog.Middleware(corsMiddleware(mux))
	if err := tracelog.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("[taskAgentSupport] server error: %v", err)
	}
}

func findMonorepoRootFrom(start string) (string, error) {
	marker := filepath.Join("trae-agent", "onlineServiceJS", "run.sh")
	dir := filepath.Clean(start)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("trae-agent/onlineServiceJS/run.sh not found")
}

func findMonorepoRoot() (string, error) {
	if cwd, err := os.Getwd(); err == nil {
		if root, err := findMonorepoRootFrom(cwd); err == nil {
			return root, nil
		}
	}
	if execPath, err := os.Executable(); err == nil {
		if root, err := findMonorepoRootFrom(filepath.Dir(execPath)); err == nil {
			return root, nil
		}
	}
	return "", fmt.Errorf("trae-agent/onlineServiceJS/run.sh not found")
}
