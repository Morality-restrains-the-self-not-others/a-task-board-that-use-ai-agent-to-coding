var lastProxyState;
var refresh;

function timeAgo(isoStr) {
  if (!isoStr) return '—';
  const d = new Date(isoStr);
  if (isNaN(d.getTime())) return isoStr;
  const sec = Math.floor((Date.now() - d.getTime()) / 1000);
  if (sec < 60) return sec + 's ago';
  if (sec < 3600) return Math.floor(sec / 60) + 'm ago';
  if (sec < 86400) return Math.floor(sec / 3600) + 'h ago';
  return Math.floor(sec / 86400) + 'd ago';
}

async function refreshCollectionStatus() {
  const data = await fetchCollectionStatus();
  if (data) renderCollectionStatus(data);
}

// ── Trace Shipping Verification ───────────────────────────────
let traceShippingVerifying = false;
let lastTraceShippingData = null;

async function fetchTraceShippingStatus() {
  try {
    const resp = await fetch('/api/trace-shipping/status');
    if (!resp.ok) return null;
    const data = await resp.json();
    if (data && data.status === 'pending' && !data.summary) return null;
    lastTraceShippingData = data;
    return data;
  } catch (err) {
    return null;
  }
}

async function runTraceShippingVerify() {
  if (traceShippingVerifying) return;
  traceShippingVerifying = true;
  const bar = document.getElementById('trace-shipping-bar');
  if (bar) {
    bar.innerHTML = '<span class="ts-label">🔍 Trace 投递验证:</span><span class="ts-stat">验证中…（写入 probe → 轮询 Loki → 汇总）</span>';
  }
  try {
    const resp = await apiFetch('/api/trace-shipping/verify?wait_seconds=15', { method: 'POST' });
    const data = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError('Trace 投递验证失败: ' + (data.error || resp.status), resolveTraceIdFromResponse(resp, resp.requestTraceId, data));
      renderTraceShippingStatus(lastTraceShippingData);
      return;
    }
    lastTraceShippingData = data;
    renderTraceShippingStatus(data);
  } catch (err) {
    showRequestError('Trace 投递验证失败: ' + err.message, err);
    renderTraceShippingStatus(lastTraceShippingData);
  } finally {
    traceShippingVerifying = false;
  }
}

function renderTraceShippingStatus(data) {
  const bar = document.getElementById('trace-shipping-bar');
  if (!bar) return;
  if (!data || !data.summary) {
    bar.innerHTML = '<span class="ts-label">🔍 Trace 投递验证:</span>'
      + '<span class="ts-stat">尚未验证</span>'
      + '<button class="ts-verify-btn" type="button" id="ts-verify-btn" onclick="runTraceShippingVerify()">立即验证</button>'
      + '<button class="lc-detail-toggle" type="button" id="ts-detail-toggle" onclick="toggleTraceShippingDetail()">详情</button>';
    return;
  }

  const s = data.summary;
  const configuredTotal = s.configured_total || s.total || 0;
  const lokiClass = data.loki_ready ? 'ts-ok' : 'ts-bad';
  const failedClass = (s.failed || 0) > 0 ? 'ts-bad' : 'ts-ok';
  let html = '<span class="ts-label">🔍 Trace 投递验证:</span>';
  if (data.verification_status === 'pending') {
    html += '<span class="ts-stat ts-warn">尚未验证</span>';
  } else if (data.verification_status === 'stale') {
    html += '<span class="ts-stat ts-warn">配置已变更，请重新验证</span>';
  }
  html += '<span class="ts-stat">配置服务: <span class="ts-ok">' + configuredTotal + '</span></span>';
  if (data.verification_status !== 'pending') {
    html += '<span class="ts-stat">Loki: <span class="' + lokiClass + '">' + (data.loki_ready ? '可达' : '不可达') + '</span></span>';
    html += '<span class="ts-stat">本地采集: <span class="ts-ok">' + (s.local_ok || 0) + '</span> / ' + configuredTotal + '</span>';
    html += '<span class="ts-stat">Loki 可见: <span class="ts-ok">' + (s.loki_ok || 0) + '</span> / ' + configuredTotal + '</span>';
    html += '<span class="ts-stat">异常: <span class="' + failedClass + '">' + (s.failed || 0) + '</span></span>';
    if (data.promtail_reload) {
      const prClass = data.promtail_reload === 'ok' ? 'ts-ok' : (data.promtail_reload === 'not_running' ? 'ts-warn' : 'ts-bad');
      html += '<span class="ts-stat">Promtail: <span class="' + prClass + '">' + esc(data.promtail_reload) + '</span></span>';
    }
  }
  if (data.probe_trace_id) {
    html += '<span class="ts-stat">probe: <code>' + esc(data.probe_trace_id) + '</code></span>';
  }
  if (data.grafana_trace_url) {
    html += '<a href="' + esc(data.grafana_trace_url) + '" target="_blank" rel="noopener noreferrer">Grafana Journey</a>';
  }
  html += '<button class="ts-verify-btn" type="button" id="ts-verify-btn" onclick="runTraceShippingVerify()">立即验证</button>';
  html += '<button class="lc-detail-toggle" type="button" id="ts-detail-toggle" onclick="toggleTraceShippingDetail()">详情</button>';
  bar.innerHTML = html;

  const tbody = document.getElementById('ts-detail-tbody');
  if (!tbody || !data.services) return;
  let rows = '';
  for (const svc of data.services) {
    const cls = (svc.issue) ? 'ts-fail' : 'ts-pass';
    const localMark = svc.local_probe_ok ? '✅' : '❌';
    const lokiMark = svc.loki_visible ? '✅' : '❌';
    const grafanaLink = svc.grafana_url
      ? '<a href="' + esc(svc.grafana_url) + '" target="_blank" rel="noopener noreferrer">查看</a>'
      : '—';
    rows += '<tr class="' + cls + '">';
    rows += '<td>' + esc(svc.service_name) + '</td>';
    rows += '<td>' + esc(svc.service_status || '?') + '</td>';
    rows += '<td>' + localMark + '</td>';
    rows += '<td>' + lokiMark + '</td>';
    rows += '<td>' + (svc.loki_log_count || 0) + '</td>';
    rows += '<td>' + grafanaLink + '</td>';
    rows += '<td>' + esc(svc.issue || '') + '</td>';
    rows += '</tr>';
  }
  tbody.innerHTML = rows;
}

