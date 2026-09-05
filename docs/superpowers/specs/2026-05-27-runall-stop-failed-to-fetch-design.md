# runAll 关闭服务 Failed to fetch 修复设计

**日期:** 2026-05-27  
**状态:** 已实现  
**范围:** `http://localhost:9999/` — 单行「关闭」（含 taskFE）

## 问题

对 `taskFE` 点击「关闭」，弹窗：

```text
Stop failed: Failed to fetch
```

`Failed to fetch` 来自 `status.html` 中 `fetch()` 的 catch 分支，表示 **HTTP 响应未到达浏览器**（连接中断/拒绝），而非 API 返回 4xx 业务错误。

## 根因

`/api/stop`（及 start/restart/group）在 HTTP 处理 goroutine 内 **同步执行** 完整停服流程（SIGTERM 等待最长 5s、`lsof` 端口清理、ownership 写入）。在以下情况浏览器侧会表现为 Failed to fetch：

1. 停服耗时较长，客户端连接在响应写回前断开（`Request.Context` 取消）  
2. 极端情况下端口清理与探活叠加导致 handler 阻塞更久  

单点 `StopServiceWithActor` 语义不变；**仅将生命周期 API 改为异步接受 + 后台执行**，UI 依赖 2s 轮询展示最终状态/错误。

## 方案

| 层 | 变更 |
|----|------|
| `ui.go` | 校验通过后 `go action(context.Background(), …)`，立即 `202` + `{"status":"accepted"}` |
| `status.html` | `accepted` / `ok` 均触发 `refresh()`；不再因长时间阻塞触发 fetch 失败 |
| 测试 | Go API 测试轮询终态；Playwright 点击关闭断言无 Failed to fetch |

## 验收

1. Playwright：`taskFE` 关闭不出现 `Failed to fetch`  
2. 刷新后状态变为 `stopped`（或 `failed` 且带 error-msg）  
3. `go test ./runAll/src/...` 通过  
