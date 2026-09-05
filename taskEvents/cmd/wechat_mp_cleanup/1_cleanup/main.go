package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/internal/handlers/wechatmpcleanup"
)

func main() {
	_, consumerCfg, _, err := config.LoadIntent("wechat_mp_cleanup", "1_cleanup")
	if err != nil {
		tracelog.Fatal("wechat_mp_cleanup", "config", err)
	}
	serviceName := config.BinaryName("wechat_mp_cleanup", "1_cleanup")
	if serviceName == "" {
		serviceName = "task-events-wechat-mp-cleanup-1-cleanup"
	}
	wechatmpcleanup.Run(serviceName, consumerCfg.Host, consumerCfg.Port)
}
