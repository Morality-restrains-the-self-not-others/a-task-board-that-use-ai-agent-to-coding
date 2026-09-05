package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
)

// settleStartupHealth 承接 startAndCheck 的启动健康收尾：金丝雀重叠就绪等待、
// 健康检查失败/成功后的 Retrying→Failed/Healthy CAS，以及进程被替换
// （superseded）时的静默返回。独立成文件以缩小 runner.go 巨石（OPT-20260903-007）。
func (r *Runner) settleStartupHealth(ctx context.Context, svc Service, node *ServiceNode, cmd *exec.Cmd) error {
	// Health check. Overlap must prove the new process group is listening
	// before trusting the shared health URL (old listener still answers).
	r.store.Update(svc.Name, StatusRetrying, "")
	if node.allowOverlapStart && node.overlapHadListeners {
		if peerErr := r.waitCanaryPeerReady(ctx, &svc, cmd.Process.Pid); peerErr != nil {
			log.Printf("[%s] canary overlap: new process group not listening: %v", svc.Name, peerErr)
			r.appendLifecycleLog(svc.Name, fmt.Sprintf("金丝雀：新进程未出现在监听集合: %v", peerErr))
			if r.startupStillOwnsProcess(svc.Name, cmd) {
				r.recordCanaryPeerFailure(svc.Name, peerErr)
				r.store.UpdateDependencyStatus(svc.Name, StatusFailed)
			}
			return peerErr
		}
	}
	err := waitHealthyWithLaunchCheck(ctx, cmd.Process, svc.HealthCheck.StartupProbeConfig(), svc.IsDetachLaunch())
	if err != nil {
		err = r.enrichStartupFailureWithRecentLogs(svc.Name, err)
		if !r.startupStillOwnsProcess(svc.Name, cmd) {
			log.Printf("[%s] health check failed after process was replaced; startup superseded", svc.Name)
			return nil
		}
		phase, code := classifyStartupFailure(err)
		committed := false
		if phase == "" || code == "" {
			committed = r.store.CompareAndSwapUpdate(svc.Name, StatusRetrying, StatusFailed, err.Error())
		} else {
			committed = r.store.RecordFailureIfStatus(svc.Name, StatusRetrying, phase, code, err.Error())
		}
		if !committed {
			log.Printf("[%s] health check failed after status left retrying; startup superseded", svc.Name)
			return nil
		}
		r.store.UpdateDependencyStatus(svc.Name, StatusFailed)
		log.Printf("[%s] health check failed: %v", svc.Name, err)
		if svc.OnFailure == "exit" {
			return fmt.Errorf("[%s] health check failed: %w", svc.Name, err)
		}
		return nil
	}

	if !r.startupStillOwnsProcess(svc.Name, cmd) {
		log.Printf("[%s] health check succeeded after process was replaced; startup superseded", svc.Name)
		return nil
	}
	if !r.store.CompareAndSwapStatus(svc.Name, StatusRetrying, StatusHealthy) {
		log.Printf("[%s] health check succeeded but status is no longer retrying; startup superseded", svc.Name)
		return nil
	}
	r.probeReadiness(ctx, svc)
	r.store.UpdateDependencyStatus(svc.Name, StatusHealthy)
	r.establishServiceOwnership(svc, cmd.Process.Pid, actorSessionIDFromContext(ctx))
	log.Printf("[%s] healthy (%s)", svc.Name, svc.HealthCheck.DisplayEndpoint())
	return nil
}
