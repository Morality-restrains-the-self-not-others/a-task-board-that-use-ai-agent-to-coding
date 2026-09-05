var _onBulkProgressDone;

async function initAllDatabases() {
  const confirmed = await showModalConfirm(
    '将按 registry 顺序执行全部 migrate_script 与 db/<app>/init.sh（不删库）。\n\n请确保相关服务已停止，否则会拒绝执行。不会自动启动服务。是否继续？',
    {opLabel: '初始化全部数据库'}
  );
  if (!confirmed) {
    return;
  }
  try {
    const resp = await apiFetch('/api/dev/init-databases?confirm=INIT_ALL', {method: 'POST'});
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError('初始化数据库失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    const summary = [
      `status: ${result.status || 'unknown'}`,
      formatStepLines('migrate', result.migrations),
      formatStepLines('init', result.inits),
    ].join('\n');
    await showModalAlert(`数据库初始化完成:\n${summary}\n\n请在 runAll 手动启动 infrastructure / platform 组。`, {type: 'success', opLabel: '初始化全部数据库'});
    refresh();
  } catch (err) {
    showRequestError('初始化数据库失败: ' + err.message, err);
  }
}

// ── Dev 数据库操作异步化（OPT-20260812-049）────────────────────────
// clear/init 后端已改为 HTTP 202 + run_id 后台执行，进度/结果经
// /api/progress?run_id=... SSE 送达（Done 事件 Detail 携带完整结果）。
// 此处复用 generic ProgressBroadcaster，与 start-all/build-all 同一通道。

// _openDevDBProgressSSE 连接 SSE 并在 done/error 时回调 onDone(ev)。
function _openDevDBProgressSSE(runId, onDone) {
  var finished = false;
  var es = null;
  try {
    es = new EventSource('/api/progress?run_id=' + encodeURIComponent(runId));
  } catch (err) {
    onDone({error: 'SSE 连接失败: ' + (err && err.message || err)});
    return;
  }
  es.onmessage = function (event) {
    if (finished) return;
    var ev;
    try { ev = JSON.parse(event.data); } catch (_) { return; }
    if (ev.current) {
      var txt = document.getElementById('prog-current-text');
      if (txt) txt.textContent = ev.current;
    }
    if (typeof updateProgress === 'function' && ev && !ev.done) {
      updateProgress(ev);
    }
    if (ev.done) {
      finished = true;
      es.close();
      onDone(ev);
    }
  };
  es.onerror = function () {
    if (finished) return;
    finished = true;
    es.close();
    onDone({error: '进度 SSE 连接中断，请刷新页面查看「开发工具日志」确认结果'});
  };
}

// finalizeDevDBProgress 在出错/意外响应时统一收尾进度面板并通知执行队列。
function finalizeDevDBProgress(op, error, failed, total) {
  var panel = document.getElementById('start-all-progress-panel');
  var opMismatch = panel && panel._progressOp && panel._progressOp !== op;
  var wasCancelled = panel && panel.classList.contains('is-cancelled');
  if (opMismatch || wasCancelled) {
    if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
      var cb = _onBulkProgressDone; _onBulkProgressDone = null;
      cb(error || '失败');
    }
    return;
  }
  updateProgress({done: true, phase: 'error', error: error || '失败', operation: op, total: total || 1, failed: failed || 1, started: 0, remaining: 0});
  if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
    var cb2 = _onBulkProgressDone; _onBulkProgressDone = null;
    cb2(error || '失败');
  }
}

