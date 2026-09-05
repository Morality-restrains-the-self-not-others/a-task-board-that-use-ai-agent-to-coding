# 开发工具统一日志查看 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 runAll Web UI 的开发工具栏新增「日志查看」按钮，可切换查看五个开发工具的操作日志并清空。

**Architecture:** Domain 层定义 `DevToolLogRecorder` 接口（DDD 步骤已完成），Infrastructure 层实现文件日志记录器（已完成）。本计划完成：API 端点、Runner 集成、前端 UI、测试。

**Tech Stack:** Go (net/http), vanilla JS/CSS (status.html embed), 无新增依赖

**Prerequisites:** Domain + Infrastructure 代码已生成并通过测试

---

### Task 1: 在 ui.go 新增 API 端点

**Files:**
- Modify: `runAll/src/ui.go`（新增 handler 注册 + handler 函数）

- [ ] **Step 1: 在 registerUIHandlers 中注册两个新路由**

在 `registerUIHandlers` 函数末尾（`mux.HandleFunc("/", ...)` 之前），添加：

```go
mux.HandleFunc("/api/dev/logs", func(w http.ResponseWriter, r *http.Request) {
    handleDevToolLogs(w, r, runner)
})
mux.HandleFunc("/api/dev/logs/clear", func(w http.ResponseWriter, r *http.Request) {
    handleDevToolLogsClear(w, r, runner)
})
```

- [ ] **Step 2: 实现 handleDevToolLogs（GET /api/dev/logs）**

在 `ui.go` 文件末尾（`startUIServer` 函数之前）添加：

```go
func handleDevToolLogs(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodGet {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runner == nil || runner.devToolLogRecorder == nil {
		writeJSONError(w, "dev tool log recorder is required")
		return
	}

	tool := strings.TrimSpace(r.URL.Query().Get("tool"))
	if tool == "" {
		writeJSONError(w, "tool is required")
		return
	}
	if !domain.IsValidDevTool(tool) {
		writeJSONError(w, fmt.Sprintf("unknown tool: %q", tool))
		return
	}

	linesRaw := r.URL.Query().Get("lines")
	if linesRaw == "" {
		writeJSONError(w, "lines is required")
		return
	}
	lines, err := strconv.Atoi(linesRaw)
	if err != nil || lines <= 0 {
		writeJSONError(w, "lines must be a positive integer")
		return
	}
	if lines > maxLogsLines {
		writeJSONError(w, fmt.Sprintf("lines must be <= %d", maxLogsLines))
		return
	}

	logLines, err := runner.devToolLogRecorder.Tail(tool, lines)
	if err != nil {
		writeJSONErrorWithStatus(w, http.StatusInternalServerError, fmt.Sprintf("failed to read logs: %v", err))
		return
	}

	writeJSON(w, map[string]any{
		"tool":  tool,
		"lines": logLines,
	})
}
```

- [ ] **Step 3: 实现 handleDevToolLogsClear（POST /api/dev/logs/clear）**

在 `handleDevToolLogs` 之后添加：

