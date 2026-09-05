// @ts-check
/**
 * 核验：开启 PayPal 时仅走 recharge_paypal_create，不再调用已改为管理员专用的直充 recharge/。
 * 不依赖本机 conf 是否已填 PayPal 密钥：接口由 route 模拟。
 * 需前端已运行（playwright.verify.config.js）。
 */
import { test, expect } from '@playwright/test';

test.describe('Billing 充值仅 PayPal', () => {
  test('paypal_enabled 时点击主按钮应 POST recharge_paypal_create', async ({ page, baseURL }) => {
    test.setTimeout(60_000);
    let paypalCreateCount = 0;

    const origin = baseURL || 'http://localhost:4000';
    const userId = 'playwright-e2e';
    const tenantId = '834421734734905344';
    await page.context().addCookies([{ name: 'userId', value: userId, url: origin }]);

    // AuthSessionGuard 会校验 /me/，仅设 userId cookie 不够
    await page.route(`**/api/accounts/users/me/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: userId,
          username: 'playwright-e2e',
          current_company: { id: tenantId, name: 'E2E Tenant' },
          companies: [{ id: tenantId, name: 'E2E Tenant' }],
        }),
      });
    });

    await page.route('**/billing/accounts/recharge_phone_status/**', async (route) => {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          has_phone: true,
          phone_masked: '139****0000',
          sms_verified: true,
          paypal_enabled: true,
          paypal_currency: 'USD',
          wechat_enabled: false,
        }),
      });
    });
    await page.route('**/billing/accounts/recharge_paypal_create/**', async (route) => {
      if (route.request().method() !== 'POST') {
        return route.continue();
      }
      paypalCreateCount += 1;
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          order_id: 'MOCK-ORDER-1',
          approval_url: 'data:text/html,%3Chtml%3E%3Cbody%3EPayPal%20mock%3C%2Fbody%3E%3C%2Fhtml%3E',
        }),
      });
    });

    const rechargeUrl = `/tenant/${tenantId}/billing/recharge/`;

    await page.goto(rechargeUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);

    await expect(page.getByText(/已绑定手机/)).toBeVisible({ timeout: 15000 });
    const payBtn = page.getByRole('button', { name: /跳转 PayPal/ });
    await expect(payBtn).toBeVisible();
    await expect(payBtn).toBeEnabled();

    await Promise.all([
      page.waitForRequest(
        (r) => r.url().includes('recharge_paypal_create') && r.method() === 'POST',
        { timeout: 20000 }
      ),
      payBtn.click(),
    ]);

    expect(paypalCreateCount).toBe(1);
  });
});
