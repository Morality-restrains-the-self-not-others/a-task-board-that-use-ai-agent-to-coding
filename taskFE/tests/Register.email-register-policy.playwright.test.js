// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Register 邮箱注册策略', () => {
  test('关闭邮箱注册后应隐藏邮箱注册入口', async ({ page, baseURL }) => {
    const origin = baseURL || 'http://127.0.0.1:4000';

    await page.route('**/api/public/system-feature-policy/', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: {
            enable_phone_login: false,
            enable_recharge_phone_verification: false,
            enable_email_register: false,
          },
        }),
      });
    });

    await page.route('**/api/public/registration-invite-policy/', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', data: { enabled: false } }),
      });
    });

    await page.route('**/api/privacy-policy/public/current/', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: 'p1', version: 'v1', title: '隐私政策', content: 'x' }),
      });
    });

    await page.route('**/api/license-agreement/public/current/', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: 'l1', version: 'v1', title: '服务协议', content: 'y' }),
      });
    });

    await page.goto(`${origin}/register/`);
    await page.waitForLoadState('networkidle');

    await expect(page.locator('[data-testid="register-method-email"]')).toHaveCount(0);
    await expect(page.locator('[data-testid="register-method-phone"]')).toBeVisible();
  });
});
