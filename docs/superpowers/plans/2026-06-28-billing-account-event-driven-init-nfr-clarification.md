# NFR 澄清: BillingAccount 事件驱动初始化

> 输入:
> - 设计文档: `design.md` (2026-06-28 — BillingAccount 事件驱动初始化 + 读写分离)
> - 价值流文档: `docs/superpowers/plans/2026-06-28-billing-account-event-driven-init-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | GET account P95 ≤ 100ms；event consumer 异步无用户感知延迟 |
| 可伸缩性 | L1 | 单实例 SQLite，日活 < 100 租户，无专项伸缩设计 |
| 可用性 | L2 | taskBill 不可达时 Kafka 重试；懒初始化安全网兜底 |
| 安全性 | L3 | 内部密钥认证 + tenant_id 隔离 + 计费审计追踪 |
| 数据一致性 | L3 | 幂等创建 + 扣费事务原子性 + at-least-once + 幂等键 = effectively-once |
| 容错机制 | L2 | 指数退避重试 + 连接超时 30s；无断路器（单消费者，故障隔离天然） |
| 可观测性 | L2 | JSON 结构化日志 + trace_id 贯穿事件链 + taskEvents 健康端点 |
| 合规与隐私 | L2 | 计费审计日志保留；暂无 GDPR/PCI-DSS 要求 |
| 可维护性 | L2 | 种子迁移幂等 (INSERT OR IGNORE)；向后兼容已有 billing_account 行 |

## 逐增量 NFR 分析

### Increment 1: 修复懒初始化 500 错误（Thin Slice）

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: taskBill `get-or-create-account` (POST) 首次创建 P95 ≤ 200ms（含 SQLite INSERT）；幂等查询 P95 ≤ 50ms
- **影响**: 仅影响首次创建 Todo 的新租户；已有账户的租户不受影响

#### NFR 类别: 数据一致性
- **等级**: L3 - 增强
- **量化目标**: `getOrCreateBillingAccount` 幂等——同一 tenant_id 并发请求仅创建一条记录
- **实现**: SQLite 串行写 + `tenant_id UNIQUE` 约束

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: 种子迁移 `INSERT OR IGNORE` 幂等，重复执行不报错

### Increment 2: 事件驱动 BillingAccount 初始化

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: taskBill 不可达时 consumer 日志告警 + Kafka 自动重试（无消息丢失）
- **降级策略**: Increment 1 的懒初始化安全网作为兜底

#### NFR 类别: 安全性
- **等级**: L3 - 增强
- **量化目标**: consumer → taskBill 调用携带 `X-TaskBill-Internal-Secret` 头；taskBill 校验不通过返回 403
- **审计**: 计费账户创建事件写入 `billing_outbox_message` → `BILLING_TRANSACTION_CREATED` Kafka 事件

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: HTTP 超时 30s；5xx 重试（DispatchRetryable）；4xx 死信（DispatchPermanent）
- **无断路器**: 单 consumer + 单 taskBill 实例，故障隔离天然成立

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**: consumer 日志含 `[billing_init]` 前缀 + company_id + taskBill HTTP 状态码

### Increment 3: Python 侧读写分离 + 诊断改进

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: `GET /api/internal/taskbill/accounts/{tenant_id}` P95 ≤ 50ms（单条主键查询）
- **对比**: 替代原 POST get-or-create（读+写混合），减少 Todo 创建路径延迟

#### NFR 类别: 安全性
- **等级**: L3 - 增强
- **量化目标**: GET 端点同样校验 `X-TaskBill-Internal-Secret`；Python 侧使用 `HTTPClient`（trust_env=False）防止代理劫持

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **量化目标**: 账户不存在时返回 404 + 友好错误 "账户正在初始化"，不静默创建
- **边界**: 极端竞态窗口（consumer 未处理 COMPANY_CREATED）→ 503 提示用户重试

---

## 质量场景

### QS-01: 新租户首次创建 Todo 计费检查
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | Web 客户端用户点击「创建任务」 |
| 刺激 | POST /api/tenant/{id}/workspace/{id}/todos/ |
| 制品 | todo_views.create() → get_billing_account() → GET taskBill/accounts/{tenant_id} |
| 环境 | 正常负载（账户已由事件消费者预创建） |
| 响应 | GET 返回 200 + 账户余额，50ms 内完成 |
| 响应度量 | 服务端 P95 ≤ 50ms（SQLite 主键查询），由 APM trace 测量 |

### QS-02: 事件消费者幂等创建 BillingAccount
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L3 |
| 刺激源 | Kafka 消息重复投递（at-least-once） |
| 刺激 | 同一 COMPANY_CREATED 事件被消费两次 |
| 制品 | 4_init_billing_account consumer → POST taskBill/get-or-create-account |
| 环境 | 正常 + 消息重复 |
| 响应 | 第一次创建账户，第二次查询已有账户（幂等） |
| 响应度量 | billing_account 表同一 tenant_id 仅一行；taskEvents idempotency store 命中后跳过 dispatch |

### QS-03: taskBill 不可达时事件重试
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | taskBill Go 进程崩溃 |
| 刺激 | consumer 发起 POST get-or-create-account 收到 connection refused |
| 制品 | 4_init_billing_account consumer |
| 环境 | taskBill 宕机 |
| 响应 | consumer 返回 DispatchRetryable → Kafka 不提交 offset → 重试 |
| 响应度量 | taskEvents 健康端点返回 degraded；日志含 `[billing_init] taskBill unreachable` |

### QS-04: 账户未就绪时 Todo 创建的降级响应
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 用户注册后立即创建 Todo（竞态窗口） |
| 刺激 | GET taskBill/accounts/{tenant_id} 返回 404 |
| 制品 | todo_views.create() 异常处理 |
| 环境 | COMPANY_CREATED 事件尚未被 consumer 处理 |
| 响应 | 返回 503 + `{"detail": "账户正在初始化，请稍后重试"}` |
| 响应度量 | 响应码 503（非 500）；日志含 tenant_id + taskBill 404 |

### QS-05: 内部密钥校验
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 未授权服务尝试调用 taskBill 内部 API |
| 刺激 | GET/POST taskBill 内部端点携带错误/缺失的 X-TaskBill-Internal-Secret |
| 制品 | taskBill requireInternalSecret() 中间件 |
| 环境 | 正常 |
| 响应 | 403 Forbidden |
| 响应度量 | 所有内部端点统一校验；consumer 调用携带正确 secret 返回 200 |

---

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 数据一致性 L3（幂等创建） | BillingAccount 创建必须是幂等操作 | `getOrCreateBillingAccount` 使用 tenant_id UNIQUE + SELECT-then-INSERT |
| 安全性 L3（审计追踪） | 计费操作需建模为领域事件 | BillingAccountCreated 事件写入 billing_outbox → Kafka |
| 可用性 L2（降级策略） | Python 侧需区分「账户不存在」和「服务不可达」 | `get_billing_account` 抛不同异常类型（NotFound vs ServiceUnavailable） |
| 容错 L2（重试 + 死信） | 事件消费者需区分可重试和永久失败 | handler 返回 DispatchRetryable (5xx/网络) vs DispatchPermanent (4xx) |
| 性能 L2（只读查询） | 读写模型分离——Todo 创建路径只读不写 | GET endpoint 独立于 POST get-or-create；不再需要 select_for_update |
| 可观测性 L2（trace_id） | 事件链需携带 trace_id | trace_id 从 Django → Kafka envelope → consumer → taskBill HTTP header |

---

## 权衡与边界

### 取舍
- **选择最终一致性（事件驱动）而非同步创建**：BillingAccount 在 COMPANY_CREATED 事件中异步创建，接受短暂竞态窗口（用户注册后秒级延迟），换取 Todo 创建路径的简洁性和 taskBill 故障隔离
- **选择 SQLite 串行写而非分布式锁**：SQLite 单写者模型天然提供事务隔离，无需额外分布式协调

### 明确不做什么
- **不在 V1 实现断路器**：单 consumer + 单 taskBill 实例，故障隔离天然成立。当 taskBill 独立扩展为多实例或 consumer 数量增加时升级
- **不做实时余额同步**：余额以 taskBill SQLite 为准，Django 侧不缓存余额
- **不支持多币种/跨境计费**（L0 数据本地化）：仅本地部署，人民币计价
- **不做 P99 < 10ms 极致性能优化**（保持 L2）：功能完整性优先

### 升级触发条件
- 日活租户 > 1000 → 性能从 L2 升级到 L3：taskBill SQLite → PostgreSQL + 读副本；GET account 增加 Redis 缓存
- 需要 SOC2/PCI-DSS 合规 → 安全从 L3 升级到 L4：全量操作审计日志 + 静态数据加密
- taskBill 成为多消费者热点 → 容错从 L2 升级到 L3：增加断路器 + 舱壁隔离
- 跨区域部署 → 数据一致性从 L3 升级到 L4：考虑分布式事务或 CQRS + 事件溯源

---

## 跳过声明

- **可伸缩性**: L1 基础。当前日活 < 100 租户，SQLite 单实例完全满足。无水平扩展需求。
- **合规与隐私**: L2 标准。无 GDPR/HIPAA/PCI-DSS 要求。计费审计日志按现有策略保留。
