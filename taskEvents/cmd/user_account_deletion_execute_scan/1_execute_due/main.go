package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/internal/handlers/useraccountdeletionexecute"
)

func main() {
	_, consumerCfg, _, err := config.LoadIntent("user_account_deletion_execute_scan", "1_execute_due")
	if err != nil {
		tracelog.Fatal("user_account_deletion_execute_scan", "config", err)
	}
	serviceName := config.BinaryName("user_account_deletion_execute_scan", "1_execute_due")
	if serviceName == "" {
		serviceName = "task-events-user-account-deletion-execute-scan-1-execute-due"
	}
	useraccountdeletionexecute.Run(serviceName, consumerCfg.Host, consumerCfg.Port)
}