// Core exec function (no confirm dialog) — used by the execution queue.
async function _execInitAllDatabases() {
  showProgress('init-db');
  document.getElementById('prog-current-line').style.display = '';
  document.getElementById('prog-current-text').textContent = '正在执行 migrate 与 init 脚本...';
  document.getElementById('prog-remaining').textContent = '等待中';
  document.getElementById('prog-started').textContent = '执行中';
  // Poll dev tool logs for real-time step updates（保留作为实时文案源；
  // SSE 负责权威的完成信号与完整结果 Detail）。
  var pollTimer = setInterval(async function() {
    try {
      var pollResp = await fetch('/api/dev/logs?tool=db-init&lines=1');
      var pollData = await parseJsonSafe(pollResp);
      if (pollData.lines && pollData.lines.length > 0) {
        var line = pollData.lines[0];
        var prefix = '[db-init] ';
        if (line.indexOf(prefix) === 0) line = line.substring(prefix.length);
        var txt = document.getElementById('prog-current-text');
        if (txt) txt.textContent = line;
      }
    } catch(_) { /* ignore poll errors */ }
  }, 800);
  try {
    const resp = await apiFetch('/api/dev/init-databases?confirm=INIT_ALL', {method: 'POST'});
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      clearInterval(pollTimer);
      // OPT-20260820-007: 409 意味着已有 init/clear 在进行中，接管其进度条/SSE 而非标失败。
      if (resp.status === 409) {
        await adoptInProgressBulkOp('init-db', resp, result, '初始化数据库已在进行中，请等待完成后再试');
        return;
      }
      showRequestError('初始化数据库失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      finalizeDevDBProgress('init-db', result.error || 'HTTP ' + resp.status, 1, 1);
      return;
    }
    if (result.status !== 'accepted' || !result.run_id) {
      clearInterval(pollTimer);
      showRequestError('初始化数据库: 意外响应 ' + JSON.stringify(result), '');
      finalizeDevDBProgress('init-db', '意外响应', 1, 1);
      return;
    }
    _openDevDBProgressSSE(result.run_id, function (ev) {
      clearInterval(pollTimer);
      _finishInitAllDatabases(ev);
    });
  } catch (err) {
    clearInterval(pollTimer);
    showRequestError('初始化数据库失败: ' + err.message, err);
    finalizeDevDBProgress('init-db', err.message, 1, 1);
  }
}

// _finishInitAllDatabases 处理 SSE 完成事件：用 ev.detail（完整结果）汇总展示。
function _finishInitAllDatabases(ev) {
  // Guard: if the panel was taken over by another operation or cancelled, skip UI updates
  var panel = document.getElementById('start-all-progress-panel');
  var opMismatch = panel && panel._progressOp && panel._progressOp !== 'init-db';
  var wasCancelled = panel && panel.classList.contains('is-cancelled');
  var skipPanelUpdate = opMismatch || wasCancelled;
  var result = ev.detail || {};
  // blocked / panic / SSE 中断等无 Detail 场景
  if (ev.error && !result.status) {
    showRequestError('初始化数据库失败: ' + ev.error, '');
    if (!skipPanelUpdate) {
      updateProgress({done: true, phase: 'error', error: ev.error, operation: 'init-db', total: 1, failed: 1, started: 0, remaining: 0});
    }
    if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
      var cb = _onBulkProgressDone; _onBulkProgressDone = null;
      cb(ev.error);
    }
    return;
  }
  // Build detailed summary — uses formatStepLines (defined in 10.js) which includes
  // per-step error messages, so the user can see exactly which script failed and why.
  var summary = formatStepLines('migrate', result.migrations);
  summary += '\n' + formatStepLines('init', result.inits);
  // Count actual step failures for progress and toast severity
  var failedCount = 0;
  var totalCount = 0;
  if (result.migrations) {
    result.migrations.forEach(function(s) { if (s.status === 'failed') failedCount++; });
    totalCount += result.migrations.length;
  }
  if (result.inits) {
    result.inits.forEach(function(s) { if (s.status === 'failed') failedCount++; });
    totalCount += result.inits.length;
  }
  var succeeded = failedCount === 0 && (result.status === 'ok' || result.status === 'completed');
  if (!skipPanelUpdate) {
    updateProgress({done: true, phase: succeeded ? 'done' : 'error', operation: 'init-db', total: totalCount, started: totalCount, failed: failedCount, remaining: 0, error: failedCount > 0 ? (failedCount + ' 步失败') : ''});
  }
  var toastMsg = '初始化数据库: status=' + (result.status || 'unknown') + ' | ' + totalCount + ' 步';
  if (failedCount > 0) {
    toastMsg += ' | ' + failedCount + ' 失败';
  }
  showToast(toastMsg, { type: succeeded ? 'success' : 'warning', duration: failedCount > 0 ? 20000 : 8000 });
  // When there are step failures, show the detailed summary in an alert so the
  // user can see the exact error messages without opening dev tool logs.
  if (failedCount > 0) {
    showModalAlert('初始化数据库 — 部分步骤失败:\n\n' + summary + '\n\n请在 runAll 查看「初始化全部数据库」开发工具日志获得更多细节。', {type: 'error', title: '⚠️ 初始化数据库失败', opLabel: '初始化全部数据库（部分失败）'});
  }
  refresh();
  if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
    var cb2 = _onBulkProgressDone; _onBulkProgressDone = null;
    cb2(null);
  }
}