```go
func handleDevToolLogsClear(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runner == nil || runner.devToolLogRecorder == nil {
		writeJSONError(w, "dev tool log recorder is required")
		return
	}

	var body struct {
		Tool string `json:"tool"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, "invalid json")
		return
	}
	body.Tool = strings.TrimSpace(body.Tool)
	if body.Tool == "" {
		writeJSONError(w, "tool is required")
		return
	}

	var err error
	if body.Tool == "all" {
		err = runner.devToolLogRecorder.ClearAll()
	} else {
		if !domain.IsValidDevTool(body.Tool) {
			writeJSONError(w, fmt.Sprintf("unknown tool: %q", body.Tool))
			return
		}
		err = runner.devToolLogRecorder.Clear(body.Tool)
	}
	if err != nil {
		writeJSONErrorWithStatus(w, http.StatusInternalServerError, fmt.Sprintf("failed to clear logs: %v", err))
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}
```

- [ ] **Step 4: 编译验证**

Run: `cd runAll/src && go build ./...`
Expected: 编译成功，无错误

---

### Task 2: 在 Runner 中集成 DevToolLogRecorder

**Files:**
- Modify: `runAll/src/runner.go`

- [ ] **Step 1: 在 Runner struct 新增字段**

找到 `Runner` struct 定义（约 line 25-35），在 `fileLogSink` 之后添加：

```go
devToolLogRecorder domain.DevToolLogRecorder
```

- [ ] **Step 2: 在 NewRunner 中初始化 recorder**

找到 `NewRunner` 函数（约 line 176-205），在 `logRepository` 赋值之后、return 之前添加：

```go
devToolLogRecorder := infrastructure.NewFileDevToolLogRecorder(monorepoRoot)
```

并在 return 的 Runner literal 中添加字段：

```go
devToolLogRecorder: devToolLogRecorder,
```

- [ ] **Step 3: 在 ClearAllObservability 中添加日志**

在 `ClearAllObservability` 函数体中（约 line 1452），`svc` 创建之前添加：

```go
r.devToolLogRecorder.Append(domain.ToolObservabilityClear, "开始清空 Grafana 可观测数据...")
```

在 `svc.ClearAll` 调用返回后：

```go
result := svc.ClearAll(ctx, r.configuredServiceNames())
if result.Status == "ok" {
    r.devToolLogRecorder.Append(domain.ToolObservabilityClear, fmt.Sprintf("清空完成: memory=%d files=%d loki=%s tempo=%s prometheus=%s",
        result.MemoryServicesCleared, result.FilesTruncated, result.LokiReset, result.TempoReset, result.PrometheusReset))
} else {
    r.devToolLogRecorder.Append(domain.ToolObservabilityClear, fmt.Sprintf("清空部分失败: status=%s", result.Status))
}
return result
```

- [ ] **Step 4: 在 ClearAllDatabases 中添加日志**

在 `ClearAllDatabases` 函数体中（约 line 1608），分步添加：

```go
r.devToolLogRecorder.Append(domain.ToolDbClear, "开始清空全部数据库...")
// 在 svc.Clear 调用前后：
r.devToolLogRecorder.Append(domain.ToolDbClear, "正在停止运行中的服务...")
result := svc.Clear(ctx)
r.devToolLogRecorder.Append(domain.ToolDbClear, fmt.Sprintf("停止服务完成: stopped=%v still_running=%v", result.ServicesStopped, result.ServicesStillRunning))
r.devToolLogRecorder.Append(domain.ToolDbClear, fmt.Sprintf("SQLite 已删除: %v", result.SQLiteRemoved))
r.devToolLogRecorder.Append(domain.ToolDbClear, fmt.Sprintf("Redis 重置: %s", result.RedisReset))
r.devToolLogRecorder.Append(domain.ToolDbClear, fmt.Sprintf("Kafka 重置: %s", result.KafkaReset))
r.devToolLogRecorder.Append(domain.ToolDbClear, fmt.Sprintf("清空数据库完成: status=%s", result.Status))
```

- [ ] **Step 5: 在 InitAllDatabases 中添加日志**

在 `InitAllDatabases` 函数体中（约 line 1630）：

```go
r.devToolLogRecorder.Append(domain.ToolDbInit, "开始初始化全部数据库...")
result := svc.Init(ctx)
for _, m := range result.Migrations {
    r.devToolLogRecorder.Append(domain.ToolDbInit, fmt.Sprintf("migrate %s: %s%s", m.Database, m.Status, m.Message))
}
for _, i := range result.Inits {
    r.devToolLogRecorder.Append(domain.ToolDbInit, fmt.Sprintf("init %s: %s%s", i.Database, i.Status, i.Message))
}
r.devToolLogRecorder.Append(domain.ToolDbInit, fmt.Sprintf("初始化数据库完成: status=%s", result.Status))
```

- [ ] **Step 6: 在 SyncMonorepoConf 中添加日志**

找到 `SyncMonorepoConf` 方法，在方法体中添加：

```go
r.devToolLogRecorder.Append(domain.ToolConfSync, "开始生成配置副本...")
// 方法执行后
r.devToolLogRecorder.Append(domain.ToolConfSync, "配置副本生成完成")
```

- [ ] **Step 7: 在 SyncConfReplicaToRemote 中添加日志**

找到 `SyncConfReplicaToRemote` 方法，在方法体中添加：

```go
r.devToolLogRecorder.Append(domain.ToolConfSyncRemote, "开始同步配置副本到远程...")
// 方法执行后
r.devToolLogRecorder.Append(domain.ToolConfSyncRemote, "配置副本远程同步完成")
```

- [ ] **Step 8: 编译验证**

Run: `cd runAll/src && go build ./...`
Expected: 编译成功，无错误

---

### Task 3: 前端 — 日志查看按钮 + 面板复用

**Files:**
- Modify: `runAll/src/status.html`

- [ ] **Step 1: 在 dev-tools-bar 中添加「日志查看」按钮**

在 `#dev-tools-bar` 末尾、「初始化全部数据库」按钮之后添加：

