// status_ui 拆分块（OPT-20260812-004）：trae-agent 镜像推送（约束 46 对称 42）独立功能块。
// 按钮「推送 trae-agent 镜像」：读取 .runall 下推送状态（pending/运行锁/水位线），
// 触发 scripts/trae-agent-docker-push.sh --if-pending --background，SSE tail 日志展示进度。

let _sseTraeAgentPushSource = null;

function disconnectTraeAgentPushSSE() {
  if (_sseTraeAgentPushSource) {
    _sseTraeAgentPushSource.close();
    _sseTraeAgentPushSource = null;
  }
}

function connectTraeAgentPushSSE() {
  disconnectTraeAgentPushSSE();
  showProgress('trae-agent-push');
  var src = new EventSource('/api/trae-agent-push/progress');
  _sseTraeAgentPushSource = src;
  src.onmessage = function (e) {
    try {
      var ev = JSON.parse(e.data);
      if (ev.phase === 'idle' || ev.phase === 'timeout' || ev.phase === 'done') {
        disconnectTraeAgentPushSSE();
        if (ev.phase === 'done' || ev.phase === 'timeout') {
          updateProgress(Object.assign({}, ev, { done: true }));
        }
        keepProgressPanelAfterStreamEnd();
        refreshTraeAgentPushStatus();
        refresh();
        return;
      }
      if (ev.phase === 'error') {
        disconnectTraeAgentPushSSE();
        showRequestError('推送 trae-agent 镜像失败: ' + (ev.error || '未知错误'));
        keepProgressPanelAfterStreamEnd();
        refreshTraeAgentPushStatus();
        return;
      }
      updateProgress(ev);
    } catch (err) {}
  };
  src.onerror = function () {
    disconnectTraeAgentPushSSE();
    if (typeof _bulkSSEOnError === 'function' && _bulkSSEOnError) {
      _bulkSSEOnError('SSE 连接中断 (trae-agent-push)');
    }
    refreshTraeAgentPushStatus();
  };
}

async function _execTraeAgentPush() {
  try {
    const resp = await apiFetch('/api/trae-agent-push', {method: 'POST'});
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      if (resp.status === 409) {
        showRequestError('trae-agent 镜像推送已在进行中', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
        connectTraeAgentPushSSE();
      } else if (resp.status === 400) {
        showRequestError('无待推送变更：' + (result.error || 'pending 登记不存在'), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      } else {
        showRequestError('推送 trae-agent 镜像失败: ' + (result.error || (result.status + ' ' + resp.status)), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      }
      refreshTraeAgentPushStatus();
      return;
    }
    if (result.status === 'accepted') {
      connectTraeAgentPushSSE();
      refreshTraeAgentPushStatus();
      refresh();
      return;
    }
    showRequestError('推送 trae-agent 镜像失败: unexpected response', resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
    refreshTraeAgentPushStatus();
  } catch (err) {
    showRequestError('推送 trae-agent 镜像失败: ' + err.message, err);
    refreshTraeAgentPushStatus();
  }
}

let _traeAgentPushShown = '';
async function refreshTraeAgentPushStatus() {
  var btn = document.getElementById('trae-agent-push-btn');
  var label = document.getElementById('trae-agent-push-label');
  if (!btn && !label) return;
  try {
    var resp = await apiFetch('/api/trae-agent-push/status');
    var result = await parseJsonSafe(resp);
    if (!resp.ok) {
      btn.disabled = true;
      btn.classList.add('is-disabled');
      if (label) { label.textContent = '状态接口不可用'; label.title = ''; }
      return;
    }
    var pending = !!result.pending;
    var running = !!result.running;
    var sha = result.last_sha || '';
    var tail = (result.log_tail && result.log_tail.length) ? result.log_tail[result.log_tail.length - 1] : '';
    var shown = (pending ? 'pending' : 'idle') + '|' + (running ? 'running' : '') + '|' + sha;
    if (shown !== _traeAgentPushShown) {
      _traeAgentPushShown = shown;
      if (running) {
        btn.disabled = true;
        btn.classList.add('is-disabled');
        btn.title = 'trae-agent 镜像推送进行中（日志见 .runall/trae_agent_docker_push.log）';
        if (label) label.textContent = '推送中…';
      } else if (pending) {
        btn.disabled = false;
        btn.classList.remove('is-disabled');
        btn.title = '有未推送的 trae-agent 镜像变更，点击推送' + (sha ? '（上次推送 ' + sha.slice(0, 8) + '）' : '');
        if (label) label.textContent = '有未推送变更';
      } else {
        btn.disabled = false;
        btn.classList.remove('is-disabled');
        btn.title = '推送 trae-agent 镜像：在 trae-agent/onlineServiceJS 执行 DOCKER_PUSH=1 ./buildDocker.sh' + (sha ? '（上次推送 ' + sha.slice(0, 8) + '）' : '');
        if (label) label.textContent = sha ? ('已推送 ' + sha.slice(0, 8)) : '';
      }
    }
  } catch (e) {
    // 状态接口暂不可用时不打扰页面
  }
}

// 绑定按钮点击（与其它 action-btn 一致，经 confirm 后走执行队列）。
function bindTraeAgentPushButton() {
  var btn = document.getElementById('trae-agent-push-btn');
  if (!btn || btn.dataset.traeBound) return;
  btn.dataset.traeBound = '1';
  btn.addEventListener('click', async function () {
    var confirmed = await showModalConfirm('将在 trae-agent/onlineServiceJS 执行 DOCKER_PUSH=1 ./buildDocker.sh，并将 onlineServiceJS 镜像推送到 registry。仅当存在未推送的 trae-agent 镜像变更时才会实际执行（--if-pending）。是否继续？', {opLabel: '推送 trae-agent 镜像'});
    if (!confirmed) return;
    _execTraeAgentPush();
  });
}

// 脚本嵌在 </body> 前，DOM 就绪后顶层执行（与 05.js 中 precise-restart 绑定一致）。
initTraeAgentPush();

function initTraeAgentPush() {
  bindTraeAgentPushButton();
  refreshTraeAgentPushStatus();
}
