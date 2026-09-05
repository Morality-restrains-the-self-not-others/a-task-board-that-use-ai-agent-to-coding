# DDD 领域建模: Trace Log Journey 日志目录对齐修复

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-28-trace-log-journey-empty-log-list-fix.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-28-trace-log-journey-empty-log-list-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-28-trace-log-journey-empty-log-list-fix-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`

## 跳过声明

**DDD 领域建模跳过。** 本修复为纯配置文件变更——将 `conf/runAll.yaml` 中 `logging.file_root: ../logs` 改为 `logging.file_root: logs`。

变更不涉及：
- 新的限界上下文或聚合
- 新的实体、值对象或领域事件
- 新的仓储接口或领域服务
- 任何业务规则的变更

修复回归到现有 `platform-observability` 限界上下文的原设计意图，无需建模新的领域概念。

## 现有领域概念（不变）

本次修复不改变现有的领域模型。相关的现有概念供参考：

| 类型 | 概念 | 位置 |
|------|------|------|
| Bounded Context | `platform-observability` | runAll/AiMonitor |
| Entity | `LogStream` (per-service 日志流) | 无变更 |
| Value Object | `LogFilePath`, `LogEntry` | runAll/src/domain/ |
| Repository | `ServiceLogRepository`, `ServiceLogFileSink` | runAll/src/domain/ |
| Domain Service | `ServiceLogPathResolution` | runAll/src/domain/ |