```html
<button id="dev-view-logs" class="logs-panel-btn dev-conf-generate-btn" type="button">日志查看</button>
```

- [ ] **Step 2: 添加开发工具日志面板的 CSS**

在 `<style>` 内末尾（`</style>` 之前）添加：

```css
.dev-log-select {
  background: #1e293b;
  color: #e0e0e0;
  border: 1px solid #475569;
  border-radius: 4px;
  padding: 3px 8px;
  font-size: 12px;
  cursor: pointer;
  margin-right: 8px;
}
.dev-log-select:focus-visible {
  outline: 2px solid var(--ra-focus-ring);
  outline-offset: 1px;
}
```

- [ ] **Step 3: 添加 JS 状态和工具选项列表**

在 `<script>` 中，现有 `logsState` 对象定义之后添加：

```javascript
const devLogState = {
  open: false,
  tool: 'conf-sync',
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

- [ ] **Step 4: 实现 openDevLogsPanel 函数**

在 `closeLogsPanel` 函数之后添加：

```javascript
function openDevLogsPanel(tool) {
  // 关闭服务日志面板（如果打开）
  if (logsState.open) {
    closeLogsPanel();
  }
  devLogState.open = true;
  devLogState.tool = tool || 'conf-sync';
  devLogState.requestSerial += 1;

  var workspace = document.getElementById('workspace');
  workspace.classList.add('logs-open');
  var panel = document.getElementById('logs-panel');
  panel.offsetHeight;
  panel.setAttribute('aria-hidden', 'false');

  var savedWidth = parseFloat(panel.style.width);
  if (!savedWidth || isNaN(savedWidth)) {
    setLogsPanelWidth(workspace.clientWidth * 0.38);
  }

  updateDevLogsPanelHeader();
  document.getElementById('logs-content').textContent = 'Loading logs...';
  stopDevLogsAutoRefresh();
  fetchDevLogsOnce();
  devLogState.timerId = setInterval(fetchDevLogsOnce, statusRefreshMs);
}

function closeDevLogsPanel() {
  devLogState.open = false;
  devLogState.tool = '';
  devLogState.requestSerial += 1;
  stopDevLogsAutoRefresh();
  var panel = document.getElementById('logs-panel');
  panel.setAttribute('aria-hidden', 'true');
  setTimeout(function () {
    if (!devLogState.open) {
      document.getElementById('workspace').classList.remove('logs-open');
    }
  }, 180);
}

function stopDevLogsAutoRefresh() {
  if (devLogState.timerId !== null) {
    clearInterval(devLogState.timerId);
    devLogState.timerId = null;
  }
}
```

- [ ] **Step 5: 实现 updateDevLogsPanelHeader 函数**

添加更新面板标题的函数：

```javascript
function updateDevLogsPanelHeader() {
  document.getElementById('logs-panel-title').textContent = '开发工具日志';
  updateDevLogsMeta('');

  // 构建带下拉和按钮的 header
  var metaEl = document.getElementById('logs-panel-meta');
  metaEl.innerHTML = '';

  // 隐藏 Loki/Grafana 按钮（开发工具日志不需要）
  document.getElementById('logs-panel-loki').style.display = 'none';
  document.getElementById('logs-panel-grafana').style.display = 'none';
  // 显示清空按钮
  document.getElementById('logs-panel-close').style.display = '';
}
```

- [ ] **Step 6: 修改面板 header，添加工具选择下拉和清空按钮**

修改现有的 `#logs-panel-header` HTML。将 `<div class="logs-panel-actions">` 中在 `#logs-panel-loki` 之后插入：

