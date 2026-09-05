# 价值流 — 项目 L2 种下创建任务自动运行

- **Date:** 2026-09-01
- **Increment:** 同用户 × 同项目 × 同 gitsite：项目详情已授权后，工作面板/Fork 自动运行不再二次 OAuth。

## 当前痛点

项目详情打项目 L2（无 ticket）；创建任务门禁只认 session `grant_ticket` → 用户已「已授权」仍被拦截。

## 目标切片（最小可行）

1. **门禁**：创建/Fork auto_run 对每个所选项目仓，session ticket **或** `validate-git-repos` + `project_id` 且 `token_available`。
2. **种下**：`ensureAutoRunAtComment` 在 ticket 之后查项目 L2，写入该【自动运行】评论 JSON，发 `COMMENT_GIT_OAUTH_GRANTED` `via=project_l2_seed`。
3. **查询**：taskProjectService GET internal grant。

不在本增量：同事另评、L1 单独放行、跨项目继承。

## 步骤（操作者视角）

1. 在项目详情完成 Git OAuth → 项目 L2 行存在。
2. 打开工作面板创建任务，所选项目 `auto_run=true`。
3. 门禁显示「已绑定 Git OAuth」，可提交。
4. 后端创建【自动运行】评论，JSON 含 `oauth_gitsite` / `oauth_remote_user_id`。
5. 克隆换票认该评论 L2。

## 测试点

- T3 / T3d：项目 L2 + L1 → bound，无 bind 链接。
- T3c：仅 L1 → 仍拦截。
- 另一项目未授权仓仍拦截。
- `auto_run=false` 无 OAuth 行。
- Fork 与创建同一套。
