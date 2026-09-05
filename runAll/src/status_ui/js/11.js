// status_ui 拆分块（OPT-20260810-040）：precise-restart SSE 独立功能块，经 manifest.json 拼接后与主文件同 scope 执行。

// ===================== 精准编译重启 (precise restart) =====================
// 智能体编程会话修改服务代码后，将服务登记到 .runall/precise_restart_services.txt；
// 点击「精准编译重启」按钮：读取登记 → 逐个编译+重启（依赖序）→ 清空登记文件。
let _ssePreciseRestartSource = null;
var _onBulkProgressDone;

function disconnectPreciseRestartSSE() {
  if (_ssePreciseRestartSource) {
    _ssePreciseRestartSource.close();
    _ssePreciseRestartSource = null;
  }
}

function connectPreciseRestartSSE() {
  disconnectPreciseRestartSSE();
  showProgress('precise-restart');
  var src = new EventSource('/api/precise-restart/progress');
  _ssePreciseRestartSource = src;
  src.onmessage = function(e) {
    try {
      var ev = JSON.parse(e.data);
      if (ev.phase === 'idle') {
        disconnectPreciseRestartSSE();
        keepProgressPanelAfterStreamEnd();
        return;
      }
      updateProgress(ev);
    } catch (err) {}
  };
  src.onerror = function() {
    disconnectPreciseRestartSSE();
    if (typeof _bulkSSEOnError === 'function' && _bulkSSEOnError) {
      _bulkSSEOnError('SSE 连接中断 (precise-restart)');
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

function preciseRestartIsEmptyResponse(resp, result) {
  if (result && result.status === 'empty') return true;
  var err = (result && result.error) ? String(result.error) : '';
  return !!(resp && resp.status === 400 && err.indexOf('no registered services') !== -1);
}

function finishPreciseRestartQueue(err) {
  if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
    var cb = _onBulkProgressDone;
    _onBulkProgressDone = null;
    cb(err || null);
  }
}

// Core exec function (no confirm dialogs) — used by the execution queue.
async function _execPreciseRestart() {
  try {
    const resp = await apiFetch('/api/precise-restart', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({session_id: selectBulkActorSessionID()})
    });
    const result = await parseJsonSafe(resp);
    if (preciseRestartIsEmptyResponse(resp, result)) {
      showEmptyPreciseRestartRegistry(result, resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      finishPreciseRestartQueue();
      return;
    }
    if (!resp.ok) {
      if (resp.status === 409) {
        await adoptInProgressBulkOp('precise-restart', resp, result, '精准编译重启已在进行中');
        return;
      }
      showRequestError('精准编译重启失败: ' + (result.error || `${resp.status}`), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      finishPreciseRestartQueue(result.error || 'HTTP ' + resp.status);
      return;
    }
    if (result.status === 'accepted' || result.status === 'ok') {
      connectPreciseRestartSSE();
      refreshPreciseRestartRegistrations();
      refresh();
      return;
    }
    showRequestError('精准编译重启失败: unexpected response', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
    finishPreciseRestartQueue('unexpected response');
  } catch (err) {
    showRequestError('精准编译重启失败: ' + err.message, err);
    finishPreciseRestartQueue(err.message);
  }
}

// 登记列表刷新：按钮旁标签（已登记 N 个服务）+ 按钮 title 展示清单。
// OPT-20260807-024/026：区分失败保留项（N 个失败待重试）、展示登记时间，
// 超过 TTL（默认 24h）的过期登记将按钮置灰提示重新登记。
let _preciseRestartRegsShown = '';
function _preciseRestartTime(unixSec) {
  if (!unixSec) return '';
  var d = new Date(unixSec * 1000);
  if (Number.isNaN(d.getTime())) return '';
  var pad = function (n) { return String(n).padStart(2, '0'); };
  return (pad(d.getMonth() + 1) + '-' + pad(d.getDate()) + ' ' + pad(d.getHours()) + ':' + pad(d.getMinutes()));
}
async function refreshPreciseRestartRegistrations() {
  var label = document.getElementById('precise-restart-reg-label');
  var btn = document.getElementById('precise-restart-btn');
  try {
    var resp = await apiFetch('/api/precise-restart/registrations');
    var result = await parseJsonSafe(resp);
    var entries = (result && Array.isArray(result.entries)) ? result.entries : [];
    var svcs = entries.map(function (e) { return e.name; });
    var shown = svcs.join('、');
    if (shown !== _preciseRestartRegsShown) {
      _preciseRestartRegsShown = shown;
      var ttlHours = (result && result.ttl_hours) ? result.ttl_hours : 24;
      var failedCount = entries.filter(function (e) { return e.state === 'failed'; }).length;
      var expiredCount = entries.filter(function (e) { return e.expired; }).length;
      var titleParts = entries.map(function (e) {
        var parts = [e.name];
        if (e.state === 'failed') parts.push('失败待重试');
        else if (e.expired) parts.push('已过期');
        if (e.resolvable === false) parts.push('未知服务');
        else if (e.buildable === false) parts.push('不可编译');
        var tm = _preciseRestartTime(e.registered_at);
        if (tm) parts.push(tm);
        return parts.join(' ');
      });
      if (label) {
        if (svcs.length) {
          var txt = '已登记 ' + svcs.length + ' 个服务';
          if (failedCount) txt += '，' + failedCount + ' 个失败待重试';
          if (expiredCount) txt += '，' + expiredCount + ' 个已过期';
          label.textContent = txt;
          label.title = '已登记: ' + titleParts.join('；');
        } else {
          label.textContent = '';
          label.title = (result && result.file_path) ? ('登记文件: ' + result.file_path) : '';
        }
      }
      if (btn) {
        if (!svcs.length) {
          btn.disabled = false;
          btn.classList.remove('is-disabled');
          btn.title = '精准编译重启：编译后金丝雀平滑重启登记服务（新进程先就绪再排空旧进程），完成后清空登记';
        } else if (expiredCount > 0) {
          btn.disabled = true;
          btn.classList.add('is-disabled');
          btn.title = '含 ' + expiredCount + ' 个超过 ' + ttlHours + 'h 的过期登记，请先重新登记（scripts/register-precise-restart.sh）\n' + titleParts.join('；');
        } else {
          btn.disabled = false;
          btn.classList.remove('is-disabled');
          btn.title = '精准编译重启（已登记: ' + titleParts.join('；') + '）';
        }
      }
    }
  } catch (e) {
    // 登记接口暂不可用时不打扰页面
  }
}
