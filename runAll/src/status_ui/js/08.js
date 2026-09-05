var _sseStartAllSource;
var _sseStopAllSource;
var _sseBuildAllSource;
var _bulkProgressResumed;
var _onBulkProgressDone;

function connectStartAllSSE() {
  disconnectStartAllSSE();
  showProgress('start');

  var src = new EventSource('/api/start-all/progress');
  _sseStartAllSource = src;

  src.onmessage = function(e) {
    try {
      var ev = JSON.parse(e.data);
      if (ev.phase === 'idle') {
        disconnectStartAllSSE();
        keepProgressPanelAfterStreamEnd();
        return;
      }
      updateProgress(ev);
    } catch (err) {
      // ignore parse errors
    }
  };

  src.onerror = function() {
    disconnectStartAllSSE();
    // Notify queue on SSE connection loss
    if (typeof _bulkSSEOnError === 'function' && _bulkSSEOnError) {
      _bulkSSEOnError('SSE 连接中断 (start-all)');
    }
    // If the panel is still active, mark as interrupted
    var panel = document.getElementById('start-all-progress-panel');
    if (panel.classList.contains('is-active') && !panel.classList.contains('is-done')) {
      document.getElementById('prog-current-text').textContent = '连接中断，进度可能不完整';
      document.getElementById('prog-error').classList.add('is-visible');
      document.getElementById('prog-error').textContent = 'SSE 连接中断';
      refresh();
    }
  };
}

