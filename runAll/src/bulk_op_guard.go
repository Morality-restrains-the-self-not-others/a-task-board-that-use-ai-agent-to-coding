package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// bulkOpGuard 统一 bulk 操作互斥（OPT-20260807-052）：
// precise-restart / start-all / stop-all / build-all / restart-all 任一进行中时，
// 其余 bulk 操作在入口 409 拒绝，而非并发执行（此前 stop-all 中途介入会打断
// precise-restart 的健康检查闭环）。
type bulkOpGuard struct {
	mu     sync.RWMutex
	active bool
	op     string
}

// bulkOpLockFile 是 bulk 操作的本地锁文件（OPT-20260810-042）：runAll 自身
// 短暂宕机 / hot-replace 时 /api/status 不可达，watchdog 门禁 fail-open 仍可能
// 在窗口内抢拉端口。bulk 开始时写锁、结束时删除，ensure_services_healthy.py
// 优先读锁文件、再回退 API，关闭该残余竞态。
const bulkOpLockFileName = "bulk_op.lock"

// bulkOpLock 锁文件内容（kind / run_id / ts）。
type bulkOpLock struct {
	Kind  string `json:"kind"`
	RunID string `json:"run_id,omitempty"`
	Ts    int64  `json:"ts"`
}

// bulkOpLockFilePath 解析锁文件路径：优先 RUNALL_PRECISE_RESTART_FILE 所在
// .runall 目录（单测可注入临时目录），否则 <monorepoRoot>/.runall。
func (r *Runner) bulkOpLockFilePath() string {
	if regPath := os.Getenv("RUNALL_PRECISE_RESTART_FILE"); regPath != "" {
		return filepath.Join(filepath.Dir(regPath), bulkOpLockFileName)
	}
	if r != nil && r.cfg != nil {
		if root, err := r.monorepoRoot(); err == nil && root != "" {
			return filepath.Join(root, ".runall", bulkOpLockFileName)
		}
	}
	return filepath.Join(".runall", bulkOpLockFileName)
}

// writeBulkOpLock 幂等写锁文件（best-effort，失败仅告警，不阻断 bulk）。
// 路径非绝对（如裸 &Runner{} 单测无 cfg/无 env）时跳过，避免污染相对 cwd。
func (r *Runner) writeBulkOpLock(op string) {
	path := r.bulkOpLockFilePath()
	if !filepath.IsAbs(path) {
		return
	}
	data, _ := json.Marshal(bulkOpLock{Kind: op, Ts: time.Now().Unix()})
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return
	}
}

// removeBulkOpLock 删除锁文件（best-effort）。
func (r *Runner) removeBulkOpLock() {
	path := r.bulkOpLockFilePath()
	if !filepath.IsAbs(path) {
		return
	}
	if err := os.Remove(path); err != nil {
		return
	}
}

// TryBeginBulk 尝试获取 bulk 互斥（空闲时成功，记录操作名并写本地锁）。
func (r *Runner) TryBeginBulk(op string) bool {
	if r == nil {
		return false
	}
	r.bulkOpActive.mu.Lock()
	defer r.bulkOpActive.mu.Unlock()
	if r.bulkOpActive.active {
		return false
	}
	r.bulkOpActive.active = true
	r.bulkOpActive.op = op
	r.writeBulkOpLock(op)
	return true
}

// EndBulk 释放 bulk 互斥（仅当持有者为该 op 时释放，防误放）并删本地锁。
func (r *Runner) EndBulk(op string) {
	if r == nil {
		return
	}
	r.bulkOpActive.mu.Lock()
	defer r.bulkOpActive.mu.Unlock()
	if r.bulkOpActive.active && r.bulkOpActive.op == op {
		r.bulkOpActive.active = false
		r.bulkOpActive.op = ""
		r.removeBulkOpLock()
	}
}

// IsBulkActive 报告是否有任一 bulk 操作进行中。
func (r *Runner) IsBulkActive() bool {
	if r == nil {
		return false
	}
	r.bulkOpActive.mu.RLock()
	defer r.bulkOpActive.mu.RUnlock()
	return r.bulkOpActive.active
}

// ActiveBulkOp 返回当前进行中的 bulk 操作名（无则空串）。
func (r *Runner) ActiveBulkOp() string {
	if r == nil {
		return ""
	}
	r.bulkOpActive.mu.RLock()
	defer r.bulkOpActive.mu.RUnlock()
	return r.bulkOpActive.op
}
