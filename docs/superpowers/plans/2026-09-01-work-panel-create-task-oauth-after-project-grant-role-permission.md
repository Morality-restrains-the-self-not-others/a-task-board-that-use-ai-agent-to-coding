# 角色权限分析 — 项目 L2 种下创建任务自动运行

- **Date:** 2026-09-01
- **Design:** `docs/superpowers/specs/2026-09-01-work-panel-create-task-oauth-after-project-grant-design.md`
- **ADR:** ADR-0055

## 结论

不引入新角色。项目 L2 查询与种下均按 **当前操作者 user_id** 限定；禁止跨用户、跨项目继承。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/internal/projects/git-oauth-grant/` | 服务间（taskTaskService） | Project grant 行 | read | `X-Internal-Secret` | ✅ 充分 | Query 必须带 `user_id`；不得按 tenant 扫全表 |
| POST 同路径（存量） | taskGitOauth 回调 | Project grant 行 | write | `X-Internal-Secret` | ✅ 充分 | 不改 |
| POST `/api/projects/validate-git-repos/`（FE 门禁） | 已登录租户成员 | Project + L1 | read | 网关会话 + `project_id` → `applyProjectGrantGate` | ✅ 充分 | `probe_access: false`；勿用 L1 `connected` 单独放行 |
| `applyProjectL2SeedToIdentities` | 任务创建者 / Fork 操作者 | Auto-run comment L2 | write | 仅查操作者 × 任务已关联 project_id × gitsite | ✅ 充分 | 无 grant 则不 stamp |
| pending `grant_ticket` 绑定链接 | 当前浏览器用户 | Pending comment grant | start OAuth | 现网 `grant_kind=pending` | ✅ | 无项目 L2 时仍走此路径 |
| 同事后续「提交并运行」 | 评论作者 | 另一条 comment L2 | write | 现网评论门禁 | ✅ 不改 | 不得用项目 L2 代替 |

## IDOR / 越权

- Internal GET 若缺少 secret → 403。
- `user_id` 由调用方传入：taskTaskService 必须用 `created_by` / `p.UserID`，禁止前端指定被种用户。
- FE validate 带 `project_id`：他人项目无成员资格时现网已 403/空结果，门禁保持拦截。

## 无新角色建模
