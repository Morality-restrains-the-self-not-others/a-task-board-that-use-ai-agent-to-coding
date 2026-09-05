var _sseRestartAllSource;
var _onBulkProgressDone;
var lifecyclePollToken;

function connectRestartAllSSE() {
  disconnectRestartAllSSE();
  showProgress('restart');
  var src = new EventSource('/api/restart-all/progress');
  _sseRestartAllSource = src;
  src.onmessage = function(e) {
    try {
      var ev = JSON.parse(e.data);
      if (ev.phase === 'idle') {
        disconnectRestartAllSSE();
        keepProgressPanelAfterStreamEnd();
        return;
      }
      // Restart-all is a single canary stream (ADR-0058).
      updateProgress(ev);
    } catch (err) {}
  };
  src.onerror = function() {
    disconnectRestartAllSSE();
    if (typeof _bulkSSEOnError === 'function' && _bulkSSEOnError) {
      _bulkSSEOnError('SSE 连接中断 (restart-all)');
    }
    var panel = document.getElementById('start-all-progress-panel');
    if (panel.classList.contains('is-active') && !panel.classList.contains('is-done')) {
      document.getElementById('prog-current-text').textContent = '连接中断，进度可能不完整';
      document.getElementById('prog-error').classList.add('is-visible');
      document.getElementById('prog-error').textContent = 'SSE 连接中断';
      refresh();
    }
  };
}

