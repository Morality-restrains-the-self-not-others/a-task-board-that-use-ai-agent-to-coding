# 计费用量页按工作空间筛选 — 设计

- 日期：2026-07-16
- 状态：已采纳（goal-mode 自动决策）
- 范围：`GET /api/tenant/{id}/billing/usages/` + SPA `/tenant/:tenant/billing/usage/`

## 问题

用量页表格已展示工作空间列，筛选区仅有时间 / 计费单元类型 / 项目；服务端 `handleUsagesList` 不接受 `workspace_id`，无法按工作空间缩小分页结果。

## 方案（已选）

与现有「项目名称」筛选同一模式：

| 层 | 改动 |
|----|------|
| taskBill | `handleUsagesList` 增加 `workspace_id` WHERE（对齐流水 filtered） |
| OpenAPI | usages GET 增加 `workspace_id` query |
| SPA | Filters 增加可搜索工作空间下拉；`buildQueryParams` 传 `workspace_id` |
| 下拉数据 | `GET /api/tenant/{id}/workspaces/` 全量 + 本地过滤（**不用**流水页未实现的 `search_workspaces`） |

## 非目标

- 不改收费写入路径
- 不新增 Django 路由
- 不做用户/任务筛选（可后续 parity）
- 无架构组件变更（无 ArchiMate 交付）

## 验收

1. 选择工作空间后，usages 请求带 `workspace_id`，列表与 total 仅含该空间
2. 重置后恢复全量
3. Go 单测覆盖过滤；Playwright mock 断言 query 含 `workspace_id`
