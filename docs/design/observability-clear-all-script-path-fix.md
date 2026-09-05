# 🧠 头脑风暴：修复可观测性清空功能脚本路径错误

- **状态**: 🎯 设计完成，待审批
- **作者**: claude
- **日期**: 2026-07-01 22:10
- **类型**: Bug 修复
- **影响范围**: `runAll/src/` + `AiMonitor/scripts/`

---

## 1. 问题描述

用户在 runAll 管理面板 (`http://183.250.1.132:9999/`) 点击「清空 Grafana 可观测数据」按钮，触发 `POST /api/observability/clear-all`，返回 `status: "partial"`：

```json
{
    "files_truncated": 39,
    "loki_reset": "error: script not found: stat ../AiMonitor/scripts/reset_observability_storage.sh: no such file or directory",
    "memory_services_cleared": 39,
    "prometheus_reset": "skipped",
    "promtail_reset": "skipped",
    "status": "partial",
    "tempo_reset": "skipped"
}
```

**问题表现**:
- ✅ 内存日志清空成功（39 服务）
- ✅ tee 日志文件截断成功（39 文件）
- ❌ Loki 数据卷重置失败（脚本找不到）
- ❌ Prometheus 重置被跳过
- ❌ Promtail 重置被跳过
- ❌ Tempo 重置被跳过

---

## 2. 根因分析

### 2.1 脚本解析路径与 CWD 不匹配

**关键事实**:
- runAll 进程 CWD: `/tmp/ram-work` (repo 根目录)
- 脚本实际路径: `/tmp/ram-work/AiMonitor/scripts/reset_observability_storage.sh`
- `ResolveObservabilityResetScript()` 使用的候选路径:

```go
// runAll/src/infrastructure/observability_storage_reset.go:103-106
candidates := []string{
    "../AiMonitor/scripts/reset_observability_storage.sh",   // → /tmp/AiMonitor/... ❌
    "../../AiMonitor/scripts/reset_observability_storage.sh", // → /AiMonitor/...    ❌
}
```

从 CWD `/tmp/ram-work` 出发:
| 候选路径 | 解析结果 | 存在? |
|----------|---------|------|
| `../AiMonitor/scripts/...` | `/tmp/AiMonitor/scripts/...` | ❌ |
| `../../AiMonitor/scripts/...` | `/AiMonitor/scripts/...` | ❌ |
| **应使用**: `AiMonitor/scripts/...` | `/tmp/ram-work/AiMonitor/scripts/...` | ✅ |

**根因**: 候选路径假设 CWD 是 `runAll/` 子目录（此时 `../AiMonitor/` 正确），但实际部署中 go binary 从 repo 根目录启动，CWD 为 `/tmp/ram-work`。

### 2.2 configPath 空值传递

```go
// ui.go:359 — 传递空字符串 configPath
result := runner.ClearAllObservability(r.Context(), "")
```

`ResolveObservabilityResetScript("")` 收到空 configPath，无法基于配置文件位置扩展候选路径（`filepath.Dir("")` = `"."`，与已有候选路径等价）。

### 2.3 错误状态传播不完整

```go
// observability_storage_reset.go:41-43
if _, err := os.Stat(script); err != nil {
    outcome.LokiReset = fmt.Sprintf("error: script not found: %v", err)
    // PromtailReset, TempoReset, PrometheusReset 保持默认值 "skipped"
    return outcome, err
}
```

当脚本找不到时，只有 `LokiReset` 被设置为错误信息，其他三个字段保持 `"skipped"`，给用户造成「这些组件被故意跳过」的误导。

### 2.4 Shell 脚本内部路径错误（次要）

```bash
# AiMonitor/scripts/reset_observability_storage.sh:76
LOCAL_PROMTAIL_RESET="${ROOT}/../scripts/runall-local-promtail.sh"
# ROOT = AiMonitor/ → 解析为 scripts/runall-local-promtail.sh ❌
# 实际路径: runAll/scripts/runall-local-promtail.sh
```

此问题仅在 Mac + `local_promtail: true` 时触发，不影响当前 Linux 部署。

---

## 3. 设计方案

### 3.1 核心修复: 扩展脚本候选路径

**文件**: `runAll/src/infrastructure/observability_storage_reset.go`

修改 `ResolveObservabilityResetScript()` 函数，增加 repo-root-relative 候选路径和二进制相对路径回退:

```go
func ResolveObservabilityResetScript(configPath string) string {
    if env := strings.TrimSpace(os.Getenv("AIMONITOR_RESET_SCRIPT")); env != "" {
        return env
    }

    candidates := []string{
        // repo-root-relative (runAll binary launched from repo root)
        "AiMonitor/scripts/reset_observability_storage.sh",
        // legacy: runAll/ subdirectory CWD
        "../AiMonitor/scripts/reset_observability_storage.sh",
        "../../AiMonitor/scripts/reset_observability_storage.sh",
    }

    // Binary-relative fallback: resolve from runAll executable location
    if exe, err := os.Executable(); err == nil {
        exeDir := filepath.Dir(exe) // runAll/bin/
        // Try: runAll/bin/../../AiMonitor/scripts/ → repo root
        candidates = append(candidates,
            filepath.Clean(filepath.Join(exeDir, "../../AiMonitor/scripts/reset_observability_storage.sh")),
            filepath.Clean(filepath.Join(exeDir, "../AiMonitor/scripts/reset_observability_storage.sh")),
        )
    }

    // Config-file-relative candidates
    if configPath != "" {
        base := filepath.Dir(configPath)
        for _, rel := range []string{
            "../AiMonitor/scripts/reset_observability_storage.sh",
            "../../AiMonitor/scripts/reset_observability_storage.sh",
            "AiMonitor/scripts/reset_observability_storage.sh",
        } {
            candidates = append(candidates, filepath.Clean(filepath.Join(base, rel)))
        }
    }

    // dedup while preserving order
    seen := map[string]bool{}
    unique := make([]string, 0, len(candidates))
    for _, p := range candidates {
        cleaned := filepath.Clean(p)
        if !seen[cleaned] {
            seen[cleaned] = true
            unique = append(unique, cleaned)
        }
    }

    for _, path := range unique {
        if _, err := os.Stat(path); err == nil {
            return path
        }
    }

    // Fallback: return the first repo-root-relative path for a clear error message
    return filepath.Clean("AiMonitor/scripts/reset_observability_storage.sh")
}
```

