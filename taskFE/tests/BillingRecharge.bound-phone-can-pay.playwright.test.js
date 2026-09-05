// @ts-check
/**
 * 核验：已绑定手机号且 recharge_phone_status.sms_verified 为 true 时，
 * 充值页主按钮应为「确认支付…」且可点击（不再卡在「请先完成短信验证」）。
 *
 * 使用 route stub 模拟修复后的接口语义，不依赖本地库内是否已有绑定数据。
 * 依赖：前端已启动（playwright.verify.config.js）。
 */
import { test, expect } from '@playwright/test';

test.describe('Billing 充值页 已绑定+sms_verified 门控', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('recharge_phone_status 返回 sms_verified 时支付按钮可用', async ({ page, baseURL }) => {
    test.setTimeout(60_000);
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
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          has_phone: true,
          phone_masked: '139****0000',
          sms_verified: true,
          paypal_enabled: true,
          paypal_currency: 'USD',
        }),
      });
    });

    const rechargeUrl = `/tenant/${tenantId}/billing/recharge/`;

    await page.goto(rechargeUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);

    await expect(page.getByText('已绑定手机')).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('已通过短信验证，可进行支付。')).toBeVisible();

    const payBtn = page.getByRole('button', { name: /跳转 PayPal/ });
    await expect(payBtn).toBeVisible();
    await expect(payBtn).toBeEnabled();
  });
});
