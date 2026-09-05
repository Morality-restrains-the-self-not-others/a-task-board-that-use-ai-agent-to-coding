# NFR 澄清: 充值 SMS 验证状态持久化修复

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-29-recharge-sms-verification-status-loss-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-29-recharge-sms-verification-status-loss-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 数据一致性 | L3 | SMS 验证写入后，同用户下一次读取必须可见（read-your-writes），无时间窗口 |
| 性能 | L1 | 无退化 — 缓存删除为 O(1)，不增加请求延迟 |
| 可观测性 | L1 | Identity 缓存命中/未命中补 info 日志 |
| 安全性 | L0 | 不适用 — 无认证/授权变更 |
| 可用性 | L0 | 不适用 — 无新依赖或故障模式引入 |
| 可伸缩性 | L0 | 不适用 — 变更不涉及数据量或并发模型 |
| 容错机制 | L1 | Identity 缓存删除失败不应阻断主流程 |
| 合规 | L0 | 不适用 |

## 逐增量 NFR 分析

### Increment 1: Identity 缓存刷新

#### NFR 类别: 数据一致性
- **等级**: L3 - 增强（金融/充值场景，read-your-writes 为刚需）
- **量化目标**: SMS 验证成功后，同用户下一次 `GET recharge_phone_status/` 必须返回 `sms_verified: true`，无时间窗口
- **质量场景**: QS-01

### Increment 2: RechargePhoneStatusView 防御性兜底

#### NFR 类别: 数据一致性
- **等级**: L3 — 双重保障，即使 Increment 1 的缓存删除因未知原因失败，SMS 缓存检查仍生效
- **量化目标**: 即使 `taskauth:identity:user:{uid}` 缓存未刷新，`billing_recharge_sms_ok:{uid}` 缓存命中时仍返回 `sms_verified: true`
- **质量场景**: QS-02

## 质量场景

### QS-01: SMS 验证后状态立即可见
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L3 |
| 刺激源 | 前端轮询 (JS `setInterval`) |
| 刺激 | `POST recharge_verify_sms/` 返回 200 后立即发起 `GET recharge_phone_status/` |
| 制品 | `RechargePhoneStatusView.get()` → `get_user_bound_phone()` → `TaskAuthIdentityClient.get_user()` |
| 环境 | 正常（单用户操作，两次请求间隔 < 3s） |
| 响应 | 返回 `has_phone: true` + `sms_verified: true` |
| 响应度量 | Identity 缓存 `taskauth:identity:user:{uid}` 在 `upsert_phone_login_method` 成功后被删除，下一次 `get_user()` 走 HTTP 获取最新 login_methods |

### QS-02: 缓存未刷新时的兜底
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L3 |
| 刺激源 | 前端轮询 |
| 刺激 | Identity 缓存因未知原因未刷新（如 Redis 网络瞬断），但 SMS 缓存 `billing_recharge_sms_ok:{uid}` 已设置 |
| 制品 | `RechargePhoneStatusView.get()` |
| 环境 | 降级（Identity 缓存返回过期数据，`has_phone` 返回 false） |
| 响应 | 仍返回 `sms_verified: true`（通过 SMS 缓存检查） |
| 响应度量 | `is_recharge_sms_verified_for_user()` 被调用且返回 `True`，不受 `has_phone` 值影响 |

### QS-03: 缓存删除失败不阻断验证流程（容错）
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L1 |
| 刺激源 | Django cache backend（如 Redis 连接池耗尽） |
| 刺激 | `cache.delete(f'taskauth:identity:user:{uid}')` 抛出异常 |
| 制品 | `upsert_phone_login_method()` |
| 环境 | 降级（cache backend 部分故障） |
| 响应 | SMS 验证成功响应正常返回，缓存删除失败被 catch 并 warning 日志，不向上传播 |
| 响应度量 | 返回 `{"message":"验证成功","sms_verified":true}` + 日志含 `identity_cache_invalidation_failed` warning |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 数据一致性 L3 (read-your-writes) | Identity 查询接口需要感知写操作副作用 — `upsert_phone_login_method` 写入后必须通知读路径刷新 | 在 `TaskAuthIdentityClient` 或对应的 Repository 接口中增加 `invalidate_user_cache(user_id)` 方法 |
| 容错 L1 (缓存删除非阻断) | 缓存刷新是 best-effort，不应作为事务的一部分 | 缓存删除逻辑放在基础设施层（`cache.delete` 在 `try/except` 中），不进入领域层 |

## 权衡与边界

### 取舍
- 选择 **cache.delete 同步刷新** 而非缩短 TTL：删除是 O(1) 操作，比调短 TTL 更精确；代价是下一次 `get_user()` 多一次 taskAuth HTTP 调用
- 选择 **双保险**（Identity 刷新 + View 兜底）而非单一修复：增加 2 行代码换取零时间窗口保证

### 明确不做什么
- 不在本次引入领域事件（`PHONE_BOUND`）来广播缓存刷新 — 过度设计，SMS 验证是单服务内的短时会话状态
- 不做 Redis pub/sub 跨实例缓存刷新 — 当前部署为单实例，无此需求
- 不缩短 Identity 缓存 TTL — 治标不治本

### 升级触发条件
- 当 Django 部署变为多 worker 且使用 locmem cache 时 → 需要切换到 Redis 集中缓存，否则多 worker 间缓存不一致
- 当 SMS 验证引入更多下游消费方（如风控、审计）时 → 引入 `PHONE_BOUND` 领域事件

## 跳过声明
- **安全性**: 跳过。Bug 修复不涉及认证/授权变更。
- **可用性**: 跳过。无新依赖或故障模式。
- **可伸缩性**: 跳过。变更不涉及数据量或并发模型。
- **合规**: 跳过。无合规相关变更。
- **可维护性**: 跳过。改动范围极小（2 文件，~10 行代码）。
