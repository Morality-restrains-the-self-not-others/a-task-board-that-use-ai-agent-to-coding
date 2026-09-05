# 设计：系统管理员用户列表增加「所属租户公司」列

- **日期**: 2026-08-23
- **页面**: `/system-admin/users/`
- **状态**: accepted（goal-mode 自动采用）
- **架构变更**: 否（复用既有内部批量查询，无新服务/协议）

## Context

超管用户列表表头「邮箱」列右侧缺少租户归属信息。用户可能加入多家公司；`is_tenant` 布尔值无法回答「属于哪家公司」。

## Decision

沿用「推荐人」列的批量回填模式：

1. **不改** `taskTenantService` 契约：继续用已有 `POST /api/internal/tenant/members/batch-get/`（`user_ids`），响应已含 `company_id` / `company_name` / `is_active`。
2. `GET /api/system-admin/users/` 在分页用户 ID 上 **best-effort** 调该接口，回填 `tenant_companies: [{id, name}]`。
3. 仅展示 **活跃** 成员关系；公司名为空时回退 `company_id`；多家公司按名称排序，前端用顿号「、」拼接；空列表显示「—」。
4. 列位置：`ID | 邮箱 | 所属租户公司 | 手机号 | …`。
5. **禁止** taskAuth 直连 `tenant_company` / `tenant_company_member`（单库单表所有权）。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 前端按用户 N+1 调租户 API | 慢；内部接口不对浏览器开放 |
| taskAuth 直连租户表 | 违反元规则 19 |
| 新建专用内部端点 | `batch-get` 已返回 `company_name` |
| 只展示 `is_tenant` | 不满足「所属租户公司」 |

## Consequences

- 租户服务不可达时列表仍 200，该列为空（与推荐人列一致）。
- 一页最多 50 用户、一次 batch-get（上限 500），无 N+1。
- `handlers_system_admin.go` 已超 500 行：列表处理抽到独立文件，避免继续膨胀。

## 契约

```json
{
  "users": [
    {
      "id": "…",
      "email": "a@example.com",
      "tenant_companies": [{"id": "c1", "name": "Acme"}]
    }
  ],
  "total": 1
}
```

字段向后兼容：仅新增键，不删既有字段。
