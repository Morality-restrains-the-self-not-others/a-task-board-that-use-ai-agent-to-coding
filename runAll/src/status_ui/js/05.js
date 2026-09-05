function stopLogsResize(event) {
  if (!layoutState.resizing) { return; }
  layoutState.resizing = false;
  document.body.classList.remove('pane-resizing');
  if (event && event.currentTarget && event.currentTarget.releasePointerCapture && event.currentTarget.hasPointerCapture) {
    try {
      if (event.currentTarget.hasPointerCapture(event.pointerId)) {
        event.currentTarget.releasePointerCapture(event.pointerId);
      }
    } catch (_) {}
  }
}

document.getElementById('services').addEventListener('click', (event) => {
  if (event.target.closest('a.port-link')) {
    return;
  }
  // Click on dependency name → scroll to that service
  const dep = event.target.closest('.dep[data-dep-name]');
  if (dep) {
    const depName = dep.getAttribute('data-dep-name');
    if (depName) {
      const targetSvc = document.querySelector(`.service[data-name="${CSS.escape(depName)}"]`);
      if (targetSvc) {
        targetSvc.scrollIntoView({ behavior: 'smooth', block: 'center' });
        // Briefly highlight the target service
        targetSvc.style.transition = 'background-color 150ms ease';
        targetSvc.style.backgroundColor = '#1e3a5f';
        setTimeout(() => {
          targetSvc.style.backgroundColor = '';
        }, 800);
      }
    }
    return;
  }
  const btn = event.target.closest('button[data-action]');
  if (!btn) {
    return;
  }
  pulseClickFeedback(btn);
  const action = btn.getAttribute('data-action');
  const name = btn.getAttribute('data-name');
  const group = btn.getAttribute('data-group');
  const sessionID = btn.getAttribute('data-session-id') || '';
  if (action === 'start' && name) {
    startService(name, sessionID);
    return;
  }
  if (action === 'stop' && name) {
    stopService(name, sessionID);
    return;
  }
  if (action === 'build') {
    if (!name) return;
    buildService(name);
    return;
  }
  if (action === 'build-group') {
    const group = btn.dataset.group;
    if (!group) return;
    pulseClickFeedback(btn);
    buildGroup(group);
    return;
  }
  if (action === 'restart') {
    if (!name) return;
    restartService(name, sessionID);
    return;
  }
  if (action === 'logs') {
    if (!name) return;
    openLogsPanel(name);
    return;
  }
  if (action === 'clear-logs') {
    if (!name) return;
    clearLogs(name);
    return;
  }
  if (action === 'stop-group' && group) {
    stopGroup(group);
    return;
  }
  if (action === 'start-group' && group) {
    startGroup(group);
    return;
  }
  if (action === 'toggle-group' && group) {
    toggleGroupCollapsed(group);
  }
});

document.getElementById('logs-panel-loki').addEventListener('click', (event) => {
  pulseClickFeedback(event.currentTarget);
  openLokiExplore();
});
document.getElementById('logs-panel-grafana').addEventListener('click', (event) => {
  pulseClickFeedback(event.currentTarget);
  openGrafanaTraceLogs();
});
document.getElementById('logs-panel-close').addEventListener('click', (event) => {
  pulseClickFeedback(event.currentTarget);
  closeLogsPanel();
});
document.getElementById('logs-panel-copy').addEventListener('click', (event) => {
  pulseClickFeedback(event.currentTarget);
  copyLogsToClipboard();
});
document.getElementById('logs-panel-refresh').addEventListener('click', (event) => {
  pulseClickFeedback(event.currentTarget);
  fetchLogsOnce();
});
document.addEventListener('keydown', (event) => {
  if (event.key === 'Escape' && logsState.open) {
    closeLogsPanel();
  }
});
var logsResizeHandle = document.getElementById('logs-resize-handle');
logsResizeHandle.addEventListener('pointerdown', handleLogsResizePointerDown);
logsResizeHandle.addEventListener('pointermove', handleLogsResizePointerMove);
logsResizeHandle.addEventListener('pointerup', stopLogsResize);
logsResizeHandle.addEventListener('pointercancel', stopLogsResize);
window.addEventListener('blur', function () { stopLogsResize(); });
window.addEventListener('resize', function () {
  if (!logsState.open) { return; }
  var panel = document.getElementById('logs-panel');
  var currentWidth = parseFloat(panel.style.width);
  if (!isNaN(currentWidth)) {
    setLogsPanelWidth(currentWidth);
  }
});

