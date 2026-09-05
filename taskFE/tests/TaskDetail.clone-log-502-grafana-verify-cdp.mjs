// @ts-check
/**
 * 核验：启动 relay → container-clone-log 非 502 → Grafana/Loki 有对应 trace 日志
 */
import { chromium } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const CDP_URL = (process.env.CDP_URL || 'http://127.0.0.1:9222').replace(/\/$/, '');
const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://183.250.1.132:4000').replace(/\/$/, '');
const API = (process.env.PLAYWRIGHT_API_ORIGIN || 'http://183.250.1.132:18081').replace(/\/$/, '');
const LOKI = (process.env.LOKI_URL || 'http://127.0.0.1:3100').replace(/\/$/, '');
const TASK_URL =
  process.env.RELAY_TASK_URL ||
  `${SITE}/tenant/850256677331562496/workspace/861623708318031872/task-detail/task_12590983282794675865/?relayToTrae=true`;
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD ;
const TOKEN = process.env.PLAYWRIGHT_API_TOKEN || 'a8266ccd793329242ddeb0f269211c62924c383c';
const TENANT = '850256677331562496';
const WS = '861623708318031872';
const TASK = 'task_12590983282794675865';
const START_TIMEOUT_MS = Number(process.env.RELAY_START_TIMEOUT_MS || 240000);

function log(msg, extra) {
  const ts = new Date().toISOString();
  if (extra !== undefined) console.log(`[${ts}] ${msg}`, extra);
  else console.log(`[${ts}] ${msg}`);
}

async function waitLocal8765(timeoutMs = 120000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const r = await fetch('http://127.0.0.1:8765/api/health/', {
        signal: AbortSignal.timeout(2000),
      });
      if (r.ok || r.status < 500) {
        log(`local :8765 ready status=${r.status}`);
        return true;
      }
    } catch {
      /* retry */
    }
    await new Promise((r) => setTimeout(r, 2000));
  }
  return false;
}

async function queryCloneLog(layerId, traceId) {
  const url = `${API}/api/cloud/compute/tenant_id/${TENANT}/workspace_id/${WS}/task_id/${TASK}container-clone-log/?layer_id=${encodeURIComponent(layerId)}`;
  const span = Math.random().toString(16).slice(2, 18);
  const r = await fetch(url, {
    headers: {
      Accept: 'application/json',
      Authorization: `Token ${TOKEN}`,
      Origin: SITE,
      'X-Trace-Id': traceId,
      'X-Parent-Span-Id': span,
      traceparent: `00-${traceId.replace(/[^a-zA-Z0-9]/g, '').padEnd(32, '0').slice(0, 32)}-${span}-01`,
    },
    signal: AbortSignal.timeout(35000),
  });
  const text = await r.text();
  let body;
  try {
    body = JSON.parse(text);
  } catch {
    body = { raw: text.slice(0, 300) };
  }
  return { status: r.status, body, traceId };
}

async function waitLokiTrace(traceId, timeoutMs = 90000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const startNs = `${Date.now() - 15 * 60 * 1000}000000`;
    const endNs = `${Date.now()}000000`;
    const q = encodeURIComponent(`{job=~".+"} |~ "${traceId}"`);
    const url = `${LOKI}/loki/api/v1/query_range?query=${q}&start=${startNs}&end=${endNs}&limit=50`;
    try {
      const r = await fetch(url, { signal: AbortSignal.timeout(8000) });
      const d = await r.json();
      const streams = d?.data?.result || [];
      let lines = 0;
      const jobs = new Set();
      for (const s of streams) {
        jobs.add(s.stream?.job || '?');
        lines += (s.values || []).length;
      }
      if (lines > 0) {
        return { ok: true, lines, jobs: [...jobs] };
      }
      log(`Loki 尚未命中 trace，继续等待… jobs_seen=${[...jobs].join(',') || '(none)'}`);
    } catch (e) {
      log('Loki 查询失败', String(e?.message || e).slice(0, 160));
    }
    await new Promise((r) => setTimeout(r, 3000));
  }
  return { ok: false, lines: 0, jobs: [] };
}

