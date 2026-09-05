package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/internal/handlers/billingprofitsharing"
)

func main() {
	_, consumerCfg, _, err := config.LoadIntent("billing_profit_sharing_scan", "1_process_pending")
	if err != nil {
		tracelog.Fatal("billing_profit_sharing_scan", "config", err)
	}
	serviceName := config.BinaryName("billing_profit_sharing_scan", "1_process_pending")
	if serviceName == "" {
		serviceName = "task-events-billing-profit-sharing-scan-1-process-pending"
	}
	billingprofitsharing.Run(serviceName, consumerCfg.Host, consumerCfg.Port)
}
