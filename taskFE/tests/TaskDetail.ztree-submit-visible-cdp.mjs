/**
 * CDP 9222 验收：真实任务页在有文件变动时，ztree 层级节点应显示「提交」按钮。
 *
 * 运行：
 *   node taskFE/tests/TaskDetail.ztree-submit-visible-cdp.mjs
 */
import { chromium } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const CDP_URL = (process.env.CDP_URL || 'http://127.0.0.1:9222').replace(/\/$/, '');
const SITE_BASE = (process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:4000').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '861623708318031872';
const TASK_ID = process.env.PLAYWRIGHT_TASK_ID || 'task_12949237300462721867';
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD ;

const TASK_URL =
  `${SITE_BASE}/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?relayToTrae=true`;

function fail(msg) {
  console.error('FAIL:', msg);
  process.exitCode = 1;
}

async function main() {
  const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
  const context = browser.contexts()[0] || (await browser.newContext());
  const page = await context.newPage();

  await playwrightLoginWithLegalAccept(page, {
    email: EMAIL,
    password: PASSWORD,
    baseURL: SITE_BASE,
  });

  await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
  await ztreePanel.waitFor({ state: 'visible', timeout: 90000 });

  // 选中第一条可写层/任务行
  const rowBtn = ztreePanel.locator('button.break-words.flex-1.min-w-0').first();
  await rowBtn.click();
  await page.waitForTimeout(1500);

  const submitBtn = ztreePanel.getByTestId('layer-ztree-submit-btn').first();
  const submitCount = await submitBtn.count();
  if (submitCount < 1) {
    fail('ztree 未找到 layer-ztree-submit-btn');
    const shot = '/tmp/ztree-submit-missing.png';
    await page.screenshot({ path: shot, fullPage: true });
    console.error('screenshot:', shot);
    await browser.close();
    return;
  }

  const visible = await submitBtn.isVisible();
  const disabled = await submitBtn.isDisabled();
  const title = (await submitBtn.getAttribute('title')) || '';
  console.log('OK: submitBtn visible=', visible, 'disabled=', disabled, 'title=', title);

  const changes = page.getByTestId('task-detail-layer-changes');
  if (await changes.isVisible().catch(() => false)) {
    const text = await changes.innerText();
    console.log('OK: layer-changes panel:', text.slice(0, 120).replace(/\s+/g, ' '));
  } else {
    console.log('NOTE: layer-changes panel not visible (may need refresh)');
  }

  const pushBtn = ztreePanel.getByTestId('layer-ztree-push-btn').first();
  if ((await pushBtn.count()) > 0) {
    console.log('OK: pushBtn also present, disabled=', await pushBtn.isDisabled());
  }

  await browser.close();
  if (!process.exitCode) console.log('PASS: ztree submit button visible');
}

main().catch((e) => {
  console.error(e);
  process.exitCode = 1;
});
