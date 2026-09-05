# [运行时] ztree「发送给AI」失败只显示裸 HTTP 403

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-22
- 编号：113
- 维护者：Trae AI 团队

## 现象

- 公网任务详情执行细节 → ztree 命令面板红色 `<p class="mt-2 text-xs text-red-600">HTTP 403</p>`。
- DOM 无 `data-traceId`，无法按 trace 查 Loki。
- 选择器落在 `#comments-container` 第 4 条评论 `details` 内 `comment-layer-ztree-command-panel`。

## 根因

1. `submitLayerGraphCommand` 只读 `resp.json().detail`；网关/Cloud/APISIX 常用 `message`/`error` 或 HTML 403。
2. `json()` 消费 body 后 `text()` 读不到，非 JSON 403 永远回退成 `HTTP 403`。
3. `formatLayerGraphCommandErrorForUser` 无 403 分支；`layerGraphCmdErrorTraceId` 有模板无写入。
4. Gateway `authorizeContainerRequest` 查 CSC 时只读 query `comment_id`，忽略 path（ADR-0010），lookup 404 被映射成含糊的 `forbidden scope`。

## 解决方案

1. `messageFromFailedResponse` 读 `_errorData.message|error|detail` 与 HTML `_rawErrorText`。
2. 403 映射为权限/登录失效或「容器配置不可用」；写入 `layerGraphCmdErrorTraceId`。
3. lookup 使用 `lookupCommentID`（scope/path 优先）；CSC 404 文案改为「容器配置不存在或尚未就绪」。

## 验证

```bash
cd taskFE/app && npx vitest run src/utils/httpError.unit.test.js src/composables/taskDetail/submitLayerGraphCommand.test.js src/composables/taskDetail/taskDetailContainerHeartbeat.test.js
cd taskContainerGateway && go test ./src -count=1 -run 'LookupCommentIDPrefersScopePathOverEmptyQuery'
```

## 关联

- `.ai/01_project_constraints/24_frontend_error_data_trace_id.md`
- `.ai/09_failure_experience/02_runtime_errors/109_ztree_push_start_vm_killed_by_stale_terminal_released.md`
