// @ts-check
/** 回归：工作面板控制台不应出现 error（需 PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD） */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const SITE_ORIGIN = (process.env.PLAYWRIGHT_SITE_ORIGIN || '').trim().replace(/\/+$/, '');

const ACCESS_CODE = 'u824976301710503936';

function loginOpts(email, password) {
  const creds = { email, password };
  if (SITE_ORIGIN) creds.baseURL = SITE_ORIGIN;
  return creds;
}

function attachConsoleErrorCollector(page, errors) {
  page.on('console', (msg) => {
    if (msg.type() === 'error') errors.push(msg.text());
  });
  page.on('pageerror', (err) => errors.push(err.message));
}

test('work-panel: no console errors after load', async ({ page }) => {
  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD for this test');

  const errors = [];
  attachConsoleErrorCollector(page, errors);

  await playwrightLoginWithLegalAccept(page, loginOpts(email, password));
  await page.goto(`/tenant/${TENANT_ID}/work-panel`);
  await page.waitForLoadState('networkidle', { timeout: 45000 }).catch(() => {});
  await page.waitForTimeout(2500);

  expect(errors.filter((t) => !t.includes('favicon'))).toEqual([]);
});

test('work-panel: no console errors after open create-task modal (accessCode)', async ({ page }) => {
  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD for this test');

  const errors = [];
  attachConsoleErrorCollector(page, errors);

  await playwrightLoginWithLegalAccept(page, loginOpts(email, password));
  const q = new URLSearchParams({ accessCode: ACCESS_CODE });
  await page.goto(`/tenant/${TENANT_ID}/work-panel/?${q.toString()}`);
  await page.waitForLoadState('networkidle', { timeout: 45000 }).catch(() => {});
  await page.waitForTimeout(2500);

  await page.locator('#create-task-btn').click();
  await page.locator('#create-task-modal').waitFor({ state: 'visible', timeout: 15000 });
  await page.waitForTimeout(500);

  expect(errors.filter((t) => !t.includes('favicon'))).toEqual([]);
});
