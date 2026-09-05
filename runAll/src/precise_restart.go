package main

// 精准编译重启（precise restart）
//
// 智能体编程会话修改某服务代码后，将服务名登记到仓库根
// `.runall/precise_restart_services.txt`（一行一个服务名，支持 # 注释）。
// runAll Web UI 的「精准编译重启」按钮点击后：
//   1. 读取登记文件，解析为服务集合（按依赖深度排序，依赖先重启）；
//   2. 逐个服务执行 restartService（ADR-0027 先编译；ADR-0058 金丝雀：新进程就绪后再排空旧进程）；
//   3. 处理完成后清空登记文件；失败的服务保留在文件中以便重试；
//   4. 写入 consumed-at 水位线，阻止 Stop hook 对「已部署但仍脏」工作树立刻重登。
// 页头「全部重新编译」（BuildAll）正常完成后同样清空登记并写水位线；中断不清空。
// 全部重启 / 单服务 ↻ 不走本路径，不编译。
//
// 登记方式（智能体约束见 .ai/01_project_constraints/42_precise_restart_service_registration.md）：
//   - scripts/register-precise-restart.sh <service-name>...
//   - 或直接向文件追加：服务名（runAll.yaml 中的 name）或工作目录名（如 taskAuth）。

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

// preciseRestartPerServiceTimeout caps compile + canary restart + health for one service
// during precise-restart. Override in tests.
var preciseRestartPerServiceTimeout = 8 * time.Minute

// preciseRestartActive 并发防护：同一时刻只允许一个精准编译重启。
type preciseRestartActive struct {
	mu     sync.RWMutex
	active bool
	runID  string
}

// TryBeginPreciseRestart 标记精准编译重启开始（空闲时成功）。
// 参与统一 bulk 互斥：任一 bulk 操作进行中即拒绝（OPT-20260807-052）。
func (r *Runner) TryBeginPreciseRestart(runID string) bool {
	if r == nil {
		return false
	}
	if !r.TryBeginBulk("precise-restart") {
		return false
	}
	r.preciseRestartActive.mu.Lock()
	defer r.preciseRestartActive.mu.Unlock()
	if r.preciseRestartActive.active {
		r.EndBulk("precise-restart")
		return false
	}
	r.preciseRestartActive.active = true
	r.preciseRestartActive.runID = runID
	return true
}

func (r *Runner) endPreciseRestart() {
	r.preciseRestartActive.mu.Lock()
	r.preciseRestartActive.active = false
	r.preciseRestartActive.runID = ""
	r.preciseRestartActive.mu.Unlock()
	r.EndBulk("precise-restart")
}

// IsPreciseRestartActive 报告是否已有精准编译重启进行中。
func (r *Runner) IsPreciseRestartActive() bool {
	if r == nil {
		return false
	}
	r.preciseRestartActive.mu.RLock()
	defer r.preciseRestartActive.mu.RUnlock()
	return r.preciseRestartActive.active
}

// GetActivePreciseRestartRunID 返回当前精准编译重启的进度 runID。
func (r *Runner) GetActivePreciseRestartRunID() string {
	if r == nil {
		return ""
	}
	r.preciseRestartActive.mu.RLock()
	defer r.preciseRestartActive.mu.RUnlock()
	return r.preciseRestartActive.runID
}

