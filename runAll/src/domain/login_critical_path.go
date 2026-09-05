package domain

import (
	"context"
	"fmt"
)

// LoginCriticalPathServices is the dependency-ordered list of runAll services
// required for public WeChat/email login after a DB clear/init window.
// Order: auth (OAuth/token) → gateway (APISIX) → SPA (taskFE).
var LoginCriticalPathServices = []string{
	"task-auth",
	"task-gateway",
	"taskFE",
}

// ServiceStarter starts a runAll-managed service (idempotent when already healthy).
type ServiceStarter interface {
	IsServiceRunning(name string) bool
	StartService(ctx context.Context, name string) error
}

// EnsureLoginCriticalPath starts login-critical services in order.
// Already-running services are skipped. Failures are collected; returns
// started names, failed names, and a non-nil error if any start failed.
func EnsureLoginCriticalPath(
	ctx context.Context,
	starter ServiceStarter,
	onProgress ProgressCallback,
) (started []string, failed []string, err error) {
	if starter == nil {
		return nil, nil, fmt.Errorf("login critical path: starter is nil")
	}
	notify := func(msg string) {
		if onProgress != nil {
			onProgress(msg)
		}
	}
	notify("正在恢复登录关键路径服务 (task-auth → task-gateway → taskFE)...")
	for _, name := range LoginCriticalPathServices {
		if starter.IsServiceRunning(name) {
			notify(fmt.Sprintf("登录关键路径 %s 已在运行，跳过", name))
			continue
		}
		notify(fmt.Sprintf("登录关键路径 %s 未运行，正在启动...", name))
		if startErr := starter.StartService(ctx, name); startErr != nil {
			notify(fmt.Sprintf("登录关键路径 %s 启动失败: %v", name, startErr))
			failed = append(failed, name)
			continue
		}
		notify(fmt.Sprintf("登录关键路径 %s 已就绪", name))
		started = append(started, name)
	}
	if len(failed) > 0 {
		return started, failed, fmt.Errorf("login critical path not ready: failed=%v", failed)
	}
	notify("登录关键路径服务已全部就绪")
	return started, nil, nil
}
