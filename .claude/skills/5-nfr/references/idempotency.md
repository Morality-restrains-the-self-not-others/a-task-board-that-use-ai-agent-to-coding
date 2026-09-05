# 副作用路径幂等性审视（NFR 硬门禁附录）

> 被 `/5-nfr` 强制引用。元规则 SSOT：`.ai/01_project_constraints/53_nfr_idempotency.md`。

## 适用范围（「副作用路径」定义）

对当前增量涉及的每一条**会产生可观察副作用**的路径做审视，包括但不限于：

| 路径类型 | 示例 |
|----------|------|
| HTTP 写接口 | `POST/PUT/PATCH/DELETE`、RPC 变更命令 |
| 领域事件消费 | Kafka intent / `taskEvents` handler、Outbox 投递 |
| Webhook / 回调 | 支付通知、云厂商回调、GitLab hook |
| 定时一次性 API | `taskEvents` timer worker 调用的 owner internal API |
| 出站副作用 | 创建云资源、扣费、发短信/邮件、调第三方写 API |

**只读路径**（纯 GET/查询、无副作用 fan-out）可标 L0，仍须写入表并写理由。

**业务重复边界**：同一用户意图被重复触达时，系统应视为「同一笔业务」的范围（例如「停这一台机器」≠「停该租户任意一台机器」）。

## 决策树（强制）

```
对增量中每条路径：
│
├─ 是否有副作用（写库 / 发事件 / 调外部写 API / 改资源）？
│   │
│   ├─ 否 → 【只读分支】标幂等性 L0；写理由（纯查询）。禁止把「以后再说」当理由。
│   │
│   └─ 是 → 【副作用分支】必须回答：
│             1. 重复触发源有哪些？（双击、客户端重试、HTTP 超时重放、
│                Kafka at-least-once、Webhook 重投、timer 重叠）
│             2. 业务重复边界是什么？（哪一组身份等于「同一笔」）
│             3. 幂等键是什么？是否与该边界同粒度？（见适配性清单）
│             4. 重放响应是什么？（返回首次结果 / 成功空操作 / 409 冲突）
│             5. 键如何持久化？（DB 唯一约束 / 幂等表 / Redis；禁止仅进程内 MemoryStore 作为资金/资源路径的唯一手段）
│             6. 并发双请求如何处理？（唯一约束冲突 → 读已有结果，禁止双写）
```

## 有副作用：何时仍可标 L0 / L1

| 等级 | 允许条件（须写入文档） |
|------|------------------------|
| L0 | 确认无副作用；或明确「不安全重试」且 UI 强制二次确认 + 无自动重试/无 Kafka 消费 |
| L1 | 自然幂等（按资源 ID 的 PUT/DELETE；`activate()`/`mark_as` 状态转移）；无额外幂等键 |
| L2 | 客户端 `Idempotency-Key` 或业务唯一键 + DB 唯一约束；Kafka at-least-once + 幂等消费 |
| L3 | 资金/配额/云资源：键 + 唯一约束 + 存储首次响应 + 并发锁；支付/KYC 路径默认 ≥ L3 |
| L4 | 端到端 exactly-once（通常不值得；默认用 L3 的 at-least-once + 幂等） |

禁止：「暂时不管」「消费者自己会去重」作为跳过理由。

## 幂等键适配性清单

逐项勾选；任一「否」则判定为**不合适幂等键**，须在 NFR 文档记录并给出纠正动作：

| # | 问题 | 期望 |
|---|------|------|
| 1 | 与业务重复边界同粒度？ | 「同一笔意图」重复命中；不同意图不得共享键 |
| 2 | 稳定且客户端可在重试时复用？ | 禁止每次重试新生成 UUID |
| 3 | 不是过粗的租户/用户身份？ | **禁止**默认用 `company_id` / `tenant_id` / `user_id` 作事件消费键 |
| 4 | 载荷若含粗字段，是否被专用键覆盖？ | 通用 `IdempotencyKeyFromEnvelope` 字段序不得让粗字段抢在业务 ID 前 |
| 5 | 写路径是状态转移而非增量累加？ | `mark_as(status)` / `activate()`，禁止裸 `increment()` |
| 6 | 资金/资源路径有 DB 唯一约束兜底？ | 进程内 `MemoryStore` 不足 |

