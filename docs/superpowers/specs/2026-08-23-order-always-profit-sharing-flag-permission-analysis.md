# 权限分析：租户下单一律打微信分账标识

- **日期**: 2026-08-23
- **设计**: `docs/superpowers/specs/2026-08-23-order-always-profit-sharing-flag-design.md`
- **结论**: 无新端点、无新角色；现有租户 pay 鉴权充分。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `wechatPrepay` 设置 SettleInfo | 租户会话用户（下单付款人） | Tenant + Order | 写（出站微信创单） | 网关已验证 + 租户订单归属 | ✅ | 分账标识不对租户回显 |
| `markOrderForProfitSharing` | 支付回调 / 内部 | Order | 写台账 | 仅支付成功路径 | ✅ | 无接收方不插行 |
| `unfreezeProfitSharing` | 支付成功内部 | Order / 微信资金 | 写出站 | 无对外 API | ✅ | 不把 Unfreeze 暴露给租户或超管按钮 |
| 租户订单 GET | 租户 | Order | 读 | ADR-0030 省略 `profit_sharing` | ✅ | 本增量不改 |

## 角色建模

不新增角色。平台员工仍只能在管理端看到佣金台账（有推荐人的行），看不到「仅微信侧打标、已解冻」的空台账（本设计故意不落空行）。

## IDOR / 越权

- Unfreeze 的 `order_id` 来自刚支付成功的内部调用，不接受客户端传入分账单号。
- `UF{order_id}` 仅服务端生成。

## 审计与日志

- 预下单已有 `profit_sharing` 布尔日志。
- 解冻须打 `order_id` / `out_order_no` / 成功或微信错误码；禁止 openid、密钥。
