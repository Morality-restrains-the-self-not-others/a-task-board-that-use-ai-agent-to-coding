# AiMonitor managed 模式 detach 启动修复

**日期**: 2026-06-02  
**状态**: 已批准（实现已完成，待交付）  
**类型**: 基础设施 Bug 修复（单文件）

## 问题陈述

runAll 中 `ai-monitor` 服务无法启动。远程路径为：

```yaml
start_command: "bash scripts/runall-remote-docker.sh stack up aimonitor"
launch_mode: detach
```

`runall-remote-docker.sh` 经 SSH 在远程 CPU 执行 `AiMonitor/run.sh managed`。

## 根因

`AiMonitor/run.sh` 的 `managed` 模式使用**前台** `compose up`（无 `-d`），导致：

1. SSH 会话永久挂在前台 compose 日志流上，无法返回 Mac
2. `runall-remote-docker.sh` 内 `wait_infra_http` 无法执行，启动脚本不结束
3. runAll `launch_mode: detach` 语义与前台 compose 冲突（redis/kafka/git-service 均为 detach）

## 方案

将 `managed` 模式改为与 `gitService/run.sh`、`dockerInfra/*/run.sh` 一致：

```bash
compose up -d --remove-orphans "$@"
print_ready_hint
```

移除 `trap` + 前台 `compose up`（仅 AiMonitor 曾采用此模式）。

## 约束与不变量

| 项 | 说明 |
|---|---|
| 停服 | 仍由 `stop_command` → `run.sh stop` → `compose down` |
| 具名容器 | `docker-compose.yaml` 中 `container_name` 不变，reset 脚本仍可用 |
| 本地 runAll | `runAll/config.yaml` 中 `./run.sh managed` + `detach` 行为一致 |
| 健康检查 | `http://${INFRA_HOST}:3000/api/health` 不变 |

## 价值流影响

影响现有 **platform-observability / ai-monitor** 相关步骤的运行时可用性，不新增 stream、不修改 `value-stream.yaml` 字段。

受影响字段（运行时依赖，非 schema 变更）：

- `ai-monitor.loki.*`
- `ai-monitor.grafana.*`
- `ai-monitor.runtime.*`

## 领域概念清单

**不适用** — 纯 Shell/Docker 基础设施脚本，无后端领域模型变更。

## 验收标准

1. `bash scripts/runall-remote-docker.sh stack up aimonitor` 在 30s 内返回 exit 0
2. `curl -sf http://${INFRA_HOST}:3000/api/health` 返回 200
3. runAll UI 启动 `ai-monitor` 状态变为 healthy
4. `runAll/src` 相关测试通过

## 非目标

- 合并 `managed` 与 `start` 分支（可选后续 simplify）
- 修改 `runall-remote-docker.sh` 调用方式
- 更新历史设计文档中「前台 managed」描述（可后续 doc 清理）
