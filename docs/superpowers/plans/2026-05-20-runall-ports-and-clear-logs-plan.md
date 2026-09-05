# runAll Ports and Clear Logs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `runAll` 首页为每个服务展示 `health_port` 与 `command_port`，并新增“清空日志”按钮，仅清空 runAll 内存日志缓冲。

**Architecture:** 复用现有 `StatusStore -> /api/status -> status.html` 主链路，在领域层先定义端口解析与日志清空契约，再落地基础设施实现与 UI 交互。采用 TDD：先写失败测试，再补最小实现，最后回归。

**Tech Stack:** Go (`net/url`, `regexp`, `net/http`), 原生 HTML/CSS/JS, `go test`.

---

## File Structure (Planned Changes)

- Modify: `runAll/src/domain/service_log_repository.go`  
  责任：领域仓储接口新增 `Clear(service string)` 契约。
- Create: `runAll/src/domain/port_resolver_service.go`  
  责任：领域端口解析服务（health URL + command best-effort）。
- Create: `runAll/src/domain/port_resolver_service_test.go`  
  责任：覆盖端口解析规则。
- Modify: `runAll/src/infrastructure/inmemory_service_log_repository.go`  
  责任：实现 `Clear`。
- Modify: `runAll/src/infrastructure/inmemory_service_log_repository_test.go`  
  责任：补充 clear 的隔离性/幂等性测试。
- Modify: `runAll/src/status.go`  
  责任：`ServiceStatus` 增加 `health_port`、`command_port` 与 setter。
- Modify: `runAll/src/runner.go`  
  责任：初始化服务元信息时写入双端口。
- Modify: `runAll/src/ui.go`  
  责任：新增 `POST /api/logs/clear` handler。
- Modify: `runAll/src/ui_test.go`  
  责任：新增 clear 接口测试与首页关键片段断言。
- Modify: `runAll/src/status.html`  
  责任：服务行展示双端口 + 清空日志按钮 + 前端请求逻辑。

---

### Task 1: 领域契约先行（DDD）— 日志仓储 Clear 接口

**Files:**
- Modify: `runAll/src/domain/service_log_repository.go`
- Modify: `runAll/src/infrastructure/inmemory_service_log_repository_test.go`
- Test: `runAll/src/infrastructure/inmemory_service_log_repository_test.go`

- [ ] **Step 1: 先写失败测试（clear 语义）**

```go
func TestInmemoryServiceLogRepository_ClearOnlyTargetService(t *testing.T) {
	repo := NewInMemoryServiceLogRepository(10)
	repo.Append("svc-a", mustLogEntry(t, "svc-a", domain.StreamStdout, "a-1"))
	repo.Append("svc-b", mustLogEntry(t, "svc-b", domain.StreamStdout, "b-1"))

	repo.Clear("svc-a")

	if got := repo.Tail("svc-a", 10); len(got) != 0 {
		t.Fatalf("svc-a logs should be cleared, got %#v", got)
	}
	if got := repo.Tail("svc-b", 10); len(got) != 1 {
		t.Fatalf("svc-b logs should remain, got %#v", got)
	}
}
```

- [ ] **Step 2: 运行测试确认失败（接口尚未定义）**

Run: `cd runAll && go test ./src/infrastructure -run TestInmemoryServiceLogRepository_ClearOnlyTargetService -v`  
Expected: FAIL（`repo.Clear undefined` 或接口不满足）。

- [ ] **Step 3: 在领域接口定义 Clear 契约（最小实现）**

```go
type ServiceLogRepository interface {
	Append(service string, entry LogEntry)
	Tail(service string, lines int) []LogEntry
	Clear(service string)
}
```

- [ ] **Step 4: 在 in-memory 仓储实现 Clear**

```go
func (r *InMemoryServiceLogRepository) Clear(service string) {
	if service == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.buffers, service)
}
```

- [ ] **Step 5: 运行仓储测试确认通过**

Run: `cd runAll && go test ./src/infrastructure -v`  
Expected: PASS（含新增 clear 用例）。

- [ ] **Step 6: 提交本任务**

