/**
 * Playwright CDP：制造孤儿占 8765 → UI 选择环境变量/镜像 → 点击「启动」
 * → 断言无 EADDRINUSE 且 server listening（registry 镜像 listen 前置 + relay 孤儿清理）。
 *
 * 运行：
 *   bash tests/TaskDetail.relay-host-port-eaddrinuse.playwright.test.js.sh
 */
import { chromium } from '@playwright/test';
import { spawnSync } from 'child_process';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';
import { mentionInstalledImageInComment } from './helpers/commentImageMentionE2e.js';

const CDP = process.env.CDP_URL || process.env.PW_CDP_URL || 'http://127.0.0.1:9222';
const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');
const TENANT = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WS = process.env.PLAYWRIGHT_WORKSPACE_ID || '861623708318031872';
const TASK = process.env.PLAYWRIGHT_RELAY_TASK_ID || 'task_12953905731855947865';
const IMAGE_ID = process.env.PLAYWRIGHT_INSTALLED_IMAGE_ID || '862588024964280320';
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');
const ORPHAN_NAME = 'relay_taskId_orphan_eaddrinuse';
const TASK_URL = `${SITE}/tenant/${TENANT}/workspace/${WS}/task-detail/${TASK}/?relayToTrae=true`;
const RELAY_BASE = `/api/cloud/compute/tenant_id/${TENANT}/workspace_id/${WS}/task_id/${TASK}relay-to-trae`;

function sh(cmd, args = []) {
  return spawnSync(cmd, args, { encoding: 'utf8' });
}

function sleepMs(ms) {
  spawnSync('sleep', [String(Math.max(0.05, ms / 1000))]);
}

function ensureOrphanOn8765() {
  sh('docker', ['rm', '-f', ORPHAN_NAME]);
  const run = sh('docker', [
    'run', '-d', '--rm', '--name', ORPHAN_NAME, '--network', 'host',
    '--entrypoint', 'node', 'trae-online-js:x86_64-latest',
    '-e',
    'require("net").createServer().listen(8765,"0.0.0.0",()=>console.log("orphan"));setInterval(()=>{},1e9)',
  ]);
  if (run.status !== 0) throw new Error(`orphan start failed: ${run.stderr || run.stdout}`);
  for (let i = 0; i < 20; i++) {
    const dial = sh('python3', [
      '-c',
      'import socket;s=socket.create_connection(("127.0.0.1",8765),1);s.close();print("ok")',
    ]);
    if (dial.status === 0) return;
    sleepMs(250);
  }
  throw new Error('orphan did not bind 8765');
}

function orphanStillRunning() {
  return String(sh('docker', ['ps', '-q', '--filter', `name=${ORPHAN_NAME}`]).stdout || '').trim().length > 0;
}

async function fetchDirectLogs() {
  const r = await fetch('http://127.0.0.1:8797/v1/status', {
    headers: { 'X-Relay-To-Trae-Secret': 'dev-secret' },
  });
  const body = await r.json();
  return {
    logs: Array.isArray(body.logs) ? body.logs.join('\n') : '',
    running: !!body.running,
    online: !!body.online_service_up,
    error: String(body.error || ''),
  };
}

