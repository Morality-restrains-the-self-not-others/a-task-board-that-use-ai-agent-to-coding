# 功能意图：超管用户列表按列过滤

## 意图

系统管理员在用户列表按某一列（或多列 AND）缩小结果，分页 total 与列表一致。

## 角色

- 系统超管（`requireSuperuser`）

## 行为

1. `GET /api/system-admin/users/` 接受列过滤 query（见设计文档），与既有 `q`、`is_archived` AND。
2. 本库字段在 SQL 过滤；`tenant_company` / `referrer` / `has_profit_sharing` 在候选 ID hydration 后过滤。
3. 200：`{ users, total }`；无命中空数组 total=0。
4. 非超管：403。
5. 日志记录 `filter_keys`，不记录过滤原文。

## 非目标

- 改变创建/编辑/归档语义。
- 推荐码申请 tab。

## 业务意图 → 事件对照

**无对应新事件（书面例外）**：只读列表过滤，不改变用户状态。
