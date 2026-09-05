// status_ui 拆分块（OPT-20260810-040）：dev logs 工具 独立功能块，经 manifest.json 拼接后与主文件同 scope 执行。
// devLogState 已移至 01.js（须在 06.js bootstrap / currentStatusRefreshMs 之前初始化）。
var closeLogsPanel;
var openLogsPanel;

// ---- 开发工具日志查看 ----
const devToolOptions = [
  { value: 'conf-sync', label: '生成配置副本' },
  { value: 'db-clear', label: '清空全部数据库' },
  { value: 'db-init', label: '初始化全部数据库' },
  { value: 'observability-clear', label: '清空 Grafana 可观测数据' },
];

function getDevToolLabel(value) {
  var opt = devToolOptions.find(function (o) { return o.value === value; });
  return opt ? opt.label : value;
}

function renderDevLogSelect() {
  var select = document.getElementById('dev-log-select');
  if (!select) return;
  select.innerHTML = '';
  devToolOptions.forEach(function (opt) {
    var option = document.createElement('option');
    option.value = opt.value;
    option.textContent = opt.label;
    select.appendChild(option);
  });
  select.value = devLogState.tool;
}

function showDevLogUI() {
  document.getElementById('logs-panel-loki').style.display = 'none';
  document.getElementById('logs-panel-grafana').style.display = 'none';
  document.getElementById('logs-panel-clear-dev').style.display = '';
  var select = document.getElementById('dev-log-select');
  if (select) {
    renderDevLogSelect();
    select.style.display = '';
  }
}

function hideDevLogUI() {
  document.getElementById('logs-panel-loki').style.display = '';
  document.getElementById('logs-panel-grafana').style.display = '';
  document.getElementById('logs-panel-clear-dev').style.display = 'none';
  var select = document.getElementById('dev-log-select');
  if (select) select.style.display = 'none';
}

function openDevLogsPanel(tool) {
  if (logsState.open) closeLogsPanel();
  // 取消 dev 面板的延迟关闭 timer（如果有），避免新面板被意外关闭
  if (devLogState.closeTimerId !== null) {
    clearTimeout(devLogState.closeTimerId);
    devLogState.closeTimerId = null;
  }
  devLogState.open = true;
  devLogState.tool = tool || 'conf-sync';
  devLogState.requestSerial += 1;
  var workspace = document.getElementById('workspace');
  workspace.classList.add('logs-open');
  var panel = document.getElementById('logs-panel');
  panel.getBoundingClientRect();
  panel.setAttribute('aria-hidden', 'false');
  var savedWidth = parseFloat(panel.style.width);
  if (!savedWidth || isNaN(savedWidth)) {
    setLogsPanelWidth(workspace.clientWidth * 0.38);
  }
  document.getElementById('logs-panel-title').textContent = '开发工具日志';
  updateDevLogsMeta('loading...');
  showDevLogUI();
  document.getElementById('logs-content').textContent = 'Loading logs...';
  stopDevLogsAutoRefresh();
  fetchDevLogsOnce();
  devLogState.timerId = setInterval(fetchDevLogsOnce, statusRefreshMs);
  restartStatusRefreshTimer();
}

