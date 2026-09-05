# [运行时] 派生/自动运行获取运行环境 404：误调 Cloud 而非 AI Provider

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-18
- 最后修改：2026-07-18
- 维护者：Trae AI 团队

## 现象

任务详情「派生任务」且选择自动运行时，错误弹层文案：

```text
派生任务失败：自动运行需要镜像配置运行环境，但获取运行环境失败: cloud runtime environments status 404: 404 page not found
```

## 根因

1. `taskTaskService.lookupImageRuntimeEnvironments` 请求 `CloudServiceURL + /api/public/image-runtime-environments/`，该公开接口只在 **taskAiProvider** 注册，Cloud 返回 Go mux 默认 `404 page not found`。
2. 即便路由正确，门禁应使用已安装镜像的 **`external_image_id`**（镜像市场 ID），而非租户 installed image 主键。
3. 前端 `forkTask` 失败时把响应 body 字符串传给 `showRequestError`；`extractTraceId` 曾把整段 JSON 当 traceId，或错误文案挂在无 `data-traceId` 的 `<p>` 上，导致排障节点缺失。

## 修复

1. TaskService 配置 `AIProviderBaseURL`；lookup 先经 Cloud installed-image lookup 解析 `external_image_id`，再 GET AI Provider public runtime-environments。
2. `forkTask` 传 `Response`；`extractTraceId` 对 JSON 字符串只抽 `trace_id`；`Modal.ui` 错误文案 `<p>` 挂 `data-traceId`。

## 验证

```bash
cd taskTaskService && go test ./src/ -run 'AutoRun|RuntimeEnv|LookupImageRuntime' -count=1
cd taskFE/app && npm run test:unit -- ./src/utils/traceId.test.js ./src/components/Modal.ui.traceId.test.js ./src/composables/taskDetail/taskDetailEditing.test.js
```
