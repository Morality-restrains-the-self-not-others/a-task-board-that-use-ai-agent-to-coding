# Idle Reuse Observability — Grafana / Loki 看板查询

本文件记录 idle-reuse 相关 Loki 查询，可直接粘贴到 Grafana Explore。

行为准则见 [OPT-20260722-037](../../.learnings/OPTIMIZATION_TODOS.md)。

## 预置查询

### 1. Boot Guard 跳过 & 随后冷启动关联

```logql
{job="task-cloud-service"} |= "idle_reuse_skip_boot_guard"
```

若短窗口内同一 instance 出现 `idle_reuse_skip_boot_guard` 后紧跟 `RunInstances` 冷启动，说明 Fork+auto_run 误抢机。

### 2. Invoker Gate 跳过统计

```logql
{job="task-cloud-service"} | logfmt | event="idle_reuse_gate_summary"
```

按 reason 聚合：`no_target_invoker`（无 invoker）、`invoker_mismatch`（跨镜像调用）。

### 3. Orphan 误删告警（应为 0）

```logql
{job="task-cloud-service"} |= "orphan_reconcile_skip_owned"
```

对同 instance 先出现 `unbound_source` 再出现 `orphan_delete` 做禁止告警。

### 4. Idle Reuse 整体成功率

```logql
sum(rate({job="task-cloud-service"} |= "idle_reuse_skip" [5m])) 
/ 
sum(rate({job="task-cloud-service"} |= "idle_reuse" [5m]))
```

低于 0.3 时告警。

## 配置

将以上查询保存为 Grafana 收藏，或导入到 `conf/grafana/dashboards/idle-reuse.json`。
