# [运行时] Git 网站授权「无法与 GitHub 交换令牌」缺少 data-traceId

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

- 页面：`https://www.daydaymoney.com/profile/git-site-oauth/`（或 `/user/{id}/profile/git-site-oauth/`）
- 可见错误：`授权失败：无法与 GitHub 交换令牌`
- 承载元素：`p.text-sm.text-danger`
- **无** `data-traceId`，无法从 DOM 对齐 `taskGitOauth` 换票失败日志

## 根因

1. 文案来自 OAuth **浏览器回跳 query** `?github=exchange_failed`，由静态 `GITHUB_CALLBACK_HINTS` 映射，并非本页 axios/fetch 的即时响应体。
2. `taskGitOauth` 的 `redirectGithubContinue` 只写 `github=<code>`，**未**把回调请求上下文中的 `trace_id` 透传到前端 URL。
3. `UserGitSiteOAuthSettings.vue` 错误 `<p>` 未绑定 `data-traceId`；`returnKey` 中间页 `GithubAppCallbackContinue` 回跳时也会丢掉 `trace_id`。

按元规则 24：确无 traceId 时应省略属性；此处后端有请求级 trace，但未传到展示层，属于契约缺口。

## 解决方案

- Go：`withRequestTraceID` / `frontendOAuthErrorRedirect`，失败回跳附加 `trace_id=`。
- Vue：错误节点 `:data-traceId="errorTraceId || undefined"`；从 query / API 响应提取并在清理 query 时去掉 `trace_id`。
- 中间页：把 `trace_id` 合并进最终 return URL。
- Toast 守卫：`toastService.error(..., { traceId })`。

## 验证

```bash
cd /tmp/ram-work/taskGitOauth && go test ./src -count=1 -run 'GithubCallbackAcceptsSignedState'
cd /tmp/ram-work/taskFE/app && npx vitest run \
  src/views/UserGitSiteOAuthSettings.test.js \
  src/utils/gitSiteOAuthCallbackUtils.test.js \
  src/tests/domain/oauth_callback/oauth_callback_domain_model.test.js
```

公网生效：重启 `taskGitOauth` + `bash taskFE/app/scripts/runall-lifecycle.sh build`（见 `OPT-20260717-042`）。
