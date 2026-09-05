# Kafka 发送有日志、消费无日志 - 分析报告

## 现象
- `kafka_events_20260318.log` 中有发送日志（Sending event USER_CREATED）
- `kafka_consumer_20260318.log` 中仅有启动日志，无消费日志（Received message / Dispatching）

## 架构
- **Producer**：Django / `init_tenant.py`（独立进程）→ 发送事件
- **Consumer**：`start_kafka_consumer` 管理命令（独立进程）→ 消费事件
- **日志**：两进程共用 `kafka_events_*.log`、`kafka_consumer_*.log`

## 可能原因

### 1. init_project 流程导致 Consumer 被提前终止（主因）
执行 `init_project.sh` 时：
1. `run.sh start` 后台启动（含 Kafka 消费者）
2. 等待服务就绪 + sleep 20 秒
3. 执行 `init_tenant.py`，发送 `USER_CREATED`
4. `init_tenant` 返回后立刻执行 `cleanup_and_stop`，终止 Consumer 进程

Consumer 可能在收到并处理消息前被 `pkill` 终止，导致消费日志未写入或被截断。

### 2. Producer 未显式 flush（次要）
`producer.py` 第 88 行注释：
```python
# 移除 flush() 调用，让消息异步发送，避免阻塞请求
```
`init_tenant` 作为短生命周期进程，发送后若进程很快退出，未 flush 的消息可能尚在缓冲区，未真正写入 Kafka。

### 3. 日志缓冲区未及时落盘
进程被 kill 时，若 Python logging 的 FileHandler 仍有缓冲未 flush，日志可能未完全写入磁盘。

## 建议修改

1. **init_tenant 发送后显式 flush**：保证消息写入 Kafka 后再继续后续逻辑
2. **init_project 在 init_tenant 完成后增加短暂等待**：给 Consumer 留出处理时间再停止服务
3. **日志 FileHandler 启用 flush**：减少异常退出时日志丢失
