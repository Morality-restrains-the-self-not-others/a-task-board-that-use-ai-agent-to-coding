# 意图：DevTools TaskPlugin 点选请求时把请求体写入任务描述

## 背景与目标

在 Chrome DevTools「TaskPlugin」面板点选 Network 请求后，任务描述会填入 URL、状态、响应体/头、请求头，但 **POST/PUT 等请求的请求体经常缺失**。根因：Chrome 扩展 `devtools.network` HAR 为效率常省略 `postData.text`；`getContent()` 只返回**响应体**。须用 `webRequest.onBeforeRequest` 的 `requestBody` 补捕获，并在点选时写入描述。

## 范围与边界

- **范围内**：TaskPlugin 单请求创建；HAR `postData` 解析增强；SW 捕获并按 tabId/method/url/时间窗匹配请求体；描述格式化包含请求体。
- **范围外**：不引入 `chrome.debugger`（避免调试横幅）；不改服务端 API；不在此变更中做敏感头/体脱敏（既有路径已可能带 Authorization）。

## 约束与风险

- 请求体可能含密钥/PII：禁止把完整请求体打进 `console` 日志。
- MV3 SW 须在顶层注册 `onBeforeRequest`，否则休眠时会漏捕。
- 大体量 body 截断，避免 SW 内存膨胀。

## 验收标准

1. 点选带 JSON/表单 body 的 POST，任务描述含 `**请求体**` 及正文。
2. HAR 已有 `postData.text` 时仍能写入（不依赖 webRequest）。
3. GET / 无 body 时不捏造请求体段落。
4. 使用说明写明点选会带上请求体。

## 实施计划

1. 抽出 `formatRequestAsTaskDescription`，用单测锁住「有 body 必写入」。
2. 增强 HAR `postData` 解析（string / text / params）。
3. SW `onBeforeRequest` + `lookupRequestBody`；DevTools 在 HAR 无 body 时查询。
4. 同步 USER_GUIDE / user-guide.js / 价值流测试点。

## 业务意图 → 事件对照

| 业务意图 | 领域事件 | MQ Topic | 说明 |
|----------|----------|----------|------|
| 点选请求填充任务描述（含请求体） | （无） | — | **无对应事件**：纯浏览器扩展本地捕获与表单填充，创建任务仍走既有 createTask HTTP，不新增服务端聚合变更 |

## 权限边界

仅当前用户在本机 DevTools 查看自己浏览会话的 Network 流量；请求体只进入该用户即将创建的任务描述。无新服务端端点、无跨租户数据。