async function restartAllServices() {
  const confirmed = await showModalConfirm(
    '将按依赖顺序对每个服务做金丝雀平滑重启：新进程先就绪再排空旧进程（非整栈先停）。\n\n是否继续？'
  );
  if (!confirmed) {
    return;
  }
  try {
    const resp = await apiFetch('/api/restart-all', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({session_id: selectBulkActorSessionID()})
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      if (resp.status === 409) {
        await adoptInProgressBulkOp('restart', resp, result, '全部重启已在进行中，请等待完成或先中断后再试');
        return;
      }
      showRequestError('全部重启失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    if (result.status === 'accepted' || result.status === 'ok') {
      connectRestartAllSSE();
      refresh();
      return;
    }
    showRequestError('全部重启失败: unexpected response', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
  } catch (err) {
    showRequestError('全部重启失败: ' + err.message, err);
  }
}

async function buildAllServices() {
  const confirmed = await showModalConfirm(
    '将按依赖顺序编译全部可编译服务。\n\n是否继续？'
  );
  if (!confirmed) {
    return;
  }
  try {
    const resp = await apiFetch('/api/build-all', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'}
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
      refresh();
      return;
    }
    showRequestError('全部重新编译失败: unexpected response', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
  } catch (err) {
    showRequestError('全部重新编译失败: ' + err.message, err);
  }
}

// Core exec functions (no confirm dialogs) — used by the execution queue.
async function _execRestartAll() {
  try {
    const resp = await apiFetch('/api/restart-all', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({session_id: selectBulkActorSessionID()})
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      if (resp.status === 409) {
        await adoptInProgressBulkOp('restart', resp, result, '全部重启已在进行中');
        return;
      }
      showRequestError('全部重启失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
        var cb = _onBulkProgressDone; _onBulkProgressDone = null;
        cb(result.error || 'HTTP ' + resp.status);
      }
      return;
    }
    if (result.status === 'accepted' || result.status === 'ok') {
      connectRestartAllSSE();
      refresh();
      return;
    }
    showRequestError('全部重启失败: unexpected response', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
    if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
      var cb2 = _onBulkProgressDone; _onBulkProgressDone = null;
      cb2('unexpected response');
    }
  } catch (err) {
    showRequestError('全部重启失败: ' + err.message, err);
    if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
      var cb3 = _onBulkProgressDone; _onBulkProgressDone = null;
      cb3(err.message);
    }
  }
}

async function _execBuildAll() {
  try {
    const resp = await apiFetch('/api/build-all', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'}
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      if (resp.status === 409) {
        await adoptInProgressBulkOp('build', resp, result, '已有全部/分组重新编译进行中');
        return;
      }
      showRequestError('全部重新编译失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
        var cb = _onBulkProgressDone; _onBulkProgressDone = null;
        cb(result.error || 'HTTP ' + resp.status);
      }
      return;
    }
    if (result.status === 'accepted' || result.status === 'ok') {
      connectBuildAllSSE();
      refresh();
      return;
    }
    showRequestError('全部重新编译失败: unexpected response', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
    if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
      var cb2 = _onBulkProgressDone; _onBulkProgressDone = null;
      cb2('unexpected response');
    }
  } catch (err) {
    showRequestError('全部重新编译失败: ' + err.message, err);
    if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
      var cb3 = _onBulkProgressDone; _onBulkProgressDone = null;
      cb3(err.message);
    }
  }
}

async function shutdownSelf() {
  const confirmed = await showModalConfirm(
    '将关闭 runAll 服务本身，但其管理的服务将继续运行。\n\n是否继续？'
  );
  if (!confirmed) {
    return;
  }
  try {
    const resp = await apiFetch('/api/shutdown-self', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'}
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError('关闭自身失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    await showModalAlert('runAll 正在关闭，管理的服务将继续运行。页面将无法继续刷新。', {type: 'warning', opLabel: '关闭 runAll 自身'});
  } catch (err) {
    showRequestError('关闭自身失败: ' + err.message, err);
  }
}

async function clearLogs(name) {
  try {
    const resp = await apiFetch('/api/logs/clear', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({name: name})
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError('Clear logs failed: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    if (logsState.open && logsState.service === name) {
      fetchLogsOnce();
    }
    refresh();
  } catch (err) {
    showRequestError('Clear logs failed: ' + err.message, err);
  }
}

function getSessionID() {
  const storageKey = 'runall_session_id';
  const existing = (window.localStorage && localStorage.getItem(storageKey)) || '';
  if (existing && existing.trim()) return existing;
  const generated = (window.crypto && typeof crypto.randomUUID === 'function')
    ? crypto.randomUUID()
    : `session-${Date.now()}`;
  if (window.localStorage) localStorage.setItem(storageKey, generated);
  return generated;
}

/** Owning session from latest /api/status row (set after hot-replace / bootstrap). */
function lookupOwningSessionID(serviceName) {
  const name = typeof serviceName === 'string' ? serviceName.trim() : '';
  if (!name || !Array.isArray(lastStatusData)) {
    return '';
  }
  for (const svc of lastStatusData) {
    if (!svc || String(svc.name || '') !== name) {
      continue;
    }
    const sid = typeof svc.session_id === 'string' ? svc.session_id.trim() : '';
    return sid;
  }
  return '';
}

/** Prefer button data-session-id, then status owning session, then local browser session. */
function selectActorSessionID(explicitSessionID, serviceName) {
  const candidate = typeof explicitSessionID === 'string' ? explicitSessionID.trim() : '';
  if (candidate) {
    return candidate;
  }
  const owning = lookupOwningSessionID(serviceName);
  if (owning) {
    return owning;
  }
  return getSessionID();
}

/** For start-all / stop-all / group: prefer a known owning session from status rows. */
function selectBulkActorSessionID() {
  const counts = new Map();
  if (Array.isArray(lastStatusData)) {
    for (const svc of lastStatusData) {
      const sid = typeof svc?.session_id === 'string' ? svc.session_id.trim() : '';
      if (!sid) {
        continue;
      }
      counts.set(sid, (counts.get(sid) || 0) + 1);
    }
  }
  let best = '';
  let bestCount = 0;
  for (const [sid, n] of counts) {
    if (n > bestCount) {
      best = sid;
      bestCount = n;
    }
  }
  if (best) {
    return best;
  }
  return getSessionID();
}

async function pollLifecycleStatus(name, expectStartable) {
  const token = ++lifecyclePollToken;
  const deadline = Date.now() + lifecyclePollTimeoutMs;
  while (Date.now() < deadline) {
    if (token !== lifecyclePollToken) {
      return true;
    }
    await refresh();
    const status = getServiceStatus(name);
    if (status && isServiceStartable(status) === expectStartable) {
      return true;
    }
    await new Promise((resolve) => setTimeout(resolve, lifecyclePollMs));
  }
  return false;
}

async function postServiceAction(url, name, label, includeSession, explicitSessionID) {
  try {
    const previousStatus = getServiceStatus(name);
    const payload = {name: name, cascade: true};
    if (includeSession) {
      payload.session_id = selectActorSessionID(explicitSessionID, name);
    }
    const resp = await apiFetch(url, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify(payload)
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError(formatActionFailure(label, result), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    if (result.status === 'accepted' || result.status === 'ok') {
      // If response includes run_id, connect SSE for live progress
      if (result.run_id) {
        connectSingleProgressSSE(result.run_id, label, name, url);
      }
      const expectStartable = url.includes('/api/stop')
        ? true
        : url.includes('/api/start')
          ? false
          : isServiceStartable(previousStatus);
      if (name && (url.includes('/api/stop') || url.includes('/api/start'))) {
        const settled = await pollLifecycleStatus(name, expectStartable);
        if (!settled) {
          showRequestError(`${label} 已提交，但 ${name} 状态未在 ${lifecyclePollTimeoutMs / 1000}s 内更新；请查看 runAll 日志或服务行错误信息`);
        }
        return;
      }
      refresh();
      return;
    }
    showRequestError(formatActionFailure(label, result), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
  } catch (err) {
    showRequestError(`${label} failed: ` + err.message, err);
  }
}

function formatActionFailure(label, result) {
  let message = `${label} failed: ` + (result.error || 'unknown error');
  const errText = String(result.error || '');
  const ownedMatch = errText.match(/owned by session "([^"]+)"/);
  if (ownedMatch) {
    message += `\n该服务归属 session: ${ownedMatch[1]}。` +
      `\nUI 重启/启停会优先使用服务行 owning session；若仍失败可 takeover:\n` +
      `  runAll -command takeover -service <name> -session-id <your-session-id>`;
  }
  if (result.cascade && typeof result.cascade === 'object') {
    const failedAt = result.cascade.failed_at || '';
    const completed = Array.isArray(result.cascade.completed) ? result.cascade.completed.join(', ') : '';
    if (failedAt) {
      message += `\nfailed at: ${failedAt}`;
    }
    if (completed) {
      message += `\ncompleted: ${completed}`;
    }
  }
  return message;
}

async function postGroupAction(url, group, label) {
  try {
    const resp = await apiFetch(url, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({group: group, session_id: selectBulkActorSessionID()})
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError(`${label} failed: ` + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    if (result.status === 'accepted' || result.status === 'ok') {
      refresh();
      return;
    }
    showRequestError(`${label} failed: unexpected response`, resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
  } catch (err) {
    showRequestError(`${label} failed: ` + err.message, err);
  }
}