function normalizeLogRows(lines) {
  if (!Array.isArray(lines)) {
    return [];
  }
  return lines.map((line) => {
    if (typeof line === 'string') {
      return line;
    }
    if (line && typeof line === 'object') {
      const ts = line.timestamp ? `[${formatLogTime(line.timestamp)}] ` : '';
      const stream = line.stream ? `(${line.stream}) ` : '';
      const message = line.message !== undefined ? String(line.message) : JSON.stringify(line);
      return `${ts}${stream}${message}`;
    }
    return String(line);
  });
}

async function parseJsonSafe(resp) {
  try {
    return await resp.json();
  } catch (_) {
    return {};
  }
}

function newRequestTraceId() {
  try {
    if (typeof crypto !== 'undefined' && crypto.randomUUID) {
      return crypto.randomUUID();
    }
  } catch (_) { /* ignore */ }
  return 'runall-' + Date.now() + '-' + Math.random().toString(36).slice(2, 12);
}

function extractTraceId(source) {
  if (source == null || source === '') return '';
  if (typeof source === 'string') return source.trim();
  if (typeof source !== 'object') return '';
  if (typeof source.traceId === 'string' && source.traceId.trim()) return source.traceId.trim();
  if (typeof source.trace_id === 'string' && source.trace_id.trim()) return source.trace_id.trim();
  return '';
}

function resolveTraceIdFromResponse(resp, requestTraceId, body) {
  try {
    const h = resp && resp.headers && (resp.headers.get('X-Trace-Id') || resp.headers.get('x-trace-id'));
    if (h && String(h).trim()) return String(h).trim();
  } catch (_) { /* ignore */ }
  const fromBody = extractTraceId(body);
  if (fromBody) return fromBody;
  return requestTraceId || '';
}

/** 带 X-Trace-Id 的 fetch；失败时 Error.traceId 可供 showRequestError 使用 */
async function apiFetch(url, opts = {}) {
  const requestTraceId = newRequestTraceId();
  const headers = Object.assign({ 'X-Trace-Id': requestTraceId }, opts.headers || {});
  let resp;
  try {
    resp = await fetch(url, Object.assign({}, opts, { headers }));
  } catch (err) {
    if (err && typeof err === 'object') err.traceId = requestTraceId;
    throw err;
  }
  resp.requestTraceId = requestTraceId;
  return resp;
}

function showRequestError(message, sourceOrTraceId) {
  const banner = document.getElementById('requestErrorBanner');
  if (!banner) {
    console.error('[runAll]', message);
    return;
  }
  const tid = extractTraceId(sourceOrTraceId);
  banner.textContent = String(message ?? '请求失败');
  if (tid) banner.setAttribute('data-traceId', tid);
  else banner.removeAttribute('data-traceId');
  banner.classList.add('is-visible');
}

// Toast Notification System 已抽至 17.js（OPT-20260823-025），
// 本文件不再重复定义，避免双份状态导致 DOM 上限失效。
// Execution queue (bulkQueue) lives in 07.js (loaded after 04, before 05).

function updateLogsMeta(text) {
  document.getElementById('logs-panel-meta').textContent = `${logsState.service || '-'} - ${text} - last ${logLines} lines`;
}

async function copyLogsToClipboard() {
  const contentEl = document.getElementById('logs-content');
  const text = contentEl ? contentEl.textContent : '';
  if (!text || text.trim() === '' || text.includes('Select a service to view logs')) {
    updateLogsMeta('copy failed: no logs');
    return;
  }
  try {
    if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
      await navigator.clipboard.writeText(text);
    } else {
      // Fallback for non-secure contexts (http://IP:port)
      fallbackCopyTextToClipboard(text);
    }
    updateLogsMeta(`copied at ${formatNowTime()}`);
  } catch (err) {
    updateLogsMeta(`copy failed: ${err.message || 'unknown error'}`);
  }
}

