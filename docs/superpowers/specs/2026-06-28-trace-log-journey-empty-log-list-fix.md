# Trace Log Journey 日志列表为空 — 根因分析与修复设计

日期：2026-06-28  
状态：待批准  
Dashboard: `http://183.250.1.132:3000/d/trace-log-journey/trace-log-journey`

---

## 1. 问题现象

Grafana「Trace Log Journey」仪表盘日志面板（Logs panel）和日志量图表（Log volume chart）均为空。面板查询：

```logql
# Panel 1 (Log stream)
{job="runall"} |= "$trace_id"

# Panel 2 (Log volume)
sum by (service) (count_over_time({job="runall"} |= "$trace_id" [$__interval]))
```

Loki 查询 `{job="runall"}` 返回 0 条结果——没有任何带 `job=runall` label 的日志流入 Loki。

---

## 2. 根因分析

### 2.1 完整数据流

```
runAll 子进程 stdout/stderr
  → streamOutput() [runner.go]
    → TeeServiceLogRepository.Append()
      → FileServiceLogSink.AppendLine()
        → 写入 /tmp/logs/<service>.log   ← 实际日志目录
                                           ↓
Promtail 容器 (aimonitor-promtail-local)    ↓
  → tail /var/log/runall/*.log            ↓
    → 挂载自 /tmp/ram-work/logs/          ← 监控目录（错误！）
```

### 2.2 目录不匹配

| 组件 | 目录 | 文件数 | 大小 |
|------|------|--------|------|
| **runAll 实际写入** | `/tmp/logs/` | 38 个 .log | 4.1 MB |
| **Promtail 监控** | `/tmp/ram-work/logs/` | 1 个 (conf-sync.log) | 2.7 KB |

**Promtail 监控的目录与 runAll 写入的目录不同，导致所有服务日志均未被采集进入 Loki。**

### 2.3 不匹配的根源

`conf/runAll.yaml` 中配置 `logging.file_root: ../logs`，该相对路径在不同组件中被不同方式解析：

| 组件 | 解析逻辑 | 结果 |
|------|----------|------|
| **runAll Go 进程** | CWD=`/tmp/ram-work`, `../logs` → `/tmp/logs` | `/tmp/logs/` ✓ |
| **Promtail 启动脚本** | `${ROOT}/logs` (忽略 `..`) | `/tmp/ram-work/logs/` ✗ |

**脚本 `runAll/scripts/runall-local-promtail.sh` 中的 `resolve_runall_log_root()` 函数：**

```bash
resolve_runall_log_root() {
  # ...
  raw="$(python3 -c "... print((d.get('logging') or {}).get('file_root') or '')")"
  if [[ "$raw" = /* ]]; then
    echo "$raw"                        # 绝对路径 → 直接使用
  else
    echo "${ROOT}/logs"                # BUG: 丢弃了相对路径的上级目录部分
  fi
}
```

脚本在 `else` 分支直接输出 `${ROOT}/logs`，完全忽略了 config 中的 `../` 前缀。config 中的 `../logs` 和 `logs` 在脚本中都被解析为同一路径。

### 2.4 验证证据

```
# runAll 进程打开的文件描述符（实际写入位置）：
/proc/2284374/fd/6  → /tmp/logs/docker-kafka.log
/proc/2284374/fd/10 → /tmp/logs/git-service.log
/proc/2284374/fd/15 → /tmp/logs/saas-backend.log
/proc/2284374/fd/41 → /tmp/logs/go-relay.log

# Promtail 容器内看到的文件：
$ docker exec aimonitor-promtail-local ls /var/log/runall/
conf-sync.log                          # ← 只有这一个文件！

# Promtail 挂载信息：
Source: /tmp/ram-work/logs → Destination: /var/log/runall

# 实际日志目录（Promtail 看不到）：
$ ls /tmp/logs/*.log | wc -l
38
$ du -sh /tmp/logs/
4.1M

# 日志中确实包含 trace_id（格式正确，可被 Promtail pipeline 解析）：
$ tail -1 /tmp/logs/saas-backend.log
{"ts": "...", "level": "info", "service": "saas-backend", "trace_id": "xHyAGrQVHLg_TnOEDeanVqyKHNpl3QZk", ...}
```

### 2.5 附加发现

