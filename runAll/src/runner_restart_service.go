package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// restartService 执行单服务重启：claim Restarting →（可选编译）→ 停止旧进程 →
// 端口释放 → 启动与健康检查。从 runner.go 独立成文件（OPT-20260903-007）。
func (r *Runner) restartService(ctx context.Context, name string) error {
	if restartServiceTestHook != nil {
		restartServiceTestHook(name)
	}
	svc := r.findService(name)
	if svc == nil {
		return fmt.Errorf("service %q not found", name)
	}

	previousStatus, err := claimServiceRestart(r.store, name)
	if err != nil {
		return err
	}
	log.Printf("[%s] restart claimed from %s", name, previousStatus)
	r.stopMonitoring(name)

	compile := compileThenSwapFrom(ctx)
	buildCmd := resolveBuildCommand(*svc)

	// 热替换（OPT-20260820-004）：静态 detach 服务（如 taskFE Docker nginx 常驻）在
	// 健康时重启不再 stop→build→start，而是保留在线容器继续服务 → 构建切 public/html
	// symlink → 复用在线容器健康检查 → 经 restart_reload_command 生效 bind-mount 配置。
	// 效果：build 期间公网不空窗，APISIX 不再因 compose stop 出现数秒 502。
	// 仅在服务当前 Healthy 且 detach 启动时启用；构建失败时旧容器继续服务旧产物（不宕机）。
	hotReplace := svc.SkipStopOnRestart && previousStatus == StatusHealthy && svc.IsDetachLaunch()
	if hotReplace {
		log.Printf("[%s] skip_stop_on_restart: keeping live container serving while build runs (hot replace)", name)
	}

	// ADR-0027 compile-then-swap：需要新二进制时先编译、成功后再停旧进程。
	// 普通 ↻ / 全部重启 compile=false，跳过 build_command（只拉 last-good）。
	if compile && buildCmd != "" && !hotReplace {
		r.store.Update(name, StatusBuilding, "")
		log.Printf("[%s] compile-then-swap: building before stop", name)
		if err := r.runBuild(ctx, svc, buildCmd); err != nil {
			log.Printf("[%s] compile-then-swap: build failed, keeping last-good process: %v", name, err)
			r.store.Update(name, previousStatus, "")
			if previousStatus == StatusHealthy {
				r.startMonitoring(ctx, *svc)
			}
			return err
		}
	}

	if !hotReplace {
		oldCmd := r.trackedCmd(name)
		hasLive := oldCmd != nil && oldCmd.Process != nil
		if shouldCanaryOverlap(previousStatus, hasLive) {
			log.Printf("[%s] canary overlap: starting peer before draining pid=%d", name, oldCmd.Process.Pid)
			node := &ServiceNode{Service: *svc, allowOverlapStart: true}
			if err := r.startAndCheck(ctx, node); err != nil {
				log.Printf("[%s] canary_overlap_fallback: %v", name, err)
				r.appendLifecycleLog(name, fmt.Sprintf("canary_overlap_fallback: %v", err))
				failedNew := r.trackedCmd(name)
				if failedNew != oldCmd {
					r.restoreTrackedCmd(name, oldCmd, failedNew)
				}
				reclaimRestartAfterCanaryFallback(r.store, name)
			} else {
				r.drainOldProcess(name, oldCmd)
				if err := r.ensureHealthyAfterManualStart(name); err != nil {
					return err
				}
				r.startMonitoring(ctx, *svc)
				return nil
			}
		}
		// 停止：stop 命令 → 进程组终止 → 端口级兜底终止 → 等待端口释放。
		// 新二进制必须真正起来：stop 之后 forceFreshStart，避免旧监听被误判为已切换。
		if err := r.runStopCommandIfConfigured(ctx, svc); err != nil {
			r.store.Update(name, previousStatus, err.Error())
			return err
		}
		if !r.stopProcess(name) {
			log.Printf("[%s] no tracked process handle for stop, relying on stop_command/port sweep", name)
		}
		// 端口级兜底：有活跃监听则走 ReleasePorts 升级终止（覆盖健康已失败但仍占端口、
		// 热替换收养、crontab watchdog 抢拉等无 cmd 句柄场景）；否则再按健康探测清场。
		if r.probeActivePortListeners(svc) {
			if err := r.ensureServicePortsReleased(ctx, svc); err != nil {
				r.store.Update(name, StatusFailed, err.Error())
				r.store.SetPID(name, 0)
				return err
			}
		} else if err := r.ensureServiceNotReachable(ctx, svc); err != nil {
			r.store.Update(name, StatusFailed, err.Error())
			r.store.SetPID(name, 0)
			return err
		}
		if err := r.waitServicePortsFree(svc); err != nil {
			r.store.Update(name, StatusFailed, err.Error())
			r.store.SetPID(name, 0)
			return err
		}
	}

	// 热替换：容器保持在线时才在此编译（与 skip_stop 同窗）。
	if compile && buildCmd != "" && hotReplace {
		r.store.Update(name, StatusBuilding, "")
		log.Printf("[%s] building...", name)
		if err := r.runBuild(ctx, svc, buildCmd); err != nil {
			log.Printf("[%s] hot-replace build failed, keeping live container: %v", name, err)
			r.store.Update(name, previousStatus, "")
			r.startMonitoring(ctx, *svc)
			return err
		}
	}

	// 启动与健康检查。热替换 forceFreshStart=false：在线且健康则「跳过启动」复用
	// 容器（内容已随 symlink 切换生效）；非热替换不得跳过启动。
	node := &ServiceNode{Service: *svc, forceFreshStart: !hotReplace}
	if err := r.startAndCheck(ctx, node); err != nil {
		r.stopMonitoring(name)
		_ = r.stopProcess(name)
		r.store.SetPID(name, 0)
		return err
	}
	// 热替换后：在复用容器内生效 bind-mount 配置变更（如 nginx -s reload），best-effort。
	if hotReplace && strings.TrimSpace(svc.RestartReloadCommand) != "" {
		if rerr := r.runLifecycleCommand(ctx, svc, svc.RestartReloadCommand); rerr != nil {
			log.Printf("[%s] hot-replace reload command failed (service stays healthy): %v", svc.Name, rerr)
		}
	}
	if err := r.ensureHealthyAfterManualStart(name); err != nil {
		return err
	}

	// Resume continuous monitoring
	r.startMonitoring(ctx, *svc)
	return nil
}

// waitServicePortsFree 等待服务全部端口释放（优雅关闭缓冲），上限 servicePortReleaseWait。
// 端口仍未释放时返回错误——禁止在占用端口上继续构建/启动（避免 EADDRINUSE）。
func (r *Runner) waitServicePortsFree(svc *Service) error {
	if svc == nil {
		return fmt.Errorf("service is required")
	}
	for _, port := range resolveServicePorts(svc) {
		if err := r.waitForPortFree(port, servicePortReleaseWait); err != nil {
			log.Printf("[%s] port %s not released after %v: %v", svc.Name, port, servicePortReleaseWait, err)
			r.appendLifecycleLog(svc.Name, fmt.Sprintf("端口 %s %v 未释放，中止重启", port, servicePortReleaseWait))
			return fmt.Errorf("[%s] %w", svc.Name, err)
		}
	}
	return nil
}

// servicePortReleaseWait 停止后等待端口释放的时长（kill-first 停止阶段与
// forceFreshStart 启动兜底共用）。包级变量以便单测缩短等待窗口。
var servicePortReleaseWait = 30 * time.Second
