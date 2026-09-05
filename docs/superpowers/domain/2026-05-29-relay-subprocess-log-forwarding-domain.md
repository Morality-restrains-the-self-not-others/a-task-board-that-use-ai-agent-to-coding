# Relay 子进程日志转发 — 领域模型

**Bounded Context:** 可观测性 / Relay 运行时（go-relay）

## 值对象

### StructuredLogLine
- `service: string`（必填）
- `trace_id?: string`
- `msg: string`
- 不变量：JSON 单行、可被 Promtail 解析

### LogBlock
- `lines: string[]`
- 合并后 `text()` 用 `\n` 连接

## 领域服务

### SubprocessLogForwarder
- `forward(line: string, defaultService: string)`: 结构化透传或包装纯文本
- `group(lines: Iterator<string>)`: 产出 `LogBlock` 序列

## 仓储接口

无持久化；输出端口为 `StdoutLogSink`（基础设施：`tracelog.ForwardChildLine`）

## 领域事件

本增量不持久化事件；可选 `ChildLogForwarded` 仅用于未来 metrics。
