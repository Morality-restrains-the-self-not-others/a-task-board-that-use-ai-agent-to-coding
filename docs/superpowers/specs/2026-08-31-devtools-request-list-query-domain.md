# 领域模型：DevTools Request List Query

- **Date:** 2026-08-31
- **Context:** taskChromePlugin 客户端（非 Go 服务）

## Bounded Context

`ChromePlugin.DevToolsRequestList` — 将已捕获的 HAR 请求投影为可过滤、可排序的列表视图。

## Value Objects

- `ResourceTypeBucket`: `xhr | doc | js | css | img | other`
- `SortKey`: `timestamp | time | method | status | url`
- `SortDir`: `asc | desc`
- `RequestListQuery`: `{ search, method, status, type, sortKey, sortDir }`
- `CapturedRequest`（既有）：`id, method, url, statusCode, canceled, type, time, timestamp, ...`

## Domain Service（纯函数端口）

```
queryRequestList(requests, query, { limit }) → CapturedRequest[]
nextSortState(current: {key, dir}, clickedKey) → {key, dir}
resourceTypeBucket(harType) → ResourceTypeBucket
```

实现落点：`taskChromePlugin/lib/request-list-query.js`（无 Chrome API）。

## 领域事件

无。纯查询/投影。创建任务事件不在本上下文。

## 不建模

SW 缓冲、HAR 补录、createTask 载荷 — 既有上下文，本增量只消费 `recentRequests`。
