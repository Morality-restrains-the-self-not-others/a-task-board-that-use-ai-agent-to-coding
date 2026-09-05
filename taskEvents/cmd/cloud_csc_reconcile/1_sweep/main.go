package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/internal/handlers/cloud_csc_reconcile"
)

func main() {
	_, consumerCfg, _, err := config.LoadIntent("cloud_csc_reconcile", "1_sweep")
	if err != nil {
		tracelog.Fatal("cloud_csc_reconcile", "config", err)
	}
	serviceName := config.BinaryName("cloud_csc_reconcile", "1_sweep")
	if serviceName == "" {
		serviceName = "task-events-cloud-csc-reconcile-1-sweep"
	}
	cloud_csc_reconcile.Run(serviceName, consumerCfg.Host, consumerCfg.Port)
}