const devGenerateConfBtn = document.getElementById('dev-generate-conf');
if (devGenerateConfBtn) {
  devGenerateConfBtn.addEventListener('click', (event) => {
    pulseClickFeedback(event.currentTarget);
    generateConfReplica();
  });
}
const devCatalogBtn = document.getElementById('dev-catalog-link');
if (devCatalogBtn) {
  devCatalogBtn.addEventListener('click', (event) => {
    pulseClickFeedback(event.currentTarget);
    openServiceUrlCatalog();
  });
}
function openServiceUrlCatalog() {
  const grafanaUrl = observabilityState.grafanaUrl || '';
  if (grafanaUrl) {
    window.open(grafanaUrl + '/d/service-url-api-catalog', '_blank', 'noopener,noreferrer');
    return;
  }
  window.open('docs/architecture/service-url-api-catalog.md', '_blank', 'noopener,noreferrer');
}
const devClearBtn = document.getElementById('dev-clear-databases');
if (devClearBtn) {
  devClearBtn.addEventListener('click', async (event) => {
    pulseClickFeedback(event.currentTarget);
    if (!await showModalConfirm(
      '将：① 先关闭 runAll 中所有正在运行的应用（含 ai-monitor、前端、task-events 等；保留 docker-redis / docker-kafka / docker-mysql）② DROP 全部用户 MySQL 库（含测试残留）并重建 registry 空库、清除 binlog ③ 删除残留 SQLite 文件（含 wal/shm）④ FLUSHALL Redis ⑤ 重建 Kafka topics。\n\n不会执行 migrate/种子。完成后请再点「初始化全部数据库」。不会自动启动服务。是否继续？',
      {opLabel: '清空全部数据库'}
    )) return;
    bulkQueue.enqueue('clear-db', '清空全部数据库', function () {
      _execClearAllDatabases();
    }, true);
  });
}
function updateStartAllButtonState() {
  const btns = [
    document.getElementById('start-all-btn'),
    document.getElementById('restart-all-btn')
  ];
  for (var i = 0; i < btns.length; i++) {
    var btn = btns[i];
    if (!btn) continue;
    if (lastDAGBootInProgress) {
      btn.disabled = true;
      btn.title = '自动引导进行中，请稍候…';
      btn.classList.add('is-disabled');
    } else {
      btn.disabled = false;
      btn.title = '';
      btn.classList.remove('is-disabled');
    }
  }
}

