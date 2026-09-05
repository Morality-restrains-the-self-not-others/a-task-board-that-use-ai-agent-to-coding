const startTime = Date.now();
let statusRefreshMs = 2000;
const statusRefreshLogsOpenMultiplier = 3;
const logLines = 200;
const minLogsPanelWidth = 320;
const minServicePanelWidth = 380;
const clickFeedbackDurationMs = 300;
const dotClass = {
  healthy: 'green', starting: 'yellow', retrying: 'yellow', restarting: 'yellow',
  pending: 'gray', stopped: 'gray', failed: 'red', skipped: 'dark'
};

function serviceDotClass(svc) {
  if (svc && svc.status === 'healthy' && svc.readiness && svc.readiness !== 'ready') {
    return 'yellow';
  }
  return dotClass[svc.status] || 'gray';
}

function isServiceFullyHealthy(svc) {
  if (!svc || svc.status === 'pending' || svc.status === '' || svc.status !== 'healthy') {
    return false;
  }
  if (svc.readiness && svc.readiness !== 'ready') {
    return false;
  }
  return true;
}

function groupHealthDotClass(services) {
  if (!Array.isArray(services) || services.length === 0) {
    return 'gray';
  }
  let anyFailed = false;
  let anyTransitional = false;
  let anyDegraded = false;
  for (const svc of services) {
    const status = typeof svc.status === 'string' ? svc.status.trim() : '';
    if (status === '' || status === 'pending') {
      continue;
    }
    if (svc.status === 'failed') {
      anyFailed = true;
      continue;
    }
    if (status === 'starting' || status === 'retrying' || status === 'restarting') {
      anyTransitional = true;
      continue;
    }
    if (svc.status === 'healthy' && svc.readiness && svc.readiness !== 'ready') {
      anyDegraded = true;
    }
  }
  if (anyFailed) {
    return 'red';
  }
  if (anyTransitional || anyDegraded) {
    return 'yellow';
  }
  if (services.every(isServiceFullyHealthy)) {
    return 'green';
  }
  return 'gray';
}

function groupHealthSummary(services) {
  const healthyCount = services.filter(isServiceFullyHealthy).length;
  return `${healthyCount}/${services.length} 正常运行`;
}

function isGroupCollapsed(groupKey) {
  return groupUIState[groupKey] === true;
}

function toggleGroupCollapsed(groupKey) {
  groupUIState[groupKey] = !isGroupCollapsed(groupKey);
  renderStatus(lastStatusData);
}

function collapseAllGroups() {
  const data = lastStatusData;
  if (!Array.isArray(data) || data.length === 0) return;
  const groupKeys = new Set();
  for (const svc of data) {
    const groupName = typeof svc.group === 'string' ? svc.group.trim() : '';
    groupKeys.add(groupName || 'ungrouped');
  }
  const allCollapsed = [...groupKeys].every(k => isGroupCollapsed(k));
  for (const key of groupKeys) {
    groupUIState[key] = !allCollapsed;
  }
  renderStatus(lastStatusData);
  updateCollapseAllButton(!allCollapsed);
}

function updateCollapseAllButton(collapsed) {
  const btn = document.getElementById('collapse-all-btn');
  if (btn) {
    btn.textContent = collapsed ? '展开全部' : '折叠全部';
  }
}

