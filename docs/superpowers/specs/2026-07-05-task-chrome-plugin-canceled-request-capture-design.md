# taskChromePlugin：Canceled 网络请求捕获修复

**日期**: 2026-07-05  
**状态**: 已实施

## 问题描述

DevTools TaskPlugin 面板的「请求列表」与「批量错误捕获」无法捕获 Chrome Network 面板中状态为 **(canceled)** 的请求。

## 根因分析

### 1. DevTools 面板路径（`devtools/devtools.js`）

| 现象 | 原因 |
|------|------|
| Canceled 请求未出现在列表 | 原实现对 `entry.response` 直接访问（`.status`、`.headers.reduce`），Canceled 请求常无完整 response 对象，回调内 **抛错中断**，整条记录丢失 |
| 无 try/catch | 解析失败静默丢失，无日志 |

Chrome 中 Canceled 请求特征（HAR）：

- `response.status === 0`
- 无 HTTP 响应头/体
- 可能含 `_errorText: net::ERR_ABORTED`
- `onRequestFinished` **会触发**，但 response 字段不完整

### 2. 批量捕获路径（`background/service-worker.js`）

| 现象 | 原因 |
|------|------|
| 自动捕获漏掉 Canceled | 仅注册 `chrome.webRequest.onCompleted`；客户端中止的请求走 **`onErrorOccurred`**（`net::ERR_ABORTED`），不会触发 onCompleted |

### 3. UI 层（次要）

- 状态码过滤器无「Canceled」选项，status=0 显示为 `0` 而非可读标签
- 批量捕获配置无 `canceled` 状态码选项

## 解决方案

### A. devtools.js — 安全解析 HAR 条目

1. 新增 `headersToObject()`：headers 非数组时返回 `{}`
2. 新增 `isCanceledHarEntry()`：status=0 / statusText 含 cancel / `_errorText` 含 ERR_ABORTED
3. 新增 `buildRequestFromHarEntry()`：对 `entry.response` 做空值保护
4. 请求对象增加 `canceled: boolean`、`error: string`
5. 整体包裹 try/catch，失败时 `console.error` 便于排查

### B. service-worker.js — 补充 onErrorOccurred

```javascript
chrome.webRequest.onErrorOccurred.addListener(handleRequestError, { urls: ['<all_urls>'] });
```

- 仅处理 `details.error === 'net::ERR_ABORTED'`
- 写入与 onCompleted 相同结构的 capture entry（statusCode=0, canceled=true）
- `matchStatusCode()` 增加 `canceled` 模式

### C. UI / 配置

- 请求列表、Popup、错误列表：status=0 显示 **Canceled**
- 状态过滤器新增「Canceled」
- 批量捕获 checkbox 新增 **Canceled**（默认勾选）
- `storage.js` 迁移：旧配置自动追加 `canceled`

## 验收标准

- [x] DevTools 打开时，页面内 AbortController 取消的 fetch 出现在 TaskPlugin 请求列表
- [x] 列表中显示 `Canceled` 而非空白/崩溃
- [x] 启用批量捕获且勾选 Canceled 时，`net::ERR_ABORTED` 进入捕获列表
- [x] 可按「Canceled」过滤；搜索 `canceled` 可命中

## 手动验证步骤

1. 加载扩展，打开任意页面 DevTools → TaskPlugin
2. Console 执行：
   ```javascript
   const c = new AbortController();
   fetch('/api/test', { signal: c.signal });
   c.abort();
   ```
3. 确认请求列表出现 `Canceled` 条目
4. 批量捕获 Tab 启用捕获 → 重复步骤 2 → 刷新错误列表确认捕获

## 已知限制

- DevTools 必须在目标页打开后才会注册 `onRequestFinished`；**晚开场景**已通过 `getHAR()` 补录缓解
- 极早期取消（请求尚未进入 HAR）仍可能漏捕，属 Chrome API 固有限制
- `onErrorOccurred` 路径无 request/response body（webRequest API 限制）

## 后续优化（2026-07-05 已实施）

### getHAR 历史补录

- DevTools 创建面板、Panel 显示、页面 `onNavigated` 时调用 `backfillFromHar()`
- 以 `startedDateTime + method + url` 为 `harKey` 去重，避免与 `onRequestFinished` 重复

### 单元测试

- 纯函数：`lib/har-request.js`（HAR 解析）、`lib/capture-status.js`（状态码匹配）
- `npm test` 覆盖 Canceled / 正常响应 / 去重 / 截断 / matchStatusCode / shouldEnrichHarBody

### HAR 补录 body 异步 enrich

- 补录后对关键请求（4xx/5xx、Canceled、非 GET）最多 25 条并发拉取 `getContent`
- Panel 通过 `requestUpdated` postMessage 刷新已选请求详情

### CI / pre-commit

- GitHub Actions：仓库根 `.github/workflows/task-chrome-plugin-test.yml`（paths 过滤 `taskChromePlugin/**`）
- 本地：`taskChromePlugin/scripts/hooks/pre-commit`（暂存 lib/devtools/background/test 时跑 `npm test`）

## 变更文件

| 文件 | 变更 |
|------|------|
| `devtools/devtools.js` | 安全 HAR 解析 + canceled 检测 |
| `background/service-worker.js` | onErrorOccurred + matchStatusCode |
| `lib/storage.js` | 默认 statusCodes 含 canceled |
| `panel/panel.js` | 展示/过滤/任务描述 |
| `panel/panel.html` | Canceled 过滤与捕获选项 |
| `panel/panel.css` | canceled 样式 |
| `popup/popup.js` | 展示/过滤 |
| `popup/popup.html` | Canceled 过滤选项 |
| `lib/har-request.js` | HAR 解析纯函数（可单测） |
| `lib/capture-status.js` | 批量捕获状态码匹配纯函数 |
| `test/har-request.test.js` | HAR 单元测试 |
| `test/capture-status.test.js` | 状态码匹配单元测试 |
| `scripts/hooks/pre-commit` | 本地提交门禁 |
| `.github/workflows/task-chrome-plugin-test.yml` | 仓库根 CI 工作流 |
| `package.json` | `npm test` 脚本 |
