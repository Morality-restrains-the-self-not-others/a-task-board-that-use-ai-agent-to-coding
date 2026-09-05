# 价值流：领域事件消费者 — 按域多进程 + 可切换传输

> 设计：`docs/superpowers/specs/2026-05-28-taskkafka-consumer-split-design.md`（v2）

## 价值摘要

**运维与开发**在 Django 滚动重启或单域消费者升级时，仍能通过 Kafka/Redis 持续处理领域事件；**终端用户**的注册链、工作区初始化、云主机启停、SSE 推送等异步副作用不再因 monolithic Python consumer 与 Django 同生共死而中断。

## Related Value Streams

| 既有流 | 关系 |
|--------|------|
| **user-auth** | **extension** — `USER_CREATED` / 注册链由 taskEvents-accounts 承接；保留 `saas-backend` 写库字段，新增 `task-events-*` 编排与健康步骤 |
| **project-workspace** / `workspace-creation-handler` | **extension** — `WORKSPACE_CREATED`、`COMPANY_CREATED` 后续步骤改由域进程 + Internal API 触发 |
| **taskauth-split** | **dependency** — 注册仍经 taskAuth；Kafka 副作用描述从「Django 内 consumer」改为「accounts 域进程」 |
| **collect-task-related-projects-scope-fix** | 无直接冲突 — 不同子系统 |
| **runall-cascade-*** | **dependency** — 新增 `task-events-*` 服务节点纳入 cascade / health |

**类型**：平台基础设施 **refactor**（非绿场用户功能），通过新价值流 `domain-events-consumer-split` 跟踪；不替换上述业务流的用户可见步骤名，仅在实现层切换消费者宿主。

## 端到端流（平台视角）

```
[Django 业务 API 提交事务]
  → [IEventPublisher 按 domainEvents.transport 发布]
  → [Kafka topic / Redis Stream]
  → [按域 taskEvents-* 进程消费 + Ack]
  → [POST /api/internal/task-events/<domain>/]
  → [Django ORM / Redis SSE]
  → [用户可见：公司/工作区/云状态/SSE 更新]
```

**隔离验收点（Increment 1 即需满足）**：

```
[仅重启 saas-backend]
  → [taskEvents-accounts 仍消费 USER_CREATED 积压]
  → [注册后公司仍创建]
```

## 价值增量

### Increment 1：薄切片 — Broker 抽象 + accounts 单域进程（Thin Slice）

**用户/运维价值：** 本地 `transport=kafka` 时，accounts 域消费者独立进程健康；Django 重启后注册链异步步骤仍可完成（至少 `USER_CREATED` → 建公司）。

**范围：**

- `taskEvents/`：`Broker` 接口、`KafkaBroker`、accounts `main`、health
- `port_config.domainEvents`（transport + consumers.accounts）
- Django Internal API：`/api/internal/task-events/accounts/`（`USER_CREATED` 最小实现）
- runAll：`task-events-accounts` 一项
- **不**下线 Python monolithic consumer（双跑禁止：切换 flag 二选一）

**依赖：** docker-infra（kafka）或 redis（Increment 3 前可仅 kafka）

**验证：**

| 步骤 | test_file |
|------|-----------|
| broker-kafka-roundtrip | `core/management/commands/test_kafka_consumption.py`（扩展或包装） |
| accounts-consumer-health | 新增 `tests/test_task_events_accounts_health.py` |
| user-created-chain | `accounts/view_test/UserViewSet_email_register_test.py` + `tests/test_workspace_creation_handler.py`（E2E 需启 accounts consumer） |

**数据字段（新增/关注）：**

- `port_config.domainEvents.transport`
- `task-events-accounts.health.status`
- `saas-backend.accounts_company.id`（仍由 Internal API 写入）

---

### Increment 2：传输可切换 — Redis + memory 对称

**价值：** CI/本地可无 Kafka 容器，用 `transport=redis` 跑通 accounts 消费；`memory` 保持现有 pytest 同步路径。

**范围：**

- Go `RedisStreamBroker`
- Django `RedisStreamEventPublisher`（或 Phase 2b 仅 consumer 侧 redis，producer 仍 kafka — 以设计待决为准）
- `port_config` 文档与 `test_port_config_merge.py` 扩展

