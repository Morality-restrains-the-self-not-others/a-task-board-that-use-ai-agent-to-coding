# 设计文档：conf/ 服务全量迁移至本机 (10.2.150.89)

> **状态:** 待审批  
> **创建日期:** 2026-06-05  
> **类型:** 配置变更（基础设施迁移）  
> **影响范围:** `conf/` 目录下所有服务配置

---

## 1. 背景与动机

当前 `conf/` 中的基础设施服务（Redis、Kafka、Portainer、AiMonitor、git-service）运行在远程 CPU 云主机 `10.2.150.119` 上，通过 SSH + Docker context (`zcpu-remote`) 管理。平台服务（Django、task-auth、task-gateway 等）运行在本机 `10.2.150.89`。

用户要求将所有服务统一安排到本机 `10.2.150.89`，消除远程依赖，简化本地开发环境。

### 当前架构

```
┌─────────────────────────────────┐     ┌──────────────────────────┐
│  本机 Mac (10.2.150.89)         │     │  远程 CPU (10.2.150.119)  │
│                                 │     │                          │
│  Django       :8001             │     │  Redis        :6379      │
│  task-auth    :8003             │     │  Kafka        :9093      │
│  task-bill    :8004             │     │  Portainer    :9000      │
│  git-oauth    :8002             │     │  AiMonitor    :3000      │
│  ai-provider  :8010             │     │  git-service  :8012      │
│  task-gateway :8080/8443        │     │  Loki         :3100      │
│  Go services  :8011-8014,8795-7 │     │  Tempo        :4317/4318 │
│  Vue frontend :4000             │     │                          │
│  taskEvents   :18020-18037      │     │                          │
└─────────────────────────────────┘     └──────────────────────────┘
         ${HOST} = 10.2.150.89               ${INFRA_HOST} = 10.2.150.119
```

### 目标架构

```
┌──────────────────────────────────────────┐
│  本机 Mac (10.2.150.89)                   │
│                                          │
│  基础设施 (原远程):                        │
│  Redis        :6379                      │
│  Kafka        :9093                      │
│  Portainer    :9000                      │
│  AiMonitor    :3000                      │
│  git-service  :8012                      │
│  Loki         :3100                      │
│  Tempo        :4317/4318                 │
│                                          │
│  平台服务 (不变):                          │
│  Django       :8001                      │
│  task-auth    :8003                      │
│  ... (同当前)                             │
│  taskEvents   :18020-18037               │
└──────────────────────────────────────────┘
         ${INFRA_HOST} = ${HOST} = 10.2.150.89
```

---

## 2. 影响分析

### 2.1 需要修改的文件

#### 权威配置源（手工编辑）

| # | 文件 | 变更内容 |
|---|------|---------|
| 1 | `conf/docker-infra/config.yaml` | `host`、`redis.host`、`kafka.bootstrapServers`、`kafkaBootstrapServers` 全部 `10.2.150.119` → `10.2.150.89` |
| 2 | `conf/task-gateway/config.yaml` | `host: 10.2.150.119` → `host: 10.2.150.89` |
| 3 | `conf/vue/config.yaml` | `apiBaseUrl: http://10.2.150.119:8080` → `http://10.2.150.89:8080` |

#### 自动生成的同步片段（通过 `sync.sh` 重新生成，禁止手工编辑）

| # | 文件 | 来源 |
|---|------|------|
| 4 | `conf/core/django/docker-infra.yaml` | `conf/docker-infra/config.yaml` → sync |
| 5 | `conf/task-sse/docker-infra.yaml` | `conf/docker-infra/config.yaml` → sync |
| 6 | `conf/domain-events/docker-infra.yaml` | `conf/docker-infra/config.yaml` → sync |

#### 文档更新

| # | 文件 | 变更内容 |
|---|------|---------|
| 7 | `conf/README.md` | 第15行：更新 `host: 10.2.150.119` → `host: 10.2.150.89` 及描述文字 |
| 8 | `conf/runAll.yaml.ai.md` | 第219行及其他引用：更新 `10.2.150.119` → `10.2.150.89` 描述 |

### 2.2 无需修改的文件

所有平台服务的 `config.yaml` 已指向 `10.2.150.89`，无需变更：

| 服务 | 文件 | 当前 host |
|------|------|-----------|
| django | `conf/core/django/config.yaml` | `10.2.150.89` ✅ |
| task-auth | `conf/task-auth/config.yaml` | `10.2.150.89` ✅ |
| ai-provider | `conf/ai-provider/config.yaml` | `10.2.150.89` ✅ |
| git-service | `conf/git-service/config.yaml` | `10.2.150.89` ✅ |
| task-bill | `conf/task-bill/config.yaml` | `10.2.150.89` ✅ |
| task-sse | `conf/task-sse/config.yaml` | `10.2.150.89` ✅ |
| task-agent-support | `conf/task-agent-support/config.yaml` | `10.2.150.89` ✅ |
| task-ai-endpoint | `conf/task-ai-endpoint/config.yaml` | `10.2.150.89` ✅ |
| task-container-gateway | `conf/task-container-gateway/config.yaml` | `10.2.150.89` ✅ |
| relay-to-trae | `conf/relay-to-trae/config.yaml` | `10.2.150.89` ✅ |
| mock-run-container | `conf/mock-run-container/config.yaml` | `10.2.150.89` ✅ |
| mock-trae-worker | `conf/mock-trae-worker/config.yaml` | `10.2.150.89` ✅ |

