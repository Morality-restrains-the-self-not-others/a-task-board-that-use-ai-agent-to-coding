# 意图：停服日志注明触发源头

## 背景与目标

任务详情评论「启动日志」出现「正在调用aliyunAPI停止服务器...」时无法判断是用户点击、空闲回收还是任务终态。目标：每条停服进度/成功文案附带 `（触发：…）`。

## 范围与边界

- 范围内：`stop-vm` API、`releaseMachineForTerminal`、`CLOUD_SERVER_STOPPED` SSE；`stop_reason` 透传与中文标签。
- 范围外：不改 Aliyun DeleteInstance 本身；不改工作区 idle_recycle 清库路径（该路径不调云 API）。

## 验收标准

1. 用户点击停止 → `（触发：用户点击停止服务器）`。
2. 指令空闲超时 → `（触发：容器指令空闲超时回收）`。
3. 冷打开从 binding `logs` 仍能看到带触发说明的停服行。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|-----------|--------|--------------|---------|
| 停止云服务器 | CLOUD_SERVER_STOPPED | Kafka | stop-vm / releaseMachineForTerminal | cloudserverstopped → DeleteInstance + SSE | — |

## 变更记录

- 2026-08-23：停服 SSE/持久化日志附加触发源头
