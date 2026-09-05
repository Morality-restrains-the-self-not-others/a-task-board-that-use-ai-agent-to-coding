var _sseStopAllSource;
var _handledProgressDoneKey;
var _onBulkProgressDone;

function buildProgressLogsText() {
  var lines = [];
  var title = document.querySelector('.progress-title');
  if (title && title.textContent) lines.push(title.textContent.trim());
  var started = document.getElementById('prog-started');
  var remaining = document.getElementById('prog-remaining');
  var failed = document.getElementById('prog-failed');
  var startedLbl = document.getElementById('prog-started-label');
  lines.push(
    ((startedLbl && startedLbl.textContent) || '已完成') + ': ' + ((started && started.textContent) || '0') +
    ', 剩余: ' + ((remaining && remaining.textContent) || '0') +
    ', 失败: ' + ((failed && failed.textContent) || '0')
  );
  var current = document.getElementById('prog-current-text');
  if (current && current.textContent) lines.push(current.textContent.trim());
  var latest = document.getElementById('prog-error');
  if (latest && latest.classList.contains('is-visible') && latest.textContent) {
    lines.push(latest.textContent.trim());
  }
  var items = document.querySelectorAll('#prog-errors-ul li');
  for (var i = 0; i < items.length; i++) {
    if (items[i].textContent) lines.push(items[i].textContent);
  }
  return lines.filter(Boolean).join('\n');
}

function flashProgressCopyBtn(ok) {
  var btn = document.getElementById('prog-copy-logs-btn');
  if (!btn) return;
  var orig = btn.getAttribute('data-orig-label') || btn.textContent;
  btn.setAttribute('data-orig-label', orig);
  btn.textContent = ok ? '已复制!' : '复制失败';
  if (ok) btn.classList.add('is-copied');
  setTimeout(function() {
    btn.textContent = orig;
    btn.classList.remove('is-copied');
  }, 1500);
}

function copyProgressLogsToClipboard() {
  var text = buildProgressLogsText();
  if (!text) {
    flashProgressCopyBtn(false);
    return;
  }
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).then(function() {
      flashProgressCopyBtn(true);
    }).catch(function() {
      fallbackCopyProgressLogs(text);
    });
  } else {
    fallbackCopyProgressLogs(text);
  }
}

// 兼容旧 onclick / 测试片段名
function copyErrorsToClipboard() {
  copyProgressLogsToClipboard();
}

function fallbackCopyProgressLogs(text) {
  var ta = document.createElement('textarea');
  ta.value = text;
  ta.style.position = 'fixed';
  ta.style.left = '-9999px';
  document.body.appendChild(ta);
  ta.select();
  try {
    flashProgressCopyBtn(!!document.execCommand('copy'));
  } catch (e) {
    flashProgressCopyBtn(false);
  }
  document.body.removeChild(ta);
}

function updateProgressCopyButton(visible) {
  var btn = document.getElementById('prog-copy-logs-btn');
  if (!btn) return;
  btn.style.display = visible ? '' : 'none';
}

function disconnectStopAllSSE() {
  if (_sseStopAllSource) {
    _sseStopAllSource.close();
    _sseStopAllSource = null;
  }
}

function showProgress(op) {
  op = op || 'start';
  var panel = document.getElementById('start-all-progress-panel');
  panel.classList.add('is-active');
  panel.classList.remove('is-done', 'has-errors', 'is-cancelled', 'is-fading');
  document.getElementById('prog-started').textContent = '0';
  document.getElementById('prog-remaining').textContent = '...';
  document.getElementById('prog-failed').textContent = '0';
  document.getElementById('prog-failed-stat').style.display = 'none';
  document.getElementById('prog-skipped-stat').style.display = 'none';
  document.getElementById('prog-skipped').textContent = '0';
  document.getElementById('prog-bar').style.width = '0%';
  document.getElementById('prog-bar').classList.remove('has-errors');
  document.getElementById('prog-current-line').style.display = 'none';
  document.getElementById('prog-current-text').textContent = '';
  document.getElementById('prog-error').classList.remove('is-visible');
  document.getElementById('prog-error').textContent = '';
  document.getElementById('prog-error').removeAttribute('data-traceId');
  document.getElementById('prog-errors-list').style.display = 'none';
  document.getElementById('prog-errors-ul').innerHTML = '';
  // Update title based on operation
  var title = document.querySelector('.progress-title');
  if (title) {
    if (op === 'stop') title.textContent = '全部关闭进度';
    else if (op === 'build') title.textContent = '全部重新编译进度';
    else if (op === 'restart') title.textContent = '全部重启进度';
    else if (op === 'precise-restart') title.textContent = '精准编译重启进度';
    else if (op === 'clear-db') title.textContent = '清空数据库进度';
    else if (op === 'init-db') title.textContent = '初始化数据库进度';
    else title.textContent = '全部启动进度';
  }
  var startedLbl = document.getElementById('prog-started-label');
  if (startedLbl) {
    if (op === 'build') startedLbl.textContent = '已编译';
    else if (op === 'stop') startedLbl.textContent = '已关闭';
    else if (op === 'restart') startedLbl.textContent = '已完成';
    else if (op === 'precise-restart') startedLbl.textContent = '已完成';
    else if (op === 'clear-db' || op === 'init-db') startedLbl.textContent = '状态';
    else startedLbl.textContent = '已启动';
  }
  // For single-step operations (clear-db / init-db), show indeterminate progress bar
  var bar = document.getElementById('prog-bar');
  if (bar) {
    if (op === 'clear-db' || op === 'init-db') {
      bar.classList.add('is-indeterminate');
    } else {
      bar.classList.remove('is-indeterminate');
    }
  }
  panel._progressOp = op;
  _handledProgressDoneKey = '';
  updateProgressCancelButton(true, false);
  updateProgressCopyButton(false);
  updateProgressClearButton(false);
}

