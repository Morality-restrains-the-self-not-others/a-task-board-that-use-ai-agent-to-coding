# 结构化日志 level 必须为小写 canonical 值

- **版本**：1.0.0
- **日期**：2026-07-15
- **适用范围**：monorepo 内所有向 Loki/Promtail 输出的结构化 JSON 日志（Go / Python / Node / runAll 编排状态行）

## 目标

消除 Loki `level` 标签大小写混用（`INFO`/`info`、`WARN`/`warning`），使 Grafana 变量与 LogQL 过滤器可稳定按级别筛选，并与 [可观测性日志规范](../03_technical_implementation/12_observability_log_shipping.md) 一致。

## 强制要求（禁止忽略）

1. **允许值**：JSON 字段 / Loki 标签 `level` **仅允许**小写：`debug` | `info` | `warn` | `error`。
2. **禁止**：输出 `INFO`、`WARN`、`ERROR`、`DEBUG`、`WARNING`、`CRITICAL` 等大写或非 canonical 字符串作为 `level`。
3. **别名归一**：
   - `warning` / `WARNING` → `warn`
   - `fatal` / `panic` / `critical` / `crit` → `error`
   - `trace` → `debug`
   - 空字符串 → `info`
4. **落点优先共享库**（禁止在业务代码散落大小写硬编码）：
   - Go：`shareLib/tracelog`（`NormalizeLevel`、`Init`/`InitConsumer` 的 slog `ReplaceAttr`、`Emit*`）
   - Python：`Saas_project/core/logging/json_trace_formatter.py`（`normalize_log_level`）
   - Node：`jsonLog` / 等价封装须 `.toLowerCase()` 并映射 `warning`→`warn`
5. **新增服务**：必须经上述共享封装输出日志；禁止新建手写 JSON logger 再写大写 level。
6. **Agent 义务**：新增/修改日志输出时若发现大写或 `warning` 未映射，**必须**在共享层或本服务 formatter 归一化后再合并。

## 允许与禁止对照

| 场景 | 正确 | 错误 |
|------|------|------|
| slog / JSON `level` | `"info"` | `"INFO"` |
| Python `logger.warning` | `"warn"` | `"warning"` |
| Grafana 过滤 | `level=~"(?i)($level)"` 或小写标签 | 依赖用户多选 `info`+`INFO` |
| Promtail | 原样提升 JSON `level` | 在 Promtail 用大小写「掩盖」上游错误（仅可作过渡兜底） |

## 验收自检

```bash
# 新产生的日志行不得再出现大写 level（替换为近期日志路径）
rg '"level":"(INFO|WARN|ERROR|DEBUG|WARNING)"' logs/*.log

# 单元测试
cd shareLib/tracelog && go test ./... -count=1
# Python（在已激活的 Django venv 中）
pytest task2app/Saas_project/core/logging/test_json_trace_formatter_level.py -q
```

## 与相关规则的关系

- 字段清单与采集架构：`.ai/03_technical_implementation/12_observability_log_shipping.md`
- Cursor alwaysApply 摘要：`.cursor/rules/log-level-lowercase.mdc`
- 根摘要：仓库根 [`.ai.md`](../../.ai.md)「结构化日志 level」一节