const startAllBtn = document.getElementById('start-all-btn');
if (startAllBtn) {
  startAllBtn.addEventListener('click', (event) => {
    pulseClickFeedback(event.currentTarget);
    // Enqueue: no confirm needed for start-all
    bulkQueue.enqueue('start-all', '全部启动', function () {
      _execStartAll();
    }, false);
  });
}
const stopAllBtn = document.getElementById('stop-all-btn');
if (stopAllBtn) {
  stopAllBtn.addEventListener('click', async (event) => {
    pulseClickFeedback(event.currentTarget);
    if (!await showModalConfirm('将按依赖顺序关闭全部服务。\n\n是否继续？', {opLabel: '全部关闭服务'})) return;
    bulkQueue.enqueue('stop-all', '全部关闭', function () {
      _execStopAll();
    }, true);
  });
}
const restartAllBtn = document.getElementById('restart-all-btn');
if (restartAllBtn) {
  restartAllBtn.addEventListener('click', async (event) => {
    pulseClickFeedback(event.currentTarget);
    if (!await showModalConfirm('将按依赖顺序对每个服务做金丝雀平滑重启：新进程先就绪再排空旧进程（非整栈先停）。\n\n是否继续？', {opLabel: '全部重启服务'})) return;
    bulkQueue.enqueue('restart-all', '全部重启', function () {
      _execRestartAll();
    }, true);
  });
}
const buildAllBtn = document.getElementById('build-all-btn');
if (buildAllBtn) {
  buildAllBtn.addEventListener('click', async (event) => {
    pulseClickFeedback(event.currentTarget);
    // OPT-20260819-040: 打开确认弹窗后立即禁用页头按钮，确认中连点不再触发
    // 第二次 showModalConfirm → 不会产生第二条 build-all 入队。
    if (buildAllBtn.dataset.confirming === '1') return;
    buildAllBtn.dataset.confirming = '1';
    buildAllBtn.disabled = true;
    buildAllBtn.classList.add('is-disabled');
    try {
      if (!await showModalConfirm('将按依赖顺序编译全部可编译服务。\n\n是否继续？', {opLabel: '全部重新编译'})) return;
      bulkQueue.enqueue('build-all', '全部重新编译', function () {
        _execBuildAll();
      }, true);
    } finally {
      delete buildAllBtn.dataset.confirming;
      buildAllBtn.disabled = false;
      buildAllBtn.classList.remove('is-disabled');
      // 队列已有 running/pending build-all 时由 updateButtonStates 重新置灰。
      if (typeof bulkQueue !== 'undefined' && typeof bulkQueue.render === 'function') {
        bulkQueue.render();
      }
    }
  });
}
function showEmptyPreciseRestartRegistry(result, sourceOrTraceId) {
  var filePath = (result && result.file_path) ? result.file_path : '';
  if (!filePath && result && result.error) {
    var err = String(result.error);
    var marker = 'registration file is empty): ';
    var i = err.indexOf(marker);
    if (i !== -1) filePath = err.slice(i + marker.length);
  }
  if (!filePath) filePath = '.runall/precise_restart_services.txt';
  // OPT-20260902-019：空登记是正常业务状态（200 {status:empty}），用 info toast 提示，
  // 不走红色请求错误横幅（也非请求失败，不设 trace 属性）。
  showToast(
    '没有已登记的服务。当前读取 ' + filePath + '。智能体编程会话修改服务代码后会将服务登记到源码仓 .runall/precise_restart_services.txt（可用 scripts/register-precise-restart.sh 登记）；clone-run 须在 cutover.env 设置 SOURCE_ROOT',
    {type: 'info', duration: 10000}
  );
}

const preciseRestartBtn = document.getElementById('precise-restart-btn');
if (preciseRestartBtn) {
  preciseRestartBtn.addEventListener('click', async (event) => {
    pulseClickFeedback(event.currentTarget);
    // 点击时读取最新登记列表用于确认弹窗
    // OPT-20260807-025：不可编译服务（无 build_command）标注「不可编译，直接重启」，
    // 避免使用者误以为编译已执行；OPT-20260807-024：失败保留项标注「失败待重试」。
    var regs = [];
    var entries = [];
    var ttlHours = 24;
    var r = null;
    try {
      var resp = await apiFetch('/api/precise-restart/registrations?fill_from_scan=1');
      r = await parseJsonSafe(resp);
      if (r && r.ttl_hours) ttlHours = r.ttl_hours;
      entries = (r && Array.isArray(r.entries)) ? r.entries : [];
      regs = entries.map(function (e) { return e.name; });
      if (!regs.length && r && Array.isArray(r.services)) regs = r.services; // 旧接口兜底
      if (!entries.length && regs.length) {
        entries = regs.map(function (n) { return {name: n}; });
      }
    } catch (e) {}
    if (!regs.length) {
      showEmptyPreciseRestartRegistry(r);
      return;
    }
    var expiredCount = entries.filter(function (e) { return e.expired; }).length;
    if (expiredCount > 0) {
      showRequestError('登记文件含 ' + expiredCount + ' 个超过有效期（' + ttlHours + 'h）的过期登记，按钮已置灰。请先重新登记（scripts/register-precise-restart.sh）');
      return;
    }
    var desc = entries.map(function (e) {
      var marks = [];
      if (e.state === 'failed') marks.push('失败待重试');
      if (e.resolvable === false) marks.push('未知服务，将标记失败');
      else if (e.buildable === false) marks.push('不可编译，直接重启');
      return marks.length ? (e.name + '（' + marks.join('，') + '）') : e.name;
    }).join('、');
    if (!await showModalConfirm(
      '将按依赖顺序对以下 ' + regs.length + ' 个已登记服务执行编译 + 金丝雀平滑重启：\n\n' + desc + '\n\n完成后清空登记文件。是否继续？',
      {opLabel: '精准编译重启'}
    )) return;
    bulkQueue.enqueue('precise-restart', '精准编译重启', function () {
      _execPreciseRestart();
    }, true);
  });
}
const collapseAllBtn = document.getElementById('collapse-all-btn');
if (collapseAllBtn) {
  collapseAllBtn.addEventListener('click', (event) => {
    pulseClickFeedback(event.currentTarget);
    collapseAllGroups();
  });
}
const shutdownSelfBtn = document.getElementById('shutdown-self-btn');
if (shutdownSelfBtn) {
  shutdownSelfBtn.addEventListener('click', (event) => {
    pulseClickFeedback(event.currentTarget);
    shutdownSelf();
  });
}
const proxyToggleBtn = document.getElementById('proxy-toggle-btn');
if (proxyToggleBtn) {
  proxyToggleBtn.addEventListener('click', (event) => {
    pulseClickFeedback(event.currentTarget);
    toggleProxy();
  });
}
const devInitBtn = document.getElementById('dev-init-databases');
if (devInitBtn) {
  devInitBtn.addEventListener('click', async (event) => {
    pulseClickFeedback(event.currentTarget);
    if (!await showModalConfirm(
      '将按 registry 顺序执行全部 migrate_script 与 db/<app>/init.sh（不删库）。\n\n请确保相关服务已停止，否则会拒绝执行。不会自动启动服务。是否继续？' +
      (typeof migratePendingConfirmSuffix === 'function' ? migratePendingConfirmSuffix() : ''),
      {opLabel: '初始化全部数据库'}
    )) return;
    bulkQueue.enqueue('init-db', '初始化全部数据库', function () {
      _execInitAllDatabases();
    }, true);
  });
}


