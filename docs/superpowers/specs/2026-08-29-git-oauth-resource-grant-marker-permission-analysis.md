# 角色权限分析：Git OAuth 资源使用标记

- **日期**: 2026-08-29
- **设计**: `2026-08-29-git-oauth-resource-grant-marker-design.md`
- **结论**: 无新角色。L2 标记仅属于**当前登录用户**；禁止代他人打标或换票。

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `*-start-from-gateway/` + grant_kind/grant_id | 已登录用户 | User | write（发起 OAuth） | gateway forward-auth `X-User-Id` | ✅ | state 内固化 uid；回调不得改 uid |
| OAuth callback MarkGrant project | 回调用户 = state.uid | Project × User | write | 项目可读（详情/云端开发） | ⚠️ | MarkGrant 须校验项目对 uid 可读；禁止 body 指定他人 user_id |
| OAuth callback MarkGrant comment | 回调用户 = state.uid | Comment × User | write | 评论作者或可发评成员 | ⚠️ | 仅允许给 **自己将作为作者** 的评论打标；禁止给同事评论打标 |
| Internal `ConsumeGrantTicket` | owner 服务（Project/Task） | User | write | 服务间 internal token | ⚠️ | ticket 绑定 uid；消费时 uid 必须等于创建任务/发评的 `X-User-Id` |
| Internal MarkGrant (project/comment) | gitOauth → owner | Resource × User | write | internal | ⚠️ | grant 的 `task2app_user_id` 必须 = state.uid；忽略调用方伪造的 user |
| GET validate-git-repos / token_status | 项目成员 | Project | read | workspace/project 访问 | ✅ | 无 L2 → 需要授权（不是别人的 grant） |
| POST access-for-user（换票） | 内部服务代 **指定 user** | User | read secret | 已有 internal | ⚠️ | **先查 L2**；L2.user 必须等于换票 user；评论路径必须等于评论 `created_by_id` |
| 创建任务 auto_run 门禁 | 创建操作者 | Task | write | 工作空间写 | ✅ | 门禁看**操作者**会话 ticket，不看 Owner |
| 排队 `startQueuedMembership` | 系统 timer | Task | write | 内部 | ⚠️ | UserID = 自动运行评论 `created_by_id`，禁止 Owner 冒充 |

## 建模

不新增 RBAC 角色。L2 是「本人使用许可」，不是租户管理员授权同事用票。

## 安全审查

- [x] **IDOR**: grant_id 必须属于当前 uid；评论 MarkGrant 校验 comment.created_by_id == uid（pending ticket 在评论创建后由同一 uid 消费）
- [x] **权限提升**: 禁止 query/body 传入他人 `user_id` 打标
- [x] **跨租户**: 项目 grant 带 `company_id`（从 `project_entries` 抄），查询必须项目归属一致
- [x] **403 vs 404**: 无项目访问权 → 与现网项目详情一致（无权限 403）
- [x] **user_id 注入**: access-for-user 仍仅 internal；L2 闸门防止「有票就能用别人的仓场景」
- [x] **敏感操作**: 不记录 refresh/access token；日志只打 grant 指纹与 remote_user_id

## 权限测试清单

| 场景 | 角色 | 操作 | 预期 |
|------|------|------|------|
| 用户 A 给自己的项目打标 | 项目成员 A | OAuth 回流 | 200，L2 行 user=A |
| 用户 A 无法消费 B 的 ticket | 用户 A | ConsumeGrantTicket(B) | 403/404 |
| 用户 B 换票跑 A 的自动运行评论 | 内部 clone | access-for-user user=B | 拒绝（评论作者是 A） |
| 无项目访问权打标 | 外人 | MarkGrant project | 403 |
| 排队开火 | timer | startQueuedMembership | UserID=评论作者 |

## 风险评级

- **高**: 换票不查 L2 → 本设计要修的洞
- **中**: 排队 Owner 覆盖 → 必须改 `UserID`
- **低**: 同事各自 OAuth 多次点击（IdP 秒过）
