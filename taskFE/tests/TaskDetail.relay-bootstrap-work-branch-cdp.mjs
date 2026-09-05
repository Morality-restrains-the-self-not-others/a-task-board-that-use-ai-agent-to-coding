// @ts-check
/**
 * 核验：relay 启动后引导日志含工作分支切换标记。
 * 经 CDP 9222 连接本机 Chrome。
 *
 * 用法：
 *   PLAYWRIGHT_SITE_ORIGIN=https://www.daydaymoney.com \
 *   RELAY_TASK_URL='https://www.daydaymoney.com/tenant/.../task-detail/task_12953905731855947865/?relayToTrae=true' \
 *   node taskFE/tests/TaskDetail.relay-bootstrap-work-branch-cdp.mjs
 */
import { chromium } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';

const CDP_URL = (process.env.CDP_URL || 'http://127.0.0.1:9222').replace(/\/$/, '');
const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');
const TASK_URL =
  process.env.RELAY_TASK_URL ||
  `${SITE}/tenant/850256677331562496/workspace/861623708318031872/task-detail/task_12953905731855947865/?relayToTrae=true`;
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD ;
const BOOTSTRAP_TIMEOUT_MS = Number(process.env.RELAY_BOOTSTRAP_TIMEOUT_MS || 300000);

function log(msg, extra) {
  const ts = new Date().toISOString();
  if (extra !== undefined) console.log(`[${ts}] ${msg}`, extra);
  else console.log(`[${ts}] ${msg}`);
}

function classifyBody(text) {
  if (/BOOTSTRAP_FAILED/i.test(text)) return 'bootstrap-failed';
  if (/HTTP Basic:\s*Access denied/i.test(text)) return 'auth-denied';
  if (/Authentication failed for/i.test(text)) return 'auth-failed';
  const checkoutDone =
    /BOOTSTRAP_PHASE=work_branch_checkout_done/i.test(text) ||
    (/【工作分支切换】/.test(text) && /\[work-branch-checkout\] ok /.test(text));
  const checkoutSkip = /BOOTSTRAP_PHASE=work_branch_checkout_skip/i.test(text);
  const bootstrapOk = text.includes('任务引导完成') || /BOOTSTRAP_COMPLETE/i.test(text);
  if (bootstrapOk && checkoutDone) return 'bootstrap-ok-checkout';
  if (bootstrapOk && checkoutSkip) return 'bootstrap-ok-skip-checkout';
  if (bootstrapOk) return 'bootstrap-ok-missing-checkout';
  return 'pending';
}

async function ensureImageSelected(page) {
  const select = page.locator('#detail-image');
  if (!(await select.isVisible().catch(() => false))) return '';
  const value = await select.inputValue().catch(() => '');
  if (value) return value;
  const options = select.locator('option:not([disabled])');
  const count = await options.count();
  for (let i = 0; i < count; i++) {
    const v = await options.nth(i).getAttribute('value');
    if (v) {
      await select.selectOption(v);
      await page.waitForTimeout(500);
      return v;
    }
  }
  return '';
}

async function main() {
  log(`CDP=${CDP_URL}`);
  log(`TASK_URL=${TASK_URL}`);

  const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
  const context = browser.contexts()[0] || (await browser.newContext());
  const page = await context.newPage();

  const result = {
    ok: false,
    startClicked: false,
    finalClass: '',
    bodyPreview: '',
    error: '',
  };

  try {
    log('网关 API 登录…');
    const loginData = await loginViaGatewayApi(page, {
      email: EMAIL,
      password: PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });
    const token = String(loginData?.token || '').trim();
    if (!token) throw new Error('登录未返回 token');
    await page.goto(`${SITE}/`, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.evaluate((t) => localStorage.setItem('authToken', t), token);
    if (loginData?.user?.id) {
      await page.context().addCookies([
        { name: 'userId', value: String(loginData.user.id), url: `${SITE}/` },
      ]);
    }
    await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    if (page.url().includes('/auth/login')) {
      throw new Error(`写入 authToken 后仍落在登录页: ${page.url()}`);
    }
    await page.waitForLoadState('networkidle', { timeout: 30000 }).catch(() => {});

    const relayTab = page.getByTestId('server-config-relay-direct-tab');
    await relayTab.waitFor({ state: 'visible', timeout: 90000 });
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
        result.bodyPreview = bodyText.slice(0, 1200).replace(/\s+/g, ' ');
        break;
      }
      const logs = await page.getByTestId('relay-to-trae-logs').innerText().catch(() => '');
      const logClass = classifyBody(logs);
      if (logClass !== 'pending') {
        result.finalClass = logClass;
        result.bodyPreview = logs.slice(0, 1200).replace(/\s+/g, ' ');
        break;
      }
      log('等待引导完成…', lastClass);
      await page.waitForTimeout(3000);
    }

    if (!result.finalClass) {
      const bodyText = await page.locator('body').innerText();
      result.finalClass = classifyBody(bodyText);
      result.bodyPreview = bodyText.slice(0, 1200).replace(/\s+/g, ' ');
    }

    if (
      result.finalClass !== 'bootstrap-ok-checkout' &&
      result.finalClass !== 'bootstrap-ok-skip-checkout'
    ) {
      throw new Error(`引导/切分支未成功: class=${result.finalClass} preview=${result.bodyPreview.slice(0, 400)}`);
    }
    if (result.finalClass === 'bootstrap-ok-skip-checkout') {
      log('⚠️ 任务未配置工作分支，checkout 跳过（仍计为引导成功）');
    }

    result.ok = true;
    log('✅ 引导完成且工作分支切换标记已出现');
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
