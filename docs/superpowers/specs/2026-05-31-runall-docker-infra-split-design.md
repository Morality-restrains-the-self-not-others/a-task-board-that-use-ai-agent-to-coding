# runAll 基础设施拆分：dockerInfra 目录 + Redis/Kafka 独立启动

**日期:** 2026-05-31  
**状态:** 已批准（方案 A）  
**范围:** 仓库根 `dockerInfra/`、`runAll.yaml` / `runAll/config.yaml`、`runAll/src/compose_lifecycle.go`、迁移 `task2app/Saas_project/docker-compose.yml`

---

## 1. 背景与问题

### 1.1 现状

Redis、Zookeeper、Kafka、Kafka UI 四者打包在 `task2app/Saas_project/docker-compose.yml`，由单一 runAll 服务 `docker-infra` 通过 `bash run-infra.sh managed` 一次性拉起：

```yaml
# runAll.yaml（节选）
- name: docker-infra
  command: "bash run-infra.sh managed"
  working_dir: task2app/Saas_project
  health_check:
    url: "http://127.0.0.1:18080"   # Kafka UI
```

问题：

| # | 问题 | 影响 |
|---|------|------|
| P1 | 基础设施与 Django 项目目录耦合 | `Saas_project/` 本应是应用代码，不应承载全 monorepo 的 Docker 栈 |
| P2 | Redis 与 Kafka 无法独立启停 | 领域事件/邮件队列已默认切到 Redis（见 `2026-05-31-message-queue-kafka-to-redis-design.md`），日常开发仍被迫拉 Kafka 三容器 |
| P3 | 单一 health 探针绑定 Kafka UI | Redis 就绪与否无法单独观测；Kafka 未启时整个 `docker-infra` 被标为 unhealthy |
| P4 | `port_config.json` 已有 `dockerInfra.*` 端口块 | 配置语义与物理目录不一致，新人难以定位 compose 文件 |

### 1.2 驱动力

- **按需启动**：默认开发路径只需 Redis；Kafka 作为可选栈单独启动（兼容 `transport=kafka`、AiProvider 镜像 hash 等遗留路径）。
- **职责清晰**：`dockerInfra/` 作为 monorepo 级本地基础设施模块，与 `runAll/`、`task2app/` 平级。
- **延续既有模式**：保留 `run-infra.sh` 已验证的 pull/up 分离、`--pull never`、UI stop → `compose down` 语义（`2026-05-28-docker-infra-no-repull-design.md`）。

---

## 2. 目标与非目标

### 2.1 目标

1. 新建仓库根目录 **`dockerInfra/`**，将 Redis 与 Kafka 相关 compose 定义迁入。
2. runAll **infrastructure 组**拆为两个独立服务，各对应一条启动命令：
   - `docker-redis` → `dockerInfra/redis/run.sh managed`
   - `docker-kafka` → `dockerInfra/kafka/run.sh managed`
3. 下游 `depends_on` 按实际传输层需求调整：默认依赖 `docker-redis`；Kafka  transport 场景可选依赖 `docker-kafka`。
4. 同步更新 `runAll.yaml`、`runAll/config.yaml`、`runAll.yaml.ai.md`、Prometheus 目标、引用路径。
5. runAll runner 识别新脚本的生命周期（detach 健康检查 + stop 时 compose down）。

### 2.2 非目标

- 不改 Redis/Kafka **端口**（仍以 `port_config.json` → `dockerInfra.redisPort` / `kafkaUiPort` / `kafkaBootstrapServers` 为准）。
- 不删除 Kafka 相关**业务代码**（仅基础设施编排拆分；Kafka compose 仍保留供可选启动）。
- 不在本阶段重构 AiMonitor、gitService 等其他 Docker 栈。
- 不合并 Redis 与 Kafka 为单一 compose（用户明确要求分开两条命令）。

---

## 3. 方案对比

### 方案 A：仓库根 `dockerInfra/{redis,kafka}/` + 两个 runAll 服务（推荐）

```
dockerInfra/
├── README.md
├── redis/
│   ├── docker-compose.yml    # 仅 redis
│   └── run.sh                # start | managed | stop
└── kafka/
    ├── docker-compose.yml    # zookeeper + kafka + kafka-ui
    └── run.sh
```

**优点：** 与 `port_config.dockerInfra` 命名一致；子目录隔离 compose project；脚本模式与 `AiMonitor/run.sh`、`run-infra.sh` 一致。  
**缺点：** 需迁移并删除旧路径；Redis 无 HTTP 探针，需 runner 小扩展（见 §4.4）。

### 方案 B：放在 `runAll/dockerInfra/`

**优点：** 路径靠近编排器。  
**缺点：** 与 `port_config.json` 的 `dockerInfra` 键语义错位；`task2app/run.sh` 等子项目引用路径变长且不直观。**不采纳。**

### 方案 C：保留 `docker-infra` 别名服务，内部顺序调用两个脚本

**优点：** `depends_on` 改动面小。  
**缺点：** 违背「分开两个命令」诉求；UI 仍是一个开关，无法独立启停 Kafka。**不采纳。**

