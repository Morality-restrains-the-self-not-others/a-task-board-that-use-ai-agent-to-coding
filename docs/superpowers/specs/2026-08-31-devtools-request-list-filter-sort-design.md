# DevTools 请求列表过滤与排序

- **Date:** 2026-08-31
- **Status:** accepted (goal-mode 自动采用)
- **Author:** cursor
- **Service:** taskChromePlugin（Chrome 扩展 DevTools 面板）

## 目标与成功标准

用户在 DevTools「云端Coding: 自动创新助手」→「单请求创建」的请求列表中，可以**过滤**并**按列排序**网络请求，以便从密集流量中点选目标请求创建任务。

| # | 可验证标准 |
|---|------------|
| S1 | 文本搜索匹配 URL / 方法 / 状态码 / `canceled`（保持现有能力） |
| S2 | 方法下拉过滤（含 GET/POST/PUT/PATCH/DELETE/OPTIONS/HEAD） |
| S3 | 状态过滤：全部 / 2xx / 3xx / 4xx / 5xx / canceled |
| S4 | 资源类型芯片：全部 / XHR / Doc / JS / CSS / Img / Other（映射 HAR `_resourceType`） |
| S5 | 点击列表表头按 方法 / 状态 / URL / 耗时 / 时间 排序；同列再点切换升/降序 |
| S6 | 默认排序：时间倒序（与现网一致，不回归） |
| S7 | 过滤+排序为纯函数，单元测试覆盖组合与边界；空结果仍显示「暂无匹配的请求」 |
| S8 | USER_GUIDE.md 与 `lib/user-guide.js` 同步说明过滤/排序 |

## 现状（代码基线）

- `panel/tabs/single-request.js` `applyRequestFilters`：已有 search + method + status；**排序写死** `timestamp` 降序。
- 列表行已展示 method / status / url / duration（`req.time`）；HAR 已带 `type`、`timestamp`。
- Popup 请求预览有搜索+状态过滤、无用户排序；本增量**不改 Popup UI**（记 OPT）。

## 🕸️ Code Review Graph 分析

- `code-review-graph update --brief` 成功（114 nodes）。
- `code-review-graph search applyRequestFilters`：**0 nodes**。根图未索引 `taskChromePlugin` 面板脚本，设计基于源码阅读（`single-request.js` / `har-request.js` / `panel.html`）。
- CRG impact：skipped_index_gap — 子仓未 register。

## 架构判断

**不更新** `docs/architecture/`：无新服务、无跨服务数据流、无基础设施变更。属既有扩展面板 UI 增强。

## Python 新增接口

不触发。无 HTTP 接口。

## 方案（采用）

抽取 `lib/request-list-query.js`（与 `har-request.js` 同模式：node + `globalThis`）：

- `resourceTypeBucket(type)` → `xhr|doc|js|css|img|other`
- `filterRequests(list, { search, method, status, type })`
- `sortRequests(list, { key, dir })`，`key ∈ timestamp|time|method|status|url`
- `nextSortState(current, clickedKey)`：同列翻转；新列数值类默认 `desc`，字符串类默认 `asc`
- `queryRequestList(list, query, { limit })`：filter → sort → slice

面板：

- 搜索栏下增加类型芯片；方法下拉补 OPTIONS/HEAD。
- `#selectedRequest` 上方固定可点击表头（`aria-sort`）。
- `applyRequestFilters` 改为读 DOM + `state.requestSort` / `state.requestTypeFilter` 后调用纯函数。
- 刷新/新请求到达后保持当前过滤与排序（不重置）。
- 清空列表不重置过滤条件（用户可能马上再捕获）；表头状态保留。

### 类型映射

| HAR `_resourceType`（大小写不敏感） | bucket |
|---|---|
| xhr, fetch, preflight | xhr |
| document, main_frame, sub_frame | doc |
| script | js |
| stylesheet | css |
| image, imageset | img |
| 其它 / 空 / unknown | other |

### 排序键

| key | 取值 | 新列默认方向 |
|-----|------|----------------|
| method | `req.method` 字符串 | asc |
| status | canceled → -1，否则 `statusCode` | desc |
| url | `req.url` | asc |
| time | `req.time`（ms） | desc |
| timestamp | `req.timestamp` | desc |

稳定排序：主键相等时用 `id` 字符串比较，避免刷新闪烁。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| Chrome Network 过滤语法（`is:xhr status-code:200`） | 学习成本高，现有下拉已覆盖主路径 |
| 仅增加排序下拉、无表头 | 不如列头点击符合 Network 心智 |
| 在 SW 侧过滤 | 列表已在 panel 内存（≤200），本地过滤足够 |
| 本增量改 Popup | 用户指定 DevTools；Popup 无多列布局 |

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 过滤/排序本地请求列表 | — | — | — | **无对应事件**：纯前端本地视图，不改变服务端事实 |
| 点选请求创建任务 | — | 既有 `createTask` | 既有后端 | 本增量不改创建路径 |

## Domain Concept Inventory

- Bounded context：ChromePlugin DevTools Request List（客户端）
- VO：`RequestListQuery`、`SortState`、`ResourceTypeBucket`
- Domain service：`queryRequestList`（纯函数）
- 无新聚合、无仓储、无 MQ

## Value Stream Impact

影响既有「DevTools 单请求创建」流的**选请求**步骤，不新增后端字段。不写入根 `value-stream.yaml`。

## 🏛️ 架构变更影响

无。current 仍为 v121（v122 为无关 target）。

## 风险

- 表头点击与行点击冒泡：表头放在列表外，不绑定在 `.request-item` 上。
- XSS：表头静态；行 URL 继续走既有 `escHtml`。
- 行数：`single-request.js` 已约 306 行；过滤逻辑外提后应下降。