### 典型判定（本仓库已踩过的坑）

| 候选键 | 通常判定 | 说明 |
|--------|---------|------|
| `event_id` / `stop_request_id` / `purchase_id` | 合适 | 与单次用户意图对齐 |
| `task_id`（任务生命周期事件） | 常合适 | `TASK_CREATED` 应按 `task_id`，不能按 `user_id` |
| `task_id` + `instance_id`（停机） | 合适 | 同租户另一台机器不得被去重吞掉 |
| 仅 `user_id` | **不合适** | 同用户后续独立操作被 `Seen` 静默跳过（见失败经验 64） |
| 仅 `company_id` / `tenant_id` | **不合适** | 同租户后续独立操作被吞（见失败经验 16） |
| 每次请求新 UUID 且客户端不回传 | **不合适** | 重试变成新业务，防不住双击/超时重放 |
| 业务自然键如 `(tenant_id, region)` 唯一购区 | 合适（L2+） | 配合唯一约束；重复购买返回已购结果 |

## NFR 文档必写小节

每个增量的 NFR 澄清文档须含：

```markdown
## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 幂等键 | 判定 | 等级 | 重放语义 / 动作 |
|------|--------|------------|--------|------|------|-----------------|
| POST /api/.../orders | 扣费+建单 | 双击、超时重试 | Idempotency-Key 或 order_id | 合适 | L3 | 返回首次 201 体；唯一约束冲突则读已有单 |
| Kafka TASK_CREATED 消费 | SSE fanout | at-least-once | event_id 或 task_id | 合适 | L2 | 禁止用 user_id；Seen 命中仍须可观测日志 |
| GET /api/.../members | 无 | — | — | 只读 | L0 | 无副作用 |
```

跳过整节 → Hard Gate 失败。

## 前端按钮（元规则 52 / ADR-0020）

用户双击是副作用路径的一等重复触发源。NFR 表写「双击」时必须同时写清：

- 前端如何锁点击（`createClickGuard` 或等价同步门闩，禁止只靠下一帧 `disabled`）
- `Idempotency-Key` 在**哪一次被接受的点击**生成，超时重试如何回传同一键

细则：`.ai/01_project_constraints/57_frontend_button_anti_replay.md`。

## 与领域模型的衔接（喂给 /6-ddd）

| 审视结论 | DDD 动作 |
|----------|---------|
| 副作用路径缺键或键过粗 | 命令/事件契约补与业务重复边界同粒度的幂等键；仓储按该键去重 |
| 操作为增量累加 | 改为状态转移方法；禁止 `increment()` 作为领域命令 |
| 资金/资源 L3 | 聚合内唯一约束（业务键）；Outbox/消费者按 `event_id` 幂等 |
| 只读 L0 | 无需幂等键；仍禁止在读路径夹带隐式写 |

## 消费实现落地（元规则 49 / ADR-0015）

NFR 文档写清的键，必须在 `taskEvents` 消费侧落地：

| 要求 | 说明 |
|------|------|
| 入口 | `eventbin.RunIntent(..., keyFn)`，禁止 nil keyFn |
| 调度 | `domain.IdempotentDispatchService`（`Seen` → skip + warn；成功才 `Mark`） |
| 键函数 | 默认 `consumer.IdempotencyKeyFromEnvelope`；载荷含粗字段时写专用函数 |
| 测试 | 同键重放只执行一次；不同 `task_id`/`event_id` 不得因同 user/tenant 塌缩 |
| 禁止 | 业务 HTTP 服务自建 `kafka.NewReader` consume loop |

细则：`.ai/01_project_constraints/54_event_consumer_idempotency.md`；门禁 `check_event_consumer_idempotency.py`。

## 质量场景最低要求

凡增量含 ≥1 条副作用路径且幂等性 ≥ L2：至少写一个 QS，刺激为「同一幂等键重复提交 / 重复投递」，响应度量为「副作用只发生一次，第二次返回首次结果或空操作成功」。
