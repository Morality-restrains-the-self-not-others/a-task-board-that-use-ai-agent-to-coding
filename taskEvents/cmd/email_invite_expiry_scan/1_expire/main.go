package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/internal/handlers/emailinviteexpiryscan"
)

func main() {
	_, consumerCfg, _, err := config.LoadIntent("email_invite_expiry_scan", "1_expire")
	if err != nil {
		tracelog.Fatal("email_invite_expiry_scan", "config", err)
	}
	serviceName := config.BinaryName("email_invite_expiry_scan", "1_expire")
	if serviceName == "" {
		serviceName = "task-events-email-invite-expiry-scan-1-expire"
	}
	emailinviteexpiryscan.Run(serviceName, consumerCfg.Host, consumerCfg.Port)
}
