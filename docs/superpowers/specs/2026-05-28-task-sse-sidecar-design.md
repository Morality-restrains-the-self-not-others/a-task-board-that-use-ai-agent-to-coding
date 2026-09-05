# taskSSE 侧车设计

日期：2026-05-28  
状态：已实现  
默认传输：**Redis**（轻量 pub/sub）

## 问题

Django `runserver` 为每个 SSE 连接占用一个线程；任务详情页长连接泄漏后导致 API 假死，relay 直接启动失败。

## 方案

独立 Node.js 服务 `taskSSE`（:8798）承载 SSE；Django 仅通过 Redis `sse:{task_id}` 或 HTTP `/internal/publish` 推送；Vite 将同源 SSE 路径代理到侧车。

## 配置

`port_config.json` → `taskSSE.transport: "redis"`（一键可改 `kafka`）。

## 验收

- `GET :8798/health` → ok
- 浏览器 EventSource 经 Vite 代理收到 connected 心跳
- Django 线程数不随 SSE 连接线性增长
