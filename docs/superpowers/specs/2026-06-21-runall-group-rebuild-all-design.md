# runAll 分组「全部重新编译」按钮 — 设计文档

Date: 2026-06-21
Status: draft

## 问题陈述

当前 runAll Web UI 中，每个分组 (group) 的 header 已有「启动本组」和「关闭本组」两个批量操作按钮，但缺少批量「重新编译」按钮。当用户修改了多个服务的代码后，需要手动逐个点击每个服务的「编译」按钮，操作繁琐。

用户需求：在每个分组的 header 中增加一个「全部重新编译」按钮，点击后将该组内所有**可编译** (buildable) 的服务依次重新编译。

## 现状分析

### 已有能力

| 能力 | 端点 | 说明 |
|------|------|------|
| 单服务编译 | `POST /api/build` `{"name":"..."}` | 对单个 buildable 服务执行编译 |
| 分组启动 | `POST /api/start-group` `{"group":"..."}` | 启动组内所有可启动服务 |
| 分组关闭 | `POST /api/stop-group` `{"group":"..."}` | 关闭组内所有运行中服务 |
| 服务 buildable 判定 | `/api/status` 返回 `buildable: bool` | 基于 `resolveBuildCommand()` 推断 |

### 当前分组 header 渲染 (JavaScript)

```javascript
const groupActions = groupKey !== 'ungrouped'
  ? `<span class="group-actions">`
    + `<button ... data-action="start-group" ...>启动本组</button>`
    + `<button ... data-action="stop-group" ...>关闭本组</button>`
    + `</span>`
  : '';
```

### 单服务编译按钮渲染

```javascript
if (buildable) {
  html += `<button class="action-btn build-btn" ... data-action="build" ...>编译</button>`;
} else {
  html += `<button class="action-btn build-btn is-disabled" ... disabled>编译</button>`;
}
```

### Runner.BuildService 现状

```go
func (r *Runner) BuildService(ctx context.Context, name string) error {
    svc := r.findService(name)
    buildCmd := resolveBuildCommand(*svc)
    // 要求服务状态为 healthy / failed / stopped 才可编译
    // CAS 状态到 StatusBuilding → runBuild → 恢复原状态
}
```

关键约束：`BuildService` 只允许对 `healthy`、`failed`、`stopped` 三种状态的服务执行编译，`starting`/`retrying`/`building` 中的服务会被拒绝。

## 方案

### 后端：新增 `POST /api/build-group` 端点

#### 路由注册 (`ui.go`)

```go
mux.HandleFunc("/api/build-group", func(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
        return
    }
    var body struct {
        Group string `json:"group"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        writeJSONError(w, "invalid json")
        return
    }
    body.Group = strings.TrimSpace(body.Group)
    if body.Group == "" {
        writeJSONError(w, "group is required")
        return
    }
    if runner == nil {
        writeJSONError(w, "runner is required")
        return
    }
    log.Printf("[api] build-group request for %s", body.Group)
    result, err := runner.BuildGroup(r.Context(), body.Group)
    if err != nil {
        writeJSONError(w, err.Error())
        return
    }
    writeJSON(w, result)
})
```

#### Runner.BuildGroup (`runner.go`)

```go
type BuildGroupResult struct {
    Status   string   `json:"status"`   // "ok" | "partial" | "none"
    Total    int      `json:"total"`    // 可编译服务总数
    Built    int      `json:"built"`     // 成功编译数
    Failed   []string `json:"failed,omitempty"`   // 失败的服务名
    Skipped  []string `json:"skipped,omitempty"`  // 状态不允许编译的服务名
    NoBuild  []string `json:"no_build,omitempty"` // 不可编译的服务名（含原因）
}