**结论：采用方案 A。**

---

## 4. 详细设计

### 4.1 目录与 Compose 拆分

#### `dockerInfra/redis/docker-compose.yml`

```yaml
services:
  redis:
    image: redis:7.0-alpine
    pull_policy: if_not_present
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  redis_data:
```

#### `dockerInfra/kafka/docker-compose.yml`

自 `task2app/Saas_project/docker-compose.yml` 迁移 `zookeeper`、`kafka`、`kafka-ui` 三段（镜像 tag、`pull_policy`、端口 2181/9092/9093/18080 保持不变）。

#### `dockerInfra/redis/run.sh` 与 `dockerInfra/kafka/run.sh`

复用 `run-infra.sh` 行为，按栈缩小 `REQUIRED_IMAGES`：

| 脚本 | REQUIRED_IMAGES | up 说明 |
|------|-----------------|---------|
| `redis/run.sh` | `redis:7.0-alpine` | `compose up -d --pull never --remove-orphans` |
| `kafka/run.sh` | cp-zookeeper:7.4.0, cp-kafka:7.4.0, kafka-ui:v0.7.2 | 同上 |

模式：`start` | `managed` | `stop`（与现 `run-infra.sh` 一致；runAll 使用 `managed`）。

### 4.2 runAll 配置变更

**删除** 单一 `docker-infra` 服务，**新增**：

```yaml
# runAll.yaml — infrastructure 组
- name: docker-redis
  command: "bash run.sh managed"
  working_dir: dockerInfra/redis
  health_check:
    tcp: "127.0.0.1:6379"          # 新增探针类型，见 §4.4
    timeout: 60
    retries: 20
    backoff:
      initial: 1.0
      max: 8.0
      multiplier: 2.0
  on_failure: skip

- name: docker-kafka
  command: "bash run.sh managed"
  working_dir: dockerInfra/kafka
  health_check:
    url: "http://127.0.0.1:18080"    # Kafka UI，与 dockerInfra.kafkaUiPort 一致
    timeout: 180
    retries: 40
    backoff:
      initial: 2.0
      max: 16.0
      multiplier: 2.0
  on_failure: skip
```

`runAll/config.yaml` 同步，`working_dir` 改为相对 `runAll/` 的 `../dockerInfra/redis` 与 `../dockerInfra/kafka`。

### 4.3 依赖图调整

```text
Level 0（并行）: docker-redis, docker-kafka, ai-monitor, git-service, git-oauth,
                 go-run-container, go-relay, value-stream, task-auth, task-bill

Level 1: task-sse  ← depends_on: [docker-redis]

Level 2: saas-backend  ← depends_on: [task-auth, git-oauth, docker-redis, task-sse]

Level 3: task-events-*  ← depends_on: [docker-redis, saas-backend]
         （不再硬依赖 docker-kafka；transport=kafka 时由开发者手动启 docker-kafka）
```

| 原 depends_on | 新 depends_on | 说明 |
|---------------|---------------|------|
| `docker-infra` | `docker-redis` | task-sse、saas-backend、5× task-events-* |
| — | （无自动）`docker-kafka` | 默认队列已 Redis；Kafka 栈 optional，UI 单独启停 |

**可选增强（非 MVP 阻塞）：** 若 `port_config.json` 中 `domainEvents.transport` / `email_queue.type` 为 `kafka`，文档注明需手动启动 `docker-kafka`；或后续增量让 runAll 读配置动态注入 depends_on。

### 4.4 runAll runner：Redis TCP 健康检查

runAll 现仅支持 HTTP `health_check.url`。Redis 无 HTTP 端点，**最小扩展**：

```yaml
health_check:
  tcp: "127.0.0.1:6379"   # 与 url 二选一；tcp 优先探测 TCP connect
  url: "http://..."       # 现有字段
```

实现要点（`config.go` + `runner.go`）：

- 校验：`url` 与 `tcp` 至少其一非空。
- 探针：`net.DialTimeout("tcp", host:port, ...)` 成功即 healthy。
- 测试：`config_test.go` 解析用例 + `runner_test.go` 模拟 listener。

### 4.5 compose 生命周期识别泛化

`compose_lifecycle.go` 当前硬编码 `run-infra.sh`。改为：

```go
// 匹配 dockerInfra/*/run.sh 或 legacy run-infra.sh
func isInfraManagedScript(command string) bool
func infraStopShellCommand(svc *Service) string // bash run.sh stop
```

- `isDetachLaunchCommand`：command 含 `run.sh managed` 且 working_dir 在 `dockerInfra/` 下 → detach。
- `composeStopShellCommand`：stop 时执行同目录 `bash run.sh stop`。

保留对 `run-infra.sh` 的兼容直至旧路径删除（一个发布周期）。

### 4.6 旧路径迁移与清理

