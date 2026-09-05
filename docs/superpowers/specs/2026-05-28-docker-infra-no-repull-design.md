# docker-infra 启动/重启禁止重复拉镜像 — 设计

**日期:** 2026-05-28  
**状态:** 已批准并实现（0-auto-flow）  
**范围:** runAll `docker-infra` 服务、`task2app/Saas_project/docker-compose.yml`、可选 `runAll/src/runner.go`

## 1. 背景与问题

在 `http://localhost:9999/` 对 **docker-infra** 执行启动或重启时，日志反复出现 `Pulling` / `Downloading`，即使本地已有 Redis、Kafka UI 等镜像。Confluent 镜像体积大（单层 ~244MB），重复拉取导致：

- 启动耗时数分钟甚至超时（`READINESS_TIMEOUT`，8080 不可用）
- 用户反复点击重启，形成「总在拉镜像」的恶性循环
- 阻塞依赖 `docker-infra` 的 `task-sse`、`saas-backend`、`task-events-*` 等上游链

### 1.1 根因（已调查确认）

| # | 根因 | 说明 |
|---|------|------|
| A | 每次 lifecycle 执行裸 `docker compose up -d` | `runAll.yaml` 无 pull/up 分离、无 `--pull never` |
| B | Compose 默认 `--pull policy` | 每次 `up` 进入 Pull 阶段；缺失镜像全量下载，已有镜像仍校验层 |
| C | `kafka-ui:latest` 浮动 tag | 易触发 registry 校验 |
| D | 无专用 stop/down | runAll 停止只杀 shell PID；`up -d` 早已退出，容器仍运行；重启再跑完整 `up` |
| E | `waitHealthyWithLaunchCheck` 与 `up -d` 不匹配 | compose 正常退出后进程结束，健康检查可能判 `launch process exited` 或超时 → 失败 → 用户重试 |

当前命令（`runAll.yaml`）：

```yaml
command: >-
  (docker compose -f docker-compose.yml up -d 2>/dev/null
  || docker-compose -f docker-compose.yml up -d)
working_dir: task2app/Saas_project
```

## 2. 目标与非目标

### 2.1 目标

1. **首次安装**：本地缺镜像时仍能自动 pull 并启动（开发者零额外步骤）。
2. **重启/再次启动**：本地镜像已存在时 **不再** 走 registry pull（日志无 `Pulling` / `Downloading`）。
3. **runAll UI 语义**：启动/重启 docker-infra 在镜像已齐时应在秒级变为 healthy（8080 就绪）。
4. **停止语义**：UI「关闭」应能 `compose down` 停掉容器（与 AiMonitor `./run.sh stop` 对齐）。
5. 同步更新 `runAll.yaml` 与 `runAll/config.yaml`。

### 2.2 非目标

- 不改 Kafka/Redis 业务配置或端口（仍以 `port_config.json` / 现有 compose 为准）。
- 不做镜像私服/离线包分发。
- 不在本阶段重构 runAll 全部 Docker 服务的通用生命周期（仅 docker-infra + 最小 runner 补丁）。
- 不合并 docker-infra 与 AiMonitor 脚本（保持各子项目独立，仅复用模式）。

## 3. 方案对比

### 方案 A：专用 `run-infra.sh` + compose 约束（推荐）

参考 `AiMonitor/run.sh`：

- `start`（默认）：检测缺镜像 → `compose pull`（仅缺失）→ `compose up -d --pull never`
- `managed`：供 runAll 托管；行为同 start，stdout 可打印 `compose ps`
- `stop`：`compose down --remove-orphans`

并在 `docker-compose.yml` 为各 service 增加 `pull_policy: if_not_present`，将 `kafka-ui:latest` 固定为已验证版本（如 `v0.7.2`）。

**优点：** 改动面小、行为可脚本化测试、与仓库内 AiMonitor 模式一致。  
**缺点：** 需维护镜像列表或 rely on compose pull。

### 方案 B：仅改 runAll.yaml 加 `--pull never`

**优点：** 一行改动。  
**缺点：** 首次无镜像时直接失败；不解决 stop/down；不解决 runner `up -d` 进程退出问题。

### 方案 C：runAll runner 通用「Docker 一次性命令」模式

在 `Service` 增加 `launch_mode: detach` 或检测 `up -d`，健康检查不再因 launch 进程退出而失败。

**优点：** 根治 E。  
**缺点：** 面较大；可与 A 分阶段做。

**结论：MVP 采用 A + C 的最小补丁**（runner 对 command 含 `compose up -d` / `docker-compose up -d` 时，launch 进程退出后继续健康检查直至 URL 就绪或超时）。

## 4. 详细设计

### 4.1 新增 `task2app/Saas_project/run-infra.sh`

```bash
#!/usr/bin/env bash
# 模式: start | managed | stop
# managed/start: 缺镜像则 pull → up -d --pull never --remove-orphans
# stop: compose down --remove-orphans
```

要点：

- `compose()` 封装：优先 `docker compose`，fallback `docker-compose`（与现 runAll 一致）。
- `ensure_images()`：对固定镜像列表 `docker image inspect`；任一缺失则 `compose pull`（一次拉齐）。
- `up_stack()`：`compose up -d --pull never --remove-orphans`。
- `stop_stack()`：`compose down --remove-orphans`。
- `managed` 与 `start` 行为相同（runAll 使用 `bash run-infra.sh managed`）。