function formatServiceStatus(svc) {
  const raw = typeof svc.status === 'string' ? svc.status.trim() : '';
  if (raw === '' || raw === 'pending') {
    if (!svc.readiness) {
      return '—';
    }
    if (svc.readiness === 'ready') {
      return '— · ready';
    }
    if (svc.readiness === 'degraded') {
      return '— · not ready';
    }
    return `— · ${svc.readiness}`;
  }
  const live = raw;
  if (!svc.readiness) {
    return live;
  }
  if (svc.readiness === 'ready') {
    return `${live} · ready`;
  }
  if (svc.readiness === 'degraded') {
    return `${live} · not ready`;
  }
  return `${live} · ${svc.readiness}`;
}
const logsState = {
  open: false,
  service: '',
  timerId: null,
  requestSerial: 0,
  lastRows: [],
};
// Shared UI state must live here (before 06.js bootstrap calls). Declaring these
// later (e.g. 09.js / 13.js) causes TDZ: restartStatusRefreshTimer() sync-reads
// devLogState via currentStatusRefreshMs(), aborts script eval, then
// loadObservabilityBar() fails with "Cannot access 'observabilityState' before initialization".
const observabilityState = {
  grafanaUrl: '',
  lokiUrl: '',
  lokiPushUrl: '',
  logShipping: '',
  grafanaLokiExplore: '',
  grafanaTempoExplore: '',
  logFileRoot: '',
};
const devLogState = {
  open: false,
  tool: 'conf-sync',
  timerId: null,
  closeTimerId: null,
  requestSerial: 0,
};
const layoutState = {
  resizing: false,
};
const groupUIState = {};
const clickFeedbackTimers = new WeakMap();
const lifecyclePollMs = 1000;
const lifecyclePollTimeoutMs = 8000;
var lastProxyState = false;
let lastDAGBootInProgress = false;
let lastStatusData = [];
let lastStatusEnvelope = null;
let statusFetchInFlight = null;
let statusRefreshTimerId = null;
var lifecyclePollToken = 0;

function currentStatusRefreshMs() {
  if (logsState.open || devLogState.open) {
    return statusRefreshMs * statusRefreshLogsOpenMultiplier;
  }
  return statusRefreshMs;
}

function applyServerPollInterval(data) {
  const ms = data && Number.isFinite(data.poll_interval_ms) ? data.poll_interval_ms : null;
  if (ms && ms >= 500 && ms <= 30000) {
    statusRefreshMs = ms;
  }
}

function restartStatusRefreshTimer() {
  if (statusRefreshTimerId !== null) {
    clearTimeout(statusRefreshTimerId);
    statusRefreshTimerId = null;
  }
  statusRefreshTimerId = setTimeout(async function () {
    await refresh();
    restartStatusRefreshTimer();
  }, currentStatusRefreshMs());
}

function isServiceStartable(status) {
  const normalized = typeof status === 'string' ? status.trim() : '';
  return normalized === ''
    || normalized === 'stopped'
    || normalized === 'failed'
    || normalized === 'skipped'
    || normalized === 'pending';
}

function getServiceStatus(name, data) {
  const rows = Array.isArray(data) ? data : lastStatusData;
  const match = rows.find((item) => item && item.name === name);
  return match && typeof match.status === 'string' ? match.status : '';
}

async function fetchStatusData() {
  if (statusFetchInFlight) {
    return statusFetchInFlight;
  }
  statusFetchInFlight = (async () => {
    const resp = await fetch('/api/status');
    if (!resp.ok) {
      throw new Error('HTTP ' + resp.status);
    }
    const data = await resp.json();
    // Support both legacy (array) and new (wrapper object) response formats
    if (Array.isArray(data)) {
      lastProxyState = false;
      lastStatusEnvelope = { services: data };
      return data;
    }
    if (data && Array.isArray(data.services)) {
      lastStatusEnvelope = data;
      lastProxyState = typeof data.use_proxy === 'boolean' ? data.use_proxy : false;
      lastDAGBootInProgress = data.dag_boot_in_progress === true;
      applyServerPollInterval(data);
      renderProxyToggle();
      updateStartAllButtonState();
      return data.services;
    }
    throw new Error('invalid status payload');
  })().finally(() => {
    statusFetchInFlight = null;
  });
  return statusFetchInFlight;
}

function pulseClickFeedback(buttonEl) {
  if (!buttonEl) {
    return;
  }
  const oldTimer = clickFeedbackTimers.get(buttonEl);
  if (oldTimer) {
    clearTimeout(oldTimer);
  }
  buttonEl.classList.remove('is-clicked');
  // Force a reflow so fast repeated clicks still retrigger the animation.
  void buttonEl.offsetWidth;
  buttonEl.classList.add('is-clicked');
  const timerId = setTimeout(() => {
    buttonEl.classList.remove('is-clicked');
    clickFeedbackTimers.delete(buttonEl);
  }, clickFeedbackDurationMs);
  clickFeedbackTimers.set(buttonEl, timerId);
}

