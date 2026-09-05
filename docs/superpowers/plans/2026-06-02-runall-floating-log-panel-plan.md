# runAll 日志面板悬浮 + 左侧拖动 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 runAll Web UI 的日志面板从 flex 并排布局改为 position:absolute 悬浮 overlay，左边缘可拖动调宽。

**Architecture:** 纯 CSS + JS 改造 `runAll/src/status.html`。`.workspace` 设为定位锚点，`.logs-panel` 改为 `position: absolute` overlay。移除 `.pane-divider`，在面板左边缘内置 resize handle。Go 后端零改动。

**Tech Stack:** HTML/CSS/JS（嵌入式，无框架）

---

## File Structure

| 文件 | 操作 | 职责 |
|------|------|------|
| `runAll/src/status.html` | 修改 | CSS 布局 + JS 交互逻辑 |

---

### Task 1: CSS — 日志面板从 flex 改为 absolute overlay

**Files:**
- Modify: `runAll/src/status.html` (style section)

- [ ] **Step 1: 修改 `.workspace` 样式，添加 `position: relative`**

找到 `.workspace` 规则（第 46 行附近），添加 `position: relative`：

```css
.workspace { margin-top: 20px; display: flex; width: 100%; flex: 1 1 auto; min-height: 0; overflow: hidden; --logs-width: 38%; position: relative; }
```

- [ ] **Step 2: 修改 `.logs-panel` 从 flex 子项改为 absolute 定位**

将当前 `.logs-panel` 规则（第 238 行附近）：

```css
.logs-panel { display: none; flex: 0 0 var(--logs-width); width: var(--logs-width); min-width: 320px; max-width: 75vw; flex-shrink: 0; min-height: 0; background: var(--ra-bg-panel); border: 1px solid #334155; border-radius: 8px; flex-direction: column; overflow: hidden; }
```

改为：

```css
.logs-panel {
  display: none;
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: var(--logs-width);
  min-width: 320px;
  max-width: 75vw;
  background: var(--ra-bg-panel);
  border: 2px solid #7c3aed;
  border-radius: 8px 0 0 8px;
  flex-direction: column;
  overflow: hidden;
  box-shadow: -4px 0 24px rgba(0, 0, 0, 0.6), 0 4px 24px rgba(0, 0, 0, 0.4);
  z-index: 10;
  transform: translateX(0);
  opacity: 1;
  transition: transform 150ms ease, opacity 150ms ease;
}
.logs-panel[aria-hidden="true"] {
  transform: translateX(100%);
  opacity: 0;
  pointer-events: none;
}
```

- [ ] **Step 3: 修改 `.workspace.logs-open` 规则**

移除对 flex 子项的控制，仅控制面板可见性。将第 241-242 行：

```css
.workspace.logs-open .pane-divider { display: block; }
.workspace.logs-open .logs-panel { display: flex; }
```

改为：

```css
.workspace.logs-open .logs-panel { display: flex; }
```

- [ ] **Step 4: 修改 `.services-pane` 确保始终全宽**

确认 `.services-pane` 规则（第 47 行）在 `logs-open` 状态下不受影响：

```css
.services-pane { flex: 1 1 auto; min-width: 320px; min-height: 0; padding-right: 8px; overflow: auto; scrollbar-gutter: stable; }
```

（无需修改，`.logs-open` 不再影响 flex 流）

- [ ] **Step 5: 移除 `.pane-divider` 和 `.pane-resizing` 相关 CSS**

删除或注释以下规则（第 236, 239, 241 行）：

```css
/* 删除: */
.pane-divider { display: none; ... }
body.pane-resizing { cursor: col-resize; user-select: none; }
.workspace.logs-open .pane-divider { display: block; }
```

- [ ] **Step 6: 添加日志面板左边缘 resize handle 样式**

在 style 区域末尾添加：

```css
.logs-resize-handle {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 6px;
  cursor: ew-resize;
  z-index: 2;
  background: linear-gradient(90deg, #7c3aed 0%, transparent 100%);
  opacity: 0.6;
  transition: opacity 120ms ease;
}
.logs-resize-handle:hover { opacity: 1; }
.logs-resize-handle::after {
  content: '';
  position: absolute;
  left: 1px;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 28px;
  border-radius: 2px;
  background: #a78bfa;
}
```

- [ ] **Step 7: 验证 CSS 变更**

启动 runAll 并打开 `http://localhost:9999/`，确认：
- 日志面板未打开时页面布局正常
- 开发者工具中 `.workspace` 有 `position: relative`
- `.logs-panel` 样式为 `position: absolute`

---

### Task 2: HTML — 添加 resize handle 元素，移除 pane-divider

**Files:**
- Modify: `runAll/src/status.html` (HTML body section)