| 路径 | 动作 |
|------|------|
| `task2app/Saas_project/docker-compose.yml` | **删除**（或短期保留 stub 打印「已迁至 dockerInfra/」并 exit 1） |
| `task2app/Saas_project/run-infra.sh` | **删除**（同上可选 stub） |
| `task2app/run.sh` | 更新提示文案：`dockerInfra/redis` 替代 `docker-infra` |
| `runAll.yaml.ai.md` | 服务表、DAG、变更日志 |
| `AiMonitor/prometheus/file_sd/runall-health-targets.json` | `docker-infra` → `docker-redis`（tcp 目标若 Prometheus 不支持可暂仅 kafka 条目 + 文档说明） |
| `scripts/docker-desktop-helper.sh` | 注释/路径更新 |

### 4.7 `dockerInfra/README.md`

说明：

- 两个栈的用途、端口、`port_config.json` 对应键。
- 手工命令：`bash dockerInfra/redis/run.sh start`、`bash dockerInfra/kafka/run.sh stop`。
- 默认开发只需 Redis；Kafka 场景与 `domainEvents.transport=kafka` 的关系。

---

## 5. 验收标准

| # | 场景 | 期望 |
|---|------|------|
| AC1 | runAll UI 仅启动 `docker-redis` | 6379 可达；Kafka UI 18080 **不可达**；task-sse / saas-backend 可级联启动 |
| AC2 | runAll UI 启动 `docker-kafka` | 18080/9093 可达；日志无重复 Pulling（沿用 pull 分离） |
| AC3 | UI 分别 stop redis / kafka | 各栈 `compose down`，端口释放 |
| AC4 | 镜像已齐时重启任一栈 | 秒级 healthy，无 registry pull |
| AC5 | `cd runAll && go test ./...` | TCP 健康检查 + 新 stop 脚本测试通过 |
| AC6 | 旧路径 `task2app/Saas_project/run-infra.sh` | 不存在或明确报错指向新路径 |

---

## 6. 价值流影响

查阅 `value-stream.yaml`：

| 维度 | 评估 |
|------|------|
| **受影响 stream** | `runall-cascade-lifecycle`（infrastructure 组上游变更）；`runall-global-start-stop-all`（planned，字段注释中的 `docker-infra.*` 应改为 `docker-redis.*` / `docker-kafka.*`） |
| **新 stream** | 不需要 |
| **字段 impact** | 无业务表字段；runtime 观测字段命名从 `docker-infra.*` 拆为两个服务名 |
| **测试 impact** | `runAll/src/compose_lifecycle_test.go`、`config_test.go`、`runner_test.go`；可选 Playwright 对 `:9999` 独立启停 |
| **status 变化** | 无 active step 回退；planned 步骤描述需同步 |
| **交叉依赖** | platform 链默认仅依赖 Redis，缩短 Level 0 等待（Kafka 可选） |

Step 3 价值流将把本变更作为 **infrastructure 组拆分增量** 切片。

---

## 7. 领域概念清单（供 Step 5 DDD）

| 概念 | 类型 | 说明 |
|------|------|------|
| **InfrastructureStack** | 聚合（两个） | `RedisStack`、`KafkaStack` 各自独立生命周期 |
| **ContainerImage** | 值对象 | `repository:tag` + 本地 inspect 结果 |
| **PullPolicy** | 值对象 | compose `if_not_present` + CLI `--pull never` |
| **StackLifecycle** | 领域服务 | ensureImages → up / down |
| **ReadinessProbe** | 值对象 | Redis: TCP 6379；Kafka: HTTP Kafka UI 18080 |
| **TransportProfile** | 值对象 | 由 `port_config` 决定下游是否需 Kafka 栈 |

**Bounded Context：** 本地开发基础设施编排（与 runAll 平台编排相邻，非 SaaS 业务域）。

---

## 8. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 开发者脚本仍引用旧 compose 路径 | stub 报错 + README + grep CI 检查 |
| Redis TCP 探针实现遗漏 | TDD 覆盖；MVP 可暂用 `on_failure: skip` + 下游 nc 作兜底（不推荐长期使用） |
| 拆分后 compose 默认 project 名导致 volume 迁移 | 首次 up 新建 volume；旧 `saas_project_redis_data` 可文档说明 `docker volume` 迁移或接受空库 |
| Prometheus 无 TCP target | kafka 仍 HTTP；redis 可后续加 redis_exporter 或 blackbox |

---

## 9. 实施顺序（供 Step 6/7）

1. 创建 `dockerInfra/redis/`、`dockerInfra/kafka/` + `run.sh` + compose  
2. runAll TCP health_check + compose_lifecycle 泛化 + 测试  
3. 更新 `runAll.yaml` / `config.yaml` 与 depends_on  
4. 删除/ stub 旧 `Saas_project` 基础设施文件  
5. 同步文档、Prometheus 目标、`task2app/run.sh` 提示  
6. 本地 AC1–AC6 验证  

---

## 10. 批准记录

- [x] 用户批准设计（**方案 A**：仓库根 `dockerInfra/{redis,kafka}/` + 两个 runAll 服务）
- [x] 目录位置确认：仓库根 `dockerInfra/`（非 `runAll/dockerInfra/`）
- [x] Redis 健康检查：TCP 探针扩展（§4.4）
- [x] 旧 compose 处理方式：直接删除
