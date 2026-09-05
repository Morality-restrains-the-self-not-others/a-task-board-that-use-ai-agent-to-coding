package main

import (
	"tracelog"

	"taskEvents/config"
	"taskEvents/internal/handlers/taskpostexpiryscan"
)

func main() {
	_, consumerCfg, _, err := config.LoadIntent("task_post_expiry_scan", "1_expire_posts")
	if err != nil {
		tracelog.Fatal("task_post_expiry_scan", "config", err)
	}
	serviceName := config.BinaryName("task_post_expiry_scan", "1_expire_posts")
	if serviceName == "" {
		serviceName = "task-events-task-post-expiry-scan-1-expire-posts"
	}
	taskpostexpiryscan.Run(serviceName, consumerCfg.Host, consumerCfg.Port)
}
