# 实施计划: Trace Log Explore level 过滤

> 设计: `docs/superpowers/specs/2026-05-30-trace-log-explore-level-filter-design.md`

**Tech Stack:** Grafana dashboard JSON, Python pytest

---

## Task 1: Dashboard JSON — level 变量 + LogQL

**Files:**
- Modify: `AiMonitor/grafana/provisioning/dashboards/files/trace-log-explore.json`

**Steps:**
1. 新增 `level` templating 变量（mirror `service`：`label_values(level)`、`includeAll`、`allValue: ".*"`）
2. 更新 expr：`{job="runall", service=~"$service", level=~"$level"} |= "$trace_id" | json | __error__=""`
3. 更新 panel title，version → 14

---

## Task 2: Dashboard 结构测试

**Files:**
- Create: `AiMonitor/scripts/test_trace_log_explore_dashboard.py`

**Steps:**
1. 断言 templating 含 `name: level`
2. 断言 expr 含 `level=~"$level"`
3. 运行: `cd AiMonitor && python -m pytest scripts/test_trace_log_explore_dashboard.py -v`

---

## Task 3: Promtail 测试加强（可选）

**Files:**
- Modify: `AiMonitor/scripts/test_promtail_config.py`

**Steps:**
1. 断言 labels stage 含 `level`

---

## Verification

```bash
cd AiMonitor && python -m pytest scripts/test_trace_log_explore_dashboard.py scripts/test_promtail_config.py -v
```
