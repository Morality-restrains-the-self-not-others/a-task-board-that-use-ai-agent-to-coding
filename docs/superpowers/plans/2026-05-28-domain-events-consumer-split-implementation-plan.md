# 领域事件消费者拆分 — 实施计划

> **For agentic workers:** 按任务顺序执行；每步先写失败测试再实现。使用 checkbox 跟踪进度。

**Goal:** 将 task2app 领域事件消费从 Django 单进程拆为按域独立的 `taskEvents` Go 服务，底层 broker 可通过 `port_config.domainEvents.transport` 在 kafka/redis/memory 间切换，满足 Django 重启不影响消费（NFR L3）。

**实施状态（2026-05-28）：** Increment 1–5 代码与编排已落地；Django 侧 Internal dispatch 测试已覆盖五域；`value-stream.yaml` 对应 step 已标 `active`。Go `broker`/`cmd` 包测试依赖 `kafka-go` 传递模块下载（网络慢时见 `DOMAIN_EVENTS.md` 验证节）。

**Architecture:** Go `event-delivery` 限界上下文（`taskEvents/domain/`）通过 `EventBrokerPort` 适配 Kafka/Redis；`IdempotentDispatchService` 调用 Django `/api/internal/task-events/<domain>/`；runAll 按域注册 `task-events-*` 服务。Producer 暂留 Django，Increment 2 补 Redis 发布。

