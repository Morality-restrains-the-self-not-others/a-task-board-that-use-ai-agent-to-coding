package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"tracelog"
)

var repoRoot string

func main() {
	var err error
	repoRoot, err = findMonorepoRoot()
	if err != nil {
		log.Fatalf("[relayToTrae] %v", err)
	}

	if err := applyPortConfig(repoRoot); err != nil {
		log.Printf("[relayToTrae] warning: failed to apply port config: %v", err)
	}

	initTimeouts()
	tracelog.Init("go-relay")
	shutdownOtel, err := tracelog.InitOtel(context.Background(), "go-relay")
	if err != nil {
		log.Printf("[relayToTrae] otel disabled: %v", err)
	} else {
		defer func() { _ = shutdownOtel(context.Background()) }()
	}

	// Kill any existing processes on the relay and onlineService ports.
	relayPortStr := strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_PORT"))
	if relayPortStr == "" {
		relayPortStr = "8797"
	}
	relayPort, err := strconv.Atoi(relayPortStr)
	if err != nil || relayPort <= 0 {
		log.Fatalf("[relayToTrae] invalid RELAY_TO_TRAE_PORT %q", relayPortStr)
	}
	onlinePort := pickPort()

	killPortProcess(relayPort)
	killPortProcess(onlinePort)

	// Find onlineServiceJS run.sh and log discovery.
	runSh := onlineServiceRunSh(repoRoot)
	if _, err := os.Stat(runSh); err != nil {
		log.Printf("[relayToTrae] warning: onlineServiceJS not found at %s", runSh)
	}

	// Graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("[relayToTrae] shutting down...")
		stopRunning()
		os.Exit(0)
	}()

	host := strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_HOST"))
	if host == "" {
		host = "0.0.0.0"
	}

	log.Printf("[relayToTrae] monorepo root: %s", repoRoot)
	log.Printf("[relayToTrae] onlineServiceJS run.sh: %s", runSh)
	log.Printf("[relayToTrae] onlineService port: %d", onlinePort)

	if err := startServer(host, relayPort); err != nil {
		log.Fatalf("[relayToTrae] server error: %v", err)
	}
}

func killPortProcess(port int) {
	killPortListeners(port)
}

// findMonorepoRoot walks upward from start until trae-agent/onlineServiceJS/run.sh exists.
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

// Helper to locate the monorepo root from cwd or the executable (supports bin/ layout).
func findMonorepoRoot() (string, error) {
	if root := strings.TrimSpace(os.Getenv("DEPLOY_ROOT")); root != "" {
		if found, err := findMonorepoRootFrom(root); err == nil {
			return found, nil
		}
	}
	if conf := strings.TrimSpace(os.Getenv("CONF_ROOT")); conf != "" {
		start := conf
		if filepath.Base(conf) == "conf" {
			start = filepath.Dir(conf)
		}
		if found, err := findMonorepoRootFrom(start); err == nil {
			return found, nil
		}
	}
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
