package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/internal/handlers/referralcodeexpiry"
)

func main() {
	_, consumerCfg, _, err := config.LoadIntent("referral_code_expiry_scan", "1_expire")
	if err != nil {
		tracelog.Fatal("referral_code_expiry_scan", "config", err)
	}
	serviceName := config.BinaryName("referral_code_expiry_scan", "1_expire")
	if serviceName == "" {
		serviceName = "task-events-referral-code-expiry-scan-1-expire"
	}
	referralcodeexpiry.Run(serviceName, consumerCfg.Host, consumerCfg.Port)
}
