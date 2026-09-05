package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 两端点同快照不变量：workspace-machine-summary 与 workspace-runtime-indicators
// 必须由同一次单扫描快照派生，杜绝「卡片实心亮起而头部已启动 0」类的结构性不一致。
// 本测试先锁定共享快照单元，再在 API 层交叉验证两个端点从同一快照输出一致结果。

func TestWorkspaceMachineSnapshotSingleScan(t *testing.T) {
	setupCloudTestDB(t)

	// 混合行集：busy/idle 已启动、starting 过渡态、stopped/released 停机（含孤儿 server_url）、同实例多任务共驻
	rows := []CloudServerConfig{
		{ID: "cfg-a", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-a", CommentID: "cmt-a",
			Platform: "mock", InstanceID: "m-a", ServerURL: "http://127.0.0.1:8080/", Region: "cn-test"},
		{ID: "cfg-b", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-b", CommentID: "cmt-b",
			Platform: "mock", InstanceID: "m-b", Region: "cn-test"},
		{ID: "cfg-c", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-c", CommentID: "cmt-c",
			Platform: "aliyun", InstanceID: "i-c", Region: "cn-test", LastRuntimeStatus: "Starting"},
		{ID: "cfg-d", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-d", CommentID: "cmt-d",
			Platform: "aliyun", InstanceID: "i-d", ServerURL: "http://10.0.0.1:8765/",
			Region: "cn-test", LastRuntimeStatus: "Stopped"},
		{ID: "cfg-e", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-e", CommentID: "cmt-e",
			Platform: "aliyun", InstanceID: "i-e", Region: "cn-test", LastRuntimeStatus: "Running"},
		{ID: "cfg-f", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-f", CommentID: "cmt-f",
			Platform: "mock", InstanceID: "m-f", ServerURL: "http://127.0.0.1:8765/", Region: "cn-test"},
		{ID: "cfg-g", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-g", CommentID: "cmt-g",
			Platform: "mock", InstanceID: "m-f", ServerURL: "http://127.0.0.1:8765/", Region: "cn-test"},
		{ID: "cfg-h", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-h", CommentID: "cmt-h",
			Platform: "aliyun", InstanceID: "i-h", ServerURL: "http://10.0.0.2:8765/",
			Region: "cn-test", LastRuntimeStatus: "Released"},
	}
	for _, r := range rows {
		if err := upsertCloudServerConfig(r); err != nil {
			t.Fatal(err)
		}
	}

	snap, err := computeWorkspaceMachineSnapshot("t1", "ws1")
	if err != nil {
		t.Fatal(err)
	}

	// 实例级集合：一次扫描产生唯一的 started/starting/busy 判定（共驻实例只计一次）
	assertBoolKey := func(m map[string]bool, key string, want bool, label string) {
		if m[key] != want {
			t.Fatalf("%s: %q = %v, want %v (set=%v)", label, key, m[key], want, m)
		}
	}
	assertBoolKey(snap.Started, "m-a", true, "Started")
	assertBoolKey(snap.Started, "m-b", true, "Started")
	assertBoolKey(snap.Started, "i-e", true, "Started")
	assertBoolKey(snap.Started, "m-f", true, "Started")
	assertBoolKey(snap.Started, "i-c", false, "Started")
	assertBoolKey(snap.Started, "i-d", false, "Started")
	assertBoolKey(snap.Started, "i-h", false, "Started")
	assertBoolKey(snap.Starting, "i-c", true, "Starting")
	assertBoolKey(snap.Starting, "m-a", false, "Starting")
	assertBoolKey(snap.Busy, "m-a", true, "Busy")
	assertBoolKey(snap.Busy, "m-f", true, "Busy")
	assertBoolKey(snap.Busy, "m-b", false, "Busy")
	assertBoolKey(snap.Busy, "i-e", false, "Busy")
	if len(snap.Started) != 4 || len(snap.Starting) != 1 || len(snap.Busy) != 2 {
		t.Fatalf("snapshot sizes: started=%d want 4, starting=%d want 1, busy=%d want 2",
			len(snap.Started), len(snap.Starting), len(snap.Busy))
	}

	counts := snap.Counts()
	if counts.StartedCount != 4 || counts.StartingCount != 1 || counts.BusyCount != 2 || counts.IdleCount != 2 {
		t.Fatalf("counts=%+v want started=4 starting=1 busy=2 idle=2", counts)
	}

	// 任务级指示器：与计数同源于一次扫描
	ind := snap.Indicators()
	byTask := map[string]workspaceRuntimeIndicator{}
	for _, it := range ind {
		byTask[it.TaskID] = it
	}
	expectFlags := map[string]workspaceRuntimeIndicator{
		"task-a": {TaskID: "task-a", MachineRunning: true, ContainerRunning: true, RunningMachineCount: 1, RunningContainerCount: 1},
		"task-b": {TaskID: "task-b", MachineRunning: true, RunningMachineCount: 1},
		"task-c": {TaskID: "task-c", MachineStarting: true},
		"task-e": {TaskID: "task-e", MachineRunning: true, RunningMachineCount: 1},
		"task-f": {TaskID: "task-f", MachineRunning: true, ContainerRunning: true, RunningMachineCount: 1, RunningContainerCount: 1},
		"task-g": {TaskID: "task-g", MachineRunning: true, ContainerRunning: true, RunningMachineCount: 1, RunningContainerCount: 1},
	}
	for tid, want := range expectFlags {
		got, ok := byTask[tid]
		if !ok || got != want {
			t.Fatalf("indicator[%s]=%+v ok=%v want %+v", tid, got, ok, want)
		}
	}
	for _, tid := range []string{"task-d", "task-h"} {
		if _, ok := byTask[tid]; ok {
			t.Fatalf("stopped/released task %s must not appear in indicators: %+v", tid, byTask[tid])
		}
	}
}

