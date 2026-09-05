package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/internal/handlers/queued_auto_run_scan"
)

func main() {
	_, consumerCfg, _, err := config.LoadIntent("queued_auto_run_scan", "1_dispatch")
	if err != nil {
		tracelog.Fatal("queued_auto_run_scan", "config", err)
	}
	serviceName := config.BinaryName("queued_auto_run_scan", "1_dispatch")
	if serviceName == "" {
		serviceName = "task-events-queued-auto-run-scan-1-dispatch"
	}
	queued_auto_run_scan.Run(serviceName, consumerCfg.Host, consumerCfg.Port)
}
