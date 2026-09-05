# relay 停止误杀 Django（task2app 挂起）修复设计

日期：2026-05-27  
状态：已实现

## 现象

task-detail `relayToTrae=true` 直接启动 → 启动 onlineServiceJS → 点击「停止」后，**task2app（Django 8001）无响应**，runAll/页面 API 全部 pending。

## 根因（Playwright + curl 复现）

`go_relayToTrae` 的 `killPortListeners` 使用：

```bash
lsof -ti :8765
```

在 macOS 上该命令返回 **监听 8765 的进程 + 所有与 8765 有 TCP 连接的客户端**。

停止流程中 `callOnlineServiceReset` 向 onlineServiceJS 发 POST 的同时，`killPortListeners` 会对上述 PID 发 SIGTERM/SIGKILL。  
若 Django runserver 线程正作为 **HTTP 客户端** 访问 `127.0.0.1:8765`（容器 API、reset 链路等），**整个 Django 进程会被误杀**，表现为 task2app 挂起。

## 修复

改为仅终止 LISTEN 进程：

```bash
lsof -tiTCP:8765 -sTCP:LISTEN
```

涉及：

- `go_relayToTrae/src/process.go`（`portListenerPIDs` + `killPortListeners`）
- `go_relayToTrae/src/main.go`（启动前清端口复用同一逻辑）
- `relayToTrae/server.py`、`relayToTrae/bootstrap.py`（Python 侧车 parity）

## 测试

- `TaskDetail.relay-to-trae-stop-django-alive.playwright.test.js`（`PLAYWRIGHT_RELAY_STOP_DJANGO=1`）
- `go test ./go_relayToTrae/src/...`

## 运维

修改后须 **重启 go_relayToTrae 二进制**（`go_relayToTrae/bin/go_relayToTrae`），否则仍运行旧 lsof 逻辑。
