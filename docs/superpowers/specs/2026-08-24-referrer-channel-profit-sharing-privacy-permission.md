# 权限分析：推荐人渠道聚合分账

## 角色

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/billing/profit-sharing/referrer-orders/` | 已登录推荐人 | Resource（本人分账） | read | `X-User-Id` = `referrer_user_id` | ✅ | 聚合后仍等值过滤；禁止 query 覆盖 referrer |
| POST `.../share-channel/` | 已登录推荐人 | Resource | write（资金 L3） | 同上 + 仅处理本人该渠道 shareable 行 | ✅ | 渠道无本人行 → 404（不暴露他人渠道存在性） |
| POST `.../{id}/share/` | 已登录推荐人 | Resource | write | 他人 404 | ✅ 充分 | 保留；列表不再返回 id |
| 匿名 | — | — | — | 无 X-User-Id → 401 | ✅ | — |
| 平台员工 / 租户管理员 | 不走本 API | System | — | 用 system-admin 队列 | ✅ | 不把订单号经推荐人 API 放出 |

## 规则

- 禁止 query/body 指定他人 `referrer_user_id`。
- 渠道聚合 JOIN `billing_referral_edge` 时必须 `e.referrer_user_id = ps.referrer_user_id`，避免错渠道。
- 不向推荐人返回受推荐人标识与订单号。
