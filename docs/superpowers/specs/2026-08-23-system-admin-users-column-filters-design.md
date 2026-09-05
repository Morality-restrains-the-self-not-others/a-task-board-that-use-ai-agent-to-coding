# 设计：系统管理用户表列过滤器

- **日期**: 2026-08-23
- **入口**: `/system-admin/users/` 用户列表表格
- **架构变更**: 无（不新增服务/聚合/消息；不更新 `docs/architecture/`）

## 背景

超管用户表仅有顶部「邮箱/ID/手机号」全局搜索，且分页（50/页）。需要在每一列表头下方提供过滤器，按列缩小结果。

## 方案（已采纳）

1. **UI**：复用账单表 `BillingUsageFiltersRow` 模式——`thead` 第二行，每列一个控件；操作列放「重置」。
2. **控件**：
   - 文本（ID / 邮箱 / 所属租户公司 / 手机号 / 推荐人）：contains，防抖 400ms + Enter/blur 立即应用。
   - 下拉：登录方式、状态、角色、分账资格。
   - 日期：注册时间、最后活跃时间（起止）。
3. **API**：扩展既有 `GET /api/system-admin/users/` query，与 `q` / `is_archived` AND 组合；重置 offset=0。
4. **本库字段**（id/email/phone/login_method/date_joined/last_login/is_active/role）：SQL `WHERE`。
5. **跨服务字段**（tenant_company / referrer / has_profit_sharing）：先本库筛出候选 ID（上限 5000），再批量 hydration 后内存过滤，再分页。
6. **安全**：仍走 `requireSuperuser`；日志只记已填过滤键名，不记邮箱/手机号原文。
7. **事件**：只读查询，不投递 MQ（书面例外）。

## 非目标

- 去掉顶部全局搜索。
- 推荐码申请 tab。
- 新增独立过滤 API。
- 导出/保存过滤预设。

## 权限

| 路径 | 角色 | 范围 |
|------|------|------|
| GET `/api/system-admin/users/?email=&role=…` | 网关已验证 + `requireSuperuser` | 跨租户用户目录 |

## Query 参数

| 参数 | 语义 |
|------|------|
| `id` | 用户 ID contains |
| `email` | 邮箱登录标识 contains |
| `phone` | 手机登录标识 contains |
| `tenant_company` | 所属租户公司名 contains |
| `login_method` | `email` \| `phone` \| `username` |
| `referrer` | 推荐人展示名 contains |
| `date_joined_from` / `date_joined_to` | 注册日期闭区间（YYYY-MM-DD） |
| `last_login_from` / `last_login_to` | 最后活跃日期闭区间 |
| `is_active` | `true` \| `false`（活跃/已禁用；归档仍由 tab 的 `is_archived` 控制） |
| `role` | `superuser` \| `staff` \| `tenant` \| `user` |
| `has_profit_sharing` | `true` \| `false` |
