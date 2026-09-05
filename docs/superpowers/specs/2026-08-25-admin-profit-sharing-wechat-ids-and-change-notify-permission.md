# 角色权限分析 — 超管微信分账单号 / 手工分账 / 动账通知

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-admin-profit-sharing-wechat-ids-and-change-notify-design.md`
- **判定**: 绿灯 ✅（资金写路径仅平台员工 + 强制审计缘由 + 微信验签）

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/profit-sharing/` 回传微信单号 | 平台员工 | System | read | 网关 verified + `IsPlatformStaff` | ✅ | 推荐人 API 仍禁止这两列 |
| POST `/api/system-admin/profit-sharing/{id}/share/` | 平台员工 | System | write（资金出站） | 须新增 | — | staff + reason 8–500 + Idempotency-Key；非 staff 403 |
| FE 分账按钮 | 平台员工 | System | write | system-admin 壳 | ✅ | 壳外无入口 |
| POST `/api/billing/profitsharing/change-notify/` | 微信支付 | System | write | 无用户会话 | ⚠️ | `core/notify` 验签解密；假通知 400 FAIL |
| 租户/推荐人路径 | 推荐人 | Tenant | — | 现网 share 仅本人 | ✅ | 不开放微信单号 |

## 角色建模

不新增角色。复用 `authz.IsPlatformStaff`。模拟登录时日志必须带 impersonation 四字段。

## 安全审查

- [x] **IDOR**: share 以路径 id 定位台账行；staff 可跨租户是需求
- [x] **权限提升**: 非 staff 403；未登录 401
- [x] **资金**: 必须 reason；幂等键 UNIQUE；processing/finished 不重复 CreateOrder
- [x] **回调伪造**: 必须 ParseNotifyRequest；mock 仅 `mode=mock`
- [x] **敏感**: 日志不写 openid / API v3 key / 完整 reason 可写（审计需要）但禁止 token

## 权限测试

| 场景 | 角色 | 预期 |
|------|------|------|
| staff GET 含微信单号 | 平台员工 | 200，有字段 |
| 非 staff POST share | 普通用户 | 403 |
| staff 无 reason | 平台员工 | 400 |
| 微信签名无效 | 外部 | 400 FAIL |
| 重复 notify.id | 微信 | 200 SUCCESS 空操作 |
