# Trace Log Explore 增加按字段值过滤设计

日期：2026-06-01  
状态：已实现（2026-06-01）  
范围：Grafana `trace-log-explore` 仪表盘 — 补齐「按某个值过滤日志」的显式能力

---

## 1. 背景与目标

### 1.1 背景

Trace Log Explore（`/d/trace-log-explore`）当前顶部变量：

| 变量 | 类型 | 作用 |
|------|------|------|
| `service` | Loki label 下拉 | 按服务名过滤（含 All） |
| `trace_id` | 文本框 | 行内文本包含（含 JSON trace_id） |
| `level` | Loki label 下拉 | 按日志级别过滤（含 All） |

**现场验证（2026-06-01）：**

- URL `var-level=error` 时面板显示 **No data** — 因近 1h 内 Loki 无 `level=error` 行，并非变量缺失。
- 展开单条日志详情后，Grafana 提供 **Filter for value / Filter out value** 按钮；点击后顶部出现 **Filters** 行（如 `detected_level = info`）及 **Filter by label values** 下拉。
- 但 `trace-log-explore.json` **未 provisioning adhoc 模板变量**，Filters 行为依赖运行时 UI，不可深链、不可发现、刷新后丢失。
- Promtail 已提升的 Loki label：`job`, `service`, `level`, `trace_id`, `msg` 等；**`trace_id` 已是 label，仪表盘仍用 `|= "$trace_id"` 行过滤**，未利用 label 精确匹配。
- JSON 字段（如 `forward_stage`, `logger`）多数 **未提升为 label**，无法通过顶部变量或 adhoc label 过滤。

**用户诉求（归纳）：** 页面缺少 **按某个值** 过滤日志的显式、可分享、可组合能力 — 不仅限于 service / level / trace_id 三个固定维度。

### 1.2 目标

1. 顶部提供 **通用搜索** 文本框，可对日志行任意子串过滤（与 trace_id 解耦或合并语义）。
2. 在 provisioning 中启用 **Ad hoc filters**（Loki label 动态过滤），使「Filter for value」与 **Filter by label values** 可深链、可持久。
3. **`trace_id` 非空时优先用 label matcher**（精确、可索引），保留行包含作为兜底（兼容无 label 历史行）。
4. 可选：增加 **`msg` 下拉**（Loki 已有 `msg` label，常见值如 `http_request`）。
5. 保持现有 URL 深链兼容（未传新变量时行为与改前一致）。

### 1.3 非目标

- 修改 Promtail pipeline 或新增高基数 label（如 `forward_stage` 本阶段不做）。
- 替换 Grafana Explore 全文检索体验。
- runAll 深链 API 默认携带 search/adhoc 参数。
- 统一各服务 `level` 大小写（INFO vs info）— 可文档说明，另案治理。

---

## 2. 价值流影响

关联 `value-stream.yaml` → **`platform-centralized-logging`**：

| 步骤 | 影响 |
|------|------|
| `increment2-grafana-trace-log-level-filter` | **扩展** — Trace Log Explore 增加 search + adhoc + trace_id label 匹配 |
| `increment2-grafana-trace-dashboard` | 无结构变更 |
| `increment2-json-log-contract` | 无变更 |
| `increment3-runall-grafana-deep-link` | 无变更（深链仍主要传 trace_id） |

**测试影响：**

- 扩展 `AiMonitor/scripts/test_trace_log_explore_dashboard.py`：断言 adhoc 变量、search 变量、LogQL 含 label/trace 逻辑。
- 现有 promtail / level 测试保持通过。

**建议 value-stream 登记（Step 3 可选）：**

| name | description |
|------|-------------|
| `ai-monitor.grafana.trace_log_explore_search` | Trace Log Explore 通用行搜索 + adhoc label 过滤 |

---

## 3. 领域概念清单（轻量）

| 概念 | 说明 |
|------|------|
| **Bounded Context: 平台可观测性** | AiMonitor Grafana 仪表盘 |
| **LogFilterVariable** | 固定模板变量（service / level / trace_id / search / msg） |
| **AdhocLogFilter** | 用户动态添加的 Loki label 键值对 |
| **LogQLSelector** | `{job="runall", service=~"$service", level=~"$level", trace_id=~"$trace_id_label"}` |
| **Domain Event** | 无（纯配置变更） |

---

## 4. 方案

### 4.1 推荐方案：search 变量 + adhoc 变量 + trace_id label 双模式

在 `AiMonitor/grafana/provisioning/dashboards/files/trace-log-explore.json` 中：

#### A. 新增 `search` 文本框

```json
{
  "label": "search",
  "name": "search",
  "type": "textbox",
  "description": "按任意子串过滤日志行（msg、路径、错误文本等）；留空不限制"
}
```

