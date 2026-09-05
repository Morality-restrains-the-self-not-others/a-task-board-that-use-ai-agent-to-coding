// status_ui 拆分块（OPT-20260810-040）：openLogsPanel 独立功能块，经 manifest.json 拼接后与主文件同 scope 执行。

function openLogsPanel(name) {
  logsState.open = true;
  logsState.service = name;
  logsState.requestSerial += 1;
  var workspace = document.getElementById('workspace');
  workspace.classList.add('logs-open');
  var panel = document.getElementById('logs-panel');
  panel.getBoundingClientRect(); /* force reflow so transition animates from hidden state */
  panel.setAttribute('aria-hidden', 'false');
  var savedWidth = parseFloat(panel.style.width);
  if (!savedWidth || isNaN(savedWidth)) {
    setLogsPanelWidth(workspace.clientWidth * 0.38);
  }
  updateLogsMeta('loading...');
  document.getElementById('logs-content').textContent = 'Loading logs...';
  stopLogsAutoRefresh();
  fetchLogsOnce();
  logsState.timerId = setInterval(fetchLogsOnce, statusRefreshMs);
  restartStatusRefreshTimer();
}

function closeLogsPanel() {
  logsState.open = false;
  logsState.service = '';
  logsState.lastRows = [];
  logsState.requestSerial += 1;
  stopLogsAutoRefresh();
  var panel = document.getElementById('logs-panel');
  panel.setAttribute('aria-hidden', 'true');
  restartStatusRefreshTimer();
  setTimeout(function () {
    if (!logsState.open) {
      document.getElementById('workspace').classList.remove('logs-open');
    }
  }, 180);
}

function stopLogsAutoRefresh() {
  if (logsState.timerId !== null) {
    clearInterval(logsState.timerId);
    logsState.timerId = null;
  }
}

async function fetchLogsOnce() {
  if (!logsState.open || !logsState.service) {
    return;
  }
  const reqId = ++logsState.requestSerial;
  const name = logsState.service;
  try {
    const query = new URLSearchParams({name, lines: String(logLines)});
    const resp = await fetch(`/api/logs?${query.toString()}`);
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      throw new Error(result.error || `HTTP ${resp.status}`);
    }
    if (!logsState.open || logsState.service !== name || reqId !== logsState.requestSerial) {
      return;
    }
    renderLogs(result);
  } catch (err) {
    if (!logsState.open || logsState.service !== name || reqId !== logsState.requestSerial) {
      return;
    }
    updateLogsMeta('refresh failed');
    document.getElementById('logs-content').textContent = `Failed to load logs: ${err.message}`;
  }
}

function renderLogs(payload) {
  const name = payload && payload.name ? String(payload.name) : logsState.service;
  const rows = normalizeLogRows(payload && payload.lines ? payload.lines : []);
  logsState.lastRows = rows;
  const content = document.getElementById('logs-content');
  // Auto-scroll to bottom only when user was already near the bottom (within 1 screen height)
  const wasNearBottom = content.scrollHeight - content.scrollTop - content.clientHeight < content.clientHeight;
  content.textContent = rows.length > 0 ? rows.join('\n') : 'No logs available.';
  if (wasNearBottom) {
    content.scrollTop = content.scrollHeight;
  }
  updateLogsMeta(`${rows.length} lines - refreshed at ${formatNowTime()}`);
  document.getElementById('logs-panel-title').textContent = `${name} logs`;
}

function extractTraceIdFromText(text) {
  const raw = String(text || '');
  const jsonMatch = raw.match(/"trace_id"\s*:\s*"([^"]+)"/);
  if (jsonMatch && jsonMatch[1]) {
    return jsonMatch[1];
  }
  const bracketMatch = raw.match(/\[trace_id=([^\]]+)\]/);
  if (bracketMatch && bracketMatch[1]) {
    return bracketMatch[1];
  }
  return '';
}

function extractTraceIdFromLogRows(rows) {
  if (!Array.isArray(rows)) {
    return '';
  }
  for (let i = rows.length - 1; i >= 0; i -= 1) {
    const tid = extractTraceIdFromText(rows[i]);
    if (tid) {
      return tid;
    }
  }
  return '';
}