function connectStopAllSSE() {
  disconnectStopAllSSE();
  showProgress('stop');
  var src = new EventSource('/api/stop-all/progress');
  _sseStopAllSource = src;
  src.onmessage = function(e) {
    try {
      var ev = JSON.parse(e.data);
      if (ev.phase === 'idle') {
        disconnectStopAllSSE();
        keepProgressPanelAfterStreamEnd();
        return;
      }
      updateProgress(ev);
    } catch (err) {}
  };
  src.onerror = function() {
    disconnectStopAllSSE();
    if (typeof _bulkSSEOnError === 'function' && _bulkSSEOnError) {
      _bulkSSEOnError('SSE 连接中断 (stop-all)');
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


function disconnectBuildAllSSE() {
  if (_sseBuildAllSource) {
    _sseBuildAllSource.close();
    _sseBuildAllSource = null;
  }
}

function connectBuildAllSSE() {
  disconnectBuildAllSSE();
  showProgress('build');
  var src = new EventSource('/api/build-all/progress');
  _sseBuildAllSource = src;
  src.onmessage = function(e) {
    try {
      var ev = JSON.parse(e.data);
      if (ev.phase === 'idle') {
        disconnectBuildAllSSE();
        keepProgressPanelAfterStreamEnd();
        return;
      }
      updateProgress(ev);
    } catch (err) {}
  };
  src.onerror = function() {
    disconnectBuildAllSSE();
    if (typeof _bulkSSEOnError === 'function' && _bulkSSEOnError) {
      _bulkSSEOnError('SSE 连接中断 (build-all)');
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

// 409「已有 bulk 进行中」时挂上进度条，而不是只显示错误横幅。
async function adoptInProgressBulkOp(kind, resp, result, message) {
  showRequestError(message, resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
  if (typeof _bulkProgressResumed !== 'undefined') {
    _bulkProgressResumed = false;
  }
  if (kind === 'build') connectBuildAllSSE();
  else if (kind === 'restart') connectRestartAllSSE();
  else if (kind === 'stop') connectStopAllSSE();
  else if (kind === 'start') connectStartAllSSE();
  else if (kind === 'precise-restart') connectPreciseRestartSSE();
  else if (kind === 'init-db' || kind === 'clear-db') {
    // OPT-20260820-007: 清库/初始化 409 时接管已在进行的 run 的 SSE，进度条续跑。
    showProgress(kind);
    var devSnap = result && result.active_bulk_progress;
    if (devSnap && devSnap.run_id && typeof _openDevDBProgressSSE === 'function') {
      _openDevDBProgressSSE(devSnap.run_id, function (ev) {
        if (kind === 'init-db' && typeof _finishInitAllDatabases === 'function') {
          _finishInitAllDatabases(ev);
        } else if (kind === 'clear-db' && typeof _finishClearAllDatabases === 'function') {
          _finishClearAllDatabases(ev);
        } else if (typeof updateProgress === 'function') {
          updateProgress(ev);
        }
      });
    }
  }
  var snap = result && result.active_bulk_progress;
  if (snap && snap.event && typeof updateProgress === 'function') {
    updateProgress(snap.event);
  }
  if (typeof refresh === 'function') {
    await refresh();
  }
}

async function startAllServices() {
  try {
    const resp = await apiFetch('/api/start-all', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({session_id: selectBulkActorSessionID()})
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      if (resp.status === 409) {
        await adoptInProgressBulkOp('start', resp, result, '全部启动已在进行中，请等待完成或先中断后再试');
        return;
      }
      showRequestError('Start all failed: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    if (result.status === 'accepted' || result.status === 'ok') {
      connectStartAllSSE();
      refresh();
      return;
    }
    showRequestError('Start all failed: unexpected response', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
  } catch (err) {
    showRequestError('Start all failed: ' + err.message, err);
  }
}

async function stopAllServices() {
  const confirmed = await showModalConfirm(
    '将按依赖顺序关闭全部服务。\n\n是否继续？'
  );
  if (!confirmed) {
    return;
  }
  try {
    const resp = await apiFetch('/api/stop-all', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({session_id: selectBulkActorSessionID()})
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      if (resp.status === 409) {
        await adoptInProgressBulkOp('stop', resp, result, '全部关闭已在进行中，请等待完成或先中断后再试');
        return;
      }
      showRequestError('Stop all failed: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    if (result.status === 'accepted' || result.status === 'ok') {
      connectStopAllSSE();
      refresh();
      return;
    }
    showRequestError('Stop all failed: unexpected response', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
  } catch (err) {
    showRequestError('Stop all failed: ' + err.message, err);
  }
}

// Core exec functions (no confirm dialogs) — used by the execution queue.
async function _execStartAll() {
  try {
    const resp = await apiFetch('/api/start-all', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({session_id: selectBulkActorSessionID()})
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      if (resp.status === 409) {
        await adoptInProgressBulkOp('start', resp, result, '全部启动已在进行中');
        return;
      }
      showRequestError('全部启动失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      // Signal queue: error
      if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
        var cb = _onBulkProgressDone; _onBulkProgressDone = null;
        cb(result.error || 'HTTP ' + resp.status);
      }
      return;
    }
    if (result.status === 'accepted' || result.status === 'ok') {
      connectStartAllSSE();
      refresh();
      return;
    }
    showRequestError('全部启动失败: unexpected response', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
    if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
      var cb2 = _onBulkProgressDone; _onBulkProgressDone = null;
      cb2('unexpected response');
    }
  } catch (err) {
    showRequestError('全部启动失败: ' + err.message, err);
    if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
      var cb3 = _onBulkProgressDone; _onBulkProgressDone = null;
      cb3(err.message);
    }
  }
}

async function _execStopAll() {
  try {
    const resp = await apiFetch('/api/stop-all', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({session_id: selectBulkActorSessionID()})
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      if (resp.status === 409) {
        await adoptInProgressBulkOp('stop', resp, result, '全部关闭已在进行中');
        return;
      }
      showRequestError('全部关闭失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
        var cb = _onBulkProgressDone; _onBulkProgressDone = null;
        cb(result.error || 'HTTP ' + resp.status);
      }
      return;
    }
    if (result.status === 'accepted' || result.status === 'ok') {
      connectStopAllSSE();
      refresh();
      return;
    }
    showRequestError('全部关闭失败: unexpected response', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
    if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
      var cb2 = _onBulkProgressDone; _onBulkProgressDone = null;
      cb2('unexpected response');
    }
  } catch (err) {
    showRequestError('全部关闭失败: ' + err.message, err);
    if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
      var cb3 = _onBulkProgressDone; _onBulkProgressDone = null;
      cb3(err.message);
    }
  }
}

var _sseRestartAllSource = null;

function disconnectRestartAllSSE() {
  if (_sseRestartAllSource) {
    _sseRestartAllSource.close();
    _sseRestartAllSource = null;
  }
}
