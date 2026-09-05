# NFR 澄清: Company Switcher & Member Join Event

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-28-company-switcher-join-event-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-28-company-switcher-join-event-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | 公司切换为客户端路由跳转，无额外服务端开销 |
| 可用性 | L1 | 无专项保证，join 事件发送失败不影响加入流程 |
| 安全性 | L1 | 复用现有 user/me companies 数据，无新增权限面 |
| 数据一致性 | L2 | MEMBER_JOINED 事件 at-least-once，join 事务 + send_event 在同一个请求内 |
| 可观测性 | L2 | 事件发送成功/失败均记录日志，带 trace_id |
| 可维护性 | L1 | 前端组件拆分清晰，后端改动仅一行 send_event |

## 逐增量 NFR 分析

### Increment 1: Company Switcher in Navbar (P0)

**跳过 NFR 类别**: 性能、可用性、安全性、可伸缩性、合规 — 纯前端 UI 组件，使用现有 API 数据，无新后端调用路径。

**唯一相关**: 可维护性 L1 — Navbar.ui.vue 新增一个 `<select>` 下拉组件，逻辑隔离在 Navbar.logic.vue 中。

### Increment 2: Post-Join Redirect (P2)

**跳过所有 NFR 类别**: 单行路由变更 (`/people/manage/` → `/work-panel/`)，无新数据流或外部依赖。

### Increment 3: MEMBER_JOINED Kafka Event (P1)

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **量化目标**: 事件在 join 事务提交后同步发送，发送失败仅记日志不阻塞
- **一致性梯度**: 最终一致 — 消费者异步处理，无分布式事务

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**: 每次 send_event 调用附带 trace_id（复用请求级 trace），日志含 event_type + member_id

## 质量场景

### QS-01: MEMBER_JOINED 事件成功发送
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 刺激源 | 被邀请用户 |
| 刺激 | POST /api/tenant/{id}/accounts/members/join/ 成功 |
| 制品 | member_views.py join 方法 |
| 环境 | 正常负载，Kafka broker 可达 |
| 响应 | send_event('MEMBER_JOINED', {...}) 被调用，消息进入 Kafka topic |
| 响应度量 | 日志含 `send_event event_type: MEMBER_JOINED` 且 Kafka producer 无报错 |

### QS-02: MEMBER_JOINED 事件发送失败不阻断加入
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L1 |
| 刺激源 | Kafka broker 不可达 |
| 刺激 | POST join/ 请求 |
| 制品 | member_views.py join 方法 |
| 环境 | Kafka 故障 |
| 响应 | join 仍返回 201，CompanyMember + WorkspaceAccess 已持久化，send_event 异常被捕获 |
| 响应度量 | 日志含 WARNING 级别错误信息，HTTP 状态码仍为 201 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 最终一致性 L2 | CompanyMember 聚合与事件通知解耦 | 仓储保存后由应用层发送事件（非领域层） |
| 可观测性 L2 | send_event 需在应用服务中调用 | 保持现有 `core.kafka.send_event` 模式 |

## 权衡与边界

### 取舍
- 选择最终一致性（L2）：join 事件异步消费，不引入分布式事务或 Outbox 模式
- 事件发送失败不重试：join 操作本身不受影响，下游可通过对账补偿

### 明确不做什么
- 不在 V1 实现 MEMBER_JOINED 的 Go 消费者（仅发布事件，消费端后续迭代）
- 不做 Outbox Pattern 或事务发件箱
- 不做事件回放或死信队列
- 不新增 Kafka topic（复用现有 `member-joined` 或通过 event_type 路由）

### 升级触发条件
- 当日活超过 1000 或出现事件丢失投诉时：升级为 Outbox Pattern + 至少一次投递保证
- 当需要实时通知（邮件/短信）新成员加入时：添加 Go 消费者

## 跳过声明
- **性能**: 跳过。公司切换为客户端路由跳转无服务端开销；join 事件发送 ~1ms 延迟可忽略。
- **可伸缩性**: 跳过。加入操作频率极低（日均 < 100 次）。
- **安全性**: 跳过。复用现有认证授权体系，无新增攻击面。
- **合规**: 跳过。无新增数据收集或跨境传输。