function toggleTraceShippingDetail() {
  const panel = document.getElementById('ts-detail-panel');
  if (!panel) return;
  const open = panel.classList.toggle('open');
  panel.setAttribute('aria-hidden', open ? 'false' : 'true');
  const btn = document.getElementById('ts-detail-toggle');
  if (btn) btn.textContent = open ? '收起' : '详情';
}

async function refreshTraceShippingStatus() {
  const data = await fetchTraceShippingStatus();
  renderTraceShippingStatus(data);
}

// ── Internal API live smoke ───────────────────────────────────
let internalAPIsSmokeVerifying = false;
let lastInternalAPIsSmokeData = null;

async function fetchInternalAPIsSmokeStatus() {
  try {
    const resp = await fetch('/api/smoke/internal-apis/status');
    if (!resp.ok) return null;
    const data = await resp.json();
    lastInternalAPIsSmokeData = data;
    return data;
  } catch (err) {
    return null;
  }
}

async function runInternalAPIsSmokeVerify() {
  if (internalAPIsSmokeVerifying) return;
  internalAPIsSmokeVerifying = true;
  const bar = document.getElementById('internal-apis-smoke-bar');
  if (bar) {
    bar.innerHTML = '<span class="ts-label">🧪 Internal API smoke:</span><span class="ts-stat">验证中…</span>';
  }
  try {
    const resp = await apiFetch('/api/smoke/internal-apis/verify', { method: 'POST' });
    const data = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError('Internal API smoke 失败: ' + (data.error || resp.status), resolveTraceIdFromResponse(resp, resp.requestTraceId, data));
      renderInternalAPIsSmokeStatus(lastInternalAPIsSmokeData);
      return;
    }
    lastInternalAPIsSmokeData = data;
    renderInternalAPIsSmokeStatus(data);
  } catch (err) {
    showRequestError('Internal API smoke 失败: ' + err.message, err);
    renderInternalAPIsSmokeStatus(lastInternalAPIsSmokeData);
  } finally {
    internalAPIsSmokeVerifying = false;
  }
}

function extractInternalAPIsSmokeFailReasons(data) {
  const reasons = [];
  const pushUnique = (line) => {
    const t = String(line || '').trim();
    if (!t || reasons.indexOf(t) >= 0) return;
    reasons.push(t);
  };
  const scan = (text) => {
    String(text || '').split('\n').forEach((raw) => {
      const line = raw.trim();
      if (line.indexOf('FAIL  ') === 0 || line.indexOf('FAIL\t') === 0) pushUnique(line);
    });
  };
  if (data) {
    scan(data.output);
    // message may already be "FAIL … · === summary: …"
    String(data.message || '').split(' · ').forEach((part) => {
      const p = part.trim();
      if (p.indexOf('FAIL  ') === 0 || p.indexOf('FAIL\t') === 0) {
        p.split(' | ').forEach(pushUnique);
      }
    });
  }
  return reasons;
}

