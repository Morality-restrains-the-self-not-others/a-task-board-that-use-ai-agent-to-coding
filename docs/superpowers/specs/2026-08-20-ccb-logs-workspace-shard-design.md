# 评论启动日志按工作空间 ID 分表 — 设计

- **Date:** 2026-08-20
- **Status:** accepted（goal-mode 自动采用）
- **Architecture change:** 是（数据对象物理分片，表名含分片序号）
- **ADR:** [ADR-0023](../../adr/0023-ccb-logs-workspace-hash-shards.md)
- **python_api_approval:** not_applicable（Go `taskCloudService`，无新 Python API）
- **Page:** 任务详情评论执行区「启动日志」`div.bg-gray-50.p-2.rounded-md.max-h-32`
- **Iteration:** ccb-logs-workspace-shard-v88

## 🕸️ Code Review Graph 分析

- CRG `update --brief` 成功（增量 6 files，risk 0.00）
- 现状调用链：`publishTaskSSE` → `logServerSchedulingToBindingBestEffort` / `appendCommentContainerBindingLog` → 单表 `cloud_comment_container_binding_logs`；`handleCommentContainerBindingsList` → `listCommentContainerBindingLogs(companyID, taskID)` **不带 workspace_id**
- 爆炸半径：taskCloudService `src/comment_container_bindings_*.go`、`events.go`、`main_test.go` 清理列表、`dataMigrate/taskCloudService/`；前端仍读 bindings JSON 的 `logs` 字段，**无 API 契约变更**

## Goal / 成功标准

1. 启动日志物理表名由 **workspace_id** 决定：`cloud_comment_container_binding_logs_{00..15}`。
2. 选片算法：`CRC32(workspace_id) % 16`（Go `crc32.ChecksumIEEE` 与 MySQL `CRC32()` 一致，可 SQL 回填）。
3. 表名只由整数格式化生成，禁止把 workspace 字符串拼进 SQL 标识符。
4. 读写路径必须携带 `workspace_id`（HTTP 已有 `X-Workspace-Id` / workspace 路由；SSE 从 statusData 或 CSC 解析）。
5. 主键改为 Snowflake 字符串（分片表禁止 AUTO_INCREMENT）。
6. 存量行经 CSC 解析 workspace 后回填到对应分片；未解析到的落入 shard 00 并保留遗留表只读。
7. 冷打开任务详情仍能还原「容器调度排队中 / 正在启动容器实例 / aliyunAPI…」时间线；跨工作空间查询互不串表。
8. 回归：分片路由单测 + 既有 binding logs timeline / server scheduling persist 测绿。

## 现状

- 单表 `cloud_comment_container_binding_logs`：时间累积、无 `workspace_id`、`AUTO_INCREMENT` PK。
- 任务详情页启动日志来自 list bindings 的 `logs[]`（再与本地 SSE 去重合并）。
- `cloud_server_events` / `cloud_job_execution_event` 已有 `workspace_id` 且按时间 RANGE 分区，**本期不改表名**（记 OPT）。

## 决策

| 方案 | 结论 |
|------|------|
| A. 16 张预创建哈希分表，表名含分片序号，键 = workspace_id | **采用**：符合冷热分离「预分 16 片」；表名由工作空间 ID 导出；运维表数固定 |
| B. 一工作空间一张表 `..._ws_{id}` | 拒绝：DDL 无界、MySQL 表数爆炸 |
| C. 仅在单表加 workspace_id 索引 / 时间分区 | 拒绝：用户明确要求**表名**按工作空间分片 |
| D. 按 tenant_id 分表 | 拒绝：页面与访问模式以 workspace 为边界；同租户多工作空间会热点 |

选片伪代码：

```go
func ccbLogTable(workspaceID string) (string, error) {
    ws := strings.TrimSpace(workspaceID)
    if ws == "" {
        return "", errWorkspaceRequired
    }
    shard := crc32.ChecksumIEEE([]byte(ws)) % 16
    return fmt.Sprintf("cloud_comment_container_binding_logs_%02d", shard), nil
}
```

列：`id VARCHAR(64)` Snowflake、`workspace_id`、`company_id`、`task_id`、`comment_id`、`binding_id`、`stage`、`message`、`created_at`。索引 `(workspace_id, task_id, comment_id, created_at)`。

## workspace_id 解析顺序

1. 显式参数（list/create/advance 的 workspace 路由 / `X-Workspace-Id`）
2. `statusData["workspace_id"]`（SSE 调度文案）
3. `cloud_server_configs.workspace_id` WHERE `task_id`（非空，最新）
4. 仍为空 → **不写分片**（best-effort 打 warn）；list 返回 400 `workspace_id required`

测试夹具一律传入非空 workspace（如 `ws1`）。

## API

无新 endpoint。既有：

- `GET/POST .../cloud/compute/comment-container-bindings/`（workspace 已在路径或头）
- 响应 `bindings[].logs[]` 字段不变（`id/stage/message/created_at`）

## 迁移

Expand/Contract：dataMigrate `030_ccb_logs_workspace_shards.sql` 建 16 表并 `INSERT…SELECT` 回填；应用改为只读写分片；遗留表保留不 DROP。

## 意图 → 事件

本增量是**既有 append-log 的存储路由**，不新增业务意图。启动进度仍走既有 SSE（`publishTaskSSE`）。无新 MQ 事件。例外理由见意图文档。

## 架构制品

v88 application-integration + enterprise-landscape：`.puml` / `.diff.archimate` / `.full.archimate` / `.mermaid.md`
