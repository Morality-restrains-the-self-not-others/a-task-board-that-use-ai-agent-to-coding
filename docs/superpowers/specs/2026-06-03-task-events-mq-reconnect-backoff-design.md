# task-events 消息队列断连 CPU 暴涨修复设计

**日期:** 2026-06-03  
**状态:** 已批准（auto-flow）  
**范围:** `taskEvents/broker`、`consumer/runner`、`server/health`、`runAll.yaml`

## 决策记录

| 能力 | 结论 |
|------|------|
| MQ 不可达 — 启动 | 有限次 Ping 重试后 **exit 1** |
| MQ 不可达 — 运行期 | **指数退避**重连，禁止 tight loop |
| Health | split probe：`/api/health/` liveness + `/api/health/ready` readiness |
| 方案 | Broker 层退避 + 启动 Ping（非仅 runAll depends_on） |

---

## 背景与问题

### 已确认根因

| 现象 | 根因 |
|------|------|
| `task-events-*` CPU 暴涨 | `broker/redis.go`、`broker/kafka.go` 连接失败时 `continue` **无 sleep** |
| Redis `Block: 2s` 无效 | 仅在连接成功时阻塞；dial 失败立即返回 |
| 18 个 intent 进程放大 | 每个独立 tight loop |
| Health 始终 `ok` | 不反映 MQ 连通性 |

### 本期目标

- MQ 不可达时单进程 CPU **< 1%**
- 启动 MQ 不可达 → 5 次退避后 **exit 1**
- 运行中断连 → 100ms–30s 指数退避，恢复后自动消费
- runAll 可区分 liveness（进程存活）与 readiness（MQ 连通）

### 非目标

- 不改 runAll 停服逻辑
- 不删 Kafka 支持
- 不改 `depends_on` 编排

---

## 领域概念清单（供 `/5-ddd` 引用）

| 类型 | 名称 | 职责 |
|------|------|------|
| 限界上下文 | domain-events 消息传输 | broker 适配与连接策略 |
| 值对象 | `BackoffPolicy` | 退避初始/上限/倍数 |
| 值对象 | `ConnectionState` | MQ 连通布尔状态 |
| 端口 | `EventBrokerPort` | Subscribe/Ack/Close（不变） |
| 领域服务 | `BrokerReconnectPolicy` | 启动 Ping + 运行退避 |

---

## 价值流影响

**受影响 stream:** `domain-events-consumer-split`、`message-queue-kafka-to-redis`

| 步骤 | 影响 |
|------|------|
| intent health 步骤 | `task-events-*.health.status` 扩展 readiness 语义 |
| integration tests | Redis 可用时行为不变 |

**新字段示例:**

```yaml
- name: task-events-user-created-0-create-company.health.readiness_status
  description: /api/health/ready 返回 ok|degraded（MQ 连通性）
- name: task-events-user-created-0-create-company.runtime.mq_connected
  description: broker ConnectionState；断连时 false
```

---

## 详细设计

### 1. 共享退避 — `broker/retry.go`

| 场景 | Initial | Max | 次数 |
|------|---------|-----|------|
| 启动 Ping | 1s | 16s | 5 |
| 运行重连 | 100ms | 30s | ∞（直到 ctx 取消） |

日志：同一退避级别每 30s 最多 warn 一次。

### 2. 启动探测 — `broker/ping.go`

| Transport | 探测 |
|-----------|------|
| Redis | `PING` |
| Kafka | TCP dial bootstrap |

调用点：`consumer/runner.go` 在 `Subscribe` 前 `PingWithRetry(5)`，失败 `log.Fatalf`。

### 3. 运行期退避 — `redis.go` / `kafka.go`

| 错误 | 行为 |
|------|------|
| `ctx.Done()` | 退出 |
| Redis `Nil` | 立即 continue |
| 连接/网络错误 | `backoff.Wait` + `connected=false` |
| 成功读消息 | `backoff.Reset` + `connected=true` |

### 4. Health — split probe

| 路径 | HTTP | status |
|------|------|--------|
| `/api/health/` | 200 | `ok` |
| `/api/health/ready` | 200/503 | `ok`/`degraded` |

`runAll.yaml` 18 个 `task-events-*` 增加 `liveness_url` + readiness `url`。

### 5. 测试

| 测试 | 验证 |
|------|------|
| `broker/retry_test.go` | 退避递增、Reset、ctx 取消 |
| `broker/ping_test.go` | dead port 5 次失败 |
| `server/health_test.go` | ready 503 when disconnected |
| 现有 integration | 回归 |

---

## 数据流

```
start → PingWithRetry(×5) → [fail exit 1]
      → health HTTP → Subscribe → loop
         ├─ MQ error → backoff.Wait
         └─ success → dispatch
runAll → GET /api/health/ (liveness)
      → GET /api/health/ready (readiness)
```
