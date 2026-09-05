// @ts-check
import { test, expect } from '@playwright/test';

test.describe('SystemAdmin 邮箱注册策略', () => {
  test('管理页应展示邮箱注册开关', async ({ page, baseURL }) => {
    const origin = baseURL || 'http://127.0.0.1:4000';

    await page.context().addCookies([{ name: 'userId', value: 'playwright-e2e', url: origin }]);

    await page.route('**/api/system-admin/system-feature-policy/', async (route) => {
      const method = route.request().method();
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              enable_phone_login: false,
              enable_recharge_phone_verification: false,
              enable_email_register: true,
            },
          }),
        });
        return;
      }
      await route.continue();
    });

    await page.goto('/system-admin/login-payment-policy/');
    await page.waitForLoadState('networkidle');

    const emailRegisterSwitch = page.locator('[data-testid="enable-email-register-switch"]');
    await expect(emailRegisterSwitch).toBeVisible();
    await expect(emailRegisterSwitch).toBeChecked();
  });
});
