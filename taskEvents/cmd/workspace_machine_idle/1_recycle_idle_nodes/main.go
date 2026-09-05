package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/internal/handlers/workspacemachineidle"
)

func main() {
	_, consumerCfg, _, err := config.LoadIntent("workspace_machine_idle", "1_recycle_idle_nodes")
	if err != nil {
		tracelog.Fatal("workspace_machine_idle", "config", err)
	}
	serviceName := config.BinaryName("workspace_machine_idle", "1_recycle_idle_nodes")
	if serviceName == "" {
		serviceName = "task-events-workspace-machine-idle-1-recycle-idle-nodes"
	}
	workspacemachineidle.Run(serviceName, consumerCfg.Host, consumerCfg.Port)
}