**验证：**

| 步骤 | test_file |
|------|-----------|
| port-config-transport | `tests/test_port_config_merge.py` |
| redis-transport-consume | 新增 `tests/test_task_events_redis_transport.py` |
| memory-mode-no-process | `tests/test_in_memory_services.py` |

**字段：**

- `port_config.domainEvents.redis.streamKeyPrefix`
- `redis.health.status`（复用 infra redis）

---

### Increment 3：projects + realtime 域进程

**价值：** 工作区创建、任务完成、SSE 推送与 accounts/cloud 解耦部署；Django 重启不影响 SSE。

**范围：**

- `task-events-projects`、`task-events-realtime`
- Internal API projects/realtime；`sse_message` 改 Redis 发布（去 Django 进程内 dict）
- runAll 注册两项

**验证：**

| 步骤 | test_file |
|------|-----------|
| workspace-creation-handler | `tests/test_workspace_creation_handler.py` |
| sse-message | 新增或扩展 SSE 相关 pytest |
| ai-assistant-reply | `tests/test_ai_task_comment.py`（若覆盖事件） |

**字段：**

- `saas-backend.projects_workspace.id`
- `task-events-realtime.health.status`
- `redis.sse.channelPrefix`（与 `taskSSE` 对齐）

---

### Increment 4：cloud 域 + 下线 monolithic consumer

**价值：** 云启停长任务不再阻塞注册；`run.sh` 不再 `pkill start_kafka_consumer`。

**范围：**

- `task-events-cloud`（4 类 cloud 事件）
- Internal API cloud（封装现有 `start_vm` / stop 服务）
- 移除 `manage.py start_kafka_consumer` 与 `run.sh` 步骤 9

**验证：**

| 步骤 | test_file |
|------|-----------|
| cloud-server-start | `tests/test_start_vm_reuse_sse_and_idempotent.py`（及相关 cloud tests） |
| cloud-platform-auth | cloud 域 authorization 测试 |

**字段：**

- `saas-backend.cloud_cloudserverevent.*`
- `task-events-cloud.health.status`

---

### Increment 5：billing 域 + 全量编排（Enhancement）

**价值：** taskBill emit 与审计消费者进程隔离；runAll 文档单一真相。

**范围：**

- `task-events-billing`
- `BILLING_TRANSACTION_CREATED` 路由
- value-stream / runAll 文档更新；禁止双 consumer 检查清单

**验证：**

| 步骤 | test_file |
|------|-----------|
| billing-transaction-event | billing_bridge 相关 pytest |

**字段：**

- `task-bill.billing_transaction.id`（emit 源）
- `task-events-billing.health.status`

---

## 增量排序（按价值，非按技术层）

1. **Increment 1** — 证明进程隔离 + 单域端到端（accounts）
2. **Increment 2** — 降低环境依赖（redis/memory 切换）
3. **Increment 3** — 解耦 projects/SSE（用户可感知的工作区与推送）
4. **Increment 4** — cloud 重负载域 + 删除旧 consumer
5. **Increment 5** — billing + 运维收尾

## 与既有 value-stream.yaml 的 reconciliation 建议

不删除 `user-auth`、`project-workspace` 等业务步骤；**新增**独立流 `domain-events-consumer-split`（或 `platform-domain-events`），避免污染业务流字段表。

可选：在 `user-auth` 的 `email-register` step 增加 **影响标注** 字段：

- `task-events-accounts.health.status` — 描述「异步建公司由该服务消费」

具体 YAML 写入位置待确认（见下方 AskQuestion）。

## 测试与 runAll 服务名映射

| runAll `services[].name` | 价值流 step |
|--------------------------|-------------|
| `task-events-accounts` | accounts-consumer-health |
| `task-events-projects` | projects-consumer-health |
| `task-events-realtime` | realtime-consumer-health |
| `task-events-cloud` | cloud-consumer-health |
| `task-events-billing` | billing-consumer-health |

## 自检

- [x] Increment 1 端到端可验收（注册 → 事件 → 公司）
- [x] 增量按价值排序，非「先写完全部 Go 再接 Django」
- [x] 与 email_queue、Saas_email 边界已区分
- [ ] YAML 写入 `value-stream.yaml` — **待用户确认路径**
