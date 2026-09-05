// @ts-check
/**
 * 核验：工作面板点击「退出」时 POST /api/accounts/users/logout/ 携带 X-CSRFToken，且返回 204（非 CSRF 403）。
 *
 * 依赖：本机 Vite + Django 已运行。
 * 凭据：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { submitLoginWithEmailPassword } from './helpers/e2eLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;

const WORK_PANEL_URL = `/tenant/${TENANT_ID}/work-panel?github=ok`;

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
  // 确保会话 cookie 已写入（Navbar 依此拉取 /users/me/）
  await page
    .waitForFunction(
      () => document.cookie.split(';').some((c) => c.trim().startsWith('userId=')),
      null,
      { timeout: 30000 }
    )
    .catch(() => {});
}

test.describe('工作面板 退出登录 CSRF', () => {
  test('POST logout 含 X-CSRFToken 且状态为 204', async ({ page }) => {
    test.setTimeout(120000);

    await login(page);
    await page.goto(WORK_PANEL_URL);
    await page.waitForLoadState('networkidle').catch(() => {});
    await page.waitForTimeout(1500);

    // 若登录后 profile 要求重新同意隐私政策，全屏遮罩会挡住导航栏
    const consentBtn = page.getByRole('button', { name: '同意并继续' });
    if (await consentBtn.isVisible().catch(() => false)) {
      await consentBtn.click();
      await page.waitForTimeout(800);
    }

    const logoutBtn = page
      .getByRole('button', { name: '退出' })
      .or(page.locator('nav[data-alias="cmp-navbar-main"] button.btn-secondary:has-text("退出")'));
    await expect(logoutBtn.first()).toBeVisible({ timeout: 45000 });

    const logoutPromise = page.waitForResponse(
      (res) =>
        res.request().method() === 'POST' &&
        res.url().includes('/api/accounts/users/logout/'),
      { timeout: 30000 }
    );

    await logoutBtn.first().click();

    const response = await logoutPromise;
    const req = response.request();
    const headers = req.headers();
    const csrfHeader =
      headers['x-csrf-token'] || headers['x-csrftoken'] || headers['X-CSRFToken'];

    expect(csrfHeader, 'logout POST 应携带 X-CSRFToken').toBeTruthy();
    expect(response.status(), '登出成功应为 204').toBe(204);
  });
});
