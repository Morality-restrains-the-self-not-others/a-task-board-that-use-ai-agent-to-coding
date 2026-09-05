// @ts-check
/** Goal loop: start → refresh → check logs → clear → stop → refresh → start until clean */
import { chromium } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const SITE = (process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:4000').replace(/\/$/, '');
const TASK_ID = '848546827193511936';
const TASK_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?relayToTrae=true`;
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD ;

const ERROR_PATTERNS = [
  /-> error\b/i,
  /\berror HTTP \d+/i,
  /HTTP 40[0-9]/,
  /HTTP 50[0-9]/,
  /无效的 access_token/,
  /REPO_CLONE_CREDENTIALS_INCOMPLETE/,
  /bootstrap \(post-listen\) error/i,
  /ECONNREFUSED/,
  /fetch failed/i,
  /unknown action/i,
  /未找到请求的 API 路径/,
];

function findLogErrors(text) {
  const lines = String(text || '').split('\n').map((l) => l.trim()).filter(Boolean);
  const hits = [];
  for (const line of lines) {
    if (ERROR_PATTERNS.some((re) => re.test(line))) {
      hits.push(line);
    }
  }
  return hits;
}

async function getLogsText(page) {
  const pre = page.locator('[data-testid="relay-to-trae-logs-clear"]').locator('xpath=ancestor::div[contains(@class,"border")]//pre').first();
  if (await pre.count()) {
    return pre.innerText().catch(() => '');
  }
  return page.locator('pre').last().innerText().catch(() => '');
}

async function waitRelayOnline(page) {
  await page.waitForTimeout(3000);
  const relayTab = page.getByTestId('server-config-relay-direct-tab');
  if (await relayTab.isVisible().catch(() => false)) {
    await relayTab.click();
  }
  const statusRow = page.getByTestId('relay-to-trae-status-row');
  await statusRow.waitFor({ state: 'visible', timeout: 60000 });
  const online = statusRow.getByText('relayToTrae 服务：在线');
  const offline = statusRow.getByText('relayToTrae 服务：未连接');
  for (let i = 0; i < 60; i++) {
    if (await online.isVisible().catch(() => false)) return 'online';
    if (await offline.isVisible().catch(() => false)) return 'offline';
    await page.waitForTimeout(1000);
  }
  return 'pending';
}

async function runCycle(page, round) {
  console.log(`\n=== Round ${round}: start → refresh → check logs ===`);
  const startBtn = page.getByTestId('relay-to-trae-start-btn');
  await startBtn.waitFor({ state: 'visible', timeout: 30000 });
  if (await startBtn.isEnabled()) {
    await startBtn.click();
  }
  const stopBtn = page.getByTestId('relay-to-trae-stop-btn');
  await stopBtn.waitFor({ state: 'visible', timeout: 120000 }).catch(() => {});

  await page.waitForTimeout(8000);
  await page.getByTestId('relay-to-trae-status-refresh').click();
  await page.waitForTimeout(5000);

  let logs = await getLogsText(page);
  let errors = findLogErrors(logs);
  console.log(`Log lines: ${logs.split('\n').filter(Boolean).length}, errors: ${errors.length}`);
  if (errors.length) {
    console.log('Errors found:');
    errors.slice(0, 20).forEach((e) => console.log('  ', e));
    return { errors, logs };
  }

  console.log('No errors — running clear → stop → refresh → start');
  await page.getByTestId('relay-to-trae-logs-clear').click();
  await page.waitForTimeout(500);
  if (await stopBtn.isVisible().catch(() => false)) {
    await stopBtn.click();
    await page.waitForTimeout(3000);
  }
  await page.getByTestId('relay-to-trae-status-refresh').click();
  await page.waitForTimeout(2000);
  if (await startBtn.isEnabled().catch(() => false)) {
    await startBtn.click();
    await stopBtn.waitFor({ state: 'visible', timeout: 120000 }).catch(() => {});
    await page.waitForTimeout(8000);
    await page.getByTestId('relay-to-trae-status-refresh').click();
    await page.waitForTimeout(5000);
  }

  logs = await getLogsText(page);
  errors = findLogErrors(logs);
  console.log(`After restart — log lines: ${logs.split('\n').filter(Boolean).length}, errors: ${errors.length}`);
  if (errors.length) {
    errors.slice(0, 20).forEach((e) => console.log('  ', e));
  }
  return { errors, logs };
}

const browser = await chromium.launch({ headless: true, channel: 'chrome' });
const context = await browser.newContext();
const page = await context.newPage();

try {
  await playwrightLoginWithLegalAccept(page, { email: EMAIL, password: PASSWORD, baseURL: SITE });
  await page.goto(`${SITE}${TASK_PATH}`);
  await page.waitForLoadState('domcontentloaded');
  if (page.url().includes('/auth/login')) {
    throw new Error('Login failed');
  }

  const health = await waitRelayOnline(page);
  console.log('relay health:', health);
  if (health === 'offline') {
    throw new Error('relayToTrae offline — start go_relayToTrae first');
  }

  let allClean = false;
  const { errors } = await runCycle(page, 1);
  if (errors.length === 0) {
    allClean = true;
    console.log('\n✅ Goal complete: logs have no errors');
  } else {
    console.log('Round 1 still has errors — need code fix before next round');
  }

  process.exitCode = allClean ? 0 : 2;
} catch (err) {
  console.error('Goal loop failed:', err);
  process.exitCode = 1;
} finally {
  await browser.close();
}