### 2.3 `runAll.yaml` 行为变化

`runAll.yaml` 使用 `${INFRA_HOST}` 和 `${HOST}` 占位符，运行时自动从 `conf/docker-infra/config.yaml` 和平台服务的 `conf_app` 映射中解析：

- **`${INFRA_HOST}`**：将从 `10.2.150.119` 变为 `10.2.150.89`（自动解析自 `docker-infra/config.yaml`）
- **`${HOST}`**：保持 `10.2.150.89` 不变
- **效果**：迁移后 `${INFRA_HOST}` = `${HOST}` = `10.2.150.89`，所有健康检查 URL 统一指向本机

`runAll.yaml` 的 `remote_docker` 块（SSH + Docker context）需要配合实际 Docker 运行位置调整，但这超出了 conf 配置变更的范围，属于 `scripts/runall-remote-docker.sh` 的职责。

---

## 3. 前置条件

迁移前需确认本机已具备以下能力：

1. **Docker 运行环境**：本机需要 Docker（Docker Desktop 或 Colima），因为 Redis、Kafka、Portainer、AiMonitor、git-service 通过 Docker Compose 栈运行
2. **端口可用性**：确认以下端口在本机未被占用：`6379`、`9093`、`9000`、`3000`、`3100`、`8012`、`18080`、`4317`、`4318`
3. **Docker 资源**：确保本机有足够的内存/CPU 运行额外的容器栈
4. **数据迁移**（可选）：如需保留远程 Redis/Kafka 中的数据，需制定迁移方案

---

## 4. 实施步骤概要

```
步骤1: 编辑权威配置源（3 个文件）
  ├── conf/docker-infra/config.yaml    (4 处替换)
  ├── conf/task-gateway/config.yaml     (1 处替换)
  └── conf/vue/config.yaml              (1 处替换)

步骤2: 运行同步脚本
  └── bash scripts/conf-sync-all.sh     (重新生成 GENERATED fragments)

步骤3: 验证
  ├── grep -rn "10.2.150.119" conf/    (确认无残留)
  ├── python3 scripts/conf-read.py snapshot-json  (确认配置完整)
  └── bash scripts/ci/check_conf_sync.sh          (CI 检查通过)

步骤4: 更新文档
  ├── conf/README.md
  └── conf/runAll.yaml.ai.md
```

---

## 5. 价值流影响

### 5.1 受影响的现有流

本次变更为基础设施配置迁移，主要影响以下流：

| 价值流 | 影响说明 |
|--------|---------|
| `platform-centralized-logging` | `observability.grafana_url` 和 `loki_url` 的 `${INFRA_HOST}` 指向变更 |
| `domain-events-consumer-split` | taskEvents 消费者的 Kafka/Redis 连接地址变更 |
| `message-queue-kafka-to-redis` | 消息队列基础设施连接地址变更 |
| `runall-cascade-lifecycle` | 基础设施服务（Redis/Kafka）的 TCP/HTTP 健康检查地址变更 |
| `platform-dev-database-reset` | `docker-redis` FLUSHALL 和 `docker-kafka` topic 重建的目标地址变更 |

### 5.2 新增/变更字段

无新增数据库字段。仅变更配置文件中已有的 host/IP 值。

### 5.3 测试影响

- 配置同步 CI 检查 (`scripts/ci/check_conf_sync.sh`) 需重新通过
- 无业务测试需要变更（host 是配置层，不影响测试逻辑）

### 5.4 跨流依赖

无新增跨流依赖。但需要注意：所有依赖 `${INFRA_HOST}` 的流在迁移后统一指向本机，如本机 Docker 基础设施未就绪，所有相关健康检查将失败。

---

## 6. 领域概念清单

本变更为纯配置迁移，不涉及新业务领域概念。但触及以下现有限界上下文的基础设施层：

| 限界上下文 | 影响 |
|-----------|------|
| **平台可观测性** (platform-centralized-logging) | Grafana/Loki/Tempo 地址变更 |
| **系统管理与策略** (message-queue, domain-events) | Redis/Kafka 连接地址变更 |
| **云平台与资源** (container-stack) | Docker 基础设施地址变更 |

---

## 7. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 本机 Docker 资源不足 | 基础设施服务启动失败 | 迁移前评估资源；可按需禁用非必要服务 (on_failure: skip) |
| 端口冲突 | 服务无法绑定 | `lsof -i :PORT` 预检 |
| 远程数据丢失 | Redis/Kafka 数据不可用 | 开发环境可重建（`clear-databases` + `init-databases`） |
| 回滚需求 | 需要切回远程 CPU | Git 版本控制，一条 `git revert` 即可 |

---

## 8. 审批检查清单

- [ ] 确认本机具备 Docker 运行环境
- [ ] 确认本机资源充足（内存/CPU）
- [ ] 确认目标端口无冲突
- [ ] 确认是否需要保留远程数据
- [ ] 审阅变更文件列表
- [ ] 批准后进入价值流映射（步骤3）或直接实施
