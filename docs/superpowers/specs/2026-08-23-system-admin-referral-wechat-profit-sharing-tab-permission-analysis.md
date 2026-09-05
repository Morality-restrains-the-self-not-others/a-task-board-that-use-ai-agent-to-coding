# 角色权限分析 — 推荐绩效抽屉微信分账 Tab

- **日期**: 2026-08-23
- **设计**: `docs/superpowers/specs/2026-08-23-system-admin-referral-wechat-profit-sharing-tab-design.md`
- **判定**: 绿灯 ✅

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/profit-sharing/?referrer_user_id=` | 平台员工 | System | read | 网关 verified + `IsPlatformStaff` | ✅ 复用列表门禁 | 非 staff 403，与全局队列一致 |
| POST `/api/system-admin/profit-sharing/refresh-wechat/` | 平台员工 | System | read（出站微信只读） | 同上 | ⚠️ 须校验 ids 属于该推荐人图 | 冒用他人 id → 400，不调微信 |
| FE 抽屉 Tab | 平台员工 | System | read | system-admin 壳 | ✅ | 壳外无入口 |
| 租户任意路径 | 租户成员 | Tenant | — | 无 tenant 前缀 | ✅ | 不注册 |
| 响应字段 | 平台员工 | System | read | ADR-0030 | ✅ | 禁止 openid / 微信分账单号 / account |

## 角色建模

不新增角色。复用 `authz.IsPlatformStaff`。

## 安全审查

- [x] **IDOR**: refresh 的 `ids` 必须落在 `referrer_user_id` JOIN 结果内
- [x] **权限提升**: 无写本地 status / 无 CreateOrder
- [x] **跨租户**: 超管跨租户只读是需求；租户不可达
- [x] **403 vs 404**: 未登录 401；非 staff 403
- [x] **user_id 注入**: `referrer_user_id` 仅过滤图，不把任意用户当 staff
- [x] **敏感操作**: 无资金写；日志不写 openid / receivers.account

## 权限测试

| 场景 | 角色 | 操作 | 预期 |
|------|------|------|------|
| staff GET 带 referrer | 平台员工 | GET | 200 |
| 非 staff GET | 普通用户 | GET | 403 |
| 未登录 | — | GET | 401 |
| staff refresh 他人 id | 平台员工 | POST | 400，不调微信 |
| 非 staff refresh | 普通用户 | POST | 403 |

## 风险

低。资金写路径仍走 timer；本增量只读 + 出站 QueryOrder。
