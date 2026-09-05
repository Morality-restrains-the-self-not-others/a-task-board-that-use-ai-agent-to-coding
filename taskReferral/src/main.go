package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"confload"
	"tracelog"
)

func main() {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		log.Fatalf("[taskReferral] monorepo root: %v", err)
	}
	loadConfig(repoRoot)
	ensureWechatProfitSharingReceiver = ensureWechatProfitSharingReceiverLive
	deleteWechatProfitSharingReceiver = deleteWechatProfitSharingReceiverLive
	disableBillEligibility = disableBillEligibilityLive
	userHasServiceAccountBound = userHasServiceAccountBoundLive

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		log.Printf("[taskReferral] migrate mode (MySQL)")
		if err := openDB(cfg.MySQLDSN, repoRoot); err != nil {
			log.Fatalf("[taskReferral] migration failed: %v", err)
		}
		if err := runDataMigrateFromDir(cfg.MySQLDSN, repoRoot); err != nil {
			log.Fatalf("[taskReferral] dataMigrate: %v", err)
		}
		log.Println("[taskReferral] migration complete")
		os.Exit(0)
	}

	tracelog.Init("task-referral")

	if err := openDB(cfg.MySQLDSN, repoRoot); err != nil {
		log.Fatalf("[taskReferral] mysql: %v", err)
	}
	defer db.Close()

	if err := openAuthDB(cfg.AuthMySQLDSN); err != nil {
		log.Fatalf("[taskReferral] auth mysql: %v", err)
	}
	defer authDB.Close()

	if err := openBillDB(cfg.BillMySQLDSN); err != nil {
		log.Fatalf("[taskReferral] bill mysql: %v", err)
	}
	defer billDB.Close()

	mux := http.NewServeMux()
	mountRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("[taskReferral] listening on %s", addr)
	handler := tracelog.Middleware(corsMiddleware(mux))
	if err := tracelog.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func findMonorepoRoot() (string, error) {
	return confload.FindMonorepoRoot()
}
