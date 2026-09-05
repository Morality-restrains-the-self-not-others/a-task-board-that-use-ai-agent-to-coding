# ECS 二次启动孤儿实例防护（OPT-20260718-018）

- **日期**: 2026-07-18
- **状态**: approved（goal-mode 自动采用）
- **关联**: OPT-20260718-018；现场案例 `task_13386794339236800866` / `task_13461153320009006866`

## 问题

同一 `task_id` 再次 `CLOUD_SERVER_STARTED` 时，`UpsertAfterStart` 覆盖 CSC.`instance_id`，旧 ECS 不再被后续 `CLOUD_SERVER_STOPPED` 删除。`ClearAfterStop` 又关闭该 task 全部 open history，本地看板不显示机器，云控台仍可见 `InstanceName=task_id`（历史实例可能为 legacy `task-{task_id}`）。

## 方案（采用）

**A. 启动前同步释放旧实例（优先于「拒绝二次启动」）**  
在 `cloudserverstarted` 调用 `RunInstances` 之前：`LoadForTask`；若已有非 mock `instance_id`，用同一授权同步 `DeleteInstance`，再 `ClearAfterStop(instance_id=旧)`。失败则 `DispatchRetryable`，不创建新机。

**B. History / CSC 按 instance 收口**  
`clearContainerReachabilityNative` / `closeOpenCloudServerConfigHistories` 接受可选 `instance_id`：只关闭匹配 history；仅当 CSC 当前 `instance_id` 等于被删实例（或入参为空兼容旧调用）时清空 CSC 绑定。

**C. InstanceName 对账回收**  
在既有 `reconcileWorkspaceMachineRuntimes` 同触发点增加：按 `InstanceName=task_id`（兼查 legacy `task-{task_id}`）DescribeInstances，云上存在且不等于 CSC 当前 `instance_id` 的实例 → 同步 DeleteInstance + 关闭对应 history。TTL 缓存防刷。

## 架构影响

无新增 Application_Component / 公网 API；变更既有 `CLOUD_SERVER_STARTED` 消费者与 taskCloudService internal clear/reconcile 行为。**不新增** `docs/architecture/` 视图 triad。

## 非目标

- 不改 InstanceName 命名规则  
- 不一次性全账号扫 ECS（仅 workspace 已有 CSC 的 task）  
- 不在本变更合入后手工删机（现场孤儿已清）

## 验收

1. 单测：二次 start 先 Stop 旧 instance；ClearAfterStop 只关匹配 history；对账回收孤儿  
2. `go test`：`taskCloudService/src`、`taskEvents/internal/handlers/cloudserverstarted`、`cloudserverstopped`  
3. OPT-20260718-018 → completed  