func (r *Runner) BuildGroup(ctx context.Context, groupName string) (*BuildGroupResult, error) {
    // 1. 查找该组所有服务
    svcs := r.findServicesByGroup(groupName)
    if len(svcs) == 0 {
        return nil, fmt.Errorf("group %q not found", groupName)
    }

    result := &BuildGroupResult{Status: "none"}

    // 2. 按依赖深度排序（浅依赖先编译）
    ordered := r.orderByDependencyDepth(svcs)

    for _, svc := range ordered {
        buildCmd := resolveBuildCommand(*svc)
        if buildCmd == "" {
            result.NoBuild = append(result.NoBuild, svc.Name)
            continue
        }
        result.Total++

        current := r.store.Get(svc.Name)
        if current == nil {
            result.Skipped = append(result.Skipped, svc.Name+" (not found in store)")
            continue
        }
        // 仅允许 healthy / failed / stopped 状态编译
        if current.Status != StatusHealthy &&
           current.Status != StatusFailed &&
           current.Status != StatusStopped {
            result.Skipped = append(result.Skipped, svc.Name)
            continue
        }

        if err := r.BuildService(ctx, svc.Name); err != nil {
            log.Printf("[build-group] %s build failed: %v", svc.Name, err)
            result.Failed = append(result.Failed, svc.Name)
        } else {
            result.Built++
        }
    }

    // 3. 计算结果状态
    if result.Built > 0 && len(result.Failed) == 0 {
        result.Status = "ok"
    } else if result.Built > 0 {
        result.Status = "partial"
    }
    // result.Status == "none" 保持不变

    return result, nil
}
```

#### 辅助方法

```go
func (r *Runner) findServicesByGroup(groupName string) []*Service {
    for gi := range r.cfg.Groups {
        if r.cfg.Groups[gi].Name == groupName {
            svcs := make([]*Service, len(r.cfg.Groups[gi].Services))
            for si := range r.cfg.Groups[gi].Services {
                svcs[si] = &r.cfg.Groups[gi].Services[si]
            }
            return svcs
        }
    }
    return nil
}

func (r *Runner) orderByDependencyDepth(svcs []*Service) []*Service {
    // 按 depends_on 链深度升序排列：无依赖的先编译，有依赖的后编译
    // 使用已有 DAG 能力或简单排序
    ...
}
```

### 前端：分组 header 增加「全部重新编译」按钮

#### JavaScript 渲染 (`status.html`)

修改 `renderStatus` 中 groupActions 的生成逻辑：

```javascript
// 判断该组是否有可编译的服务
const hasBuildable = services.some(svc => svc.buildable === true);

const groupActions = groupKey !== 'ungrouped'
  ? `<span class="group-actions">`
    + `<button class="action-btn" type="button" data-action="start-group" data-group="${esc(groupKey)}">启动本组</button>`
    + `<button class="action-btn" type="button" data-action="stop-group" data-group="${esc(groupKey)}">关闭本组</button>`
    + (hasBuildable
        ? `<button class="action-btn build-btn rebuild-all-btn" type="button" data-action="build-group" data-group="${esc(groupKey)}">全部重新编译</button>`
        : `<button class="action-btn build-btn is-disabled" type="button" disabled aria-disabled="true" title="本组无可编译服务">全部重新编译</button>`)
    + `</span>`
  : '';