```html
<select id="dev-log-select" class="dev-log-select" aria-label="选择开发工具"></select>
<button id="logs-panel-clear-dev" class="logs-panel-btn" type="button" style="display:none">清空日志</button>
```

- [ ] **Step 7: 实现 fetchDevLogsOnce 和 renderDevLogs**

```javascript
async function fetchDevLogsOnce() {
  if (!devLogState.open || !devLogState.tool) { return; }
  var reqId = ++devLogState.requestSerial;
  var tool = devLogState.tool;
  try {
    var query = new URLSearchParams({tool: tool, lines: String(logLines)});
    var resp = await fetch('/api/dev/logs?' + query.toString());
    var result = await parseJsonSafe(resp);
    if (!resp.ok) { throw new Error(result.error || 'HTTP ' + resp.status); }
    if (!devLogState.open || devLogState.tool !== tool || reqId !== devLogState.requestSerial) { return; }
    renderDevLogs(result);
  } catch (err) {
    if (!devLogState.open || devLogState.tool !== tool || reqId !== devLogState.requestSerial) { return; }
    updateDevLogsMeta('refresh failed');
    document.getElementById('logs-content').textContent = 'Failed to load logs: ' + err.message;
  }
}

function renderDevLogs(payload) {
  var tool = payload && payload.tool ? String(payload.tool) : devLogState.tool;
  var rows = Array.isArray(payload && payload.lines) ? payload.lines : [];
  var content = document.getElementById('logs-content');
  content.textContent = rows.length > 0 ? rows.join('\n') : '暂无日志';
  updateDevLogsMeta(rows.length + ' lines - refreshed at ' + formatNowTime());
}

function updateDevLogsMeta(text) {
  document.getElementById('logs-panel-meta').textContent =
    (devLogState.tool || '-') + ' - ' + (text || 'loading...') + ' - last ' + logLines + ' lines';
}
```

- [ ] **Step 8: 实现 clearDevLogs 函数**

```javascript
async function clearDevLogs() {
  if (!devLogState.tool) { return; }
  var confirmed = window.confirm('确认清空「' + (getDevToolLabel(devLogState.tool) || devLogState.tool) + '」的日志吗？');
  if (!confirmed) { return; }
  try {
    var resp = await fetch('/api/dev/logs/clear', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({tool: devLogState.tool})
    });
    var result = await parseJsonSafe(resp);
    if (!resp.ok) {
      alert('清空日志失败: ' + (result.error || resp.status));
      return;
    }
    fetchDevLogsOnce();
  } catch (err) {
    alert('清空日志失败: ' + err.message);
  }
}

function getDevToolLabel(value) {
  var opt = devToolOptions.find(function(o) { return o.value === value; });
  return opt ? opt.label : '';
}
```

- [ ] **Step 9: 修改 openLogsPanel（服务日志）在处理时隐藏开发工具 UI 元素**

修改 `openLogsPanel` 函数，在开头添加：

```javascript
// 关闭开发工具日志面板（如果打开）
if (devLogState.open) {
  closeDevLogsPanel();
}
// 恢复服务日志面板的 header 样式
document.getElementById('logs-panel-loki').style.display = '';
document.getElementById('logs-panel-grafana').style.display = '';
document.getElementById('logs-panel-clear-dev').style.display = 'none';
var devSelect = document.getElementById('dev-log-select');
if (devSelect) { devSelect.style.display = 'none'; }
```

- [ ] **Step 10: 修改 closeLogsPanel 恢复服务日志 header**

在 `closeLogsPanel` 末尾（但 setTimeout 之前）添加：

```javascript
document.getElementById('logs-panel-loki').style.display = '';
document.getElementById('logs-panel-grafana').style.display = '';
document.getElementById('logs-panel-clear-dev').style.display = 'none';
var devSelect = document.getElementById('dev-log-select');
if (devSelect) { devSelect.style.display = 'none'; }
```

- [ ] **Step 11: 绑定事件监听器**

在现有 `window.addEventListener('resize', ...)` 之后、`</script>` 之前添加：

