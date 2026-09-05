# [运行时] 项目详情「无法启动 OAuth 授权」且错误节点无 data-traceId

## 基本信息

- 案例编号：FE-20260718-OAUTH-START-TRACE
- 录入日期：2026-07-18
- 最后更新：2026-07-18
- 关联服务：taskGitOauth（:8002）、APISIX `up-gitOauth`、front_project ProjectDetail

## 失败现象

- 页面：`/tenant/{id}/projects/{proj}/` 项目详情 Git 仓库 / 子仓行
- 可见错误：`无法启动 OAuth 授权`
- 承载元素：`p.mt-1.text-xs.text-red-600`（`ProjectDetailGitReposSection`）
- **无** `data-traceId`，无法从 DOM 对齐网关 / taskGitOauth 日志

## 根因

1. **服务侧**：`task-git-oauth` 进程变为 zombie / `failed`（runAll 状态 failed，`:8002` 无监听）。浏览器经网关调用 `/api/accounts/*/oauth/start-from-gateway/` 得不到 `authorize_url`（上游不可达或 5xx），前端落入通用文案「无法启动 OAuth 授权」。
2. **前端侧**：`useProjectDetailGitRepos.startRepoOAuthConnect` 只把 `detail`/`message` 写入 `repoOAuthActionErrorByUrl`，模板 `<p class="…text-red-600">` 未绑定 `data-traceId`（违反元规则 24）。同模式亦存在于创建项目页 `useCreateProjectGitRepoRows`。

## 修复

1. runAll：`POST /api/restart` `task-git-oauth` → health 200；直连带 `X-User-Id` 可返回 `authorize_url`。
2. 前端：按 URL 保存 `repoOAuthActionErrorTraceIdByUrl`（`response.traceId` / body / `Error.traceId`）；错误 `<p>` 绑定 `:data-traceId`。
3. 任务详情 OAuth 启动抛错时把 `traceId` 挂到 `Error`，经 `showRequestError` 进入 toast。

## 验证

```bash
curl -sS http://127.0.0.1:8002/api/health/
curl -sS -H 'X-User-Id: 1' -H 'X-Gateway-Auth-Verified: 1' \
  'http://127.0.0.1:8002/api/accounts/github/oauth/start-from-gateway/?next=/&return_key=t'
cd taskFE/app && npx vitest run \
  src/composables/useProjectDetailGitRepos.oauth-traceId.test.js
bash scripts/runall-lifecycle.sh build
```

## 预防

- runAll 对 `task-git-oauth` failed 应可被 UI/告警及时发现；僵尸进程勿长期停留。
- 凡自绘 `text-red-600` 请求错误节点，同步挂 `data-traceId`（见 `.ai/01_project_constraints/24_frontend_error_data_trace_id.md`）。
