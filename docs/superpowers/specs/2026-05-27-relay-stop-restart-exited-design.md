# relay 停止后再次启动 exited (code=1) 修复设计

日期：2026-05-27  
状态：已实现

## 现象

task-detail 直接启动：启动 → 停止 → 再启动，面板提示 `onlineServiceJS exited (code=1)`，onlineServiceJS 无法再次运行。

## 根因

1. **Django 异步 stop/start 竞态**：`relay_to_trae_stop` 与 `_relay_to_trae_start_async_task` 各在线程中调用 go_relay，UI 在 stop 返回后立即允许再次 start，但 go_relay 侧 `/v1/stop` 可能仍在执行（reset 最长 45s + killPortListeners）。
2. **go_relay 无互斥**：`/v1/start` 已拉起新 onlineServiceJS 后，`/v1/stop` 的 `killPortListeners` 清理 8765 端口时会 **误杀新进程**，子进程 exit code=1，经 SSE 显示为 `onlineServiceJS exited (code=1)`。
3. **二次 start 未清 orphan 端口**：`handleStart` 仅在 `state.Running` 时 `stopRunning()`，stop 后 `Running=false` 但端口仍可能被占用时未主动 `killPortListeners`。

## 修复

| 层 | 变更 |
|----|------|
| Django `relay_to_trae_proxy.py` | `_relay_lifecycle_lock` 串行化 `/v1/stop` 与 `/v1/start` HTTP 调用 |
| go_relay `handlers.go` | `lifecycleMu` 串行化 `handleStart`/`handleStop`；提取 `stopOnlineService` |
| go_relay `process.go` | `ensureOnlineServicePortFree()` 在 start 前清理 orphan listener |
| stop 收尾 | `stopOnlineService` 清空 `state.Error` |

## 测试

- `test_relay_start_waits_for_stop_lifecycle_lock`（Django）
- `TaskDetail.relay-to-trae-stop-restart.playwright.test.js`（Mock UI 回归）
- `go test ./go_relayToTrae/src/...`

## 关联

- 前置修复：`lsof -tiTCP:PORT -sTCP:LISTEN` 避免误杀 Django（见 `2026-05-27-relay-stop-kills-django-design.md`）