**Tech Stack:** Go 1.22, franz-go 或 confluent-kafka-go, go-redis, Django 4.x, pytest, runAll.yaml, port_config.json

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-28-taskkafka-consumer-split-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-28-domain-events-consumer-split-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-05-28-domain-events-consumer-split-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-05-28-domain-events-consumer-split-ddd-domain-model.md`

---

## 文件结构（锁定）

| 路径 | 职责 |
|------|------|
| `taskEvents/domain/*` | 已有：信封、路由、派发、端口 |
| `taskEvents/src/config.go` | 读 `port_config.domainEvents` |
| `taskEvents/src/broker/kafka.go` | `EventBrokerPort` Kafka 实现 |
| `taskEvents/src/broker/redis.go` | Increment 2 |
| `taskEvents/src/http/domain_command_client.go` | `DomainCommandPort` |
| `taskEvents/src/idempotency/memory.go` | 进程内幂等（Increment 1） |
| `taskEvents/cmd/accounts/main.go` | accounts 域消费者入口 |
| `taskEvents/run.sh` | start/stop/status accounts\|cloud\|... |
| `task2app/conf/port_config.json` | `domainEvents` 段 |
| `task2app/Saas_project/accounts/urls_task_events_internal.py` | Internal 路由 |
| `task2app/Saas_project/accounts/task_events_internal_views.py` | USER_CREATED 处理 |
| `runAll.yaml` | `task-events-accounts` 服务 |
| `value-stream.yaml` | 已追加 `domain-events-consumer-split` |

**切换门禁:** 环境变量 `DOMAIN_EVENTS_CONSUMER=go` 时禁止启动 `manage.py start_kafka_consumer`（防双消费）。

---

## Increment 1: accounts 薄切片 + Kafka + 进程隔离

### Task 1.1: port_config 与配置加载

**Files:**
- Modify: `task2app/conf/port_config.json`
- Create: `task2app/Saas_project/tests/test_domain_events_port_config.py`
- Create: `taskEvents/src/config.go`
- Create: `taskEvents/src/config_test.go`

- [x] **Step 1:** 在 `port_config.json` 增加 `domainEvents`（transport、kafka、consumers.accounts.port=8005、groupId、events 列表）
- [ ] **Step 2:** 写失败测试 `test_domain_events_port_config.py` 断言 JSON 结构与 merge 规则
- [ ] **Step 3:** 实现 `taskEvents/src/config.go`（`findMonorepoRoot` 对齐 taskBill）
- [ ] **Step 4:** `cd taskEvents && go test ./src/... -run Config`

### Task 1.2: Kafka Broker 适配器

**Files:**
- Create: `taskEvents/src/broker/kafka.go`
- Create: `taskEvents/src/broker/kafka_test.go`（可用 mock 或 integration build tag）

- [ ] **Step 1:** `go get` 选定 Kafka 客户端依赖
- [ ] **Step 2:** 实现 `Subscribe` / `Ack` / `Close`，消息体解析为 `domain.EventEnvelope`
- [ ] **Step 3:** 单元测试：JSON envelope 解析（不依赖真实 broker）
- [ ] **Step 4:** 手动验证脚本：对照 `task2app/scripts/test_kafka_consumption.sh`

### Task 1.3: Domain command HTTP 客户端

**Files:**
- Create: `taskEvents/src/http/domain_command_client.go`
- Create: `taskEvents/src/http/domain_command_client_test.go`

- [ ] **Step 1:** 测试：mock server 返回 502 → `DispatchRetryable`，200 → `DispatchSuccess`
- [ ] **Step 2:** 实现 POST + `X-TaskEvents-Internal-Secret` + 30s timeout
- [ ] **Step 3:** `go test ./src/http/...`

### Task 1.4: accounts 消费者进程 + health

**Files:**
- Create: `taskEvents/cmd/accounts/main.go`
- Create: `taskEvents/run.sh`
- Create: `taskEvents/src/health.go`
- Create: `task2app/Saas_project/tests/test_task_events_accounts_health.py`

- [ ] **Step 1:** pytest：请求 `http://127.0.0.1:8005/api/health/`（可先 skip 若服务未启）
- [ ] **Step 2:** `main.go` 组装 `ConsumerSession`（events: USER_CREATED, COMPANY_CREATED）、router、`IdempotentDispatchService`、消费循环
- [ ] **Step 3:** health 返回 transport、domain、subscribed_events
- [ ] **Step 4:** `./run.sh start accounts` 本地验证

### Task 1.5: Django Internal API — USER_CREATED

**Files:**
- Create: `task2app/Saas_project/accounts/task_events_internal_views.py`
- Create: `task2app/Saas_project/accounts/urls_task_events_internal.py`
- Modify: `task2app/Saas_project/saas_project/urls.py`
- Modify: `task2app/Saas_project/accounts/views` 或抽取 `ensure_default_company` 应用服务

- [ ] **Step 1:** pytest：POST internal user-created 幂等（同 user_id 两次 → 一个 Company）
- [ ] **Step 2:** 实现 view：校验 secret，调用与 `0_create_company.active.py` 相同逻辑
- [ ] **Step 3:** 注册 URL
- [ ] **Step 4:** `pytest accounts/view_test/UserViewSet_email_register_test.py`（需 `DOMAIN_EVENTS_CONSUMER=go` + accounts 进程运行，或 mock）

### Task 1.6: runAll 编排 + 禁用旧 consumer

**Files:**
- Modify: `runAll.yaml`
- Modify: `task2app/run.sh`（`DOMAIN_EVENTS_CONSUMER=go` 时跳过步骤 9）
- Modify: `task2app/.env_django.yaml.example` 文档

- [ ] **Step 1:** runAll 增加 `task-events-accounts`，depends_on saas-backend + docker-infra
- [ ] **Step 2:** run.sh 步骤 9 加门禁注释与条件跳过
- [ ] **Step 3:** 手工 QS-01：只重启 Django，accounts consumer 仍运行并处理积压

### Task 1.7: 价值流测试文件

**Files:**
- Create: `task2app/Saas_project/tests/test_task_events_accounts_health.py`
- Modify: `value-stream.yaml` step `increment1-accounts-thin-slice` status → active（文件存在后）

- [ ] **Step 1:** 实现 health pytest
- [ ] **Step 2:** `cd valueStream && go test ./...`

---

## Increment 2: transport redis + Django Redis 发布

### Task 2.1: Redis Stream Broker

**Files:**
- Create: `taskEvents/src/broker/redis.go`
- Create: `task2app/Saas_project/tests/test_task_events_redis_transport.py`

- [ ] **Step 1:** 实现 `RedisStreamBroker`（XREADGROUP / XACK）
- [ ] **Step 2:** 配置校验：禁止 kafka+redis 双消费同一事件
- [ ] **Step 3:** E2E：transport=redis，无 Kafka 容器，注册→建公司

### Task 2.2: Django RedisEventPublisher

**Files:**
- Create: `task2app/Saas_project/core/services/redis_stream_event_publisher.py`
- Modify: `task2app/Saas_project/core/services/registry.py`

- [ ] **Step 1:** `test_in_memory_services` 仍通过
- [ ] **Step 2:** `domainEvents.transport=redis` 时 `get_event_publisher()` 返回 Redis 实现
- [ ] **Step 3:** `pytest tests/test_port_config_merge.py` 扩展

---

## Increment 3: projects + realtime 域

### Task 3.1: task-events-projects

**Files:**
- Create: `taskEvents/cmd/projects/main.go`
- Create: `task2app/Saas_project/projects/task_events_internal_views.py`
- Modify: `runAll.yaml`, `port_config.json` consumers.projects

- [ ] **Step 1:** WORKSPACE_CREATED internal API + pytest `test_workspace_creation_handler.py`
- [ ] **Step 2:** projects 消费者仅订阅 projects 域 events

### Task 3.2: task-events-realtime (SSE)

**Files:**
- Create: `taskEvents/cmd/realtime/main.go`
- Create: `task2app/Saas_project/tests/test_task_events_realtime_sse.py`
- Refactor: `sse_message` handler → Redis publish（去掉 Django 进程内 dict 依赖）

- [ ] **Step 1:** 对齐 `port_config.taskSSE.redis.channelPrefix`
- [ ] **Step 2:** QS：SSE Redis publish P95 开发环境可观测

---

## Increment 4: cloud 域 + 下线 Python consumer

### Task 4.1: task-events-cloud

**Files:**
- Create: `taskEvents/cmd/cloud/main.go`
- Create: `task2app/Saas_project/cloud/task_events_internal_views.py`（封装 start_vm/stop 服务）
- Modify: `task2app/run.sh` 删除 `start_kafka_consumer` 默认路径

- [ ] **Step 1:** 按事件拆分 internal 端点（可先 1 个聚合 dispatch）
- [ ] **Step 2:** `pytest tests/test_start_vm_reuse_sse_and_idempotent.py` 回归
- [ ] **Step 3:** 确认无 `saas-project-group` 第二消费者

---

## Increment 5: billing + 文档

### Task 5.1: task-events-billing

- [ ] Internal API：billing_transaction_created 审计
- [ ] runAll + value-stream step `increment5-billing-consumer` → active

### Task 5.2: 文档

- [ ] 更新 `KAFKA_INTEGRATION.md` → `DOMAIN_EVENTS.md`
- [ ] `init_project.sh` 不再 kill task-events 进程

---

## 验证命令汇总

```bash
# Go 领域层
cd taskEvents && go test ./domain/... ./src/...

# Django
cd task2app/Saas_project && DJANGO_SETTINGS_MODULE=saas_project.settings_test \
  pytest tests/test_domain_events_port_config.py \
         tests/test_task_events_accounts_health.py \
         accounts/view_test/UserViewSet_email_register_test.py -q

# 价值流配置
cd valueStream && go test ./...

# 隔离手测（QS-01）
# 1. runAll 启 task-events-accounts + saas-backend
# 2. 注册用户 → 观察 consumer 日志 / 公司行
# 3. 仅 restart saas-backend → 再注册 → 两用户均应有公司
```

---

## 风险检查清单（每 Increment 末）

- [ ] 无同 topic 双 consumer group 并行
- [ ] offset 仅在 Internal API 成功后提交
- [ ] `USER_CREATED` 幂等键 `user_id` 实测
- [ ] memory 模式不启动 task-events-*

---

## 完成后

调用 `/7-build-构建`，从 **Task 1.1** 开始 TDD 执行；或先 `/2-worktrees-工作隔离` 在独立 worktree 实施。
