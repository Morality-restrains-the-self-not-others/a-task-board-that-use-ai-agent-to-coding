// @ts-check
/**
 * 核验：任务详情「批准在本任务使用凭据」POST 返回 200（非 DRF SessionAuthentication 的 CSRF 403）。
 *
 * 从 ViewSet 转发到 @api_view 时，内层 WrappedAPIView 仍用全局 DEFAULT_AUTHENTICATION_CLASSES，
 * Session 在前会触发 enforce_csrf →「CSRF cookie not set」；须在 github_task_credential_views 上声明
 * SessionAuthenticationWithoutCSRF。
 *
 * 默认跳过（需本机 Vite + Django + 真实数据）。运行示例（工作目录 task2app/playwright）：
 *   GITHUB_CREDENTIAL_APPROVE_PLAYWRIGHT=1 \\
 *   PLAYWRIGHT_TASK_ACCESS_CODE='你的 accessCode' \\
 *   npx playwright test -c playwright.verify.config.js \\
 *     tests/TaskDetail.github-credential-approve-no-csrf-403.playwright.test.js
 *
 * 可选：PLAYWRIGHT_TASK_ID、PLAYWRIGHT_WORKSPACE_ID、PLAYWRIGHT_TENANT_ID
 */
import { test, expect } from '@playwright/test';
import { submitLoginWithEmailPassword } from './helpers/e2eLogin.js';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const ENABLED = process.env.GITHUB_CREDENTIAL_APPROVE_PLAYWRIGHT === '1';
const ACCESS_CODE = (process.env.PLAYWRIGHT_TASK_ACCESS_CODE || '').trim();

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const TASK_ID = process.env.PLAYWRIGHT_TASK_ID || '840179061849448448';

async function login(page) {
  await page.goto('/auth/login/');
  await page.waitForLoadState('domcontentloaded');

  const emailPasswordTab = page.locator('text=邮箱').or(page.locator('button:has-text("邮箱")')).first();
  if (await emailPasswordTab.isVisible()) {
    await emailPasswordTab.click();
    await page.waitForTimeout(300);
  }

  await submitLoginWithEmailPassword(page, EMAIL, PASSWORD);
  await page.waitForLoadState('domcontentloaded');
  await page.waitForTimeout(2000);
  await page
    .waitForFunction(
      () => document.cookie.split(';').some((c) => c.trim().startsWith('userId=')),
      null,
      { timeout: 30000 },
    )
    .catch(() => {});
}

test.describe('TaskDetail GitHub 凭据批准（无 CSRF 403）', () => {
  test.skip(!ENABLED, '设置 GITHUB_CREDENTIAL_APPROVE_PLAYWRIGHT=1 后运行');
  test.skip(!ACCESS_CODE, '设置 PLAYWRIGHT_TASK_ACCESS_CODE（任务详情 URL 的 accessCode）');

  test('POST github-credential-approve 返回 200 且 JSON 含 approved', async ({ page }) => {
    test.setTimeout(120000);

    const path = taskDetailPathWithMockQuery(
      `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=${encodeURIComponent(ACCESS_CODE)}`,
    );

    await login(page);

    const approveUrlPart = `/api/cloud/compute/github-credential-approve/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/task_id/${TASK_ID}/`;

    await page.goto(path);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    const consentBtn = page.getByRole('button', { name: '同意并继续' });
    if (await consentBtn.isVisible().catch(() => false)) {
      await consentBtn.click();
      await page.waitForTimeout(800);
    }

    const panel = page.getByTestId('task-github-credential-panel');
    await expect(panel).toBeVisible({ timeout: 60000 });

    const approveBtn = page.getByRole('button', { name: '批准在本任务使用凭据' });
    await expect(approveBtn).toBeVisible({ timeout: 30000 });

    if (await approveBtn.isDisabled().catch(() => true)) {
      test.skip(true, '已批准或按钮不可用，跳过');
    }

    const respPromise = page.waitForResponse(
      (res) =>
        res.request().method() === 'POST' &&
        res.url().includes(approveUrlPart),
      { timeout: 45000 },
    );

    await approveBtn.click();

    const response = await respPromise;
    const text = await response.text().catch(() => '');
    expect(
      response.status(),
      `期望 200，实际 ${response.status()}，body 前 500 字: ${text.slice(0, 500)}`,
    ).toBe(200);
    expect(text, '响应体应含 approved').toContain('approved');
    expect(text, '不应为 CSRF cookie 错误').not.toContain('CSRF cookie not set');
  });
});