```javascript
// 开发工具日志查看按钮
var devViewLogsBtn = document.getElementById('dev-view-logs');
if (devViewLogsBtn) {
  devViewLogsBtn.addEventListener('click', function (event) {
    pulseClickFeedback(event.currentTarget);
    openDevLogsPanel('conf-sync');
  });
}

// 工具选择下拉
var devLogSelect = document.getElementById('dev-log-select');
if (devLogSelect) {
  renderDevLogSelect();
  devLogSelect.addEventListener('change', function () {
    devLogState.tool = devLogSelect.value;
    devLogState.requestSerial += 1;
    updateDevLogsPanelHeader();
    document.getElementById('logs-content').textContent = 'Loading logs...';
    fetchDevLogsOnce();
  });
}

function renderDevLogSelect() {
  var select = document.getElementById('dev-log-select');
  if (!select) { return; }
  select.innerHTML = '';
  devToolOptions.forEach(function (opt) {
    var option = document.createElement('option');
    option.value = opt.value;
    option.textContent = opt.label;
    select.appendChild(option);
  });
  select.value = devLogState.tool;
  select.style.display = 'none';
}

// 开发工具清空日志按钮
var clearDevLogsBtn = document.getElementById('logs-panel-clear-dev');
if (clearDevLogsBtn) {
  clearDevLogsBtn.addEventListener('click', function (event) {
    pulseClickFeedback(event.currentTarget);
    clearDevLogs();
  });
}

// 修改 closeLogsPanel 以处理开发工具日志
var origCloseLogsPanel = closeLogsPanel;
closeLogsPanel = function() {
  if (devLogState.open) {
    closeDevLogsPanel();
  } else {
    origCloseLogsPanel();
  }
};

// 修改 openDevLogsPanel —— 打开时更新下拉和按钮可见性
var origOpenDevLogsPanel = openDevLogsPanel;
openDevLogsPanel = function(tool) {
  origOpenDevLogsPanel(tool);
  // 显示开发工具特有 UI 元素
  document.getElementById('logs-panel-loki').style.display = 'none';
  document.getElementById('logs-panel-grafana').style.display = 'none';
  document.getElementById('logs-panel-clear-dev').style.display = '';
  if (devLogSelect) {
    devLogSelect.style.display = '';
    devLogSelect.value = devLogState.tool;
  }
};
```

- [ ] **Step 12: 编译 + 运行测试验证**

Run: `cd runAll/src && go build ./... && go test ./... -count=1`
Expected: 编译成功，已有测试全部通过

---

### Task 4: 新增 API 端点测试

**Files:**
- Modify: `runAll/src/ui_test.go`

- [ ] **Step 1: 添加 GET /api/dev/logs 测试**

在 `ui_test.go` 末尾添加测试函数：

```go
func TestUIDevToolLogs_Get(t *testing.T) {
	root := t.TempDir()
	rec := infrastructure.NewFileDevToolLogRecorder(root)
	rec.Append(domain.ToolDbClear, "test line 1")
	rec.Append(domain.ToolDbClear, "test line 2")

	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{devToolLogRecorder: rec}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner)

	req := httptest.NewRequest("GET", "/api/dev/logs?tool=db-clear&lines=10", nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var payload struct {
		Tool  string   `json:"tool"`
		Lines []string `json:"lines"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if payload.Tool != "db-clear" {
		t.Errorf("tool = %q, want db-clear", payload.Tool)
	}
	if len(payload.Lines) != 2 {
		t.Errorf("lines count = %d, want 2", len(payload.Lines))
	}
}

