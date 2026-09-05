package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/internal/handlers/oidcsssoidempotencycleanup"
)

func main() {
	_, consumerCfg, _, err := config.LoadIntent("oidc_sso_idempotency_cleanup", "1_cleanup")
	if err != nil {
		tracelog.Fatal("oidc_sso_idempotency_cleanup", "config", err)
	}
	serviceName := config.BinaryName("oidc_sso_idempotency_cleanup", "1_cleanup")
	if serviceName == "" {
		serviceName = "task-events-oidc-sso-idempotency-cleanup-1-cleanup"
	}
	oidcsssoidempotencycleanup.Run(serviceName, consumerCfg.Host, consumerCfg.Port)
}
