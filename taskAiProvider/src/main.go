package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"confload"
	"taskAiProvider/infrastructure"
	"tracelog"
)

func main() {
	const service = "ai-provider"
	tracelog.Init(service)

	root, err := confload.FindMonorepoRoot()
	if err != nil {
		root, err = infrastructure.FindMonorepoRoot()
		if err != nil {
			tracelog.Fatal(service, "monorepo root", err)
		}
	}
	cfg, err := infrastructure.LoadConfig(root)
	if err != nil {
		tracelog.Fatal(service, "config", err)
	}
	db, err := infrastructure.OpenDB(cfg.DatabasePath)
	if err != nil {
		tracelog.Fatal(service, fmt.Sprintf("db open %s", cfg.DatabasePath), err)
	}
	defer db.Close()

	// OPT-20260831-019: missing SPA dist must Fatal before listen — handleSPA
	// silently ServeFile-404s every non-/api path, and runAll 直启 ./bin/taskAiProvider
	// 会绕过 run.sh start 的 dist 硬失败。启动失败比线上 404 更易发现。
	if err := ensureFrontendDist(cfg.FrontendDistDir); err != nil {
		tracelog.Fatal(service, "frontend dist missing", err)
	}

	app := NewApp(cfg, db)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	tracelog.Emit("info", fmt.Sprintf("listening on %s db=%s", addr, cfg.DatabasePath), "", nil)
	if err := tracelog.ListenAndServe(addr, tracelog.Middleware(mux)); err != nil {
		tracelog.Fatal(service, "server", err)
	}
}

// ensureFrontendDist returns an error when the SPA bundle (index.html) is absent,
// so ai-provider fails before entering listen instead of silently 404-ing /.
func ensureFrontendDist(distDir string) error {
	if _, err := os.Stat(filepath.Join(distDir, "index.html")); err != nil {
		return fmt.Errorf("%s — run build.sh to build the SPA", distDir)
	}
	return nil
}
