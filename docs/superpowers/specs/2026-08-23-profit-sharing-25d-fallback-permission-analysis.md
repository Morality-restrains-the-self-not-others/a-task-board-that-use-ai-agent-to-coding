# 权限分析：25 天分账兜底

- **日期**: 2026-08-23
- **结论**: 无新对外 API、无新角色、无租户 IDOR 面扩大。

## 改动点

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `POST /api/internal/taskbill/profit-sharing/process-pending/` | `taskEvents` timer | System / 内部 | 写（出站微信分账） | `requireInternalSecret` | ✅ 充分 | 不暴露给浏览器 / 租户 JWT |
| `queryFallbackProfitSharings` / `markUnmarkedFallback` | 同上 | `billing_profit_sharing` + `billing_resource_order` | 读+写 | 仅内部进程 | ✅ | 扫描全库 LIMIT，无租户入参 |
| 微信 `CreateOrder` | 平台商户号 | 资金 | 出站写 | 仓内 wechatpay-go | ✅ | 幂等键 `out_profit_sharing_no` |

## 角色

不新增。租户、推荐人、超管 UI 均不调用本路径。超管待分账队列仍为只读/核单，不替代本兜底。

## 数据

内部扫描可读跨租户 `paid_at` 与分账状态。日志只打 `order_id` / `out_profit_sharing_no` / status，不打 openid、姓名、金额以外的 PII。金额已在现网 success 日志中，本增量不新增金额字段到兜底 skip 日志。