// OPT-20260809-028：无 task 绑定的已启动实例（task_id=”，如容器绑定在 comment_id
// 上但任务已删除的残留）不得进入头部计数集合 —— 卡片指示器按 task_id 聚合，
// 无 task 即无卡片，若计入会产生「头部已启动 N > 卡片亮起数」。
func TestWorkspaceMachineSnapshotExcludesNoTaskBound(t *testing.T) {
	setupCloudTestDB(t)

	rows := []CloudServerConfig{
		{ID: "cfg-a", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-a", CommentID: "cmt-a",
			Platform: "mock", InstanceID: "m-a", Region: "cn-test"},
		{ID: "cfg-orphan", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "",
			Platform: "mock", InstanceID: "m-orphan", Region: "cn-test"},
	}
	for _, r := range rows {
		if err := upsertCloudServerConfig(r); err != nil {
			t.Fatal(err)
		}
	}

	snap, err := computeWorkspaceMachineSnapshot("t1", "ws1")
	if err != nil {
		t.Fatal(err)
	}

	// 无 task 绑定的实例不应计入 started（头部计数），也不产生任何指示器
	if snap.Started["m-orphan"] {
		t.Fatalf("no-task-bound instance must be excluded from Started, got %v", snap.Started)
	}
	if snap.Busy["m-orphan"] || snap.Starting["m-orphan"] {
		t.Fatalf("no-task-bound instance must be excluded from Busy/Starting: %v", snap)
	}
	if len(snap.Started) != 1 || snap.Started["m-a"] != true {
		t.Fatalf("Started=%v want only m-a", snap.Started)
	}
	counts := snap.Counts()
	if counts.StartedCount != 1 || counts.BusyCount != 0 || counts.IdleCount != 1 {
		t.Fatalf("counts=%+v want started=1 busy=0 idle=1", counts)
	}
	ind := snap.Indicators()
	if len(ind) != 1 || ind[0].TaskID != "task-a" {
		t.Fatalf("indicators=%+v want only task-a", ind)
	}
}

func TestWorkspaceEndpointsShareOneSnapshot(t *testing.T) {
	setupCloudTestDB(t)

	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-1", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-1", CommentID: "cmt-1",
		Platform: "mock", InstanceID: "m-1", ServerURL: "http://127.0.0.1:8080/", Region: "cn-test",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-2", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-2", CommentID: "cmt-2",
		Platform: "mock", InstanceID: "m-2", Region: "cn-test",
	}); err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-3", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-3", CommentID: "cmt-3",
		Platform: "aliyun", InstanceID: "i-3", Region: "cn-test", LastRuntimeStatus: "Starting",
	}); err != nil {
		t.Fatal(err)
	}

	getJSON := func(routeKey, path string, out interface{}) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-User-Id", "test-user")
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		req.Header.Set("X-Workspace-Id", "ws1")
		rec := httptest.NewRecorder()
		handleCloudWorkspaceRoutes(rec, req, routeKey)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatal(err)
		}
	}

	var summary map[string]interface{}
	getJSON("cloud/compute/workspace-machine-summary/",
		"/api/tenant/t1/workspace/ws1/cloud/compute/workspace-machine-summary/", &summary)
	if summary["started_count"] != float64(2) || summary["busy_count"] != float64(1) ||
		summary["idle_count"] != float64(1) || summary["starting_count"] != float64(1) {
		t.Fatalf("summary=%v want started=2 busy=1 idle=1 starting=1", summary)
	}

	var ind struct {
		Status     string                      `json:"status"`
		Count      int                         `json:"count"`
		Indicators []workspaceRuntimeIndicator `json:"indicators"`
	}
	getJSON("cloud/compute/workspace-runtime-indicators/",
		"/api/tenant/t1/workspace/ws1/cloud/compute/workspace-runtime-indicators/", &ind)
	byTask := map[string]workspaceRuntimeIndicator{}
	for _, it := range ind.Indicators {
		byTask[it.TaskID] = it
	}
	if !byTask["task-1"].MachineRunning || !byTask["task-1"].ContainerRunning {
		t.Fatalf("task-1=%+v", byTask["task-1"])
	}
	if !byTask["task-2"].MachineRunning || byTask["task-2"].ContainerRunning {
		t.Fatalf("task-2=%+v", byTask["task-2"])
	}
	if !byTask["task-3"].MachineStarting || byTask["task-3"].MachineRunning {
		t.Fatalf("task-3=%+v", byTask["task-3"])
	}
	if ind.Count != 3 {
		t.Fatalf("indicator count=%d want 3", ind.Count)
	}

	// 报告缺陷方向的一致性锁：任何 machine_running 卡片 ⇒ 头部已启动 ≥ 1。
	// （本数据集 started=2，卡片亮起 2 个 —— 若两端点各自独立计算漂移，此处即失效。）
	if int(summary["started_count"].(float64)) < 1 {
		t.Fatalf("card-lit implies started>=1 violated: %v", summary)
	}
}
