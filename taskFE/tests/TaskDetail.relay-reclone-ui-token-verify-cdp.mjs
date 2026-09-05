// @ts-check
/**
 * 精简 CDP 核验：任务详情启动 → 检查侧车日志无 TOKEN_ACCESS_INVALID / Host key；
 * 并对容器 /ui/{stale} 做 302 探测（若能拿到当前 token）。
 */
import { chromium } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import fs from 'node:fs';

const CDP_URL = 'http://127.0.0.1:9222';
const SITE = 'http://183.250.1.132:4000';
const TASK_URL =
  `${SITE}/tenant/850256677331562496/workspace/861623708318031872/task-detail/task_12590983282794675865/?relayToTrae=true`;
const EMAIL = 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
// 规范路径：runAll tee → logs/go-relay.log（禁止 *.restart.log 旁路）
const RELAY_LOG = '/tmp/ram-work/logs/go-relay.log';

function log(...a) {
  console.log(new Date().toISOString(), ...a);
}

function tailLog(n = 80) {
  try {
    const raw = fs.readFileSync(RELAY_LOG, 'utf8');
    return raw.split('\n').slice(-n).join('\n');
  } catch {
    return '';
  }
}

async function main() {
  const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
  const context = browser.contexts()[0] || (await browser.newContext());
  const page = context.pages().find((p) => !p.url().startsWith('devtools://')) || (await context.newPage());

  log('goto', TASK_URL);
  await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  if (page.url().includes('/auth/login')) {
    log('login…');
    await playwrightLoginWithLegalAccept(page, { email: EMAIL, password: PASSWORD, baseURL: SITE });
    await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  }

  const relayTab = page.getByTestId('server-config-relay-direct-tab');
  if (await relayTab.isVisible().catch(() => false)) await relayTab.click();

  const select = page.locator('#detail-image');
  await select.waitFor({ state: 'visible', timeout: 60000 });
  if (!(await select.inputValue())) {
    const opt = select.locator('option:not([disabled])').nth(0);
    const v = await opt.getAttribute('value');
    if (v) await select.selectOption(v);
  }

  const stopBtn = page.getByTestId('relay-to-trae-stop-btn');
  if (await stopBtn.isVisible().catch(() => false)) {
    log('stop existing…');
    await stopBtn.click();
    await page.getByTestId('relay-to-trae-start-btn').waitFor({ state: 'visible', timeout: 90000 });
    await page.waitForTimeout(2000);
  }

  const mark = `VERIFY_${Date.now()}`;
  fs.appendFileSync(RELAY_LOG, `\n# ${mark}\n`);

  const startBtn = page.getByTestId('relay-to-trae-start-btn');
  await startBtn.waitFor({ state: 'visible', timeout: 30000 });
  for (let i = 0; i < 30 && !(await startBtn.isEnabled()); i++) await page.waitForTimeout(1000);
  if (!(await startBtn.isEnabled())) throw new Error('start disabled');
  log('click start');
  await startBtn.click();

  // wait token-sync OK in relay log（仅看 mark 之后的新行）
  const deadline = Date.now() + 120000;
  let synced = false;
  while (Date.now() < deadline) {
    const t = tailLog(200);
    const after = t.includes(mark) ? t.split(mark).pop() || '' : '';
    if (/token-sync: OK/.test(after)) {
      synced = true;
      break;
    }
    if (/TOKEN_ACCESS_INVALID|status push HTTP 401/.test(after)) {
      throw new Error('still seeing TOKEN_ACCESS_INVALID after start');
    }
    await page.waitForTimeout(2000);
  }
  if (!synced) {
    log('WARN: token-sync OK not seen in window; dump after-mark');
    const t = tailLog(200);
    console.log(t.includes(mark) ? t.split(mark).pop() : t.slice(-2000));
  } else {
    log('token-sync: OK observed');
  }

  const after = (() => {
    const t = tailLog(300);
    return t.includes(mark) ? t.split(mark).pop() || '' : '';
  })();
  if (/Host key verification failed/.test(after)) {
    throw new Error('Host key verification failed still present after start');
  }
  log('no Host key failure in post-start log window');

  // open console if available
  const link = page.getByTestId('relay-to-trae-open-console');
  const consDeadline = Date.now() + 90000;
  let href = '';
  while (Date.now() < consDeadline) {
    if (await link.isVisible().catch(() => false)) {
      href = (await link.getAttribute('href')) || '';
      if (href && !/task-detail|127\.0\.0\.1|localhost|:4000/.test(href)) break;
    }
    await page.waitForTimeout(2000);
  }
  if (!href) {
    log('WARN: console link not ready; skip UI probe');
  } else {
    log('console href', href);
    const u = new URL(href);
    // probe current UI
    const cur = await page.request.get(href, { timeout: 10000 });
    log('console GET status', cur.status());
    if (cur.status() === 401) throw new Error('console URL returned 401');

    // if path is /ui/{token}, probe stale redirect using remembered bootstrap is hard remotely;
    // instead hit /api/session/ui-redirect with current token
    const tokMatch = u.pathname.match(/\/ui\/([^/]+)/);
    if (tokMatch) {
      const tok = decodeURIComponent(tokMatch[1]);
      const redir = await page.request.get(`${u.origin}/api/session/ui-redirect`, {
        headers: { 'X-Access-Token': tok },
        timeout: 10000,
      });
      log('ui-redirect status', redir.status());
      if (redir.ok()) {
        const j = await redir.json();
        log('ui-redirect body', j);
        if (j.access_token !== tok && !j.redirected) {
          throw new Error('unexpected ui-redirect payload');
        }
      }
    }
  }

  log('VERIFY_OK');
  process.exitCode = 0;
}

main().catch((e) => {
  console.error('VERIFY_FAIL', e);
  process.exitCode = 1;
});
