package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"confload"
	"taskGitOauth/infrastructure"
	"tracelog"
)

func main() {
	root, err := confload.FindMonorepoRoot()
	if err != nil {
		root, err = infrastructure.FindMonorepoRoot()
		if err != nil {
			log.Fatalf("[taskGitOauth] %v", err)
		}
	}
	cfg, err := infrastructure.LoadConfig(root)
	if err != nil {
		log.Fatalf("[taskGitOauth] config: %v", err)
	}
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		db, err := infrastructure.OpenDB(cfg.DatabasePath, root)
		if err != nil {
			log.Fatalf("[taskGitOauth] db open: %v", err)
		}
		defer db.Close()
		if err := db.EnsureSchema(root); err != nil {
			log.Fatalf("[taskGitOauth] migrate: %v", err)
		}
		log.Println("[taskGitOauth] migration complete")
		return
	}
	db, err := infrastructure.OpenDB(cfg.DatabasePath, root)
	if err != nil {
		log.Fatalf("[taskGitOauth] db open %s: %v", cfg.DatabasePath, err)
	}
	defer db.Close()

	app := NewApp(cfg, db)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	tracelog.Init("taskGitOauth")
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("[taskGitOauth] listening on %s db=%s", addr, cfg.DatabasePath)
	if err := tracelog.ListenAndServe(addr, tracelog.Middleware(app.gatewayUserMiddleware(mux))); err != nil {
		log.Printf("[taskGitOauth] server error: %v", err)
		os.Exit(1)
	}
}
