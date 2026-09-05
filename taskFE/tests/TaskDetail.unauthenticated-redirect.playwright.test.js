// @ts-check
/**
 * 回归：未登录或仅有无效 userId Cookie 时，访问 task-detail（含 relayToTrae）应跳转登录页。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const SITE_ORIGIN = String(process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://localhost:4000').replace(/\/$/, '');
const SITE_ORIGIN_127 = 'http://127.0.0.1:4000';
const TASK_ID = '846269443533955072';
const REPORTED_TASK_ID = '847744505890045952';
const TASK_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?relayToTrae=true`;
const REPORTED_TASK_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${REPORTED_TASK_ID}/?relayToTrae=true`;

test.describe('TaskDetail unauthenticated redirect', () => {
  test('无 Cookie 时跳转登录页', async ({ page, context }) => {
    await context.clearCookies();
    await page.goto(`${SITE_ORIGIN}${TASK_PATH}`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await expect(page).toHaveURL(/\/auth\/login\/?$/, { timeout: 15000 });
  });

  test('仅有无效 userId Cookie 时跳转登录页', async ({ page, context }) => {
    await context.clearCookies();
    await context.addCookies([{ name: 'userId', value: '999999999', url: `${SITE_ORIGIN}/` }]);
    await page.goto(`${SITE_ORIGIN}${TASK_PATH}`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await expect(page).toHaveURL(/\/auth\/login\/?$/, { timeout: 15000 });
  });

  test('未认证访问时写入 postLoginRedirect', async ({ page, context }) => {
    await context.clearCookies();
    await page.goto(`${SITE_ORIGIN}${TASK_PATH}`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    const stored = await page.evaluate(() => localStorage.getItem('postLoginRedirect'));
    expect(stored).toBe(TASK_PATH);
  });

  test('127.0.0.1 用户报告 taskId 无 Cookie 时跳转登录页', async ({ page, context }) => {
    await context.clearCookies();
    await page.goto(`${SITE_ORIGIN_127}${REPORTED_TASK_PATH}`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await expect(page).toHaveURL(/\/auth\/login\/?$/, { timeout: 15000 });
  });

  test('无效 authToken 时清除并跳转登录页', async ({ page, context }) => {
    await context.clearCookies();
    await page.addInitScript(() => {
      localStorage.setItem('authToken', 'invalid-stale-token-12345');
    });
    await page.goto(`${SITE_ORIGIN}${REPORTED_TASK_PATH}`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await expect(page).toHaveURL(/\/auth\/login\/?$/, { timeout: 15000 });
    const token = await page.evaluate(() => localStorage.getItem('authToken'));
    expect(token).toBeNull();
  });
});