- [ ] **Step 1: 移除 `.pane-divider` 元素**

删除第 368 行的 divider HTML：

```html
<!-- 删除: -->
<div id="pane-divider" class="pane-divider" role="separator" aria-orientation="vertical" aria-label="Resize panels"></div>
```

- [ ] **Step 2: 在 `.logs-panel` 内添加 resize handle**

在 `<aside id="logs-panel">` 内部，`logs-panel-header` 之前添加：

```html
<div id="logs-resize-handle" class="logs-resize-handle" aria-hidden="true"></div>
```

---

### Task 3: JS — 更新 open/close 逻辑

**Files:**
- Modify: `runAll/src/status.html` (script section)

- [ ] **Step 1: 修改 `openLogsPanel()` — 移除 flex 相关逻辑**

找到 `openLogsPanel` 函数（第 960 行附近），当前代码：

```javascript
function openLogsPanel(name) {
  logsState.open = true;
  logsState.service = name;
  logsState.requestSerial += 1;
  const workspace = document.getElementById('workspace');
  workspace.classList.add('logs-open');
  setLogsPanelWidth(workspace.clientWidth * 0.38);
  const panel = document.getElementById('logs-panel');
  panel.setAttribute('aria-hidden', 'false');
  updateLogsMeta('loading...');
  document.getElementById('logs-content').textContent = 'Loading logs...';
  stopLogsAutoRefresh();
  fetchLogsOnce();
  logsState.timerId = setInterval(fetchLogsOnce, statusRefreshMs);
}
```

修改为（仅改 `panel.setAttribute` 的时机和默认宽度初始化逻辑）：

```javascript
function openLogsPanel(name) {
  logsState.open = true;
  logsState.service = name;
  logsState.requestSerial += 1;
  const workspace = document.getElementById('workspace');
  workspace.classList.add('logs-open');
  var panel = document.getElementById('logs-panel');
  panel.setAttribute('aria-hidden', 'false');
  // Use saved width or default 38%
  var savedWidth = parseFloat(panel.style.width);
  if (!savedWidth || isNaN(savedWidth)) {
    setLogsPanelWidth(workspace.clientWidth * 0.38);
  }
  updateLogsMeta('loading...');
  document.getElementById('logs-content').textContent = 'Loading logs...';
  stopLogsAutoRefresh();
  fetchLogsOnce();
  logsState.timerId = setInterval(fetchLogsOnce, statusRefreshMs);
}
```

- [ ] **Step 2: 修改 `closeLogsPanel()` — 保持简洁**

找到 `closeLogsPanel` 函数（第 976 行），确认逻辑不变（仅移除 class + 设置 aria-hidden）：

```javascript
function closeLogsPanel() {
  logsState.open = false;
  logsState.service = '';
  logsState.lastRows = [];
  logsState.requestSerial += 1;
  stopLogsAutoRefresh();
  var workspace = document.getElementById('workspace');
  workspace.classList.remove('logs-open');
  var panel = document.getElementById('logs-panel');
  panel.setAttribute('aria-hidden', 'true');
}
```

- [ ] **Step 3: 修改 `setLogsPanelWidth()` — 直接设置面板 style.width**

找到 `setLogsPanelWidth` 函数（第 1492 行），当前代码：

```javascript
function setLogsPanelWidth(px) {
  const workspace = document.getElementById('workspace');
  const maxWidth = Math.max(minLogsPanelWidth, workspace.clientWidth - minServicePanelWidth - 16);
  const clamped = Math.min(Math.max(px, minLogsPanelWidth), maxWidth);
  workspace.style.setProperty('--logs-width', `${clamped}px`);
}
```

修改为直接设置面板 width：

```javascript
function setLogsPanelWidth(px) {
  var workspace = document.getElementById('workspace');
  var maxWidth = Math.min(workspace.clientWidth * 0.75, workspace.clientWidth - 16);
  var clamped = Math.min(Math.max(px, minLogsPanelWidth), maxWidth);
  var panel = document.getElementById('logs-panel');
  panel.style.width = clamped + 'px';
}
```

---

### Task 4: JS — 左边缘拖拽 resize handle

**Files:**
- Modify: `runAll/src/status.html` (script section)

- [ ] **Step 1: 删除旧的 pane-divider 事件处理代码**

删除以下代码块（第 1628-1640 行）：

```javascript
// 删除:
const paneDividerEl = document.getElementById('pane-divider');
paneDividerEl.addEventListener('pointerdown', handleDividerPointerDown);
paneDividerEl.addEventListener('pointermove', handleDividerPointerMove);
paneDividerEl.addEventListener('pointerup', handleDividerPointerUp);
paneDividerEl.addEventListener('pointercancel', handleDividerPointerUp);
window.addEventListener('blur', stopPaneResize);
window.addEventListener('resize', () => {
  const workspace = document.getElementById('workspace');
  const currentWidth = parseFloat(workspace.style.getPropertyValue('--logs-width'));
  if (!Number.isNaN(currentWidth)) {
    setLogsPanelWidth(currentWidth);
  }
});
```

