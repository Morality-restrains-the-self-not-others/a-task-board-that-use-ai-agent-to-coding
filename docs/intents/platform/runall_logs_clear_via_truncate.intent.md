# Intent: runAll tee 日志清空走 truncate API

## 目标

清空 `logging.file_root`（默认 `logs/`）下服务 tee 日志时，必须经 runAll 的 truncate 路径（`FileServiceLogSink.Truncate*` / `POST /api/logs/clear` / `POST /api/logs/clear-all`），禁止对 `logs/*.log` 使用 `rm`/`unlink`。

## 背景

runAll 长期持有日志文件 FD。若外部 `rm` 后重建同名文件，写入落到 deleted inode，Promtail 只 tail 新路径 → Loki 找不到 probe（Trace 投递验证失败）。

## 验收

1. `POST /api/logs/clear-all` 清空内存缓冲并 truncate 文件，响应含 `"method":"truncate"`。
2. 清空后文件路径仍存在且 inode 保持（非 rm 重建）；后续 append 仍写入该路径。
3. `runAll/scripts/truncate-ram-work-logs.sh` 优先调用 `:9999/api/logs/clear-all`；API 成功时不对 `logs/` 做 shell 截断；全文不对 tee `*.log` 使用 `rm`。
4. `/api/dev/logs/clear` 仅清开发工具日志，不承担服务 tee 清空职责。
5. `taskGateway/logs/*.log`（APISIX uid 636 / 0644）经 `docker exec` in-place truncate，并 `chmod a+rw`；`taskGateway/run.sh` start/reload 后同样 widen 权限，宿主 truncate 不再 skip。



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：运维日志 truncate API，不产生业务领域事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: runAll tee 日志清空走 truncate API | — | — | — | — | 运维日志 truncate API，不产生业务领域事件 |
## 变更记录

- 2026-07-15：新增 bulk clear-all API；修正 cron 脚本误调 `/api/dev/logs/clear` 与错误端口 18080。
- 2026-07-15：修复 taskGateway 日志宿主不可写——容器内 truncate + chmod；run.sh 启动后 ensure_logs_host_writable。
