# Trace Log Explore 增加 level 过滤设计

日期：2026-05-30  
状态：待批准  
范围：Grafana `trace-log-explore` 仪表盘新增 `level` 模板变量，与现有 `service`、`trace_id` 并列过滤

---

## 1. 背景与目标

### 1.1 背景

用户在使用 Trace Log Explore 仪表盘（`http://127.0.0.1:3000/d/trace-log-explore/...`）排查链路日志时，已有：

| 变量 | 类型 | 作用 |
|------|------|------|
| `service` | Loki query 下拉 | 按服务名过滤（含 All） |
| `trace_id` | 文本框 | 按行内文本 / JSON trace_id 过滤 |

Promtail 已在 JSON pipeline 中将 `level` 提升为 Loki label（`AiMonitor/promtail/promtail.yaml`），各服务 JSON 日志契约也统一输出小写 level（如 `info`、`error`、`warn`）。

**缺口：** 仪表盘 LogQL 未使用 `level` label，用户无法快速只看 error/warn 级别日志。

### 1.2 目标

1. 在 Trace Log Explore 顶部增加 **level** 下拉过滤，行为与 **service** 一致（含 All、动态选项）。
2. 更新面板 LogQL，使 `service` + `level` + `trace_id` 三者可组合过滤。
3. 保持现有 URL 深链兼容（未传 `var-level` 时默认 All）。

### 1.3 非目标

- 修改 Promtail pipeline（`level` label 已存在）。
- 修改各服务日志格式或 level 枚举。
- runAll UI 深链默认带 level（可选增量，本阶段不做）。
- 新增 Loki 数据源或 retention 策略变更。

---

## 2. 价值流影响

关联 `value-stream.yaml` → **`platform-centralized-logging`**：

| 步骤 | 影响 |
|------|------|
| `increment2-grafana-trace-dashboard` | **扩展**：Trace Log Explore 增加 level 变量与 LogQL |
| `increment2-json-log-contract` | 无变更（已输出 `level` 字段） |
| `increment3-runall-grafana-deep-link` | 无变更（深链仍只传 trace_id） |

**测试影响：**

- 新增 `AiMonitor/scripts/test_trace_log_explore_dashboard.py`（或扩展现有 promtail 测试）校验 dashboard JSON 结构。
- 现有 `test_promtail_config.py` 可补充断言 `level` 在 labels stage 中（已有 pipeline，可选加强）。

---

## 3. 领域概念清单（轻量）

| 概念 | 说明 |
|------|------|
| **Bounded Context: 平台可观测性** | AiMonitor Grafana 仪表盘配置 |
| **LogFilterVariable** | 仪表盘模板变量（service / level / trace_id） |
| **LogQLSelector** | `{job="runall", service=~"$service", level=~"$level"}` |
| **Domain Event** | 无（纯配置变更，无运行时事件） |

---

## 4. 方案

### 4.1 推荐方案：Grafana 模板变量 + LogQL label matcher

在 `AiMonitor/grafana/provisioning/dashboards/files/trace-log-explore.json` 中：

**新增变量 `level`：**

```json
{
  "allValue": ".*",
  "includeAll": true,
  "label": "level",
  "name": "level",
  "type": "query",
  "datasource": { "type": "loki", "uid": "loki" },
  "query": "label_values(level)",
  "definition": "label_values(level)",
  "refresh": 2,
  "sort": 1
}
```

与 `service` 变量对称：`allValue: ".*"` + `includeAll: true`，选 All 时不限制 level。

**更新面板 LogQL：**

```
{job="runall", service=~"$service", level=~"$level"} |= "$trace_id" | json | __error__=""
```

- `level=~"$level"`：选 All 时 `$level` 为 `.*`，匹配有/无 level label 的流（无 label 的行在选具体 level 时被排除，符合预期）。
- `trace_id` 仍用 `|=` 行过滤，与现有行为一致。

**面板标题：** 更新为 `RunAll logs (service + level + optional trace_id)`。

### 4.2 备选方案（不采用）

| 方案 | 原因不采用 |
|------|-----------|
| LogQL `| json | level=~"$level"` 管道过滤 | 依赖 JSON 解析，慢于 label matcher；Promtail 已有 label |
| 固定枚举 dropdown（debug/info/warn/error） | 与 Loki 实际 label 值可能不同步 |
| runAll 深链强制带 level | 用户未要求；增加 API 复杂度 |

---

## 5. 行为说明

| 场景 | 预期 |
|------|------|
| level = All | 显示所有级别（与改前一致） |
| level = error | 仅显示 Promtail 已打 `level=error` 的日志行 |
| 旧日志无 level label | 在选具体 level 时不显示；选 All 仍可见 |
| URL `var-level=error` | 直接打开并过滤 error |
| 与 trace_id 组合 | 先 label 过滤，再文本匹配 trace_id |

---

## 6. 实现清单（供 Step 6 计划引用）

1. 修改 `trace-log-explore.json`：新增 `level` 变量、更新 expr 与 title、version bump。
2. 新增 dashboard 结构测试（Python）：断言 templating 含 `level`、expr 含 `level=~"$level"`。
3. （可选）`test_promtail_config.py` 断言 labels stage 含 `level`。
4. 本地验证：Grafana 重启/热加载后，选 level=info/error 可见差异。

**不涉及：** runAll Go 代码、Promtail YAML、value-stream.yaml 字段新增（除非 Step 3 决定登记新 step）。

---

## 7. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 历史非 JSON 日志无 level label | 文档说明；选 All 仍可查 |
| label 高基数 | level 仅 ~5 值，远低于 trace_id |
| Grafana provisioning 缓存 | 重启 ai-monitor 或 bump dashboard version |

---

## 8. 验收标准

- [ ] 仪表盘顶部出现 level 下拉，选项来自 Loki `label_values(level)`
- [ ] 选 `error` 后日志面板仅含 error 级别行
- [ ] 选 All + 填写 trace_id 行为与改前一致
- [ ] 自动化测试通过（dashboard JSON + 现有 promtail 测试）
