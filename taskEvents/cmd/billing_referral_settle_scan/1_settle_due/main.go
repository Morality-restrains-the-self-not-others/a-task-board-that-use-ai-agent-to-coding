package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/internal/handlers/billingreferralsettle"
)

func main() {
	_, consumerCfg, _, err := config.LoadIntent("billing_referral_settle_scan", "1_settle_due")
	if err != nil {
		tracelog.Fatal("billing_referral_settle_scan", "config", err)
	}
	serviceName := config.BinaryName("billing_referral_settle_scan", "1_settle_due")
	if serviceName == "" {
		serviceName = "task-events-billing-referral-settle-scan-1-settle-due"
	}
	billingreferralsettle.Run(serviceName, consumerCfg.Host, consumerCfg.Port)
}