```

#### 按钮样式

复用现有 `.build-btn` 样式类，可选增加专属微调：

```css
.rebuild-all-btn {
  border-color: #2563eb;
  color: #bfdbfe;
}
.rebuild-all-btn:active,
.rebuild-all-btn.is-clicked {
  background: #1e3a8a;
  border-color: #3b82f6;
  color: #fff;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.75), 0 0 14px rgba(59, 130, 246, 0.55);
}
```

#### 事件处理

在全局 click 委托中增加 `build-group` action：

```javascript
// 在现有事件委托中增加:
if (action === 'build-group') {
  const group = btn.dataset.group;
  await buildGroup(group);
  return;
}
```

新增 `buildGroup` 函数：

```javascript
async function buildGroup(group) {
  try {
    const resp = await fetch('/api/build-group', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({group: group})
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      alert('全部重新编译失败: ' + (result.error || `${resp.status}`));
      return;
    }
    // 汇总结果
    let msg = `编译完成：${result.built}/${result.total} 成功`;
    if (result.failed && result.failed.length > 0) {
      msg += `\n失败: ${result.failed.join(', ')}`;
    }
    if (result.skipped && result.skipped.length > 0) {
      msg += `\n跳过: ${result.skipped.join(', ')}`;
    }
    if (result.no_build && result.no_build.length > 0) {
      msg += `\n不可编译: ${result.no_build.join(', ')}`;
    }
    alert(msg);
    refresh();
  } catch (err) {
    alert('全部重新编译失败: ' + err.message);
  }
}
```

### 编译顺序

组内编译按**依赖深度升序**执行：无依赖的服务先编译，有依赖的后编译。这样确保：
- 基础库/公共模块先于上层服务编译
- 编译失败不会阻塞后续无依赖服务的编译

### 错误处理策略

- **继续策略**：一个服务编译失败不阻塞后续服务的编译（best-effort）
- **状态保护**：只编译 `healthy`/`failed`/`stopped` 状态的服务；`building`/`starting`/`retrying` 中的服务跳过
- **结果汇总**：API 返回详细的分项结果（succeeded / failed / skipped / no_build），UI 以 alert 呈现

## 涉及的变更文件

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `runAll/src/ui.go` | 修改 | 新增 `/api/build-group` 路由；可能需要新增 `findServicesByGroup` 辅助 |
| `runAll/src/runner.go` | 修改 | 新增 `BuildGroup`、`findServicesByGroup`、`orderByDependencyDepth` 方法 |
| `runAll/src/status.html` | 修改 | 分组 header 增加「全部重新编译」按钮 + JS 逻辑 |
| `runAll/src/ui_test.go` | 修改 | 新增 `/api/build-group` 端点的测试 |
| `runAll/src/runner_test.go` | 修改 | 新增 `BuildGroup` 方法的测试 |

## 价值流影响

查看 `value-stream.yaml`，此变更影响以下已有流：

- **runAll Web UI 依赖链一键启停** (line ~1198)：该流涵盖了分组按钮交互，新增的「全部重新编译」按钮属于该流的自然扩展
- **runAll 编译按钮** (line ~181)：涉及单服务编译，分组编译是对该能力的批量化

不需要新建价值流，此需求是对现有「分组操作」和「服务编译」两条流的组合增强。

## 领域概念清单

| 概念 | 类型 | 说明 |
|------|------|------|
| Service (managed_service) | 聚合根 | 被管理的服务，已有 `buildable` 判定逻辑 |
| Group | 值对象/分组 | 服务的逻辑分组，已有 group header 渲染 |
| BuildCommand | 值对象 | `resolveBuildCommand()` 推断出的编译命令 |
| BuildGroupResult | 值对象 | 分组编译的结果摘要 |
| ServiceLifecycle | 领域服务 | 管理服务状态转换，`BuildService` 是其一部分 |

## 设计决策

1. **组内编译顺序**：按依赖深度升序（浅依赖先编译），而非按名称排序或并行编译
   - 理由：确保基础库先于上层服务编译完成；并行编译可能造成资源竞争（多个 `go build` 同时写 `$GOPATH/pkg`）
2. **继续策略**：单个失败不阻塞整体
   - 理由：用户目标是尽可能多地编译成功；单个失败不应阻止其他独立服务的编译
3. **不在 header 中隐藏按钮**：即使组内没有 buildable 服务，也显示 disabled 按钮
   - 理由：保持 UI 一致性；tooltip 解释原因（"本组无可编译服务"）
4. **复用现有 `BuildService`**：`BuildGroup` 内部调用 `BuildService`，而非重写编译逻辑
   - 理由：DRY；`BuildService` 已有完善的状态 CAS + 日志 + 错误处理