#### B. 新增 adhoc 模板变量

```json
{
  "name": "Filters",
  "type": "adhoc",
  "datasource": { "type": "loki", "uid": "loki" },
  "label": "Filters",
  "hide": 0
}
```

Grafana 会将 adhoc 过滤器自动注入 Loki 查询；与日志详情 **Filter for value** 联动。

#### C. 更新 LogQL

```
{job="runall", service=~"$service", level=~"$level", trace_id=~"$trace_id_regex"} 
  |= "$search" 
  | json 
  | __error__=""
```

**`trace_id` 变量处理（Grafana 变量链或 expr 内联）：**

- 用户填写 `trace_id` 时：`trace_id_regex = $trace_id`（label 精确匹配，Promtail 已写入 label）。
- 留空时：`trace_id_regex = .*`（不限制）。
- 若需兼容 **无 trace_id label 的历史行**：保留 `|= "$trace_id"` 作为第二段（仅当 trace_id 非空时生效）；推荐用 Grafana **custom allValue** 或 chained variable 实现，避免空字符串误匹配。

**简化实现（推荐 MVP）：**

```
{job="runall", service=~"$service", level=~"$level"} 
  |= "$trace_id" 
  |= "$search" 
  | json 
  | __error__=""
```

- `search` 承担「按某个值过滤」的通用入口。
- `trace_id` 保持现有语义；adhoc 负责 label 级动态过滤（含 `msg`、`detected_level` 等）。
- 后续增量再将 `trace_id` 升级为 label matcher。

#### D. 可选：`msg` 下拉

```json
{
  "name": "msg",
  "type": "query",
  "query": "label_values(msg)",
  "includeAll": true,
  "allValue": ".*",
  "label": "msg"
}
```

LogQL 增加 `msg=~"$msg"`（`msg` 已是 Loki label）。

#### E. 面板标题

更新为：`RunAll logs (service + level + trace_id + search + filters)`

### 4.2 备选方案

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| 仅文档教用户点「Filter for value」 | 零代码 | 不可深链、不可发现、无数据时无法用 | 不采用 |
| 把 trace_id 改名为 search | 少一个变量 | 破坏 runAll 深链 `var-trace_id` 契约 | 不采用 |
| Promtail 提升 `forward_stage` 为 label | label 过滤快 | 改动面大、需评估基数 | 后续增量 |
| 跳转 Grafana Explore 替代 | Explore 检索更强 | 脱离 curated 仪表盘、丢失变量组合 | 不采用 |

### 4.3 行为说明

| 场景 | 预期 |
|------|------|
| search 留空 | 与改前一致（除 adhoc 外） |
| search = `502` | 仅显示行内包含 `502` 的日志 |
| search + level=error | 先 label 过滤 level，再子串匹配 |
| 点击字段 Filter for value | 顶部 Filters 出现 chip，面板即时过滤；URL 可带 adhoc 参数 |
| Filter by label values 下拉 | 选手动添加 label 过滤（如 `msg=http_request`） |
| level=error 且无 error 日志 | 仍显示 No data（预期，非 bug） |
| 旧 URL 无 var-search | 默认空，兼容 |

---

## 5. 实现清单（供 Step 6 计划引用）

1. 修改 `trace-log-explore.json`：新增 `search`、adhoc `Filters`、（可选）`msg`；更新 expr 与 title；version bump。
2. 扩展 `test_trace_log_explore_dashboard.py`：断言变量与 LogQL 片段。
3. 本地验证：search 过滤子串；adhoc 过滤 `msg`；与 level/service 组合；深链 URL 可复现。
4. （可选）README / 仪表盘 description 说明 level 大小写与 No data 含义。

**不涉及：** runAll Go、Promtail YAML（本阶段）。

---

## 6. 风险与缓解

| 风险 | 缓解 |
|------|------|
| adhoc 与固定变量重复过滤 | 文档说明优先级：label adhoc AND 固定变量 AND search |
| `msg` label 基数 | 仅常见枚举（http_request 等）；监控 label 数量 |
| search 空字符串 `|=` 行为 | 留空时不追加管道段，或 Grafana 变量 default `""` + 查询模板条件 |
| level 大小写不一致（INFO/info） | level 下拉展示全部真实值；长期统一 JSON 契约 |

---

## 7. 验收标准

- [x] 顶部可见 **search** 输入框，输入任意子串后面板过滤
- [x] 顶部可见 **Filters** adhoc 区域，可通过下拉或日志详情 Filter for value 添加 label 过滤
- [x] 现有 service / level / trace_id 行为不变（含 runAll 深链 `var-trace_id`）
- [x] URL 可分享带 `var-search` 与 adhoc 参数的过滤状态
- [x] 自动化 dashboard JSON 测试通过