- [ ] **Step 2: 删除旧的 divider handler 函数**

删除 `handleDividerPointerDown`、`handleDividerPointerMove`、`handleDividerPointerUp`、`stopPaneResize` 函数（第 1499-1538 行）。

- [ ] **Step 3: 添加新的 resize handle 事件处理**

在 script 区域添加以下代码（在 `setLogsPanelWidth` 之后）：

```javascript
function handleLogsResizePointerDown(event) {
  if (!logsState.open) { return; }
  layoutState.resizing = true;
  event.preventDefault();
  if (event.currentTarget.setPointerCapture) {
    event.currentTarget.setPointerCapture(event.pointerId);
  }
  document.body.classList.add('pane-resizing');
}

function handleLogsResizePointerMove(event) {
  if (!layoutState.resizing || !logsState.open) { return; }
  var workspace = document.getElementById('workspace');
  var rect = workspace.getBoundingClientRect();
  var newWidth = rect.right - event.clientX;
  setLogsPanelWidth(newWidth);
}

function stopLogsResize(event) {
  if (!layoutState.resizing) { return; }
  layoutState.resizing = false;
  document.body.classList.remove('pane-resizing');
  if (event && event.currentTarget && event.currentTarget.releasePointerCapture && event.currentTarget.hasPointerCapture) {
    try {
      if (event.currentTarget.hasPointerCapture(event.pointerId)) {
        event.currentTarget.releasePointerCapture(event.pointerId);
      }
    } catch (_) {}
  }
}

var logsResizeHandle = document.getElementById('logs-resize-handle');
logsResizeHandle.addEventListener('pointerdown', handleLogsResizePointerDown);
logsResizeHandle.addEventListener('pointermove', handleLogsResizePointerMove);
logsResizeHandle.addEventListener('pointerup', stopLogsResize);
logsResizeHandle.addEventListener('pointercancel', stopLogsResize);
window.addEventListener('blur', function () { stopLogsResize(); });
```

- [ ] **Step 4: 更新 window resize 处理**

在 init 代码区域添加 window resize 处理（替换之前删除的）：

```javascript
window.addEventListener('resize', function () {
  if (!logsState.open) { return; }
  var panel = document.getElementById('logs-panel');
  var currentWidth = parseFloat(panel.style.width);
  if (!isNaN(currentWidth)) {
    setLogsPanelWidth(currentWidth);
  }
});
```

---

### Task 5: 验证与测试

- [ ] **Step 1: 编译 runAll**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/runAll && go build -o bin/runAll ./src/
```

Expected: 编译成功（status.html 通过 embed 嵌入，Go 代码未改动）

- [ ] **Step 2: 启动 runAll 并手动验证**

```bash
# 启动 runAll
cd /Users/task2app/gitClone/ramDisk/ram-mount && ./run.sh
```

打开 `http://localhost:9999/`：

1. 确认页面正常加载，服务列表全宽显示 ✓
2. 点击某服务的「日志」按钮 ✓
3. 确认日志面板从右侧滑入，覆盖在服务列表上方 ✓
4. 确认服务列表宽度未变化（未被挤压） ✓
5. 拖拽面板左边缘紫色手柄 ✓
6. 确认面板宽度实时调整，min 320px、max 75vw ✓
7. 点击 Close 按钮关闭面板 ✓
8. 按 ESC 键关闭面板 ✓
9. 打开日志 → 切换到另一个服务的日志 → 面板宽度保持 ✓

- [ ] **Step 3: 运行现有测试确保无回归**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/runAll && go test ./... -count=1
```

Expected: 所有现有测试通过（UI 变更不影响 Go 逻辑）

---

### Task 6: 关闭视觉伴随器并提交

- [ ] **Step 1: 关闭视觉伴随器**

```bash
/Users/task2app/.claude/plugins/cache/claude-plugins-official/superpowers/5.1.0/skills/brainstorming/scripts/stop-server.sh /Users/task2app/gitClone/ramDisk/ram-mount/.superpowers/brainstorm/94705-1780413303/state
```

- [ ] **Step 2: 提交变更**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
git add runAll/src/status.html docs/superpowers/
git commit -m "feat: float log panel over services pane with draggable left edge

- Change .logs-panel from flex child to position:absolute overlay
- Remove .pane-divider, add inline resize handle on panel left edge
- Preserve existing log fetch/refresh/logic unchanged
- Services pane stays full width, no longer squeezed

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```
