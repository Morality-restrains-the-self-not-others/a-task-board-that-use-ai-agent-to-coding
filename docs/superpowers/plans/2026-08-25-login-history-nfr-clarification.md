# NFR 澄清：登录历史

- **日期**: 2026-08-25
- **默认档**: L2；认证写路径升 L3 幂等审视（每次成功登录是独立事实，非重复边界）

## 类别

| 类别 | 档 | 说明 |
|------|----|------|
| 可用性 | L2 | 历史插入 fail-open，登录仍成功 |
| 性能 | L2 | 列表 `user_id + logged_in_at` 索引；limit≤100 |
| 安全 | L3 | 认证数据；IDOR 禁止；不存 identifier |
| 可观测性 | L2 | `login_history_recorded` / `_insert_failed` + trace_id |
| 可伸缩性 | 见表 | 按月分区 |

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 可伸缩性 | 动作/理由 |
|------|---------|----------|----------|-----------|
| `POST /api/auth/` / `.../login/` | 无路径 ID | — | L1 写按 user_id | 写入带 user_id；表按时间分区 |
| `POST /api/auth/admin-login/` | 无 | — | L1 | 同上，entry=admin |
| `GET /api/auth/login-history/` | 会话 user_id | 是（用户） | L1 | `WHERE user_id=? ORDER BY logged_in_at DESC` |
| `/profile/login-history/` 等 | cookie userId 仅导航 | 否（非分片） | L0 UI | 数据分片在 API 的 user_id |
| `GET /api/system-admin/users/{id}/login-history/` | id=user | 是 | L1 | 点查+列表按 user_id |
| `/system-admin/users/:id/login-history/` | user id | 是 | L1 | 与 API 同键 |
| `USER_LOGGED_IN` Kafka key | user_id | 是 | L1 | 与业务重复边界同粒度 |

## 幂等性强制审视

| 路径 | 副作用 | 档 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|----|------------|--------------|--------|----------|
| 成功登录写历史 | 有 | L3 | 双击/重试/Kafka 重放 | **每一次成功认证** 都是新事实 | 无自然幂等键；每次 INSERT 新 Snowflake id | 允许同一用户连续多行；不把 user_id 当幂等键 |
| GET 列表 | 无 | L0 | — | — | — | 只读 |
| USER_LOGGED_IN 发布 | 有（事件） | L2 | 登录重试 | 同一次 HTTP 成功一次 | 随登录发生；不 Mark 消费 | 本迭代无新消费者；存量消费者须容忍增补字段 |

## 领域模型影响

新实体 `LoginHistoryRecord`，聚合根按「一次成功认证」。仓储按 `user_id` 查询。不引入新 MQ topic。
