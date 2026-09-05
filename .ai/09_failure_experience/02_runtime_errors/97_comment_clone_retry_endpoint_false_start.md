# [运行时] 评论级克隆「手动重试」显示「容器未启动」

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-17
- 最后修改：2026-08-17
- 维护者：Trae AI 团队

## 现象

任务详情评论执行细节已出现「手动重试」（`comment-execution-clone-progress-retry`），点击后同级红字：

`data-testid="comment-execution-clone-progress-retry-status"` 可见文本 **「容器未启动」**。

无 `data-traceId`，Network 中没有 `POST /api/cloud/repo-reclone`。

复现：`task_877083769524219904` 冷打开，引导克隆已失败，容器仍在跑（进度条/启动日志可见）。

## 根因

`onRepoReclone` 在已有 `comment_id` 之后，仍用**任务页单例** `containerEndpointRegistered` 短路：

```js
if (!containerEndpointRegistered.value) {
  recloneStatusByUrl.value = { ...recloneStatusByUrl.value, [u]: '容器未启动' }
  return
}
```

该旗标来自 `container-task-ui-context` 的 `container_endpoint_registered`（业务 HTTP `server_url` 且算作 running）。引导克隆失败时常尚未登记业务端点，但评论级 CSC 已存在、容器已在跑。Cloud `handleRepoReclone` 本可按 `comment_id` 转发 `/api/repos/reclone`。

同类：`35_bootstrap_clone_fail_no_repo_name.md`（失败态重试不得只绑 endpoint）、`93_task_detail_layer_not_found_and_server_url_unreadiness.md`（执行日志已按 comment_id 放行）、`96_comment_clone_fail_no_manual_retry.md`（上一刀：按钮缺失）。

## 解决方案

- 有 `comment_id` 时去掉任务级 endpoint 门禁，直接 POST；无 comment_id 仍展示「缺少评论ID」。
- 真实不可达由 Cloud 返回 409/404，前端展示 `detail` 并写入 `data-traceId`。
- 单测 T29：`containerEndpointRegistered=false` + payload 含 commentId → 仍 POST，状态不是「容器未启动」。

## 预防

- 评论级容器转发（reclone / exec-log / compute）不得用任务页单例 `containerEndpointRegistered` 作为唯一门禁。
- 纯前端短路文案不得冒充「容器没起来」；引导克隆失败进度可见即说明容器已跑过。

## 验证

```bash
cd taskFE/app && npx vitest run \
  src/composables/taskDetail/taskDetailFetchFns.repoCloneIdentity.test.js
```

公网：`cd taskFE/app && npm run build` 后硬刷新任务详情，点「手动重试」应发出 `POST /api/cloud/repo-reclone/.../comment_id/{id}/`。

后续：重试成功后自动任务不续跑，见 `98_comment_clone_retry_no_autorun_resume.md`。
