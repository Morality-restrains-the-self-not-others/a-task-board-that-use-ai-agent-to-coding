// @ts-check
/**
 * 回归：任务详情「分支策略」面板加载时不应出现
 * workPanelBranchHelpers / workBranchPresetOptions 命名导出缺失的 SyntaxError。
 *
 * 默认联调 URL（relayToTrae）与手工复现一致；可用 PLAYWRIGHT_TASK_DETAIL_URL 覆盖。
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const SITE_ORIGIN = String(process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://daydaymoney.com').replace(/\/$/, '');
const TASK_URL =
  process.env.PLAYWRIGHT_TASK_DETAIL_URL ||
  `${SITE_ORIGIN}/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/843742455533076480/?relayToTrae=true`;

const EXPORT_ERROR_RE =
  /does not provide an export named ['"]WORK_BRANCH_PRESET_OPTIONS['"]|workPanelBranchHelpers|workBranchPresetOptions/i;

test('task detail branch strategy panel loads without module export SyntaxError', async ({ page }) => {
  test.setTimeout(120000);

  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, '设置 PLAYWRIGHT_TEST_EMAIL 与 PLAYWRIGHT_TEST_PASSWORD');

  /** @type {string[]} */
  const pageErrors = [];
  /** @type {string[]} */
  const consoleErrors = [];

  page.on('pageerror', (error) => {
    pageErrors.push(String(error?.message || error));
  });
  page.on('console', (message) => {
    if (message.type() !== 'error') return;
    const text = message.text();
    if (text.includes('text/event-stream') || text.includes('SSE连接错误')) return;
    consoleErrors.push(text);
  });

  await playwrightLoginWithLegalAccept(page, { email, password, baseURL: SITE_ORIGIN });
  await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 90000 });
  await expect(page.getByText('分支策略', { exact: false }).first()).toBeVisible({ timeout: 60000 });

  const exportRelatedPageErrors = pageErrors.filter((m) => EXPORT_ERROR_RE.test(m));
  const exportRelatedConsoleErrors = consoleErrors.filter((m) => EXPORT_ERROR_RE.test(m));

  expect(exportRelatedPageErrors, exportRelatedPageErrors.join(' | ')).toEqual([]);
  expect(exportRelatedConsoleErrors, exportRelatedConsoleErrors.join(' | ')).toEqual([]);
});