function closeDevLogsPanel() {
  devLogState.open = false;
  devLogState.tool = '';
  devLogState.requestSerial += 1;
  stopDevLogsAutoRefresh();
  hideDevLogUI();
  if (devLogState.closeTimerId !== null) {
    clearTimeout(devLogState.closeTimerId);
    devLogState.closeTimerId = null;
  }
  var panel = document.getElementById('logs-panel');
  panel.setAttribute('aria-hidden', 'true');
  restartStatusRefreshTimer();
  devLogState.closeTimerId = setTimeout(function () {
    devLogState.closeTimerId = null;
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

async function fetchDevLogsOnce() {
  if (!devLogState.open || !devLogState.tool) return;
  var reqId = ++devLogState.requestSerial;
  var tool = devLogState.tool;
  try {
    var query = new URLSearchParams({ tool: tool, lines: String(logLines) });
    var resp = await fetch('/api/dev/logs?' + query.toString());
    var result = await parseJsonSafe(resp);
    if (!resp.ok) throw new Error(result.error || 'HTTP ' + resp.status);
    if (!devLogState.open || devLogState.tool !== tool || reqId !== devLogState.requestSerial) return;
    renderDevLogs(result);
  } catch (err) {
    if (!devLogState.open || devLogState.tool !== tool || reqId !== devLogState.requestSerial) return;
    updateDevLogsMeta('refresh failed');
    document.getElementById('logs-content').textContent = 'Failed to load logs: ' + err.message;
  }
}

function renderDevLogs(payload) {
  var rows = Array.isArray(payload && payload.lines) ? payload.lines : [];
  document.getElementById('logs-content').textContent = rows.length > 0 ? rows.join('\n') : '暂无日志';
  updateDevLogsMeta(rows.length + ' lines - refreshed at ' + formatNowTime());
}

function updateDevLogsMeta(text) {
  var tool = devLogState.tool || '-';
  document.getElementById('logs-panel-title').textContent = '开发工具: ' + getDevToolLabel(tool);
  document.getElementById('logs-panel-meta').textContent = tool + ' - ' + (text || 'loading...') + ' - last ' + logLines + ' lines';
}

async function clearDevLogs() {
  if (!devLogState.tool) return;
  var label = getDevToolLabel(devLogState.tool) || devLogState.tool;
  if (!await showModalConfirm('确认清空「' + label + '」的日志吗？', {opLabel: '清空开发工具日志: ' + label})) return;
  try {
    var resp = await apiFetch('/api/dev/logs/clear', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tool: devLogState.tool })
    });
    var result = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError('清空日志失败: ' + (result.error || resp.status), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    fetchDevLogsOnce();
  } catch (err) {
    showRequestError('清空日志失败: ' + err.message, err);
  }
}

// 修改 closeLogsPanel 以处理开发工具日志面板
var origCloseLogsPanel = closeLogsPanel;
closeLogsPanel = function () {
  if (devLogState.open) {
    closeDevLogsPanel();
  } else {
    origCloseLogsPanel();
  }
};

// 修改 openLogsPanel 以关闭开发工具日志面板
var origOpenLogsPanel = openLogsPanel;
openLogsPanel = function (name) {
  if (devLogState.open) closeDevLogsPanel();
  // 确保 dev 面板的延迟关闭 timer 不会影响服务日志面板
  if (devLogState.closeTimerId !== null) {
    clearTimeout(devLogState.closeTimerId);
    devLogState.closeTimerId = null;
  }
  hideDevLogUI();
  origOpenLogsPanel(name);
};

// 开发工具日志事件绑定
var devViewLogsBtn = document.getElementById('dev-view-logs');
if (devViewLogsBtn) {
  devViewLogsBtn.addEventListener('click', function (event) {
    pulseClickFeedback(event.currentTarget);
    openDevLogsPanel('conf-sync');
  });
}

var devLogSelect = document.getElementById('dev-log-select');
if (devLogSelect) {
  devLogSelect.addEventListener('change', function () {
    devLogState.tool = devLogSelect.value;
    devLogState.requestSerial += 1;
    updateDevLogsMeta('loading...');
    document.getElementById('logs-content').textContent = 'Loading logs...';
    fetchDevLogsOnce();
  });
}

var clearDevLogsBtn = document.getElementById('logs-panel-clear-dev');
if (clearDevLogsBtn) {
  clearDevLogsBtn.addEventListener('click', function (event) {
    pulseClickFeedback(event.currentTarget);
    clearDevLogs();
  });
}

// ── Bootstrap（OPT-20260811-011）────────────────────────────────
// 顶层初始化统一在最后一个 JS 片段末尾执行，确保运行前所有 const *State
// 声明已初始化（observabilityState/devLogState/logsState/layoutState/…），
// 从根上消除「状态未初始化」TDZ 复发。
loadObservabilityBar();
refreshCollectionStatus();
refreshTraceShippingStatus();
refreshInternalAPIsSmokeStatus();
refresh();
restartStatusRefreshTimer();