// ── Log Collection Status ───────────────────────────────────────
let lastCollectionData = null;

async function fetchCollectionStatus() {
  try {
    const resp = await fetch('/api/log-collection-status');
    if (!resp.ok) return null;
    const data = await resp.json();
    lastCollectionData = data;
    return data;
  } catch (err) {
    return null;
  }
}

function renderCollectionStatus(data) {
  const bar = document.getElementById('log-collection-bar');
  if (!bar || !data) return;

  const active = data.active_services || 0;
  const total = data.total_services || 0;
  const hasFileSink = data.has_file_sink;

  let html = '<span class="lc-label">📊 日志收集:</span>';
  html += '<span class="lc-stat">活动日志服务: <span class="lc-val">' + active + '</span> / <span class="lc-val">' + total + '</span></span>';
  if (!hasFileSink) {
    html += '<span class="lc-stat">⚠️ <span class="lc-warn">文件输出未启用</span></span>';
  }
  html += '<button class="lc-detail-toggle" type="button" id="lc-detail-toggle" onclick="toggleCollectionDetail()">详情</button>';
  bar.innerHTML = html;

  // Render detail table
  const tbody = document.getElementById('lc-detail-tbody');
  if (!tbody || !data.services) return;
  let rows = '';
  for (const svc of data.services) {
    const cls = svc.is_active ? 'lc-active' : 'lc-inactive';
    const lastLog = svc.last_log_at ? timeAgo(svc.last_log_at) : '—';
    const activeMark = svc.is_active ? '✅' : '❌';
    rows += '<tr class="' + cls + '">';
    rows += '<td>' + esc(svc.service_name) + '</td>';
    rows += '<td>' + esc(svc.service_status || '?') + '</td>';
    rows += '<td>' + svc.total_lines + '</td>';
    rows += '<td>' + esc(lastLog) + '</td>';
    rows += '<td>' + activeMark + '</td>';
    rows += '</tr>';
  }
  tbody.innerHTML = rows;
}

function toggleCollectionDetail() {
  const panel = document.getElementById('lc-detail-panel');
  if (!panel) return;
  const open = panel.classList.toggle('open');
  panel.setAttribute('aria-hidden', open ? 'false' : 'true');
  const btn = document.getElementById('lc-detail-toggle');
  if (btn) btn.textContent = open ? '收起' : '详情';
}