async function prepareUiGates(page) {
  const tab = page.getByTestId('server-config-relay-direct-tab');
  if (await tab.count()) {
    await tab.click();
    await page.waitForTimeout(800);
  }

  const envSel = page.getByTestId('feature-params-source-selector');
  if (await envSel.count()) {
    const val = await envSel.inputValue().catch(() => '');
    if (!val) {
      await envSel.selectOption('company');
      await page.waitForTimeout(400);
    }
  }

  const imageSelect = page.locator('#detail-image');
  if (await imageSelect.count()) {
    const cur = await imageSelect.inputValue().catch(() => '');
    if (!cur) {
      await imageSelect.selectOption(IMAGE_ID).catch(async () => {
        const opts = await imageSelect.locator('option').all();
        for (const o of opts) {
          const v = await o.getAttribute('value');
          if (v) {
            await imageSelect.selectOption(v);
            break;
          }
        }
      });
    }
  } else if (await page.getByTestId('comment-content-editor').count()) {
    await mentionInstalledImageInComment(page);
  }

  // Stale repo banner: acknowledge to unblock
  const ack = page.getByTestId('relay-to-trae-ack-stale-repo-btn');
  if ((await ack.count()) && (await ack.isVisible().catch(() => false))) {
    await ack.click();
    await page.waitForTimeout(500);
  }

  const startBtn = page.getByTestId('relay-to-trae-start-btn');
  if (!(await startBtn.count())) throw new Error('start button missing');

  // If still disabled, surface hint for diagnosis
  for (let i = 0; i < 15; i++) {
    if (await startBtn.isEnabled()) return startBtn;
    const hint = page.getByTestId('relay-to-trae-start-disabled-hint');
    const oauth = page.getByTestId('relay-to-trae-oauth-unbound-guide');
    const text = [
      (await hint.count()) ? await hint.innerText() : '',
      (await oauth.count()) && (await oauth.isVisible()) ? 'oauth-unbound' : '',
    ]
      .filter(Boolean)
      .join(' | ');
    console.log(`[e2e] start still disabled (${i}): ${text || '(no hint)'}`);
    await page.waitForTimeout(1000);
  }
  throw new Error('start button remains disabled after gate prep');
}

async function main() {
  console.log('[e2e] stop tracked relay + seed orphan…');
  const browser = await chromium.connectOverCDP(CDP, { timeout: 15000 });
  const ctx = browser.contexts()[0] || (await browser.newContext());
  const page = await ctx.newPage();

  try {
    const login = await loginViaGatewayApi(page, {
      email: EMAIL,
      password: PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });
    const token = String(login?.token || '').trim();
    if (!token) throw new Error('login token missing');
    const hdr = {
      Authorization: `Token ${token}`,
      Accept: 'application/json',
      'Content-Type': 'application/json',
      Origin: SITE,
    };

    await page.request.post(`${GATEWAY}${RELAY_BASE}/stop/`, {
      headers: hdr,
      data: { task_id: TASK },
    });
    await page.waitForTimeout(1500);
    ensureOrphanOn8765();
    console.log('[e2e] orphan=', orphanStillRunning());

    await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForTimeout(2500);

    const startBtn = await prepareUiGates(page);
    // Re-seed orphan in case stop/page-load cleared it
    if (!orphanStillRunning()) ensureOrphanOn8765();

    await startBtn.click();
    console.log('[e2e] UI start clicked');

    let logs = '';
    let ok = false;
    const deadline = Date.now() + 240_000;
    while (Date.now() < deadline) {
      const direct = await fetchDirectLogs();
      logs = direct.logs;
      if (/EADDRINUSE|port \d+ already in use/i.test(logs + direct.error)) {
        throw new Error(`EADDRINUSE present:\n${logs.slice(-2500)}\nerror=${direct.error}`);
      }
      if (/server listening on http:\/\/0\.0\.0\.0:\d+/i.test(logs) && (direct.running || direct.online)) {
        ok = true;
        break;
      }
      if (/still in use before docker run/i.test(logs + direct.error)) {
        throw new Error(`port cleanup failed: ${direct.error}\n${logs.slice(-1500)}`);
      }
      await page.waitForTimeout(2000);
    }
    if (!ok) throw new Error(`timeout waiting for listening\n${logs.slice(-2500)}`);
    if (orphanStillRunning()) throw new Error('orphan should be removed');

    console.log('[e2e] PASS UI click start');
    console.log(
      logs
        .split('\n')
        .filter((l) => /orphan|overlay|EADDRINUSE|listening|container started|port |pull ok/i.test(l))
        .slice(-30)
        .join('\n'),
    );
  } finally {
    await page.close().catch(() => {});
    // Disconnect without closing shared CDP Chrome
    try {
      browser.close = async () => {};
    } catch {
      /* ignore */
    }
  }
  process.exit(0);
}

main().catch((e) => {
  console.error('[e2e] FAIL', e);
  process.exit(1);
});