func TestUIDevToolLogs_GetMissingTool(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{devToolLogRecorder: infrastructure.NewFileDevToolLogRecorder(t.TempDir())}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner)

	req := httptest.NewRequest("GET", "/api/dev/logs?lines=10", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestUIDevToolLogs_InvalidTool(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{devToolLogRecorder: infrastructure.NewFileDevToolLogRecorder(t.TempDir())}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner)

	req := httptest.NewRequest("GET", "/api/dev/logs?tool=bad-tool&lines=10", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestUIDevToolLogs_Clear(t *testing.T) {
	root := t.TempDir()
	rec := infrastructure.NewFileDevToolLogRecorder(root)
	rec.Append(domain.ToolDbClear, "test line")

	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{devToolLogRecorder: rec}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner)

	body := strings.NewReader(`{"tool":"db-clear"}`)
	req := httptest.NewRequest("POST", "/api/dev/logs/clear", body)
	req.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	lines, _ := rec.Tail(domain.ToolDbClear, 10)
	if len(lines) != 0 {
		t.Errorf("after clear, Tail returned %d lines, want 0", len(lines))
	}
}

func TestUIDevToolLogs_ClearAll(t *testing.T) {
	root := t.TempDir()
	rec := infrastructure.NewFileDevToolLogRecorder(root)
	for _, tool := range domain.AllDevTools {
		rec.Append(tool, "test")
	}

	store := NewStatusStore()
	store.Init([]string{"svc"})
	runner := &Runner{devToolLogRecorder: rec}

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner)

	body := strings.NewReader(`{"tool":"all"}`)
	req := httptest.NewRequest("POST", "/api/dev/logs/clear", body)
	req.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	for _, tool := range domain.AllDevTools {
		lines, _ := rec.Tail(tool, 10)
		if len(lines) != 0 {
			t.Errorf("after ClearAll, Tail(%q) returned %d lines, want 0", tool, len(lines))
		}
	}
}
```

需要添加 import：
```go
import (
    // ... existing imports ...
    "runAll/src/infrastructure"
)
```

- [ ] **Step 2: 运行测试**

Run: `cd runAll/src && go test ./... -run "DevTool" -v`
Expected: 所有 DevTool 相关测试通过

---

### Task 5: UI 片段测试（status.html 包含新元素）

**Files:**
- Modify: `runAll/src/ui_test.go`（扩展现有 TestUIHomePage）

- [ ] **Step 1: 在 TestUIHomePage 的 requiredSnippets 中添加新元素验证**

找到 `TestUIHomePage` 函数中的 `requiredSnippets` 切片（约 line 512-580），添加新元素：

```go
`id="dev-view-logs"`,
`id="dev-log-select"`,
`id="logs-panel-clear-dev"`,
`function openDevLogsPanel(name)`,
`function closeDevLogsPanel()`,
`function fetchDevLogsOnce()`,
`function renderDevLogs(payload)`,
`function clearDevLogs()`,
`devToolOptions`,
`devLogState`,
`/api/dev/logs`,
`/api/dev/logs/clear`,
`class="dev-log-select"`,
```

- [ ] **Step 2: 运行 UI 测试**

Run: `cd runAll/src && go test -run "TestUIHomePage" -v`
Expected: PASS（验证 status.html 包含所有新增 UI 元素）

---

### Task 6: 全量测试 + 最终验证

**Files:**
- (all modified files)

- [ ] **Step 1: 运行全部测试**

Run: `cd runAll/src && go test ./... -count=1`
Expected: 全部测试通过

- [ ] **Step 2: 运行 race 检测**

Run: `cd runAll/src && go test -race ./domain/... ./infrastructure/...`
Expected: 无 data race

- [ ] **Step 3: 最终提交**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
git add runAll/src/domain/dev_tool_log_recorder.go \
        runAll/src/domain/dev_tool_log_recorder_test.go \
        runAll/src/infrastructure/dev_tool_log_recorder.go \
        runAll/src/infrastructure/dev_tool_log_recorder_test.go \
        runAll/src/ui.go \
        runAll/src/ui_test.go \
        runAll/src/runner.go \
        runAll/src/status.html \
        docs/superpowers/specs/2026-06-03-dev-tools-log-viewer-design.md \
        docs/superpowers/plans/2026-06-03-dev-tools-log-viewer-value-stream.md \
        docs/superpowers/plans/2026-06-03-dev-tools-log-viewer-nfr-clarification.md \
        value-stream.yaml
git commit -m "feat: add dev tools log viewer with file-based logging

- Add DevToolLogRecorder domain interface + FileDevToolLogRecorder infra impl
- Add GET /api/dev/logs and POST /api/dev/logs/clear endpoints
- Integrate logging into all five dev tool operations
- Add log viewer button, tool selector dropdown, and clear button to UI
- Logs persisted to logs/<tool>.log files, survive restarts

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```
