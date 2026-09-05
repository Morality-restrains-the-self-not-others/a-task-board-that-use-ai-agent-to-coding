package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"tracelog"
)

func main() {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		log.Fatalf("[taskBill] monorepo root: %v", err)
	}
	loadConfig(repoRoot)
	loadWechatPayConfig(repoRoot)
	loadPaypalConfig(repoRoot)
	initInvoiceFileMediaRoot(repoRoot)
	tracelog.Init("task-bill")
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		// Explicit migrate CLI only — business server must not auto-migrate (9999 / migrate.sh).
		if err := runDataMigrateFromDir(cfg.MySQLDSN, repoRoot); err != nil {
			log.Fatalf("[taskBill] migrations: %v", err)
		}
		return
	}

	if err := openDB(cfg.MySQLDSN); err != nil {
		log.Fatalf("[taskBill] mysql: %v", err)
	}
	defer db.Close()

	// 幂等：补建已有赠送资源的订单记录（新手礼包等）
	go func() {
		n, err := backfillGrantOrders()
		if err != nil {
			log.Printf("[taskBill] backfillGrantOrders: %v", err)
		} else if n > 0 {
			log.Printf("[taskBill] backfillGrantOrders: created %d orders", n)
		}
	}()

	mux := http.NewServeMux()
	mountRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("[taskBill] listening on %s", addr)
	if err := tracelog.ListenAndServe(addr, tracelog.Middleware(corsMiddleware(tracelog.MetricsMiddleware(mux)))); err != nil {
		log.Fatal(err)
	}
}
