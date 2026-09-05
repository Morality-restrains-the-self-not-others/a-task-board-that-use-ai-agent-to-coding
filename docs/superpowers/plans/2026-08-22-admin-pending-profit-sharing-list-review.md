# 代码审查 — 管理员待分账订单列表

- **日期**: 2026-08-22
- **对照计划**: `docs/superpowers/plans/2026-08-22-admin-pending-profit-sharing-list-plan.md`

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | staff 默认 open 不含 finished；非法 status 400；401/403；无 openid。Tab query 与退款兼容。 |
| Readability | handler / queue 查询分离；面板与退款列表同构。 |
| Architecture | 扩展 taskBill + APISIX 前缀；无新服务；沿用 ADR-0030。 |
| Security | IsPlatformStaff；响应剥离 openid/微信单号；租户无入口。 |
| Performance | 使用既有 status+settle 索引；limit≤50。 |

## 安全审计

- [x] 无密钥入代码/日志
- [x] 边界鉴权 401/403
- [x] SQL 参数化 IN + LIMIT
- [x] 错误带 trace_id；前端 data-traceId
- [x] CORS 沿用网关既有 origin
- [x] 无 SSRF（无出站 URL）

## Intent→Event

纯查询例外已写入意图文档。无新消费者。

## Log Audit

- 未认证 WARN `admin_profit_sharing_list_unauthorized`
- 非 staff WARN `admin_profit_sharing_list_forbidden`
- 查询失败 ERROR
- 成功 INFO：status_filter / total / limit / offset / returned；无 openid

## CRG

`code-review-graph update --brief` 已在流水线开头执行。本增量新增独立 handler，不改 `orderJSON`。

## 发现

| 级别 | 项 | 处理 |
|------|----|------|
| Nit | 管理端立即分账/重试 | 记 OPT-20260822-056 |
| Nit | 公网硬刷新验收 | 记同一 OPT 的部署步 |

无 Critical / Required 阻断。
