# 权限分析：微信支付 Native 充值

- **日期**：2026-07-14
- **设计**：`docs/superpowers/specs/2026-07-14-wechat-pay-recharge-design.md`

## 端点权限矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有/计划检查 | 是否缺失 | 建议 |
|--------|------|----------|------|---------------|----------|------|
| POST `recharge_wechat_create` | 已登录用户 | Tenant × BillingAccount | create | 网关 token + 租户成员 + SMS 已验证（策略开启时） | ⚠️ 待实现 | 复用 `RechargePaypalCreateView` 门禁：`resolve_tenant_id` + `is_recharge_sms_verified_for_user` |
| GET `recharge_wechat_status` | 已登录用户 | Tenant × WechatRechargeOrder | read | 订单 `user_id` 须等于当前用户 | ⚠️ 待实现 | 与 PayPal status 相同：`pending.user_id == request.user.id` |
| POST `/api/billing/wechat/notify/` | 微信服务器 | 全局回调 | notify | **无 session**；验签 + 解密 | ⚠️ 待实现 | 公钥验签失败 → 401/403；禁止 CSRF session 依赖 |
| POST internal `wechat/mock-complete` | 内部调用方 | Internal | write | `X-Internal-Secret` + `mode=mock` | ⚠️ 待实现 | live 模式硬拒绝 403 |
| GET `recharge_phone_status` | 已登录用户 | User | read | 已有 | ✅ | 扩展字段 `wechat_enabled` 无额外权限 |

## 创建订单 — 详细要求

| 检查项 | 说明 |
|--------|------|
| 认证 | 网关注入 `X-Auth-User-Id` / session；未登录 → 401 |
| 租户成员 | `tenant_id` 路径参数须属于当前用户可访问租户；否则 404/403 |
| SMS 门禁 | `is_recharge_phone_verification_required()` 为 true 时，须 `is_recharge_sms_verified_for_user(user)`；否则 400 |
| 金额 | 与 PayPal 相同校验：预设档位或自定义上限 |
| 配置 | `wechat_enabled=false`（未配置且非 mock）→ 503 |

**角色**：不引入新角色。任意满足 SMS 门禁的租户成员可为自己租户充值（与 PayPal 一致）。

## 回调 — 详细要求

| 检查项 | 说明 |
|--------|------|
| 访问性 | 公网 HTTPS；无需 Cookie / Bearer |
| 验签 | `Wechatpay-Signature` 等头 + 平台公钥；失败拒绝 |
| 幂等 | 同一 `out_trade_no` 重复通知仅入账一次 |
| 信息泄露 | 验签失败响应不含密钥/完整 body |

## mock-complete — 详细要求

| 检查项 | 说明 |
|--------|------|
| Secret | `X-Internal-Secret`（或项目统一 internal header）与配置常量时序安全比较 |
| 模式 | 仅 `conf/billing/wechatPay/` → `mode=mock` 时可用 |
| 环境 | 生产 live 配置下路由应 403 或不存在 |
| 归属 | 可选：校验 `out_trade_no` 存在于 pending 缓存 |

## IDOR / 越权风险

| 风险 | 缓解 |
|------|------|
| 用户 A 查询用户 B 的订单状态 | status API 校验 pending 内 `user_id` |
| 用户 A 为租户 B 充值（非成员） | `resolve_tenant_id` + 成员关系 |
| 伪造微信回调 | 验签 + 商户号/appid 匹配 |
| 外部调用 mock-complete | internal secret + mode=mock 双门禁 |
| 回调重放 | 微信 timestamp 窗口 + 幂等 txn |

## 结论

无架构级阻断项。实现时必须：

1. 创建/查状态与 PayPal 同级鉴权，不可仅依赖 `out_trade_no` 猜测。
2. 回调 **公开但验签**，不得因「内部接口」而跳过签名。
3. `mock-complete` **双重门禁**（secret + mode），禁止 live 环境暴露。