function fallbackCopyTextToClipboard(text) {
  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.style.position = 'fixed';
  textarea.style.left = '-9999px';
  textarea.style.top = '0';
  textarea.setAttribute('readonly', '');
  document.body.appendChild(textarea);
  textarea.select();
  textarea.setSelectionRange(0, textarea.value.length);
  const success = document.execCommand('copy');
  document.body.removeChild(textarea);
  if (!success) {
    throw new Error('execCommand copy failed');
  }
}

function formatNowTime() {
  return formatDateTime(new Date());
}

function formatLogTime(ts) {
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) {
    return String(ts);
  }
  return formatDateTime(d);
}

function formatDateTime(d) {
  const pad = (n) => String(n).padStart(2, '0');
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function formatTime(ts) {
  const d = new Date(ts);
  const pad = (n) => String(n).padStart(2, '0');
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function esc(s) {
  const el = document.createElement('span');
  el.textContent = s;
  return el.innerHTML;
}

function escAttr(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;');
}

function normalizePortValue(port) {
  if (port === undefined || port === null) {
    return '';
  }
  const value = String(port).trim();
  if (value === '') {
    return '';
  }
  for (const ch of value) {
    if (ch < '0' || ch > '9') {
      return '';
    }
  }
  return value;
}

function resolveHealthHref(url, fallbackPort) {
  const candidateURL = typeof url === 'string' ? url.trim() : '';
  if (candidateURL && !candidateURL.startsWith('tcp://')) {
    return candidateURL;
  }
  if (!fallbackPort) {
    return '';
  }
  return `http://localhost:${fallbackPort}`;
}

function normalizePortHref(href) {
  const trimmedHref = typeof href === 'string' ? href.trim() : '';
  if (!trimmedHref) {
    return '';
  }
  try {
    const parsed = new URL(trimmedHref);
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      return '';
    }
    return parsed.href;
  } catch (_) {
    return '';
  }
}

function parseTcpProbeLabel(url) {
  const trimmed = typeof url === 'string' ? url.trim() : '';
  if (!trimmed.startsWith('tcp://')) {
    return '';
  }
  const label = trimmed.slice('tcp://'.length).trim();
  return label;
}

function httpEndpointLabel(href) {
  const normalizedHref = normalizePortHref(href);
  if (!normalizedHref) {
    return '';
  }
  try {
    const parsed = new URL(normalizedHref);
    const port = parsed.port || (parsed.protocol === 'https:' ? '443' : '80');
    return `${parsed.hostname}:${port}`;
  } catch (_) {
    return '';
  }
}

function healthHostFromService(svc) {
  const healthPort = normalizePortValue(svc.health_port);
  const healthHref = normalizePortHref(resolveHealthHref(svc.url, healthPort));
  if (!healthHref) {
    return 'localhost';
  }
  try {
    return new URL(healthHref).hostname || 'localhost';
  } catch (_) {
    return 'localhost';
  }
}

function healthPortCell(svc) {
  const tcpLabel = parseTcpProbeLabel(svc.url);
  if (tcpLabel) {
    return esc(tcpLabel);
  }
  const healthPort = normalizePortValue(svc.health_port);
  const healthHref = resolveHealthHref(svc.url, healthPort);
  return portLink(healthHref, healthPort);
}

function commandPortCell(svc) {
  const commandPort = normalizePortValue(svc.command_port);
  if (!commandPort) {
    return '-';
  }
  const host = healthHostFromService(svc);
  return portLink(`http://${host}:${commandPort}`, commandPort);
}

function portLink(href, fallbackPort = '') {
  let normalizedHref = normalizePortHref(href);
  if (!normalizedHref && fallbackPort) {
    normalizedHref = normalizePortHref(resolveHealthHref('', fallbackPort));
  }
  const label = httpEndpointLabel(normalizedHref) || normalizePortValue(fallbackPort);
  if (!label) {
    return '-';
  }
  if (!normalizedHref) {
    return esc(label);
  }
  return `<a class="port-link" href="${escAttr(normalizedHref)}" target="_blank" rel="noopener noreferrer">${esc(label)}</a>`;
}

function setLogsPanelWidth(px) {
  var workspace = document.getElementById('workspace');
  var maxWidth = Math.min(workspace.clientWidth * 0.75, workspace.clientWidth - 16);
  var clamped = Math.min(Math.max(px, minLogsPanelWidth), maxWidth);
  var panel = document.getElementById('logs-panel');
  panel.style.width = clamped + 'px';
}

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

