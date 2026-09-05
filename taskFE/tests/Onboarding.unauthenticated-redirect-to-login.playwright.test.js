// @ts-check
/**
 * E2E: Onboarding 页面 — 未登录访问必须重定向到登录页
 *
 * 覆盖：
 * 1. 未登录（无任何会话 cookie）访问 /onboarding/ → 重定向到 /auth/login/
 * 2. 重定向 URL 携带 next=/onboarding/ 回跳参数
 *
 * 运行：npx playwright test Onboarding.unauthenticated-redirect-to-login.playwright.test.js
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();

const BASE_URL = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  process.env.BASE_URL ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');

test.describe('Onboarding 未登录重定向', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 环境下后端可能不可用');

  test('未登录访问 /onboarding/ 应重定向到登录页', async ({ page }) => {
    // 全新 context：无 userId/token cookie，确保未登录态
    await page.goto(`${BASE_URL}/onboarding/`, {
      waitUntil: 'domcontentloaded',
      timeout: 30000,
    });

    // 等待重定向落定（Vue mounted 内 roles 401 → 跳转登录页）
    await page.waitForURL((url) => url.pathname.startsWith('/auth/login/'), {
      timeout: 15000,
    });

    const url = new URL(page.url());
    expect(url.pathname).toBe('/auth/login/');
    // 携带 next 回跳参数，登录后可回到 onboarding 继续创建公司
    expect(url.searchParams.get('next')).toBe('/onboarding/');
    // 登录表单可见
    await expect(page.locator('[data-alias="cmp-login-form"]')).toBeVisible({ timeout: 10000 });
  });
});
