// @ts-check
/**
 * 核验：登录后点击「模拟启动（onlineService …）」时，
 * mock-trae-online-stream 不应返回 403 + Invalid token（会话 + CSRF + 宽松 Token 认证）。
 * 凭据可通过环境变量覆盖：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

test.describe('任务详情 mock-trae-online-stream', () => {
  test('模拟启动 POST 不应为 403 Invalid token', async ({ page }) => {
    test.skip(process.env.PRE_COMMIT === '1', 'pre-commit 无全栈/Docker 时任务页无模拟启动按钮，请本地或 CI 单独执行');
    test.setTimeout(120000);
    const taskDetailUrl = taskDetailPathWithMockQuery(
      `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/828234211043913728/?accessCode=u824976301710503936`
    );

    const streamResponses = [];
    page.on('response', async (response) => {
      const url = response.url();
      if (!url.includes('mock-trae-online-stream')) return;
      let body = '';
      let contentType = '';
      try {
        contentType = (response.headers()['content-type'] || '').toLowerCase();
        body = await response.text();
      } catch {
        // ignore
      }
      streamResponses.push({ url, status: response.status(), body, contentType });
    });

    await page.goto('/auth/login/');
    await page.waitForLoadState('domcontentloaded');

    const emailPasswordTab = page.locator('text=邮箱').or(page.locator('button:has-text("邮箱")')).first();
    if (await emailPasswordTab.isVisible()) {
      await emailPasswordTab.click();
      await page.waitForTimeout(300);
    }

    await page.locator('#email').fill(EMAIL);
    await page.locator('#password').fill(PASSWORD);
    await page.locator('form').first().evaluate((form) => form.requestSubmit());
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    await page.goto(taskDetailUrl);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    const btn = page.getByRole('button', { name: /模拟启动（onlineService/ });
    await expect(btn).toBeVisible({ timeout: 20000 });
    await btn.click();

    await page.waitForTimeout(500);
    await expect
      .poll(
        () => streamResponses.length,
        { timeout: 120000, message: '应发起 mock-trae-online-stream 请求' }
      )
      .toBeGreaterThan(0);

    const last = streamResponses[streamResponses.length - 1];
    const body = last.body || '';
    expect(
      body.includes('Invalid token'),
      `不应返回 Invalid token，HTTP ${last.status}，片段: ${body.slice(0, 600)}`
    ).toBe(false);
    if ((last.contentType || '').includes('application/json')) {
      let payload = null;
      try {
        payload = JSON.parse(body);
      } catch {
        payload = null;
      }
      const message = typeof payload?.message === 'string' ? payload.message : '';
      expect(
        /入队|startVM|Worker/i.test(message),
        `JSON 响应应体现新链路（入队 -> startVM -> Worker），实际 message: ${message || '<empty>'}`
      ).toBe(true);
    }
  });
});
