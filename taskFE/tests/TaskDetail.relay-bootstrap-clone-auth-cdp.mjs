// @ts-check
/**
 * 核验：多仓 bootstrap 克隆不得出现 HTTP Basic Access denied / BOOTSTRAP_FAILED。
 * 经 CDP 9222 连接本机 Chrome。
 *
 * 用法：
 *   cd task2app/playwright && node taskFE/tests/TaskDetail.relay-bootstrap-clone-auth-cdp.mjs
 */
import { chromium } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const CDP_URL = (process.env.CDP_URL || 'http://127.0.0.1:9222').replace(/\/$/, '');
const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://183.250.1.132:4000').replace(/\/$/, '');
const TASK_URL =
  process.env.RELAY_TASK_URL ||
  `${SITE}/tenant/850256677331562496/workspace/861623708318031872/task-detail/task_12590983282794675865/?relayToTrae=true`;
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD ;
const BOOTSTRAP_TIMEOUT_MS = Number(process.env.RELAY_BOOTSTRAP_TIMEOUT_MS || 300000);

function log(msg, extra) {
  const ts = new Date().toISOString();
  if (extra !== undefined) console.log(`[${ts}] ${msg}`, extra);
  else console.log(`[${ts}] ${msg}`);
}

function classifyBody(text) {
  if (/HTTP Basic:\s*Access denied/i.test(text)) return 'auth-denied';
  if (/BOOTSTRAP_FAILED/i.test(text)) return 'bootstrap-failed';
  if (/Authentication failed for/i.test(text)) return 'auth-failed';
  if (text.includes('任务引导完成') || /BOOTSTRAP_COMPLETE/i.test(text)) return 'bootstrap-ok';
  return 'pending';
}

async function ensureImageSelected(page) {
  const select = page.locator('#detail-image');
  if (!(await select.isVisible().catch(() => false))) {
    log('未找到 #detail-image，跳过镜像选择');
    return '';
  }
  const value = await select.inputValue().catch(() => '');
  if (value) {
    log(`镜像已选: ${value}`);
    return value;
  }
  const options = select.locator('option:not([disabled])');
  const count = await options.count();
  for (let i = 0; i < count; i++) {
    const v = await options.nth(i).getAttribute('value');
    if (v) {
      await select.selectOption(v);
      log(`已选择镜像: ${v}`);
      await page.waitForTimeout(500);
      return v;
    }
  }
  log('镜像下拉无可用选项，继续尝试启动（可能已由任务默认镜像覆盖）');
  return '';
}

async function main() {
  log(`CDP=${CDP_URL}`);
  log(`TASK_URL=${TASK_URL}`);

  const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
  const context = browser.contexts()[0] || (await browser.newContext());
  const page = (await context.newPage());

  const result = {
    ok: false,
    startClicked: false,
    finalClass: '',
    bodyPreview: '',
    error: '',
  };

  try {
    await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    if (page.url().includes('/auth/login')) {
      log('需要登录…');
      await playwrightLoginWithLegalAccept(page, { email: EMAIL, password: PASSWORD, baseURL: SITE });
      await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    }
    await page.waitForLoadState('networkidle', { timeout: 30000 }).catch(() => {});

    const relayTab = page.getByTestId('server-config-relay-direct-tab');
    await relayTab.waitFor({ state: 'visible', timeout: 60000 });
    await relayTab.click();
    log('已切换到「直接启动」Tab');

    await ensureImageSelected(page);

    const statusRow = page.getByTestId('relay-to-trae-status-row');
    await statusRow.getByText(/relayToTrae 服务：在线/).waitFor({ state: 'visible', timeout: 60000 });
    log('relayToTrae 服务在线');

    const stopBtn = page.getByTestId('relay-to-trae-stop-btn');
    if (await stopBtn.isVisible().catch(() => false)) {
      log('先停止已有实例…');
      await stopBtn.click();
      await page.getByTestId('relay-to-trae-start-btn').waitFor({ state: 'visible', timeout: 90000 });
      await page.waitForTimeout(1500);
    }

    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    const enableDeadline = Date.now() + 60000;
    while (Date.now() < enableDeadline) {
      if (await startBtn.isEnabled().catch(() => false)) break;
      await page.waitForTimeout(1500);
    }
    if (!(await startBtn.isEnabled())) throw new Error('启动按钮一直禁用');

    log('点击「启动」…');
    await startBtn.click();
    result.startClicked = true;

    const deadline = Date.now() + BOOTSTRAP_TIMEOUT_MS;
    let lastClass = 'pending';
    while (Date.now() < deadline) {
      const bodyText = await page.locator('body').innerText();
      lastClass = classifyBody(bodyText);
      if (lastClass !== 'pending') {
        result.finalClass = lastClass;
        result.bodyPreview = bodyText.slice(0, 800).replace(/\s+/g, ' ');
        break;
      }
      // 也扫容器日志区域
      const logs = await page.getByTestId('relay-to-trae-logs').innerText().catch(() => '');
      const logClass = classifyBody(logs);
      if (logClass !== 'pending') {
        result.finalClass = logClass;
        result.bodyPreview = logs.slice(0, 800).replace(/\s+/g, ' ');
        break;
      }
      log('等待引导完成…', lastClass);
      await page.waitForTimeout(3000);
    }

    if (!result.finalClass) {
      const bodyText = await page.locator('body').innerText();
      result.finalClass = classifyBody(bodyText);
      result.bodyPreview = bodyText.slice(0, 800).replace(/\s+/g, ' ');
    }

    if (result.finalClass !== 'bootstrap-ok') {
      throw new Error(`引导未成功: class=${result.finalClass} preview=${result.bodyPreview.slice(0, 300)}`);
    }
    if (/HTTP Basic:\s*Access denied|BOOTSTRAP_FAILED|Authentication failed for/i.test(result.bodyPreview)) {
      throw new Error(`仍出现认证失败文案: ${result.bodyPreview.slice(0, 300)}`);
    }

    result.ok = true;
    log('✅ 引导克隆成功，无 HTTP Basic Access denied');
  } catch (err) {
    result.error = String(err?.message || err);
    console.error('失败:', result.error);
    process.exitCode = 1;
  } finally {
    console.log('\n=== RESULT_JSON ===');
    console.log(JSON.stringify(result, null, 2));
    await page.close().catch(() => {});
  }
}

main();
