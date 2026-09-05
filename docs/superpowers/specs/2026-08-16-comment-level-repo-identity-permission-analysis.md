# 权限分析：评论级仓库身份

> 设计：`docs/superpowers/specs/2026-08-16-comment-level-repo-identity-design.md`

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST 评论 `repo_identities` | 工作区可写成员 | Workspace + Task + Comment | write | 既有 comment create（workspace + task） | ✅ | 身份必须属于当前用户/租户可用的 git_identities；github_user_id 必须是该用户已连接账号 |
| GET 评论列表回读 identities | 工作区成员 | Comment | read | 既有 list | ✅ | 身份 ID 非密钥；不回传 token |
| snapshot?comment_id= | 内部服务 | Task + Comment | read | internal | ✅ | 须校验 comment 属于该 task |
| github-credential-status（只读连接列表） | 当前用户 | User | read | 既有 | ✅ | composer 仍用；不作为运行真源写入 |
| github-credential-approve | 工作区成员 | Task | write | 既有 | ⚠️ UI 停用 | 保留 API；前端运行路径不再调用 |
| 关联项目只读 | 工作区成员 | Task | read | 既有 | ✅ | 编辑关联项目仍走原 PATCH 任务 |

租户 page/region：**不触发**。无新租户控制台页面。

## 角色与权限建模

无新角色。`RepoIdentitySelection` 写入者 = 评论 `created_by_id`。不得替其他用户选用其未授权的 GitHub 账号。

## 安全审查结论

- [x] IDOR：snapshot 的 `comment_id` 必须属于 path 上的 task_id
- [x] 权限提升：客户端不可伪造他人 github_user_id 换票（Cloud 用当前用户 credential-summary 校验）
- [x] 跨租户：comment create 已带 tenant_id
- [x] 403 vs 404：沿用既有评论 API
- [x] 密钥：JSON 只存 ID，不存 OAuth token
- [x] XSS：仓库 URL 仍文本展示，不 v-html

**评级：绿灯**（approve API 停用为 advisory，不删 endpoint）

## 权限测试清单

| 场景 | 角色 | 操作 | 预期 |
|------|------|------|------|
| 工作区成员提交并运行+身份 | member | POST comment | 201，JSON 落库 |
| 未认证 | 匿名 | POST | 401 |
| 跨任务 comment_id snapshot | internal | GET snapshot | 空回退或 404，不得串任务身份 |
| 选用未连接的 github_user_id | member | POST | 400 |
