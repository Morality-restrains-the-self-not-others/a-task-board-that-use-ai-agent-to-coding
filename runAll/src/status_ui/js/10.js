// status_ui 拆分块（OPT-20260810-040）：formatStepLines 独立功能块，经 manifest.json 拼接后与主文件同 scope 执行。
var _onBulkProgressDone;

function formatStepLines(label, steps) {
  if (!steps || steps.length === 0) {
    return `${label}: (无)`;
  }
  return `${label}:\n` + steps.map((s) => `  - ${s.database}: ${s.status}${s.message ? ' (' + s.message + ')' : ''}`).join('\n');
}

async function clearAllDatabases() {
  const confirmed = await showModalConfirm(
    '将：① 先关闭 runAll 中所有正在运行的应用（含 ai-monitor、前端、task-events 等；保留 docker-redis / docker-kafka / docker-mysql）② DROP 全部用户 MySQL 库（含测试残留）并重建 registry 空库、清除 binlog ③ 删除残留 SQLite 文件（含 wal/shm）④ FLUSHALL Redis ⑤ 重建 Kafka topics。\n\n不会执行 migrate/种子。完成后请再点「初始化全部数据库」。不会自动启动服务。是否继续？',
    {opLabel: '清空全部数据库'}
  );
  if (!confirmed) {
    return;
  }
  try {
    const resp = await apiFetch('/api/dev/clear-databases?confirm=CLEAR_ALL', {method: 'POST'});
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError('清空数据库失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    const stillRunning = result.services_still_running || [];
    const summary = [
      `status: ${result.status || 'unknown'}`,
      `stopped: ${(result.services_stopped || []).join(', ') || '-'}`,
      stillRunning.length ? `still running: ${stillRunning.join(', ')}` : '',
      `sqlite: ${(result.sqlite_removed || []).join(', ') || '-'}`,
      `mysql: ${result.mysql_reset || '-'}`,
      `redis: ${result.redis_reset || '-'}`,
      `kafka: ${result.kafka_reset || '-'}`,
    ].filter(Boolean).join('\n');
    const hint = stillRunning.length
      ? '\n\n仍有服务未关闭，已中止删库。请先在 runAll 手动关闭上述服务，或检查是否由 runAll 外部启动。'
      : '\n\n下一步：点击「初始化全部数据库（开发）」执行 migrate 与 init.sh。';
    await showModalAlert(`数据库清空结果:\n${summary}${hint}`, {type: (stillRunning.length ? 'warning' : 'success'), opLabel: '清空全部数据库'});
    refresh();
  } catch (err) {
    showRequestError('清空数据库失败: ' + err.message, err);
  }
}

// Core exec function (no confirm dialog) — used by the execution queue.
// OPT-20260812-049: 后端异步化，POST 返回 {status:'accepted', run_id}，
// 完成信号与完整结果 Detail 经 /api/progress?run_id=... SSE 送达。
async function _execClearAllDatabases() {
  showProgress('clear-db');
  document.getElementById('prog-current-line').style.display = '';
  document.getElementById('prog-current-text').textContent = '正在停止服务并清空数据库...';
  document.getElementById('prog-remaining').textContent = '等待中';
  document.getElementById('prog-started').textContent = '执行中';
  // Poll dev tool logs for real-time step updates（保留作为实时文案源）
  var pollTimer = setInterval(async function() {
    try {
      var pollResp = await fetch('/api/dev/logs?tool=db-clear&lines=1');
      var pollData = await parseJsonSafe(pollResp);
      if (pollData.lines && pollData.lines.length > 0) {
        var line = pollData.lines[0];
        var prefix = '[db-clear] ';
        if (line.indexOf(prefix) === 0) line = line.substring(prefix.length);
        var txt = document.getElementById('prog-current-text');
        if (txt) txt.textContent = line;
      }
    } catch(_) { /* ignore poll errors */ }
  }, 800);
  try {
    const resp = await apiFetch('/api/dev/clear-databases?confirm=CLEAR_ALL', {method: 'POST'});
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      clearInterval(pollTimer);
      // OPT-20260820-007: 409 意味着已有 init/clear 在进行中，接管其进度条/SSE 而非标失败。
      if (resp.status === 409) {
        await adoptInProgressBulkOp('clear-db', resp, result, '清库已在进行中，请等待完成后再试');
        return;
      }
      showRequestError('清空数据库失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      finalizeDevDBProgress('clear-db', result.error || 'HTTP ' + resp.status, 1, 1);
      return;
    }
    if (result.status !== 'accepted' || !result.run_id) {
      clearInterval(pollTimer);
      showRequestError('清空数据库: 意外响应 ' + JSON.stringify(result), '');
      finalizeDevDBProgress('clear-db', '意外响应', 1, 1);
      return;
    }
    _openDevDBProgressSSE(result.run_id, function (ev) {
      clearInterval(pollTimer);
      _finishClearAllDatabases(ev);
    });
  } catch (err) {
    clearInterval(pollTimer);
    showRequestError('清空数据库失败: ' + err.message, err);
    finalizeDevDBProgress('clear-db', err.message, 1, 1);
  }
}

// _finishClearAllDatabases 处理 SSE 完成事件：用 ev.detail（完整结果）汇总展示。
function _finishClearAllDatabases(ev) {
  // Guard: if the panel was taken over by another operation or cancelled, skip UI updates
  var panel = document.getElementById('start-all-progress-panel');
  var opMismatch = panel && panel._progressOp && panel._progressOp !== 'clear-db';
  var wasCancelled = panel && panel.classList.contains('is-cancelled');
  var skipPanelUpdate = opMismatch || wasCancelled;
  var result = ev.detail || {};
  if (ev.error && !result.status) {
    showRequestError('清空数据库失败: ' + ev.error, '');
    if (!skipPanelUpdate) {
      updateProgress({done: true, phase: 'error', error: ev.error, operation: 'clear-db', total: 1, failed: 1, started: 0, remaining: 0});
    }
    if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
      var cb = _onBulkProgressDone; _onBulkProgressDone = null;
      cb(ev.error);
    }
    return;
  }
  const stillRunning = result.services_still_running || [];
  var succeeded = (result.status === 'ok' || result.status === 'completed') && stillRunning.length === 0;
  var msgParts = [];
  if (result.services_stopped && result.services_stopped.length) {
    msgParts.push('已停止: ' + result.services_stopped.join(', '));
  }
  if (result.sqlite_removed && result.sqlite_removed.length) {
    msgParts.push('SQLite: ' + result.sqlite_removed.length + ' 个文件已清除');
  }
  if (result.mysql_reset) msgParts.push('MySQL: ' + result.mysql_reset);
  if (result.redis_reset) msgParts.push('Redis: ' + result.redis_reset);
  if (result.kafka_reset) msgParts.push('Kafka: ' + result.kafka_reset);
  var summary = msgParts.join(' | ') || '完成';
  if (stillRunning.length) {
    summary += ' (仍有 ' + stillRunning.length + ' 个服务未关闭)';
  }
  if (!skipPanelUpdate) {
    updateProgress({done: true, phase: succeeded ? 'done' : 'error', operation: 'clear-db', total: 1, started: 1, failed: succeeded ? 0 : 1, remaining: 0, error: stillRunning.length ? ('仍有 ' + stillRunning.length + ' 个服务未关闭') : ''});
  }
  showToast('清空数据库: ' + summary, { type: succeeded ? 'success' : 'warning', duration: 8000 });
  refresh();
  if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
    var cb2 = _onBulkProgressDone; _onBulkProgressDone = null;
    cb2(null);
  }
}


