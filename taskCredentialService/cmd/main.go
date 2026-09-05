package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"confload"
	"taskCredentialService/infrastructure"
	"taskCredentialService/interfaces"
	"tracelog"
)

func main() {
	monorepoRoot, err := findMonorepoRoot()
	if err != nil {
		log.Fatalf("[task-credential-service] %v", err)
	}

	host, port, gitoauthBase := loadListenConfig(monorepoRoot)

	// Compose application services (dependency injection).
	svc, err := infrastructure.Compose(monorepoRoot, gitoauthBase, 5)
	if err != nil {
		log.Fatalf("[task-credential-service] composition failed: %v", err)
	}
	defer svc.Shutdown()

	// Set up HTTP routes.
	tracelog.Init("task-credential-service")
	h := interfaces.NewHandlers(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	addr := fmt.Sprintf("%s:%s", host, port)
	log.Printf("[task-credential-service] listening on %s", addr)
	log.Printf("[task-credential-service] monorepo root: %s", monorepoRoot)

	if err := tracelog.ListenAndServe(addr, tracelog.Middleware(mux)); err != nil {
		log.Fatalf("[task-credential-service] server error: %v", err)
	}
}

func loadListenConfig(monorepoRoot string) (host, port, gitoauthBase string) {
	host = strings.TrimSpace(os.Getenv("TASK_CREDENTIAL_SERVICE_HOST"))
	port = strings.TrimSpace(os.Getenv("TASK_CREDENTIAL_SERVICE_PORT"))
	gitoauthBase = strings.TrimSpace(os.Getenv("GITOAUTH_BASE_URL"))

	var block struct {
		Host         string `yaml:"host"`
		Port         int    `yaml:"port"`
		GitoauthBase string `yaml:"gitoauth_base"`
	}
	if err := confload.ReadAppConfig(monorepoRoot, "container/task-credential-service", &block); err == nil {
		if host == "" && strings.TrimSpace(block.Host) != "" {
			host = strings.TrimSpace(block.Host)
		}
		if port == "" && block.Port > 0 {
			port = fmt.Sprintf("%d", block.Port)
		}
		if gitoauthBase == "" && strings.TrimSpace(block.GitoauthBase) != "" {
			gitoauthBase = strings.TrimSpace(block.GitoauthBase)
		}
	}

	if host == "" {
		host = "0.0.0.0"
	}
	if port == "" {
		port = "8015"
	}
	if gitoauthBase == "" {
		gitoauthBase = "http://127.0.0.1:8002"
	}
	return host, port, gitoauthBase
}

func findMonorepoRoot() (string, error) {
	return confload.FindMonorepoRoot()
}