### 3.2 传递 configPath 到 ClearAllObservability

**文件**: `runAll/src/ui.go`

```go
// 修改前:
result := runner.ClearAllObservability(r.Context(), "")

// 修改后: 使用 runner 持有的配置路径
result := runner.ClearAllObservability(r.Context(), r.cfgPath)
```

**文件**: `runAll/src/runner.go`

在 `Runner` 结构体中新增字段:
```go
type Runner struct {
    cfg     *Config
    cfgPath string  // NEW: stored config file path for relative resolution
    // ...
}
```

在 `NewRunner` 中存储 configPath。`ClearAllObservability` 传递该路径给 `ResolveObservabilityResetScript`。

### 3.3 完善错误状态传播

**文件**: `runAll/src/infrastructure/observability_storage_reset.go`

```go
func (r *ScriptObservabilityStorageResetter) Reset(ctx context.Context) (domain.StorageResetOutcome, error) {
    // ...
    if _, err := os.Stat(script); err != nil {
        errMsg := fmt.Sprintf("error: script not found: %v", err)
        outcome.LokiReset = errMsg
        outcome.PromtailReset = errMsg    // NEW
        outcome.TempoReset = errMsg       // NEW
        outcome.PrometheusReset = errMsg  // NEW
        return outcome, err
    }
    // ...
}
```

### 3.4 修复 Shell 脚本内部路径

**文件**: `AiMonitor/scripts/reset_observability_storage.sh`

```bash
# 修改前:
LOCAL_PROMTAIL_RESET="${ROOT}/../scripts/runall-local-promtail.sh"

# 修改后:
LOCAL_PROMTAIL_RESET="${ROOT}/../runAll/scripts/runall-local-promtail.sh"
```

---

## 4. 测试策略

### 4.1 单元测试

**文件**: `runAll/src/infrastructure/observability_storage_reset_test.go`

新增测试用例:
- `TestResolveObservabilityResetScript_RepoRoot` — 验证从 repo root CWD 可以找到脚本
- `TestResolveObservabilityResetScript_EnvOverride` — 验证环境变量优先
- `TestResolveObservabilityResetScript_BinaryRelative` — 验证 binary-relative 回退
- `TestScriptObservabilityStorageResetter_NotFound_AllStatuses` — 验证脚本未找到时所有状态字段都被设置

### 4.2 现有测试保持通过

`TestAPIObservabilityClearAll` 测试通过内存 runner（无真实脚本），不受影响。

### 4.3 集成验证

部署后，在 runAll 管理面板点击「清空 Grafana 可观测数据」，预期:
```json
{
    "loki_reset": "ok",
    "prometheus_reset": "ok",
    "promtail_reset": "ok",
    "tempo_reset": "ok",
    "status": "ok"
}
```

---

## 5. 影响分析

| 维度 | 影响 |
|------|------|
| **变更文件** | `runAll/src/infrastructure/observability_storage_reset.go` (核心修复) |
|  | `runAll/src/runner.go` (存储 configPath) |
|  | `runAll/src/ui.go` (传递 configPath) |
|  | `runAll/src/infrastructure/observability_storage_reset_test.go` (新增测试) |
|  | `AiMonitor/scripts/reset_observability_storage.sh` (路径修复) |
| **破坏性变更** | 无 — 纯修复，扩展候选路径 + 保持向后兼容 |
| **架构变更** | 无 — 纯 Bug 修复，不涉及架构 |
| **部署依赖** | 需要重新编译 `runAll/bin/runAll` |
| **回滚方案** | 设置 `AIMONITOR_RESET_SCRIPT` 环境变量指向正确脚本路径即可绕过 |

---

## 6. 领域概念清单

本次为 Bug 修复，不引入新的领域概念。涉及的现有上下文:

| 上下文 | 实体 |
|--------|------|
| **Observability** (可观测性) | `StorageResetOutcome` — 重置操作结果值对象 |
| **Infrastructure** (基础设施) | `ScriptObservabilityStorageResetter` — 脚本执行适配器 |
| **runAll Orchestrator** | `Runner` — 编排器聚合根 |

---

## 7. 价值流影响

**现有流**: `increment4-runall-clear-all-observability` (记录在 `conf/value-stream.yaml`)

本次修复不改变价值流步骤，仅修复实现缺陷。

---

## 🏛️ 架构变更影响

**无架构变更** — 纯 Bug 修复。不创建新版本架构文件。