Run:
```bash
git add runAll/src/domain/service_log_repository.go runAll/src/infrastructure/inmemory_service_log_repository.go runAll/src/infrastructure/inmemory_service_log_repository_test.go
git commit -m "feat(runall): add log repository clear contract and in-memory implementation"
```

---

### Task 2: 领域服务（DDD）— 端口解析规则与测试

**Files:**
- Create: `runAll/src/domain/port_resolver_service.go`
- Create: `runAll/src/domain/port_resolver_service_test.go`
- Test: `runAll/src/domain/port_resolver_service_test.go`

- [ ] **Step 1: 写失败测试（health URL 端口规则）**

```go
func TestResolveHealthPort(t *testing.T) {
	tests := []struct{
		name string
		raw  string
		want string
	}{
		{"explicit port", "http://localhost:8080/health", "8080"},
		{"http default", "http://localhost/health", "80"},
		{"https default", "https://svc.internal/health", "443"},
		{"invalid", "://bad-url", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveHealthPort(tt.raw); got != tt.want {
				t.Fatalf("ResolveHealthPort(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: 写失败测试（command 端口优先级）**

```go
func TestResolveCommandPort(t *testing.T) {
	tests := []struct{
		name string
		cmd  string
		want string
	}{
		{"long flag", "server --port 7001", "7001"},
		{"long flag equals", "server --port=7002", "7002"},
		{"short flag", "server -p 7003", "7003"},
		{"env prefix", "PORT=7004 server", "7004"},
		{"host colon", "server --listen 0.0.0.0:7005", "7005"},
		{"no match", "server --verbose", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveCommandPort(tt.cmd); got != tt.want {
				t.Fatalf("ResolveCommandPort(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 3: 运行测试确认失败（函数尚未实现）**

Run: `cd runAll && go test ./src/domain -run 'TestResolveHealthPort|TestResolveCommandPort' -v`  
Expected: FAIL（undefined functions）。

- [ ] **Step 4: 实现端口解析领域服务（最小可用）**

```go
func ResolveHealthPort(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u == nil {
		return ""
	}
	if p := u.Port(); p != "" {
		return p
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}
```

```go
func ResolveCommandPort(command string) string {
	candidates := []string{
		findByRegex(`--port(?:=|\s+)(\d{1,5})`, command),
		findByRegex(`(?:^|\s)-p\s+(\d{1,5})(?:\s|$)`, command),
		findByRegex(`(?:^|\s)PORT=(\d{1,5})(?:\s|$)`, command),
		findByRegex(`(?:^|[^0-9]):(\d{1,5})(?:[^0-9]|$)`, command),
	}
	for _, p := range candidates {
		if isValidPort(p) {
			return p
		}
	}
	return ""
}

func findByRegex(pattern, src string) string {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(src)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func isValidPort(raw string) bool {
	n, err := strconv.Atoi(raw)
	return err == nil && n >= 1 && n <= 65535
}
```

- [ ] **Step 5: 运行领域测试确认通过**

Run: `cd runAll && go test ./src/domain -v`  
Expected: PASS。

- [ ] **Step 6: 提交本任务**

Run:
```bash
git add runAll/src/domain/port_resolver_service.go runAll/src/domain/port_resolver_service_test.go
git commit -m "feat(runall): add domain port resolver service"
```

---

### Task 3: 状态模型与 Runner 接线（双端口进入 /api/status）

**Files:**
- Modify: `runAll/src/status.go`
- Modify: `runAll/src/runner.go`
- Modify: `runAll/src/ui_test.go`
- Test: `runAll/src/ui_test.go`

- [ ] **Step 1: 写失败测试（/api/status 包含双端口字段）**

```go
func TestAPIStatus_IncludesPortFields(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	store.SetHealthPort("svc", "8080")
	store.SetCommandPort("svc", "9090")

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var result []map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&result)
	if result[0]["health_port"] != "8080" || result[0]["command_port"] != "9090" {
		t.Fatalf("missing port fields: %#v", result[0])
	}
}
```

- [ ] **Step 2: 运行测试确认失败（setter/字段不存在）**

Run: `cd runAll && go test ./src -run TestAPIStatus_IncludesPortFields -v`  
Expected: FAIL（缺少字段或方法）。

- [ ] **Step 3: 修改 `ServiceStatus` 与 `StatusStore` setter**

```go
type ServiceStatus struct {
	Name        string `json:"name"`
	Status      Status `json:"status"`
	HealthPort  string `json:"health_port"`
	CommandPort string `json:"command_port"`
}

func (s *StatusStore) SetHealthPort(name, port string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.HealthPort = port
	}
}

func (s *StatusStore) SetCommandPort(name, port string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svc, ok := s.services[name]; ok {
		svc.CommandPort = port
	}
}
```

- [ ] **Step 4: 在 Runner 初始化阶段写入端口**

```go
for _, svc := range services {
	store.SetCommand(svc.Name, svc.Command)
	store.SetURL(svc.Name, svc.HealthCheck.URL)
	store.SetHealthPort(svc.Name, domain.ResolveHealthPort(svc.HealthCheck.URL))
	store.SetCommandPort(svc.Name, domain.ResolveCommandPort(svc.Command))
}
```

- [ ] **Step 5: 运行相关测试确认通过**

Run: `cd runAll && go test ./src -run 'TestAPIStatus|TestAPIStatus_IncludesPortFields' -v`  
Expected: PASS。

- [ ] **Step 6: 提交本任务**

Run:
```bash
git add runAll/src/status.go runAll/src/runner.go runAll/src/ui_test.go
git commit -m "feat(runall): include health and command ports in status payload"
```

---

### Task 4: Clear Logs API（后端）— `/api/logs/clear`

**Files:**
- Modify: `runAll/src/ui.go`
- Modify: `runAll/src/ui_test.go`
- Test: `runAll/src/ui_test.go`

- [ ] **Step 1: 写失败测试（clear 成功路径）**

```go
func TestAPILogsClear_Success(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-clear",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9981"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	entry, err := domain.NewLogEntry(time.Now(), "svc-clear", domain.StreamStdout, "line-1")
	if err != nil {
		t.Fatalf("NewLogEntry: %v", err)
	}
	runner.logRepository.Append("svc-clear", entry)

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner)

	req := httptest.NewRequest(http.MethodPost, "/api/logs/clear", strings.NewReader(`{"name":"svc-clear"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
	if got := runner.logRepository.Tail("svc-clear", 10); len(got) != 0 {
		t.Fatalf("logs not cleared: %#v", got)
	}
}
```

- [ ] **Step 2: 写失败测试（method/name/unknown service）**

```go
func TestAPILogsClear_BadRequest(t *testing.T) {
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{
					Name:        "svc-clear-bad",
					Command:     "echo running",
					HealthCheck: HealthCheck{URL: "http://localhost:9980"},
				},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, runner)

	tests := []struct {
		name   string
		method string
		body   string
		status int
	}{
		{"method not allowed", http.MethodGet, "", http.StatusMethodNotAllowed},
		{"missing name", http.MethodPost, `{}`, http.StatusBadRequest},
		{"unknown service", http.MethodPost, `{"name":"missing"}`, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/logs/clear", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
```

- [ ] **Step 3: 运行测试确认失败（接口未实现）**

Run: `cd runAll && go test ./src -run 'TestAPILogsClear_Success|TestAPILogsClear_BadRequest' -v`  
Expected: FAIL（404 或 handler 缺失）。

- [ ] **Step 4: 在 `registerUIHandlers` 增加 clear handler**

```go
mux.HandleFunc("/api/logs/clear", func(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, "invalid json")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeJSONError(w, "name is required")
		return
	}
	if runner == nil || runner.logRepository == nil {
		writeJSONError(w, "runner is required")
		return
	}
	if runner.findService(name) == nil {
		writeJSONError(w, fmt.Sprintf("service %q not found", name))
		return
	}
	runner.logRepository.Clear(name)
	writeJSON(w, map[string]string{"status": "ok"})
})
```

- [ ] **Step 5: 运行 API 测试确认通过**

Run: `cd runAll && go test ./src -run 'TestAPILogs' -v`  
Expected: PASS（含 clear 新增用例）。

- [ ] **Step 6: 提交本任务**

Run:
```bash
git add runAll/src/ui.go runAll/src/ui_test.go
git commit -m "feat(runall): add clear logs api endpoint"
```

---

### Task 5: 首页 UI 展示端口与清空日志按钮

**Files:**
- Modify: `runAll/src/status.html`
- Modify: `runAll/src/ui_test.go`
- Test: `runAll/src/ui_test.go`

- [ ] **Step 1: 写失败测试（首页包含端口与清空按钮片段）**

```go
func TestUIHomePage_IncludesPortsAndClearLogsAction(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	body := rec.Body.String()

	requiredSnippets := []string{
		`data-action="clear-logs"`,
		`/api/logs/clear`,
		`health ${esc(svc.health_port || "-")}`,
		`command ${esc(svc.command_port || "-")}`,
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(body, snippet) {
			t.Fatalf("home page missing snippet %q", snippet)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll && go test ./src -run TestUIHomePage_IncludesPortsAndClearLogsAction -v`  
Expected: FAIL（片段不存在）。

- [ ] **Step 3: 更新 `status.html` 服务行渲染与交互**

```javascript
html += `<span class="ports">Ports: health ${esc(svc.health_port || "-")} / command ${esc(svc.command_port || "-")}</span>`;
html += `<button class="action-btn clear-logs-btn" type="button" data-action="clear-logs" data-name="${esc(svc.name)}">清空日志</button>`;
```

```javascript
async function clearLogs(name) {
  await postServiceAction('/api/logs/clear', name, 'Clear logs');
  if (logsState.open && logsState.service === name) {
    fetchLogsOnce();
  }
}
```

- [ ] **Step 4: 在 click handler 分支接入 clear-logs**

```javascript
if (action === 'clear-logs') {
  clearLogs(name);
  return;
}
```

- [ ] **Step 5: 运行页面相关测试确认通过**

Run: `cd runAll && go test ./src -run 'TestUIHomePage|TestUILogsModal' -v`  
Expected: PASS。

- [ ] **Step 6: 提交本任务**

Run:
```bash
git add runAll/src/status.html runAll/src/ui_test.go
git commit -m "feat(runall): show dual ports and add clear logs button in status page"
```

---

### Task 6: 全量回归与文档同步

**Files:**
- Modify: `docs/superpowers/specs/2026-05-20-runall-ports-and-clear-logs-design.md` (仅当实现偏差需要回写)
- Test: `runAll/src/...` 全量

- [ ] **Step 1: 运行 runAll 全量测试**

Run: `cd runAll && go test ./...`  
Expected: PASS（无回归）。

- [ ] **Step 2: 手工冒烟验证 UI**

Run:
```bash
cd runAll
go run ./src -config config.yaml -ui-port :9999
```
Expected:
- 服务行展示 `health / command` 端口（无值显示 `-`）；
- “清空日志”按钮可点击；
- 清空后当前服务日志视图变为空，再有新输出会继续出现。

- [ ] **Step 3: 若与 spec 有偏差，更新 spec 并说明原因**

```markdown
## Implementation Notes
- None
```

- [ ] **Step 4: 最终提交（如有未提交变更）**

Run:
```bash
git add runAll docs/superpowers/specs/2026-05-20-runall-ports-and-clear-logs-design.md
git commit -m "test(runall): validate ports and clear-logs feature end-to-end"
```

---

## DDD Structure Validation Checklist

- [ ] 领域接口（`ServiceLogRepository.Clear`）先于基础设施实现落地。
- [ ] 端口解析逻辑位于 `domain`，不依赖基础设施。
- [ ] 基础设施层只实现仓储，不定义领域契约。
- [ ] 事件命名沿用过去式（本次可复用既有事件定义，非必须扩展）。
- [ ] 任务顺序保持 domain -> infrastructure -> interfaces/UI。

## Self-Review

- Spec coverage: 端口双来源展示、清空日志 API、UI 按钮、错误处理、测试策略均已映射到 Task 1-6。
- Placeholder scan: 无 `TODO/TBD/later` 占位。
- Type consistency: 统一使用 `health_port` / `command_port` / `Clear(service string)`。
