// @ts-check
/**
 * CDP 核验：任务详情启动 selected_image 后，go_relay 日志须含镜像 Id/digest（sha256），
 * 便于核对实际拉取的镜像。
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

function tailLog(n = 120) {
  try {
    const raw = fs.readFileSync(RELAY_LOG, 'utf8');
    return raw.split('\n').slice(-n).join('\n');
  } catch {
    return '';
  }
}

function afterMark(mark) {
  const t = tailLog(400);
  return t.includes(mark) ? t.split(mark).pop() || '' : '';
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

  const mark = `IMAGE_HASH_VERIFY_${Date.now()}`;
  fs.appendFileSync(RELAY_LOG, `\n# ${mark}\n`);

  const startBtn = page.getByTestId('relay-to-trae-start-btn');
  await startBtn.waitFor({ state: 'visible', timeout: 30000 });
  for (let i = 0; i < 30 && !(await startBtn.isEnabled()); i++) await page.waitForTimeout(1000);
  if (!(await startBtn.isEnabled())) throw new Error('start disabled');
  log('click start');
  await startBtn.click();

  const deadline = Date.now() + 180000;
  let hashLine = '';
  while (Date.now() < deadline) {
    const after = afterMark(mark);
    const m = after.match(/\[relayToTrae\] docker pull ok:.*id=sha256:[0-9a-f]+.*/i);
    if (m) {
      hashLine = m[0];
      break;
    }
    if (/docker pull failed/.test(after)) {
      throw new Error('docker pull failed after start:\n' + after.slice(-1500));
    }
    await page.waitForTimeout(2000);
  }
  if (!hashLine) {
    const after = afterMark(mark);
    console.error('post-start log window:\n', after.slice(-3000));
    throw new Error('expected docker pull ok with id=sha256:… in relay logs');
  }
  log('hash log line:', hashLine);

  const after = afterMark(mark);
  if (!/image_id=sha256:[0-9a-f]+/i.test(after) && !/id=sha256:[0-9a-f]+/i.test(after)) {
    throw new Error('sha256 image id missing from post-start logs');
  }

  // Prefer RepoDigest when registry provides it
  const hasDigest = /digest=\S+@sha256:[0-9a-f]+/i.test(after) || /image_digest=\S+@sha256:[0-9a-f]+/i.test(after);
  log('repo digest present:', hasDigest);

  console.log('VERIFY_OK image-hash-log');
  await browser.close().catch(() => {});
}

main().catch((e) => {
  console.error('VERIFY_FAIL', e);
  process.exit(1);
});
