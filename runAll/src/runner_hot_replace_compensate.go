package main

import (
	"context"
	"log"
)

// compensateHotReplaceDownServices (OPT-20260812-048)
//
// shutdown-self 热替换后 adoptListeningServices 只回填「探针仍通过/仍有端口或 ownership
// 证据」的存活服务；替换前已 down 且配置了健康探针的服务不会自动重启，用户必须手工
// start-all 才恢复（现场复现：热替换后 /api/status 仅 43/59 healthy）。热替换本应零
// 中断，这里在 adopt 完成后对「配置了探针 + 有 ownership 记录（曾被启动，非手工停止）
// + 仍未 healthy」的服务按 DAG 依赖序自动补 start，避免部署 runAll 后托管服务掉线无人
// 拉起。
//
// 边界：
//   - 仅热替换（GracefulShutdownSelf）路径触发；普通 UI 启动仍保持「services idle」
//     契约，不会自动拉起任何服务。
//   - 手工停止的服务 StopService 会删除 ownership 记录，不会被补偿拉起。
//   - 只有配置了健康探针的服务参与补偿（无探针则无法确认起停语义，维持人工 start-all）。
func (r *Runner) compensateHotReplaceDownServices(ctx context.Context) int {
	if r == nil || r.cfg == nil || r.store == nil || r.ownershipRepo == nil {
		return 0
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var candidates []string
	for _, svc := range r.cfg.Flatten() {
		if !serviceHealthProbeConfigured(svc) {
			continue
		}
		if r.isServiceUp(svc.Name) {
			continue
		}
		ownership, err := r.ownershipRepo.FindByServiceName(svc.Name)
		if err != nil || ownership.ServiceName == "" {
			continue
		}
		candidates = append(candidates, svc.Name)
	}
	if len(candidates) == 0 {
		return 0
	}

	log.Printf("[runAll] hot-replace compensation: %d down service(s) with probe+ownership will be auto-started: %v",
		len(candidates), candidates)
	levels, err := r.buildParallelStartLevels(candidates)
	if err != nil {
		log.Printf("[runAll] hot-replace compensation plan failed: %v", err)
		return 0
	}

	started := 0
	for _, level := range levels {
		for _, name := range level {
			if err := r.StartService(ctx, name); err != nil {
				log.Printf("[runAll] hot-replace compensation start %s: %v", name, err)
				continue
			}
			started++
			r.appendLifecycleLog(name, "热替换补偿：adopt 未收养（替换前已 down），自动 start 恢复")
		}
	}
	log.Printf("[runAll] hot-replace compensation done: started=%d/%d", started, len(candidates))
	return started
}

// isServiceUp reports whether the status store currently considers the service
// up (adopt or a concurrent start already owns it).
func (r *Runner) isServiceUp(name string) bool {
	st := r.store.Get(name)
	if st == nil {
		return false
	}
	switch st.Status {
	case StatusHealthy, StatusStarting, StatusRestarting, StatusBuilding:
		return true
	default:
		return false
	}
}
