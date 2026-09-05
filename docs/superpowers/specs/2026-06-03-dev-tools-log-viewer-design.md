# dev-tools-log-viewer — 开发工具日志查看器

**日期：** 2026-06-03
**状态：** approved
**范围：** runAll 管理页面 (http://localhost:9999/) 的开发工具日志查看功能

## 1. 需求概述

在开发工具栏（dev-tools-bar）新增「日志查看」按钮，点击后复用现有日志面板，可切换查看五个开发工具的日志，并提供清空日志按钮。

### 1.1 覆盖的开发工具

| 工具 | 工具名常量 | 日志文件 |
|------|-----------|---------|
| 生成配置副本 | `conf-sync` | `logs/conf-sync.log` |
| 同步配置副本 | `conf-sync-remote` | `logs/conf-sync-remote.log` |
| 清空全部数据库 | `db-clear` | `logs/db-clear.log` |
| 初始化全部数据库 | `db-init` | `logs/db-init.log` |
| 清空 Grafana 可观测数据 | `observability-clear` | `logs/observability-clear.log` |

### 1.2 关键设计决策

| 决策 | 选择 |
|------|------|
| 日志存储 | 文件持久化，每个工具独立文件，重启后可追溯 |
| 日志查看 UI | 复用现有右侧日志面板，新增下拉选择器切换工具 |
| 日志捕获机制 | Domain 层新增 `DevToolLogRecorder` 接口 + 文件实现 |
| 后端 API | 新增 `GET /api/dev/logs` 和 `POST /api/dev/logs/clear` |

## 2. 架构

```
┌─────────────────────────────────────────────────────────┐
│  status.html (前端)                                      │
│  ┌──────────────────────────────────────────────────┐   │
│  │ dev-tools-bar                                    │   │
│  │ [生成配置][同步配置][清空DB][初始化DB] [日志查看] │   │
│  └──────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────┐   │
│  │ logs-panel (复用)                                 │   │
│  │ [工具: ▼ conf-sync] [清空日志] [Copy] [Close]    │   │
│  │ ┌──────────────────────────────────────────────┐ │   │
│  │ │ 日志内容区（文件尾部 N 行，2s 自动刷新）       │ │   │
│  │ └──────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
         │  GET /api/dev/logs?tool=conf-sync&lines=200
         │  POST /api/dev/logs/clear  {tool: "conf-sync"}
         ▼
┌─────────────────────────────────────────────────────────┐
│  ui.go (新增 handler)                                    │
│  GET  /api/dev/logs       → 读文件尾 N 行                │
│  POST /api/dev/logs/clear → 清空指定工具日志文件          │
└─────────────────────────────────────────────────────────┘
         │  DevToolLogRecorder (domain 接口)
         ▼
┌─────────────────────────────────────────────────────────┐
│  infrastructure/dev_tool_log_recorder.go (新增)          │
│  Append(tool, message) → 追加时间戳行到文件              │
│  Tail(tool, lines)     → 返回文件尾 N 行                 │
│  Clear(tool)           → 清空文件                        │
│  ClearAll()            → 清空全部五个文件                 │
└─────────────────────────────────────────────────────────┘
         │  五个独立日志文件
         ▼
  logs/conf-sync.log
  logs/conf-sync-remote.log
  logs/db-clear.log
  logs/db-init.log
  logs/observability-clear.log
```

## 3. Domain 层

### 3.1 DevToolLogRecorder 接口

**新文件：** `runAll/src/domain/dev_tool_log_recorder.go`

```go
package domain

type DevToolLogRecorder interface {
    Append(tool string, message string) error
    Tail(tool string, lines int) ([]string, error)
    Clear(tool string) error
    ClearAll() error
}
```

### 3.2 工具名常量

```go
const (
    ToolConfSync           = "conf-sync"
    ToolConfSyncRemote     = "conf-sync-remote"
    ToolDbClear            = "db-clear"
    ToolDbInit             = "db-init"
    ToolObservabilityClear = "observability-clear"
)
```

### 3.3 合法工具名白名单

```go
var AllDevTools = []string{
    ToolConfSync,
    ToolConfSyncRemote,
    ToolDbClear,
    ToolDbInit,
    ToolObservabilityClear,
}
```

## 4. Infrastructure 层

### 4.1 文件日志记录器

**新文件：** `runAll/src/infrastructure/dev_tool_log_recorder.go`

基于 `os.File` 实现：
- 日志目录：`<monorepoRoot>/logs/`，不存在时自动 `os.MkdirAll`
- 文件名：`<tool>.log`
- `Append`：追加 `[2026-06-03 14:30:05] message\n`，使用 `os.O_APPEND|os.O_CREATE|os.O_WRONLY`
- `Tail`：打开文件 seek 到末尾，逐字节回退读取 N 行（最大读 64KB buffer）
- `Clear`：`os.OpenFile` + `os.O_TRUNC` 清空文件
- `ClearAll`：遍历五个文件名逐个 Truncate；文件不存在则跳过

### 4.2 Runner 集成

**修改 `runner.go`：**

在 `Runner` struct 新增字段：
```go
devToolLogRecorder domain.DevToolLogRecorder
```

初始化时创建实例：
```go
devToolLogRecorder := infrastructure.NewFileDevToolLogRecorder(monorepoRoot)
```

五个开发工具方法执行时写入日志，例如：
```go
func (r *Runner) ClearAllDatabases(ctx context.Context, configPath string) domain.DatabasePlatformClearResult {
    r.devToolLogRecorder.Append(domain.ToolDbClear, "开始清空全部数据库...")
    // ... 各步骤中写日志
    r.devToolLogRecorder.Append(domain.ToolDbClear, "清空数据库完成")
}
```

## 5. API

### 5.1 GET /api/dev/logs

**参数：**
- `tool`（query，必填）：工具名，需在白名单中
- `lines`（query，必填）：返回行数，1-2000

**成功响应：**
```json
{
  "tool": "db-clear",
  "lines": ["[2026-06-03 14:30:01] 开始清空数据库...", "..."]
}
```

**错误响应：**
- 400 `{"error": "tool is required"}`
- 400 `{"error": "unknown tool: xxx"}`
- 400 `{"error": "lines must be a positive integer"}`
- 500 `{"error": "failed to read logs"}`

### 5.2 POST /api/dev/logs/clear

**Body：**
```json
{"tool": "db-clear"}
```
或
```json
{"tool": "all"}
```

**成功响应：**
```json
{"status": "ok"}
```

**错误响应：**
- 400 `{"error": "tool is required"}`
- 400 `{"error": "unknown tool: xxx"}`（`"all"` 除外）

## 6. Frontend

### 6.1 开发工具栏变更

在 `#dev-tools-bar` 末尾新增按钮：
```html
<button id="dev-view-logs" class="logs-panel-btn dev-conf-generate-btn" type="button">
  日志查看
</button>
```

### 6.2 日志面板复用

点击「日志查看」→ 打开右侧面板，与现有服务日志面板共用 DOM 结构，但行为有差异：

**开发工具模式特征：**
- 面板标题：`开发工具日志`
- 面板 meta 行显示工具名 + 刷新时间
- 工具选择下拉框（使用 `<select>` 切换五个工具）
- 「清空日志」按钮（需 confirm 确认后调用 `POST /api/dev/logs/clear`）
- 「Copy」按钮（复用现有 `copyLogsToClipboard`）
- 「Close」按钮（复用现有 `closeLogsPanel`）
- Loki / Grafana 按钮隐藏
- 2 秒自动刷新（复用现有定时器机制）

### 6.3 JS 状态

```javascript
const devLogState = {
  open: false,
  tool: '',
  timerId: null,
  requestSerial: 0,
};

const devToolOptions = [
  { value: 'conf-sync', label: '生成配置副本' },
  { value: 'conf-sync-remote', label: '同步配置副本' },
  { value: 'db-clear', label: '清空全部数据库' },
  { value: 'db-init', label: '初始化全部数据库' },
  { value: 'observability-clear', label: '清空 Grafana 可观测数据' },
];
```

### 6.4 与现有服务日志面板共存

开发工具日志面板和服务日志面板互斥：
- 打开服务日志 → 关闭开发工具日志（如果打开）
- 打开开发工具日志 → 关闭服务日志（如果打开）
- 共享同一个 `#logs-panel` DOM，通过标志位区分当前模式

## 7. 数据流

```
1. 用户点击「日志查看」→ 面板打开（默认选中第一个工具 conf-sync）
2. 用户在下拉中选择「清空全部数据库」
3. 前端 GET /api/dev/logs?tool=db-clear&lines=200
   → 后端 Tail("db-clear", 200) → 返回已有日志
4. 用户点击 dev-tools-bar 中的「清空全部数据库（开发）」
   → POST /api/dev/clear-databases?confirm=CLEAR_ALL
   → 后端逐行写入 logs/db-clear.log
5. 前端每 2s 自动刷新 → 用户实时看到新日志追加
6. 用户点击「清空日志」→ confirm → POST /api/dev/logs/clear
   → 文件清空 → 刷新显示「暂无日志」
```

## 8. 错误处理

| 场景 | 处理方式 |
|------|---------|
| 日志文件不存在（首次查看） | `Tail` 返回空数组，前端显示「暂无日志」 |
| 工具名无效 | API 返回 400，前端 alert 错误信息 |
| 文件写入失败 | `Append` 返回 error，调用方 `log.Printf` 记录 |
| 日志目录不存在 | `Append` 自动 `os.MkdirAll` 创建 |
| 并发写入 | 文件 append mode，OS 级别保证原子性 |
| runner 为 nil | API 返回 400 `"runner is required"` |

## 9. 测试策略

| 层 | 文件 | 内容 |
|----|------|------|
| Domain 单元测试 | `domain/dev_tool_log_recorder_test.go` | 接口契约验证 |
| Infrastructure 集成测试 | `infrastructure/dev_tool_log_recorder_test.go` | 文件读写、Tail 边界、并发写入、目录自动创建 |
| API 测试 | `ui_test.go` | GET /api/dev/logs（正常/缺参/非法tool）、POST /api/dev/logs/clear（单个/all/非法tool） |
| UI 片段测试 | `ui_test.go` | status.html 包含下拉选择器、日志查看按钮、清空按钮、五个 tool 选项 |

## 10. Value Stream 影响

- **影响现有 stream：** `platform-dev-database-reset`（新增字段 `runall.runtime.dev_tool_log` 描述日志查看功能）
- **新增 stream：** 不需要 — 这是现有开发工具的 UI 增强，不引入新的业务价值流
- **字段变更：** 无数据库表字段变更；新增文件系统日志文件

## 11. 涉及文件清单

| 操作 | 文件 |
|------|------|
| 新增 | `runAll/src/domain/dev_tool_log_recorder.go` |
| 新增 | `runAll/src/infrastructure/dev_tool_log_recorder.go` |
| 修改 | `runAll/src/runner.go`（新增字段 + 操作中写日志） |
| 修改 | `runAll/src/ui.go`（新增两个 handler） |
| 修改 | `runAll/src/status.html`（前端 UI + JS） |
| 新增 | `runAll/src/domain/dev_tool_log_recorder_test.go` |
| 新增 | `runAll/src/infrastructure/dev_tool_log_recorder_test.go` |
| 修改 | `runAll/src/ui_test.go`（扩展测试） |

## 12. 文件行数估算

所有新增/修改文件均在 500 行限制内：
- `dev_tool_log_recorder.go`（domain）：~25 行
- `dev_tool_log_recorder.go`（infrastructure）：~120 行
- `ui.go` 新增：~80 行
- `status.html` 新增：~100 行 JS + ~20 行 HTML
- `runner.go` 新增：~40 行