function renderInternalAPIsSmokeStatus(data) {
  const bar = document.getElementById('internal-apis-smoke-bar');
  if (!bar) return;
  const status = (data && data.status) || 'pending';
  let statusClass = 'ts-warn';
  let statusText = '尚未验证';
  if (status === 'ok') { statusClass = 'ts-ok'; statusText = '通过'; }
  else if (status === 'failed') { statusClass = 'ts-bad'; statusText = '失败'; }
  else if (status === 'skipped') { statusClass = 'ts-warn'; statusText = '跳过'; }
  const failReasons = (status === 'failed') ? extractInternalAPIsSmokeFailReasons(data) : [];
  let html = '<span class="ts-label">🧪 Internal API smoke:</span>';
  html += '<span class="ts-stat"><span class="' + statusClass + '">' + statusText + '</span></span>';
  if (failReasons.length) {
    const detail = failReasons.join(' | ');
    html += '<span class="ts-fail-detail" title="' + esc(detail) + '">原因: ' + esc(detail.slice(0, 220)) + (detail.length > 220 ? '…' : '') + '</span>';
  } else if (data && data.message) {
    html += '<span class="ts-stat" title="' + esc(String(data.message)) + '">' + esc(String(data.message).slice(0, 120)) + '</span>';
  }
  if (data && data.trigger) {
    html += '<span class="ts-stat">来源: ' + esc(data.trigger) + '</span>';
  }
  html += '<button class="ts-verify-btn" type="button" id="ias-verify-btn" onclick="runInternalAPIsSmokeVerify()">立即验证</button>';
  bar.innerHTML = html;
}

async function refreshInternalAPIsSmokeStatus() {
  const data = await fetchInternalAPIsSmokeStatus();
  renderInternalAPIsSmokeStatus(data);
}

// ── Proxy Toggle ─────────────────────────────────────────────────
function renderProxyToggle() {
  const btn = document.getElementById('proxy-toggle-btn');
  if (!btn) return;
  if (lastProxyState) {
    btn.textContent = '代理: 开启';
    btn.classList.remove('proxy-off');
    btn.classList.add('proxy-on');
    btn.title = '代理已开启 — 点击关闭（服务启动时将移除 HTTP_PROXY 等环境变量）';
  } else {
    btn.textContent = '代理: 关闭';
    btn.classList.remove('proxy-on');
    btn.classList.add('proxy-off');
    btn.title = '代理已关闭（默认）— 点击开启（服务启动时保留 HTTP_PROXY 等环境变量）';
  }
}

async function toggleProxy() {
  const newState = !lastProxyState;
  try {
    const resp = await apiFetch('/api/proxy-config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ use_proxy: newState })
    });
    const result = await parseJsonSafe(resp);
    if (!resp.ok) {
      showRequestError('代理切换失败: ' + (result.error || resp.status), resolveTraceIdFromResponse(resp, resp.requestTraceId, result));
      return;
    }
    lastProxyState = result.use_proxy;
    renderProxyToggle();
  } catch (err) {
    showRequestError('代理切换失败: ' + err.message, err);
  }
}

// Modified refresh to also update collection status.
var _origRefresh = refresh;
refresh = async function () {
  await _origRefresh();
  refreshCollectionStatus();
  refreshTraceShippingStatus();
  refreshInternalAPIsSmokeStatus();
};

// 顶层 bootstrap 调用统一在 13.js 末尾执行（全部 const *State 声明之后，防 TDZ 复发）。
// body { overflow: hidden } — wheel over page chrome (title/actions) does nothing.
// Chrome bars live inside .services-pane so they scroll away; still forward
// wheels from page-header / progress panel onto the pane.
function isVerticallyScrollable(el) {
  if (!el || el === document.body || el === document.documentElement) {
    return false;
  }
  const style = getComputedStyle(el);
  const oy = style.overflowY;
  if (oy !== 'auto' && oy !== 'scroll' && oy !== 'overlay') {
    return false;
  }
  return el.scrollHeight > el.clientHeight + 1;
}

function findVerticalScrollableAncestor(start) {
  let el = start instanceof Element ? start : null;
  while (el && el !== document.body && el !== document.documentElement) {
    if (isVerticallyScrollable(el)) {
      return el;
    }
    el = el.parentElement;
  }
  return null;
}

document.addEventListener('wheel', (event) => {
  if (event.defaultPrevented || event.ctrlKey) {
    return;
  }
  const pane = document.querySelector('.services-pane');
  if (!pane) {
    return;
  }
  const scrollable = findVerticalScrollableAncestor(event.target);
  if (scrollable && scrollable !== pane) {
    return;
  }
  if (scrollable === pane) {
    return;
  }
  const maxScroll = pane.scrollHeight - pane.clientHeight;
  if (maxScroll <= 0) {
    return;
  }
  const next = Math.min(maxScroll, Math.max(0, pane.scrollTop + event.deltaY));
  if (next === pane.scrollTop) {
    return;
  }
  pane.scrollTop = next;
  event.preventDefault();
}, { passive: false });