固定镜像列表（与 compose 一致）：

- `confluentinc/cp-zookeeper:7.4.0`
- `confluentinc/cp-kafka:7.4.0`
- `redis:7.0-alpine`
- `provectuslabs/kafka-ui:<pinned>`（见 4.2）

### 4.2 `docker-compose.yml` 变更

```yaml
services:
  zookeeper:
    image: confluentinc/cp-zookeeper:7.4.0
    pull_policy: if_not_present
  kafka:
    image: confluentinc/cp-kafka:7.4.0
    pull_policy: if_not_present
  redis:
    image: redis:7.0-alpine
    pull_policy: if_not_present
  kafka-ui:
    image: provectuslabs/kafka-ui:v0.7.2   # 替换 latest
    pull_policy: if_not_present
```

移除 obsolete 的顶层 `version: '3'`（Compose V2 警告，可选清理）。

### 4.3 runAll 配置变更

`runAll.yaml` 与 `runAll/config.yaml`：

```yaml
- name: docker-infra
  command: "bash run-infra.sh managed"
  working_dir: task2app/Saas_project   # config.yaml 仍用 ../task2app/Saas_project
```

**停止路径：** runAll 当前无 `stop_command`。MVP 二选一：

- **4.3a（推荐）** 在 `run-infra.sh` 内文档化；runAll `StopService` 对 docker-infra 通过 **扩展 runner**：若 `working_dir` 存在 `run-infra.sh`，stop 时执行 `bash run-infra.sh stop`（小补丁 `runner.go` + 测试）。
- **4.3b** 仅依赖 port fallback 杀 8080 监听（现状，不 down 容器）— **不采纳**，无法真正停止栈。

采用 **4.3a**。

### 4.4 runAll runner 补丁（最小）

**健康检查（问题 E）：**

在 `waitHealthyWithLaunchCheck` 中，若 launch 命令匹配 `up -d`（compose），则 **进程退出不立即失败**；继续按 backoff 探测 `health_check.url` 直至成功或 `timeout/retries` 耗尽。

**停止（问题 D）：**

`stopService` 在 `stopProcess` 之后、若服务 command 指向 `run-infra.sh`（或 generic：`run*.sh stop` 模式），执行 `bash run-infra.sh stop`（working_dir 内）。

### 4.5 验收标准

| # | 场景 | 期望 |
|---|------|------|
| AC1 | 本地无 Confluent 镜像，首次 UI 启动 | 一次 `compose pull`，随后 up；最终 8080 healthy |
| AC2 | 镜像已齐，UI 重启 docker-infra | 日志 **无** `Pulling`/`Downloading`；30s 内 healthy |
| AC3 | UI 关闭 docker-infra | 容器停止，8080/6379/9093 不可达；status `stopped` |
| AC4 | 再次 UI 启动（AC3 后） | 仅 up，不 pull（镜像仍在） |
| AC5 | `runAll` Go 测试 | 新增/更新 detach compose 健康检查与 stop 脚本测试 |

## 5. 价值流影响

查阅 `value-stream.yaml`：

- **受影响：** `infrastructure` 组 / 二期全局启停验收（注释：`启用 runAll docker-infra 后改回 docker-infra.*`）。
- **无新 stream**；不修改业务表字段。
- **测试影响：** runAll runner 单测；可选 Playwright 对 `:9999` docker-infra 启停（非 MVP 阻塞）。
- Step 3 价值流将把本变更作为 **infrastructure 组可靠性增量** 切片。

## 6. 领域概念清单（供 Step 5 DDD 轻量输入）

| 概念 | 类型 | 说明 |
|------|------|------|
| InfrastructureStack | 聚合根 | Redis + Kafka + ZK + Kafka UI 组合 |
| ContainerImage | 值对象 | `repository:tag` + 本地是否存在 |
| PullPolicy | 值对象 | `if_not_present` / `never` / `on_missing` |
| StackLifecycle | 领域服务 | ensureImages → up / down |
| ReadinessProbe | 值对象 | HTTP 8080，与 runAll health_check 对齐 |

Bounded Context：**本地开发基础设施编排**（与 runAll 平台编排上下文相邻，非 SaaS 业务域）。

## 7. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 固定 kafka-ui 版本与 compose 不兼容 | 选用广泛使用的 `v0.7.2`；文档注明升级路径 |
| `pull_policy` 需 Compose V2 | 与现网一致（已用 docker-compose v2.33） |
| runner 补丁误伤非 compose 服务 | 仅匹配 `up -d` 子串 + 测试覆盖 |
| 首次 pull 仍慢 | 接受；仅发生一次，UI 显示 pulling 合理 |

## 8. 实施顺序（供 Step 6/7）

1. `run-infra.sh` + 可执行权限  
2. `docker-compose.yml` pull_policy + pin tag  
3. `runAll.yaml` / `config.yaml` command 更新  
4. `runner.go` detach health + script stop + 测试  
5. 本地 AC1–AC4 手工验证  
6. 更新 `runAll.yaml.ai.md` 变更日志（一行）

## 9. 批准记录

- [ ] 用户批准设计  
- [ ] 0-auto-flow 继续 Step 3→9
