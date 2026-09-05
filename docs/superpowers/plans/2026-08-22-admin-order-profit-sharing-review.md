# Review — 管理员订单分账只读

- **日期**: 2026-08-22
- **范围**: 本增量 diff（taskBill 管理员详情、taskFE 展开、056 BIGINT）

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | staff GET 返回 receiver/amount；租户 GET 无 `profit_sharing` 键；空数组与 401/403/404 覆盖 |
| Readability | 分账 JSON 单独 `profit_sharing_admin.go`，未污染 `orderJSON` |
| Architecture | ADR-0030；管理员详情独立资源；APISIX 已有 `orders/*` |
| Security | IsPlatformStaff；无 openid；租户契约省略键 |
| Performance | `idx_profit_sharing_order` 点查 |

## 安全清单

- [x] 无密钥在代码/日志
- [x] 输入 order_id 解析失败 400
- [x] SQL 参数化
- [x] 认证+授权覆盖管理员详情
- [x] 错误不暴露 openid
- [x] 非 SSRF

## Log Audit

- 401/403 WARN `admin_order_detail_*`
- 成功 INFO `share_count`（无 PII）
- 分账查询失败 ERROR

## Intent→Event

纯查询例外，已写意图文档。

## 发现

- Critical：无（056 修复 INTEGER 无法存 snowflake，属本增量必要）
- Required：无
- Nit：管理端接收方目前为 user_id，未解析显示名（OPT）