// PreciseRestart 根据登记文件执行精准编译重启：
//  1. 读取登记 → 解析服务（未知登记名记为失败）；
//  2. 按依赖深度排序（依赖先重启）；
//  3. 逐个 restartService（ADR-0027 compile-then-swap：编译 → 停止 → 启动 → 健康检查），经 runID 广播进度；
//  4. 处理完成后清除本批成功项；失败项保留；并发新登记保留；写入 consumed-at 水位线。
//
// 返回保留在登记文件中的服务名（成功且无并发新登记则文件已清空）。
func (r *Runner) PreciseRestart(ctx context.Context, actorSessionID string) ([]string, error) {
	if r == nil {
		return nil, fmt.Errorf("runner is required")
	}
	path := preciseRestartFile(r.cfgPath)
	registeredEntries, err := readRegistrationEntries(path)
	if err != nil {
		return nil, err
	}
	if len(registeredEntries) == 0 {
		return nil, fmt.Errorf("no registered services (registration file: %s)", path)
	}
	// 磁盘 YAML 可能在进程启动后新增服务（如新的 task-events-* 消费者）。
	// 重载失败则保留内存配置，已知服务仍可重启。
	added, rerr := r.reloadConfigFromDisk()
	logPreciseRestartReload(added, rerr)
	// 原登记时间戳索引（别名归一化后保留首次登记时间，供中断/失败回写）。
	originalTs := make(map[string]int64)
	registrations := make([]string, 0, len(registeredEntries))
	for _, e := range registeredEntries {
		registrations = append(registrations, e.Name)
		if _, ok := originalTs[e.Name]; !ok {
			originalTs[e.Name] = e.RegisteredAt
		}
	}

	// 解析登记名 → 服务（别名归一化：task-auth 与 taskAuth 同服务只重启一次）；
	// 未知名保留在文件中并计入失败。
	var failed []string
	var allErrors []string
	var resolved []*Service
	seenSvc := make(map[string]bool)
	originalResolved := make(map[string]bool) // 本批已解析服务名（收尾时用于识别并发新登记）
	for _, name := range registrations {
		// 原始登记名（含别名）一律视为本批，收尾时不得当「并发新登记」保留。
		originalResolved[name] = true
		svcs := r.resolveRegisteredServices(name)
		if len(svcs) == 0 {
			failed = append(failed, name)
			allErrors = append(allErrors, fmt.Sprintf("%s: unknown service (not in runAll config)", name))
			continue
		}
		for _, svc := range svcs {
			originalResolved[svc.Name] = true
			if seenSvc[svc.Name] {
				continue
			}
			seenSvc[svc.Name] = true
			// 别名登记时把时间戳挂到规范服务名，供中断回写 pending 使用。
			if _, ok := originalTs[svc.Name]; !ok {
				originalTs[svc.Name] = originalTs[name]
			}
			resolved = append(resolved, svc)
		}
	}
	ordered := r.orderByDependencyDepth(resolved)

	runID := r.GetActivePreciseRestartRunID()
	if runID == "" {
		runID = fmt.Sprintf("precise-restart-%d", time.Now().UnixNano())
	}
	// 复用 start-all 的进度广播通道，事件 Operation 用 "restart"。
	publish := func(ev StartAllProgressEvent) {
		if runID != "" && r.progressBroadcaster != nil {
			r.progressBroadcaster.Publish(runID, ev)
		}
	}

	total := len(ordered)
	// 与 start-all/stop-all/build-all 一致：凡 Failed>0 必须附带 Errors，
	// 供 status UI 展示错误列表与「复制日志」按钮（omit 会导致按钮丢失）。
	mkEv := func(built int, current string, done bool, phase string) StartAllProgressEvent {
		remaining := total - built - len(failed)
		if remaining < 0 {
			remaining = 0
		}
		return StartAllProgressEvent{
			Total: total, Started: built, Failed: len(failed), Skipped: 0,
			Current: current, Remaining: remaining,
			Done: done, Phase: phase, Operation: "restart",
			Error:  lastErrorString(allErrors),
			Errors: append([]string(nil), allErrors...),
		}
	}
	publish(mkEv(0, "", false, "starting"))

	var built int
	var succeeded []string
	finishWith := func(done bool, phase string, cancelledErr error) {
		publish(mkEv(built, "", done, phase))
		if cancelledErr != nil {
			log.Printf("[precise-restart] aborted: %v", cancelledErr)
		}
		if runID != "" && r.progressBroadcaster != nil {
			go func() {
				time.Sleep(30 * time.Second)
				r.progressBroadcaster.CloseRun(runID)
			}()
		}
	}

	if deployModeActive() && len(ordered) > 0 {
		compileNames := make([]string, 0, len(ordered))
		for _, svc := range ordered {
			compileNames = append(compileNames, svc.Name)
		}
		publish(mkEv(0, "source-compile", false, "progress"))
		if err := prepareDeploySourceArtifactsFn(ctx, compileNames, false); err != nil {
			log.Printf("[precise-restart] source compile/install failed: %v", err)
			allErrors = append(allErrors, err.Error())
			finishWith(true, "error", err)
			return registrations, err
		}
	}

	for i, svc := range ordered {
		select {
		case <-ctx.Done():
			// 中断：仅保留尚未处理的服务 + 已失败项；已成功项丢弃（避免「跑完仍已登记」）。
			pending := make([]string, 0, len(ordered)-i)
			for _, s := range ordered[i:] {
				pending = append(pending, s.Name)
			}
			keepEntries := keepEntriesOnAbort(succeeded, pending, failed, originalTs, time.Now().Unix())
			if werr := writeRegistrationEntries(path, keepEntries); werr != nil {
				log.Printf("[precise-restart] keep registrations on abort: %v", werr)
			}
			// 已成功部分也算消费，刷新水位线，阻止 Stop hook 立刻把成功项重登回来。
			if werr := writePreciseRestartConsumedAt(path, time.Now().Unix()); werr != nil {
				log.Printf("[precise-restart] write consumed-at on abort: %v", werr)
			}
			keep := make([]string, 0, len(keepEntries))
			for _, e := range keepEntries {
				keep = append(keep, e.Name)
			}
			finishWith(true, progressPhaseForContextErr(ctx), ctx.Err())
			return keep, ctx.Err()
		default:
		}

		publish(mkEv(built, svc.Name, false, "progress"))
		if skipOrchestratorRestart(svc) {
			log.Printf("[precise-restart] skip restart of orchestrator %s (artifacts already installed)", svc.Name)
			succeeded = append(succeeded, svc.Name)
			built++
			publish(mkEv(built, "", false, "progress"))
			continue
		}
		// Heartbeat while a single service rebuild/restart can take minutes,
		// so the UI progress bar does not look frozen with no updates.
		hbStop := make(chan struct{})
		go func(name string, startedAt int) {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-hbStop:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
					publish(mkEv(startedAt, name, false, "progress"))
				}
			}
		}(svc.Name, built)
		// Per-service deadline: one hung stop/build/health must not freeze the
		// whole precise-restart panel indefinitely (hot-replace then leaves a
		// zombie "current" that cancel cannot clear).
		svcCtx, svcCancel := context.WithTimeout(ctx, preciseRestartPerServiceTimeout)
		err := r.RestartServiceWithActor(withCompileThenSwap(svcCtx), svc.Name, actorSessionID)
		svcCancel()
		close(hbStop)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(svcCtx.Err(), context.DeadlineExceeded) {
				err = fmt.Errorf("timed out after %s: %w", preciseRestartPerServiceTimeout, err)
			}
			log.Printf("[precise-restart] %s failed: %v", svc.Name, err)
			failed = append(failed, svc.Name)
			allErrors = append(allErrors, fmt.Sprintf("%s: %v", svc.Name, err))
			publish(mkEv(built, "", false, "progress"))
			continue
		}
		succeeded = append(succeeded, svc.Name)
		built++
		publish(mkEv(built, "", false, "progress"))
	}

	// 处理完成：清除本批成功项；失败项保留；运行中并发追加的新登记保留。
	keep, err := rewriteRegistrationsAfterRun(path, originalResolved, failed, succeeded)
	if err != nil {
		log.Printf("[precise-restart] rewrite registrations after run: %v", err)
		allErrors = append(allErrors, fmt.Sprintf("rewrite registrations: %v", err))
		// 回退：至少按旧语义保留失败项，避免成功项残留。
		keep = dedupeStrings(append([]string(nil), failed...))
		if rerr := retainFailedEntries(path, keep); rerr != nil {
			log.Printf("[precise-restart] retainFailedEntries fallback: %v", rerr)
		}
	}
	if werr := writePreciseRestartConsumedAt(path, time.Now().Unix()); werr != nil {
		log.Printf("[precise-restart] write consumed-at: %v", werr)
	}
	if len(keep) == 0 {
		log.Printf("[precise-restart] run finished: %d built, 0 failed, registration file cleared: %s", built, path)
	} else {
		log.Printf("[precise-restart] run finished: %d built, %d failed, retained registrations: %v", built, len(failed), keep)
	}
	finishWith(true, "done", nil)
	return keep, nil
}

func lastErrorString(errs []string) string {
	if len(errs) == 0 {
		return ""
	}
	return errs[len(errs)-1]
}

func dedupeStrings(in []string) []string {
	var out []string
	seen := make(map[string]struct{})
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
