# 权限分析：工作面板按可访问人或小组过滤

- 日期：2026-07-15
- 设计文档：`docs/superpowers/specs/2026-07-15-work-panel-access-filter-design.md`
- 结论：**无新增端点**；复用既有接口的既有鉴权；前端过滤不扩大数据可见范围。

## 1. 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET workspace-permissions | 已登录且可访问工作空间的用户 | Workspace | read | 网关 token + Go workspace 归属/access | ✅ 充分 | 不变 |
| GET workspace-collaborators | 同上 | Workspace | read | 同上 | ✅ 充分 | 不变 |
| GET accounts/groups/{id}/members/ | 租户内已登录成员 | Tenant/Group | read | IsAuthenticated + 同公司 queryset | ✅ 充分 | 仅请求本工作空间 access 内的 group_id，前端不枚举全租户组 |
| 客户端 filterTodosByAccess | 当前页用户 | UI | read（视图裁剪） | 任务列表本身已按工作空间拉取 | ✅ | 过滤只能缩小，不能扩大 |

## 2. IDOR / 越权风险

| 风险 | 评估 | 缓解 |
|------|------|------|
| 用他人 workspace_id 拉 permissions | 既有 Go/网关校验 | 无新面 |
| 用非本空间 group_id 拉 members | ViewSet 限同公司；组员名单属租户级信息 | UI 仅展示 access 内小组；不新增跨租户 |
| 伪造 company_member_id 过滤 | 纯客户端；不影响服务端授权 | 无 |

## 3. 角色建模

本迭代**不引入**新角色或权限粒度。

## 4. 对设计的补充要求

1. 小组下拉选项 **必须** 来自 `workspace-permissions` 的 group 行，禁止用「公司全部小组」列表。
2. 人选项 **必须** 来自 permissions 的 user 行（有 `company_member_id`），与 collaborators 交集校验可选。
3. 日志禁止输出完整成员邮箱列表。
