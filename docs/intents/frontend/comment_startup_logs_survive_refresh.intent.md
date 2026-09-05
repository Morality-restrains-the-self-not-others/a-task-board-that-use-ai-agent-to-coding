# 意图：释放后刷新仍显示启动日志

## 背景与目标

评论「启动日志」只在 SSE 会话内存中可见；刷新后 binding 已 `released`，面板 `serverStatus` 为空导致整块不渲染，尽管 list API 已返回完整 `logs`。

目标：释放/完成/终止后仍展示启动日志时间线（含停服触发说明）。

## 范围与边界

- 范围内：`mapBindingLifecycleToServerStatus`；启动面板在 `statusLogs.length > 0` 时也渲染。
- 范围外：不改分片落库；不把心跳行重新并入启动日志。

## 验收标准

1. `released` binding 的 `buildPerBindingServerStatusProps.serverStatus === 'stopped'`。
2. 面板在仅有 statusLogs、无 running 标志时仍显示历史行。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|-----------|--------|--------------|---------|
| 冷打开还原启动日志 | — | — | comment-container-bindings list | 前端 mergeBackendBindingLogs | 纯查询展示 |

## 变更记录

- 2026-08-23：released 后面板仍渲染后端 logs
