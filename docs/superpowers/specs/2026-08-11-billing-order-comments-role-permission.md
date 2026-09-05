# 订单评论 — 角色权限分析

- **Date:** 2026-08-11
- **Iteration:** billing-order-comments-v76

## 角色矩阵

| 角色 | GET comments | POST comments |
|------|--------------|---------------|
| 匿名 | 401 | 401 |
| 租户成员 `billing:view` | ✅ 本租户订单 | ❌ 403 |
| 租户管理员 `billing:manage` | ✅ | ✅ author_side=tenant |
| 平台员工 IsPlatformStaff | ✅（admin API） | ✅ author_side=system_admin |
| 非 staff 登录用户调 admin API | 403 | 403 |

## 威胁模型（简）

- 跨租户读写：handler 校验 `order.tenant_id == path/query tenant_id`
- 内容注入：JSON 存储 + FE 文本渲染（Vue text interpolation，禁 v-html）
- 日志：禁止记录完整 content；事件 payload 不含正文

## 决策

沿用既有粗码 `billing:view` / `billing:manage` 与 `IsPlatformStaff`，不新增 permission code。