async function main() {
  const browser = await chromium.connectOverCDP(CDP_URL);
  const ctx = browser.contexts()[0] || (await browser.newContext());
  const page = await ctx.newPage();

  const cloneLogHits = [];
  page.on('response', async (resp) => {
    const u = resp.url();
    if (u.includes('/cloud/compute/container-clone-log/')) {
      const headers = resp.request().headers();
      cloneLogHits.push({
        status: resp.status(),
        url: u,
        traceId: headers['x-trace-id'] || '',
      });
      log(`intercept clone-log status=${resp.status()}`, headers['x-trace-id'] || '');
    }
  });

  try {
    log('goto', TASK_URL);
    await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    if (page.url().includes('/auth/login')) {
      log('need login');
      await playwrightLoginWithLegalAccept(page, { email: EMAIL, password: PASSWORD });
      await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    }

    const relayTab = page.getByTestId('server-config-relay-direct-tab');
    if (await relayTab.isVisible().catch(() => false)) {
      await relayTab.click();
      log('switched relay tab');
    }

    const select = page.locator('#detail-image');
    await select.waitFor({ state: 'visible', timeout: 60000 });
    if (!(await select.inputValue().catch(() => ''))) {
      const opt = select.locator('option:not([disabled])').nth(0);
      const v = await opt.getAttribute('value');
      if (v) await select.selectOption(v);
    }

    const stopBtn = page.getByTestId('relay-to-trae-stop-btn');
    if (await stopBtn.isVisible().catch(() => false)) {
      log('stop existing relay…');
      await stopBtn.click();
      await page.getByTestId('relay-to-trae-start-btn').waitFor({ state: 'visible', timeout: 60000 });
      await page.waitForTimeout(2000);
    }

    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    await startBtn.waitFor({ state: 'visible', timeout: 60000 });
    const deadline = Date.now() + 60000;
    while ((await startBtn.isDisabled().catch(() => true)) && Date.now() < deadline) {
      await page.waitForTimeout(1000);
    }
    if (await startBtn.isDisabled()) throw new Error('start button disabled');
    log('click start');
    await startBtn.click();

    const up = await waitLocal8765(START_TIMEOUT_MS);
    if (!up) throw new Error('local :8765 not up after start');

    // Discover a layer_id from bootstrap or recent UI traffic
    let layerId = '';
    const hitDeadline = Date.now() + 90000;
    while (Date.now() < hitDeadline && !layerId) {
      for (const h of cloneLogHits) {
        const m = h.url.match(/layer_id=([^&]+)/);
        if (m) {
          layerId = decodeURIComponent(m[1]);
          break;
        }
      }
      if (!layerId) await page.waitForTimeout(2000);
    }
    if (!layerId) {
      // fallback: probe bootstrap-clone-log via API for layer_id
      const traceId = `web-${Date.now()}-verifyboot`;
      const span = Math.random().toString(16).slice(2, 18);
      const bootUrl = `${API}/api/cloud/compute/tenant_id/${TENANT}/workspace_id/${WS}/task_id/${TASK}container-bootstrap-clone-log/`;
      const br = await fetch(bootUrl, {
        headers: {
          Accept: 'application/json',
          Authorization: `Token ${TOKEN}`,
          Origin: SITE,
          'X-Trace-Id': traceId,
          'X-Parent-Span-Id': span,
          traceparent: `00-${'a'.repeat(32)}-${span}-01`,
        },
        signal: AbortSignal.timeout(20000),
      });
      const bj = await br.json().catch(() => ({}));
      layerId = String(bj.layer_id || '').trim();
      log('bootstrap-clone-log', { status: br.status, layerId, keys: Object.keys(bj) });
    }
    if (!layerId) throw new Error('no layer_id available for clone-log check');

    const verifyTrace = `web-${Date.now()}-clonelogfix`;
    log('GET container-clone-log', { layerId, verifyTrace });
    const result = await queryCloneLog(layerId, verifyTrace);
    log('clone-log result', { status: result.status, bodyKeys: Object.keys(result.body || {}) });

    if (result.status !== 200) {
      throw new Error(`clone-log expected 200 got ${result.status}: ${JSON.stringify(result.body).slice(0, 400)}`);
    }

    const loki = await waitLokiTrace(verifyTrace, 90000);
    log('loki result', loki);
    if (!loki.ok) {
      throw new Error(`Grafana/Loki has no logs for trace ${verifyTrace}`);
    }

    console.log(
      JSON.stringify(
        {
          ok: true,
          cloneLogStatus: result.status,
          layerId,
          traceId: verifyTrace,
          lokiLines: loki.lines,
          lokiJobs: loki.jobs,
          intercepted: cloneLogHits.slice(-5),
        },
        null,
        2,
      ),
    );
  } finally {
    await page.close().catch(() => {});
  }
}

main().catch((e) => {
  console.error('VERIFY_FAILED', e);
  process.exit(1);
});