- `/tmp/runall-logs/` 目录包含 38 个 **0 字节** 的空文件，创建于 `Jun 28 00:11`（早于 runAll 启动时间 `00:15`），属于历史残留
- `/tmp/ram-work/logs/conf-sync.log` 是唯一被 Promtail 采集的文件，但其内容不包含 `trace_id` 字段（是配置同步日志）

---

## 3. 修复方案

### 方案 A：统一使用项目根目录下的 `logs/`（推荐）

**修改 `conf/runAll.yaml`：**
```yaml
logging:
  file_root: logs   # 改为相对于 CWD（项目根 /tmp/ram-work）的路径
```

**影响分析：**
- runAll CWD 为 `/tmp/ram-work`，`logs` → `/tmp/ram-work/logs/`
- 脚本 `${ROOT}/logs` = `/tmp/ram-work/logs/`
- 两者一致，无需修改脚本

**风险：**
- 日志目录从 `/tmp/logs/` 变为 `/tmp/ram-work/logs/`，需重启 runAll 生效
- 旧的 `/tmp/logs/` 日志不会被自动迁移（可手动清理）

### 方案 B：修复脚本的相对路径解析

**修改 `runAll/scripts/runall-local-promtail.sh`：**
```bash
resolve_runall_log_root() {
  # ...
  if [[ "$raw" = /* ]]; then
    echo "$raw"
  else
    # 正确解析相对于 runAll 工作目录的相对路径
    # runAll CWD 为 ${ROOT}（项目根），config 中 ../logs → ${ROOT}/../logs
    echo "$(cd "${ROOT}" && realpath "$raw")"
  fi
}
```

**风险：**
- 依赖 `realpath` 命令（需确认 CI 环境可用）
- 只修复了脚本端，没有解决配置本身语义模糊的问题

### 方案 C：使用绝对路径

**修改 `conf/runAll.yaml`：**
```yaml
logging:
  file_root: /tmp/ram-work/logs
```

**风险：**
- 硬编码绝对路径，不便于在不同机器/CI 之间移植

### 推荐

**方案 A**（改配置）+ **方案 B**（加固脚本）：

1. 将 `conf/runAll.yaml` 的 `file_root` 从 `../logs` 改为 `logs`
2. 加固脚本的相对路径解析作为防御性措施（防止类似问题再次出现）

---

## 4. 价值流影响

| 现有流 | 影响 |
|--------|------|
| `platform-centralized-logging` | 修复：日志采集路径对齐后，Grafana 可正常查询 |
| 所有依赖 trace_id 检索的调试流程 | 恢复：trace-log-journey / trace-log-explore 仪表盘可用 |

**字段影响：** 无 schema 变更。仅配置文件路径修改。

**测试影响：**
- `AiMonitor/scripts/test_trace_log_explore_dashboard.py` — 验证修复后仪表盘可查询到日志
- `AiMonitor/scripts/verify_trace_stack.py` — 端到端验证 trace stack

---

## 5. 领域概念清单

本次为纯 bug 修复（配置路径错误），不涉及新领域概念。

| 类型 | 相关项 |
|------|--------|
| **Bounded Context** | `platform-observability`（日志采集配置） |
| **配置项** | `runAll.Logging.FileRoot` — 服务日志 tee 输出根目录 |

---

## 6. 实施步骤

1. **修改 `conf/runAll.yaml`**：`file_root: ../logs` → `file_root: logs`
2. **（可选）加固 `runAll/scripts/runall-local-promtail.sh`**：修复相对路径解析
3. **重启 runAll**：使新的 file_root 生效
4. **重启 Promtail**：`bash runAll/scripts/runall-local-promtail.sh down && bash runAll/scripts/runall-local-promtail.sh up`
5. **验证**：在 Grafana 中打开 trace-log-journey，确认日志列表有数据
6. **清理**：移除 `/tmp/runall-logs/` 残留目录和 `/tmp/logs/` 旧日志

---

## 7. 结论

**根因：** `conf/runAll.yaml` 中的 `logging.file_root: ../logs` 被 runAll（Go）正确解析为 `/tmp/logs/`，但被 Promtail 启动脚本错误解析为 `/tmp/ram-work/logs/`。Promtail 监控了错误的目录，导致所有服务日志均未被采集到 Loki，Grafana 仪表盘显示为空。

**修复：** 将 `file_root` 改为不含 `..` 的相对路径 `logs`，使两个组件解析到同一目录 `/tmp/ram-work/logs/`。
