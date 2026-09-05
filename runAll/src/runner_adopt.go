package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// AdoptRunningManagedServices is the public name for hot-replace StatusStore
// adopt/reconcile (OPT-20260723-011 / OPT-20260722-065).
func (r *Runner) AdoptRunningManagedServices(ctx context.Context) int {
	return r.adoptListeningServices(ctx)
}

// adoptProbeRetryAttempts / adoptProbeRetryInterval：热替换后编排器与真实监听态
// 短暂不一致时（健康探针瞬时 EOF/竞态），对「有监听但健康失败」做短退避重试，
// 而非一次失败即放弃收养（OPT-20260810-041）。
var (
	adoptProbeRetryAttempts = 3
	adoptProbeRetryInterval = 200 * time.Millisecond
)

// checkProbeWithRetry 先立即探测一次；失败后按短退避重试至多 adoptProbeRetryAttempts
// 次总尝试，仍失败才返回最后一次错误（由调用方决定跳过收养）。
func checkProbeWithRetry(ctx context.Context, hc HealthCheck) error {
	var lastErr error
	for i := 0; i < adoptProbeRetryAttempts; i++ {
		if lastErr = checkProbe(ctx, hc); lastErr == nil {
			return nil
		}
		if i < adoptProbeRetryAttempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(adoptProbeRetryInterval):
			}
		}
	}
	return lastErr
}

// adoptListeningServices scans configured services whose ports are already
// listening (or that still have a live ownership PID after /api/shutdown-self
// hot-replace) and back-fills StatusStore + ownership so the UI is not blank.
func (r *Runner) adoptListeningServices(ctx context.Context) int {
	if r == nil || r.cfg == nil || r.store == nil {
		return 0
	}
	if ctx == nil {
		ctx = context.Background()
	}
	adopted := 0
	for _, svc := range r.cfg.Flatten() {
		if r.tryAdoptOneListeningService(ctx, svc) {
			adopted++
		}
	}
	return adopted
}

func (r *Runner) tryAdoptOneListeningService(ctx context.Context, svc Service) bool {
	cur := r.store.Get(svc.Name)
	if cur != nil {
		switch cur.Status {
		case StatusHealthy, StatusStarting, StatusRestarting, StatusBuilding:
			return false
		}
	}

	pid := 0
	if r.ownershipRepo != nil {
		if ownership, err := r.ownershipRepo.FindByServiceName(svc.Name); err == nil && ownership.ServiceName != "" {
			if ownership.PID > 0 && isPIDAlive(ownership.PID) {
				pid = ownership.PID
			}
		}
	}

	ports := resolveServicePorts(&svc)
	var listenerPIDsOnPort []int
	for _, port := range ports {
		pids, err := r.listenerPIDsForPort(port)
		if err != nil || len(pids) == 0 {
			continue
		}
		listenerPIDsOnPort = pids
		break
	}
	if pid <= 0 && len(listenerPIDsOnPort) > 0 {
		pid = r.discoverRunningPIDFromListeners(&svc)
		if pid <= 0 {
			pid = listenerPIDsOnPort[0]
		}
	}
	hasProbe := serviceHealthProbeConfigured(svc)
	probeIsEvidence := false
	if pid <= 0 && len(listenerPIDsOnPort) == 0 {
		// 无端口监听、无存活 ownership PID：若配置了健康探针，则以探针通过作为存活
		// 证据（OPT-20260811-012）。shutdown-self 热替换后 docker/netns 发布的端口对
		// lsof 不可见（docker-proxy 常以 root 运行）、exec 型健康检查无端口可解析，
		// 若 ownership PID 又因历史重启失效，服务会被证据门禁拒之门外导致 /api/status
		// 全空。探针通过即证明服务存活，应回填为 healthy（PID 未知则保持 0）。
		if !hasProbe {
			return false
		}
		if err := checkProbeWithRetry(ctx, svc.HealthCheck); err != nil {
			log.Printf("[runAll] adopt skip %s: no port/ownership evidence and health failed after %d attempts: %v",
				svc.Name, adoptProbeRetryAttempts, err)
			return false
		}
		probeIsEvidence = true
	}

	if hasProbe && !probeIsEvidence {
		// 热替换后瞬时健康失败（探针 EOF/竞态）时短退避重试，避免误放弃收养。
		if err := checkProbeWithRetry(ctx, svc.HealthCheck); err != nil {
			log.Printf("[runAll] adopt skip %s: evidence present (PID≈%d listeners=%v) but health failed after %d attempts: %v",
				svc.Name, pid, listenerPIDsOnPort, adoptProbeRetryAttempts, err)
			if pid > 0 && cur != nil && cur.PID <= 0 {
				r.store.SetPID(svc.Name, pid)
			}
			return false
		}
	}

	if !adoptPIDMatchesDeployLayout(svc, pid) {
		return false
	}

	if pid > 0 {
		r.store.SetPID(svc.Name, pid)
		if r.ownershipRepo != nil {
			existing, err := r.ownershipRepo.FindByServiceName(svc.Name)
			if err != nil || existing.ServiceName == "" || existing.PID != pid {
				r.establishServiceOwnership(svc, pid, defaultOwnershipSessionID)
			}
		}
	}
	r.store.Update(svc.Name, StatusHealthy, "")
	if hasProbe {
		r.probeReadiness(ctx, svc)
	} else {
		r.store.SetReadiness(svc.Name, ReadinessReady, "")
	}
	r.store.SetLastChecked(svc.Name, time.Now())
	r.store.UpdateDependencyStatus(svc.Name, StatusHealthy)
	evidence := fmt.Sprintf("端口/ownership 已存活且健康 (PID=%d)", pid)
	if probeIsEvidence {
		evidence = "健康探针通过（无端口/ownership 证据，PID 未知按 0 处理）"
	}
	r.appendLifecycleLog(svc.Name, fmt.Sprintf(
		"热替换收养：%s，StatusStore 回填为 healthy",
		evidence,
	))
	log.Printf("[runAll] adopted %s as healthy (PID=%d)", svc.Name, pid)
	r.startMonitoring(context.Background(), svc)
	return true
}

var readProcessExePath = func(pid int) (string, error) {
	target, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(target, " (deleted)"), nil
}

func adoptPIDMatchesDeployLayout(svc Service, pid int) bool {
	if !deployModeActive() || pid <= 0 {
		return true
	}
	start := strings.TrimSpace(svc.EffectiveStartCommand())
	if !isDeployFlatBinStart(start) {
		return true
	}
	root := strings.TrimSpace(os.Getenv("DEPLOY_ROOT"))
	if root == "" {
		return true
	}
	exe, err := readProcessExePath(pid)
	if err != nil || strings.TrimSpace(exe) == "" {
		log.Printf("[runAll] adopt skip %s: cannot read /proc/%d/exe: %v", svc.Name, pid, err)
		return false
	}
	binDir := filepath.Clean(filepath.Join(root, "bin"))
	if filepath.Dir(filepath.Clean(exe)) == binDir {
		return true
	}
	log.Printf("[runAll] adopt skip %s: exe %s is not under %s (leftover source-tree ELF)", svc.Name, exe, binDir)
	return false
}

func serviceHealthProbeConfigured(svc Service) bool {
	if svc.HealthCheck.UsesExec() && strings.TrimSpace(svc.HealthCheck.Exec) != "" {
		return true
	}
	if svc.HealthCheck.UsesTCP() && strings.TrimSpace(svc.HealthCheck.TCP) != "" {
		return true
	}
	return strings.TrimSpace(svc.HealthCheck.URL) != ""
}

func isPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil || proc == nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