async function refresh() {
  try {
    const data = await fetchStatusData();
    lastStatusData = data;
    renderStatus(data);
    resumeActiveBulkProgress(lastStatusEnvelope);
    refreshPreciseRestartRegistrations();
    if (typeof refreshTraeAgentPushStatus === 'function') {
      refreshTraeAgentPushStatus();
    }
  } catch (err) {
    // Keep showing last successful state; update uptime only
    const elapsed = Math.floor((Date.now() - startTime) / 1000);
    document.getElementById('uptime').textContent = `Uptime: ${elapsed}s (fetch error)`;
  }
}

async function buildService(name) {
  try {
    const resp = await apiFetch('/api/build', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({name: name, cascade: true})
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError('编译失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    await showModalAlert(name + ' 编译完成', {type: 'success', opLabel: '编译服务: ' + name});
    refresh();
  } catch (err) {
    showRequestError('编译失败: ' + err.message, err);
  }
}

async function buildGroup(group) {
  const confirmed = await showModalConfirm(
    '将按依赖顺序编译本组可编译服务：' + group + '\n\n是否继续？'
  );
  if (!confirmed) {
    return;
  }
  try {
    const resp = await apiFetch('/api/build-group', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({group: group})
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      if (resp.status === 409) {
        await adoptInProgressBulkOp('build', resp, result, '已有全部/分组重新编译进行中，请等待完成或先中断后再试');
        return;
      }
      showRequestError('全部重新编译失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    if (result.status === 'accepted' || result.status === 'ok') {
      connectBuildAllSSE();
      var title = document.querySelector('.progress-title');
      if (title) title.textContent = '分组重新编译进度 (' + group + ')';
      refresh();
      return;
    }
    showRequestError('全部重新编译失败: unexpected response', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
  } catch (err) {
    showRequestError('全部重新编译失败: ' + err.message, err);
  }
}

async function restartService(name, sessionID) {
  await postServiceAction('/api/restart', name, 'Restart', true, sessionID);
}

async function stopService(name, sessionID) {
  // OPT-20260811-033: 单服务 stop 默认级联。先预览受影响服务，
  // 若会连带停止依赖服务（如 task-auth → task-gateway → taskFE）则弹确认，避免误伤全站。
  const dependents = await fetchStopPlanDependents(name, sessionID);
  if (dependents && dependents.length > 0) {
    const confirmed = await showModalConfirm(
      '停止 ' + name + ' 将连带停止以下依赖服务：\n' + dependents.join(', ') +
      '\n\n是否继续？'
    );
    if (!confirmed) {
      return;
    }
  }
  await postServiceAction('/api/stop', name, 'Stop', true, sessionID);
}

async function fetchStopPlanDependents(name, sessionID) {
  try {
    const resp = await apiFetch('/api/stop', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        name: name,
        cascade: true,
        preview: true,
        session_id: selectActorSessionID(sessionID, name)
      })
    });
    if (!resp.ok) {
      return null;
    }
    const result = await parseJsonSafe(resp);
    if (result && result.status === 'preview' && Array.isArray(result.services)) {
      return result.services.filter((s) => s !== name);
    }
    return null;
  } catch (err) {
    return null;
  }
}

async function startService(name, sessionID) {
  await postServiceAction('/api/start', name, 'Start', true, sessionID);
}

async function stopGroup(group) {
  await postGroupAction('/api/stop-group', group, 'Stop group');
}

async function startGroup(group) {
  await postGroupAction('/api/start-group', group, 'Start group');
}

async function generateConfReplica() {
  try {
    const resp = await apiFetch('/api/conf/sync', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: '{}'
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError('生成配置副本失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    if (result.status === 'accepted' || result.status === 'ok') {
      await showModalAlert('已开始生成配置副本（conf-sync），请查看 conf-sync 日志', {type: 'success'});
      refresh();
      return;
    }
    showRequestError('生成配置副本失败: unexpected response', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
  } catch (err) {
    showRequestError('生成配置副本失败: ' + err.message, err);
  }
}

var _sseStartAllSource = null;
var _sseStopAllSource = null;
var _sseBuildAllSource = null;
var _handledProgressDoneKey = '';

function disconnectStartAllSSE() {
  if (_sseStartAllSource) {
    _sseStartAllSource.close();
    _sseStartAllSource = null;
  }
}