function updateProgressCancelButton(visible, disabled) {
  var btn = document.getElementById('prog-cancel-btn');
  if (!btn) return;
  btn.style.display = visible ? '' : 'none';
  btn.disabled = !!disabled;
  btn.textContent = disabled ? '正在中断…' : '中断';
}

function cancelOpFromQueueType(type) {
  if (type === 'precise-restart') return 'precise-restart';
  if (type === 'restart-all') return 'restart';
  if (type === 'build-all') return 'build';
  if (type === 'stop-all') return 'stop';
  if (type === 'start-all') return 'start';
  return '';
}

async function cancelProgressOperation(forcedOp) {
  var panel = document.getElementById('start-all-progress-panel');
  var op = forcedOp || (panel && panel._progressOp) || 'start';
  var panelActive = panel && panel.classList.contains('is-active') && !panel.classList.contains('is-done');
  // Queue interrupt may run before/without progress panel; still cancel backend.
  if (!panelActive && !forcedOp) {
    return;
  }
  var url = '/api/start-all/cancel';
  if (op === 'stop') url = '/api/stop-all/cancel';
  else if (op === 'build') url = '/api/build-all/cancel';
  else if (op === 'restart') url = '/api/restart-all/cancel';
  else if (op === 'precise-restart') url = '/api/precise-restart/cancel';
  else if (op === 'clear-db' || op === 'init-db') {
    // These operations have no server-side cancel; mark cancelled and keep logs until 清空.
    // The queue interrupt will handle _onBulkProgressDone cleanup.
    if (panel) {
      panel.classList.add('is-done', 'is-cancelled');
      updateProgressCancelButton(false, false);
      keepProgressPanelAfterStreamEnd();
    }
    return;
  }
  if (panelActive) updateProgressCancelButton(true, true);
  try {
    const resp = await apiFetch(url, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'}
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      if (panelActive) updateProgressCancelButton(true, false);
      showRequestError('中断失败: ' + (result.error || resp.status), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
  } catch (err) {
    if (panelActive) updateProgressCancelButton(true, false);
    showRequestError('中断失败: ' + err.message, err);
  }
}

function updateProgress(ev) {
  var panel = document.getElementById('start-all-progress-panel');
  var op = ev.operation || 'start';
  // Keep bulk restart chrome while underlying stop/start events stream in.
  if (panel && panel._progressOp === 'restart') {
    op = 'restart';
  }
  if (panel && panel._progressOp === 'precise-restart') {
    op = 'precise-restart';
  }
  panel.classList.add('is-active');
  // Remove indeterminate animation once real progress events arrive
  var bar = document.getElementById('prog-bar');
  if (bar) bar.classList.remove('is-indeterminate');

  document.getElementById('prog-started').textContent = ev.started;
  document.getElementById('prog-remaining').textContent = ev.remaining;

  var skipped = ev.skipped || 0;
  if (skipped > 0) {
    document.getElementById('prog-skipped-stat').style.display = '';
    document.getElementById('prog-skipped').textContent = skipped;
  }

  if (ev.failed > 0) {
    document.getElementById('prog-failed-stat').style.display = '';
    document.getElementById('prog-failed').textContent = ev.failed;
    document.getElementById('prog-bar').classList.add('has-errors');
    panel.classList.add('has-errors');
  }

  var total = ev.total;
  var done = ev.started + ev.failed + skipped;
  var pct = total > 0 ? Math.round((done / total) * 100) : 0;
  document.getElementById('prog-bar').style.width = pct + '%';

  var actionText = '正在启动: ';
  if (op === 'stop') actionText = '正在关闭: ';
  else if (op === 'build') actionText = '正在编译: ';
  else if (op === 'restart') {
    if (ev.operation === 'stop') actionText = '正在关闭: ';
    else actionText = '正在启动: ';
  }
  else if (op === 'precise-restart') actionText = '正在编译重启: ';
  if (ev.current) {
    document.getElementById('prog-current-line').style.display = '';
    document.getElementById('prog-current-text').textContent = actionText + ev.current;
  } else if (!ev.done) {
    document.getElementById('prog-current-line').style.display = '';
    document.getElementById('prog-current-text').textContent = '等待当前批次完成...';
  } else {
    document.getElementById('prog-current-line').style.display = 'none';
  }

  if (ev.error) {
    var latestErr = document.getElementById('prog-error');
    latestErr.classList.add('is-visible');
    latestErr.textContent = '最新错误: ' + ev.error;
    latestErr.title = ev.error;
    // Stamp run_id so ops can jump from the banner to the matching runAll logs in Loki.
    // Only set when a real run_id/trace is present; never fall back to "unknown".
    if (ev.run_id) {
      latestErr.setAttribute('data-traceId', ev.run_id);
    } else {
      latestErr.removeAttribute('data-traceId');
    }
    updateProgressCopyButton(true);
  }
  if (ev.errors && ev.errors.length > 0) {
    var ul = document.getElementById('prog-errors-ul');
    ul.innerHTML = '';
    for (var i = 0; i < ev.errors.length; i++) {
      var li = document.createElement('li');
      li.textContent = ev.errors[i];
      ul.appendChild(li);
    }
    document.getElementById('prog-errors-list').style.display = '';
    updateProgressCopyButton(true);
  } else if (ev.failed > 0) {
    // failed 计数已出但 errors 尚未到达时仍提供复制入口（复制当前进度摘要）
    updateProgressCopyButton(true);
  }

  if (typeof bulkQueue !== 'undefined' && bulkQueue.applyProgress) {
    bulkQueue.applyProgress(ev, op);
  }

  if (ev.done) {
    var doneKey = progressDoneKey(ev);
    var doneReplay = isProgressDoneReplay(_handledProgressDoneKey, ev);
    _handledProgressDoneKey = doneKey;
    panel.classList.add('is-done');
    document.getElementById('prog-current-line').style.display = 'none';
    updateProgressCancelButton(false, false);
    if (ev.phase === 'cancelled') {
      panel.classList.add('is-cancelled');
      document.getElementById('prog-error').classList.add('is-visible');
      if (op === 'stop') document.getElementById('prog-error').textContent = '全部关闭操作已中断';
      else if (op === 'build') document.getElementById('prog-error').textContent = '全部重新编译操作已中断';
      else if (op === 'restart') document.getElementById('prog-error').textContent = '全部重启操作已中断';
      else if (op === 'precise-restart') document.getElementById('prog-error').textContent = '精准编译重启操作已中断';
      else document.getElementById('prog-error').textContent = '全部启动操作已中断';
    }
    if (!doneReplay) {
      if (ev.failed > 0 && ev.errors && ev.errors.length > 0) {
        for (var ei = 0; ei < ev.errors.length; ei++) {
          if (typeof showToast === 'function') showToast(ev.errors[ei], {type: 'error', duration: 8000});
        }
      } else if (ev.error && typeof showToast === 'function') {
        showToast(ev.error, {type: 'error', duration: 8000});
      }
      if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
        var cb = _onBulkProgressDone;
        _onBulkProgressDone = null;
        var queueErr = null;
        if (ev.failed > 0) {
          queueErr = (ev.errors && ev.errors.length > 0) ? ev.errors[0] : (ev.error || '部分服务执行失败');
        } else if (ev.phase === 'cancelled') {
          queueErr = null;
        }
        cb(queueErr);
      }
    }
    if (op === 'stop') {
      disconnectStopAllSSE();
    } else if (op === 'build') {
      disconnectBuildAllSSE();
    } else if (op === 'restart') {
      disconnectRestartAllSSE();
    } else if (op === 'precise-restart') {
      disconnectPreciseRestartSSE();
    } else {
      disconnectStartAllSSE();
    }
    if (!doneReplay) {
      refresh();
      keepProgressPanelAfterStreamEnd();
    }
  }
}

function connectSingleProgressSSE(runID, label, name, url) {
  // Determine operation type from URL or label
  var op = 'start';
  if (url && url.includes('stop')) op = 'stop';
  else if (url && url.includes('restart')) op = 'restart';

  showProgress(op);
  // For single service, show compact info (no bulk cancel)
  updateProgressCancelButton(false, false);
  document.getElementById('prog-current-line').style.display = '';
  document.getElementById('prog-current-text').textContent = '手动操作: ' + label.toLowerCase() + ' ' + name;
  document.getElementById('prog-started').textContent = '0';
  document.getElementById('prog-remaining').textContent = '1';

  var src = new EventSource('/api/progress?run_id=' + encodeURIComponent(runID));
  src.onmessage = function(e) {
    try {
      var ev = JSON.parse(e.data);
      if (ev.phase === 'idle') {
        src.close();
        return;
      }
      updateProgress(ev);
      if (ev.done) {
        src.close();
        keepProgressPanelAfterStreamEnd();
      }
    } catch (err) {}
  };
  src.onerror = function() {
    src.close();
  };
}

var _bulkProgressResumed = false;

function bulkKindLabel(kind) {
  if (kind === 'precise-restart') return '精准编译重启';
  if (kind === 'restart-all') return '全部重启';
  if (kind === 'build-all') return '全部重新编译';
  if (kind === 'stop-all') return '全部关闭';
  if (kind === 'start-all') return '全部启动';
  if (kind === 'clear-db') return '清空全部数据库';
  if (kind === 'init-db') return '初始化全部数据库';
  return kind || '批量操作';
}

function resolveBulkProgressSnap(data) {
  if (data && data.execution_queue && data.execution_queue.current && data.execution_queue.current.run_id) {
    return data.execution_queue.current;
  }
  if (data && data.active_bulk_progress && data.active_bulk_progress.run_id) {
    return data.active_bulk_progress;
  }
  return null;
}

function hydrateExecQueueFromStatus(data) {
  if (typeof bulkQueue === 'undefined' || !bulkQueue.adoptExternalOp) return;
  var snap = resolveBulkProgressSnap(data);
  if (!snap || !snap.run_id || (snap.event && snap.event.done)) {
    bulkQueue.clearExternalOpIfIdle();
  } else {
    bulkQueue.adoptExternalOp(snap.kind || '', snap.label || bulkKindLabel(snap.kind), snap.event || null);
  }
  var pending = data && data.execution_queue && data.execution_queue.pending;
  if (typeof bulkQueue.restorePending === 'function') {
    bulkQueue.restorePending(pending || []);
  }
}

// 将 /api/status 的 active_bulk_progress / execution_queue 同步到队列条（SSE 空窗或刷新后仍可见进度）。
function syncBulkQueueFromStatus(data) {
  hydrateExecQueueFromStatus(data);
}

function bulkProgressSSELive(kind, op) {
  if (kind === 'restart-all') return !!_sseRestartAllSource;
  if (kind === 'precise-restart') return !!_ssePreciseRestartSource;
  if (op === 'build') return !!_sseBuildAllSource;
  if (op === 'stop') return !!_sseStopAllSource;
  if (op === 'start') return !!_sseStartAllSource;
  return false;
}

function resumeActiveBulkProgress(data) {
  syncBulkQueueFromStatus(data);
  var snap = resolveBulkProgressSnap(data);
  if (!snap || !snap.run_id) {
    _bulkProgressResumed = false;
    return;
  }
  var kind = snap.kind || '';
  var op = snap.operation || '';
  if (!op) {
    if (kind === 'build-all') op = 'build';
    else if (kind === 'stop-all') op = 'stop';
    else if (kind === 'precise-restart') op = 'restart';
    else if (kind === 'init-db' || kind === 'clear-db') op = kind;
    else op = 'start';
  }
  if (shouldReconnectBulkProgressSSE(snap) && !bulkProgressSSELive(kind, op) && !((kind === 'init-db' || kind === 'clear-db') && _bulkProgressResumed)) {
    _bulkProgressResumed = true;
    if (kind === 'restart-all') {
      connectRestartAllSSE();
    } else if (kind === 'precise-restart') {
      connectPreciseRestartSSE();
    } else if (kind === 'init-db' || kind === 'clear-db') {
      showProgress(kind);
      if (typeof _openDevDBProgressSSE === 'function') {
        _openDevDBProgressSSE(snap.run_id, function (ev) {
          if (kind === 'init-db' && typeof _finishInitAllDatabases === 'function') {
            _finishInitAllDatabases(ev);
          } else if (kind === 'clear-db' && typeof _finishClearAllDatabases === 'function') {
            _finishClearAllDatabases(ev);
          } else if (typeof updateProgress === 'function') {
            updateProgress(ev);
          }
        });
      }
    } else if (op === 'build') {
      connectBuildAllSSE();
    } else if (op === 'stop') {
      connectStopAllSSE();
    } else {
      connectStartAllSSE();
    }
  }
  if (snap.event) updateProgress(snap.event);
}


