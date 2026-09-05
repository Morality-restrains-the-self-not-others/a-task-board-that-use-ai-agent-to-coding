# [运行时] 创建任务勾选自动运行未启动机器节点

## 现象

创建任务时勾选「自动运行」（`auto_run=true`），任务创建成功且详情页显示自动运行为「是」，但机器节点未自动启动；需手动在任务详情点「启动服务器」。

## 环境与上下文

- 前端已于 2026-07-13 移除创建成功后的二次 `start-vm` 调用（依赖后端触发）
- 意图文档 `005_create_task_auto_run_backend_start` 已勾选验收，但 `conf/value-stream.yaml` 中步骤长期为 `planned`
- 典型路径：`POST /api/tenant/.../workspace/.../todos/` → 201，无后续 `POST .../cloud/compute/start-vm(-auto)/`

## 根因

`taskTaskService` `handleCreateTask` / `handleUpdateTask` 仅将 `auto_run` 写入 DB，未实现：

1. `validateAutoRunPrerequisites`（CloudServiceURL / 镜像 / 运行模版门禁）
2. `scheduleTaskAutoRun` → 异步 POST Cloud `start-vm(-auto)/`，`X-Trace-Id` = task_id

## 修复

在 `taskTaskService` 落地意图 005：

- `auto_run.go` / `auto_run_startvm.go`：门禁、构建请求、异步触发
- `cloud_client.go`：`startVM` 出站调用
- 挂接 create（`auto_run=true`）与 update（false→true 或 `force_auto_run`）
- `auto_run_test.go`；value-stream 步骤改为 `active`

## 预防

- 意图勾选完成前，value-stream 不得标 `active`；文档与代码同步落地
- 前端移除二次 start-vm 前，须确认后端触发已合入并有测例
