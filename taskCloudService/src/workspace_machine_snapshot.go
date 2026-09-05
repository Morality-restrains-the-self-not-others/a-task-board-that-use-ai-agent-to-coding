package main

import (
	"strings"
	"sync"
)

var workspaceMachineSnapshotScan = scanWorkspaceMachineSnapshot

type workspaceMachineSnapshotFlight struct {
	wg   sync.WaitGroup
	snap *workspaceMachineSnapshot
	err  error
}

var (
	workspaceMachineSnapshotMu       sync.Mutex
	workspaceMachineSnapshotInflight = map[string]*workspaceMachineSnapshotFlight{}
)

func workspaceMachineSnapshotKey(companyID, workspaceID string) string {
	return trim(companyID) + "|" + trim(workspaceID)
}

// workspaceMachineSnapshot 是 workspace 机器运行时状态的单一数据源快照：
// 一次扫描 cloud_server_configs 即同时产出头部摘要计数与卡片运行态指示器，
// 从结构上保证「卡片实心亮起」与「头部已启动 N」永远来自同一份判定结果，
// 杜绝两个端点各自独立扫描导致的漂移。
type workspaceMachineSnapshot struct {
	Started  map[string]bool // instance_id → 已启动（含 mock 与 legacy 空状态）
	Starting map[string]bool // instance_id → 过渡态（Starting/Pending/Initializing）
	Busy     map[string]bool // instance_id → 已启动且有可达 server_url
	// TaskIndicators 按 task_id 聚合卡片指示器；仅含至少一个标志为真的任务
	TaskIndicators map[string]*workspaceRuntimeIndicator
}

// computeWorkspaceMachineSnapshot 只读 last_runtime_status（OPT-20260827-029）：
// 看板 GET 不得打 Aliyun Describe / 孤儿回收。Describe 走 cloud_csc_reconcile timer。
// 并行 summary+indicators 经 singleflight 共用一次扫描。
func computeWorkspaceMachineSnapshot(companyID, workspaceID string) (*workspaceMachineSnapshot, error) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	if companyID == "" || workspaceID == "" {
		return emptyWorkspaceMachineSnapshot(), nil
	}
	key := workspaceMachineSnapshotKey(companyID, workspaceID)
	workspaceMachineSnapshotMu.Lock()
	if f, ok := workspaceMachineSnapshotInflight[key]; ok {
		workspaceMachineSnapshotMu.Unlock()
		f.wg.Wait()
		return f.snap, f.err
	}
	f := &workspaceMachineSnapshotFlight{}
	f.wg.Add(1)
	workspaceMachineSnapshotInflight[key] = f
	workspaceMachineSnapshotMu.Unlock()

	f.snap, f.err = workspaceMachineSnapshotScan(companyID, workspaceID)
	workspaceMachineSnapshotMu.Lock()
	delete(workspaceMachineSnapshotInflight, key)
	workspaceMachineSnapshotMu.Unlock()
	f.wg.Done()
	return f.snap, f.err
}

func emptyWorkspaceMachineSnapshot() *workspaceMachineSnapshot {
	return &workspaceMachineSnapshot{
		Started:        map[string]bool{},
		Starting:       map[string]bool{},
		Busy:           map[string]bool{},
		TaskIndicators: map[string]*workspaceRuntimeIndicator{},
	}
}

// scanWorkspaceMachineSnapshot 单次扫描 cloud_server_configs，不调用云厂商。
func scanWorkspaceMachineSnapshot(companyID, workspaceID string) (*workspaceMachineSnapshot, error) {
	snap := emptyWorkspaceMachineSnapshot()
	if companyID == "" || workspaceID == "" {
		return snap, nil
	}

	rows, err := db.Query(
		`SELECT task_id, COALESCE(comment_id,''), COALESCE(instance_id,''), COALESCE(server_url,''), COALESCE(last_runtime_status,'')
		 FROM cloud_server_configs
		 WHERE company_id=? AND workspace_id=?`,
		companyID, workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var taskID, commentID, instanceID, serverURL, lastStatus string
		if err := rows.Scan(&taskID, &commentID, &instanceID, &serverURL, &lastStatus); err != nil {
			return nil, err
		}
		taskID = strings.TrimSpace(taskID)
		commentID = strings.TrimSpace(commentID)
		// 无 task 绑定的实例不进入头部计数。任务级模板行（comment_id=''）不作运行实例。
		if taskID == "" || commentID == "" {
			continue
		}
		status := effectiveMachineRuntimeStatus(instanceID, lastStatus)
		if machineRuntimeCountsAsStarted(instanceID, status) {
			snap.Started[instanceID] = true
			if strings.TrimSpace(serverURL) != "" {
				snap.Busy[instanceID] = true
			}
		} else if machineRuntimeCountsAsStarting(instanceID, status) {
			snap.Starting[instanceID] = true
		}
		cur, ok := snap.TaskIndicators[taskID]
		if !ok {
			cur = &workspaceRuntimeIndicator{TaskID: taskID}
			snap.TaskIndicators[taskID] = cur
		}
		if machineRuntimeCountsAsStarted(instanceID, status) {
			cur.MachineRunning = true
			cur.RunningMachineCount++
		} else if machineRuntimeCountsAsStarting(instanceID, status) {
			cur.MachineStarting = true
		}
		if containerReachabilityCountsAsRunning(instanceID, status, serverURL) {
			cur.ContainerRunning = true
			cur.RunningContainerCount++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return snap, nil
}

// Counts 从快照派生头部摘要计数：计数与指示器同源，绝不产生第二份独立判定。
func (s *workspaceMachineSnapshot) Counts() workspaceMachineCounts {
	counts := workspaceMachineCounts{
		StartedCount:  len(s.Started),
		StartingCount: len(s.Starting),
		BusyCount:     len(s.Busy),
	}
	counts.IdleCount = counts.StartedCount - counts.BusyCount
	return counts
}

// Indicators 从快照派生卡片指示器列表（仅含至少一个标志为真的任务，顺序稳定）。
func (s *workspaceMachineSnapshot) Indicators() []workspaceRuntimeIndicator {
	out := make([]workspaceRuntimeIndicator, 0, len(s.TaskIndicators))
	for _, cur := range s.TaskIndicators {
		if cur.MachineRunning || cur.MachineStarting || cur.ContainerRunning {
			out = append(out, *cur)
		}
	}
	return out
}
