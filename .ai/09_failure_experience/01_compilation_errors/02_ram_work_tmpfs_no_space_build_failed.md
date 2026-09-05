# [编译] `/tmp/ram-work` tmpfs 满导致 runAll `build failed: exit status 1`

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

- runAll 批量构建时报大量服务失败，错误形如：
  `[task-events-project-updated-1-process-project-update] build failed: exit status 1`
- 同类失败可同时波及多个 `task-events-*`、`task-task-service`、`task-cloud-service`、`task-ai-comment`。
- 直接 `go build` 可见底层错误：
  `write …: no space left on device`
- `df -h /tmp/ram-work` 显示 tmpfs **100%**（容量约 10G），而根分区 `/` 仍有充足空间。

## 根因

1. 工作区挂在 **tmpfs**（`/tmp/ram-work`），容量有限（约 10G）。
2. 运行日志（`logs/`、`taskGateway/logs/`）与大量 Go 二进制（尤其 `taskEvents/bin/**`，单文件约 20–32MB）快速占满可用空间。
3. runAll 只透传 `go build` / `build.sh` 的 exit status，**不区分**「磁盘满」与「代码编译错误」，表现为统一的 `build failed: exit status 1`。

## 解决方案

1. 确认：`df -h /tmp/ram-work`；必要时 `go build -o /tmp/testbin ./…` 复现 `no space left on device`。
2. 安全腾挪（不删源码）：
   - 截断大日志：`find logs -name '*.log' -size +1M -exec truncate -s 0 {} +`（及 `taskGateway/logs/`）。
   - 删除孤儿产物：如误落在包目录旁的 `go build` 输出、过时 `bin/` 别名、`tmp/` 临时二进制。
3. 按服务重建：
   - `taskEvents`：`./run.sh build <event>/<intent>`
   - `taskTaskService` / `taskCloudService` / `taskAIComment`：`./build.sh`

## 预防

- 批量 build 前检查 tmpfs 可用空间（建议预留 ≥1GB）。
- 定期轮转/截断 `logs/` 与网关 access log，避免长期堆积到数百 MB。
- 排查 `build failed: exit status 1` 时，**先看磁盘再看代码**。

## 验证

```bash
df -h /tmp/ram-work
cd /tmp/ram-work/taskEvents && ./run.sh build project_updated/1_process_project_update
cd /tmp/ram-work/taskTaskService && ./build.sh
cd /tmp/ram-work/taskCloudService && ./build.sh
cd /tmp/ram-work/taskAIComment && ./build.sh
```

## 相关

- runAll 构建入口：`runAll/src/runner.go`（`build failed` 包装）
- 事件构建：`taskEvents/run.sh` → `build_intent`
