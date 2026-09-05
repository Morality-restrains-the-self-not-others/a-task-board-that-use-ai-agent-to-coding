// @ts-check
/**
 * 核验：微信支付启用时，主按钮走 recharge_wechat_create 并展示扫码弹层。
 * 接口由 route 模拟，不依赖真实微信商户号。
 */
import { test, expect } from '@playwright/test';

test.describe('Billing 充值微信支付', () => {
  test('wechat_enabled 时点击主按钮应 POST recharge_wechat_create 并出现扫码弹层', async ({ page, baseURL }) => {
    test.setTimeout(60_000);
    let wechatCreateCount = 0;

    const origin = baseURL || 'http://localhost:4000';
    const userId = 'playwright-e2e';
    const tenantId = '850256677331562496';
    await page.context().addCookies([{ name: 'userId', value: userId, url: origin }]);

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
          paypal_currency: 'HKD',
          wechat_enabled: true,
        }),
      });
    });

    await page.route('**/billing/accounts/recharge_wechat_create/**', async (route) => {
      if (route.request().method() !== 'POST') {
        return route.continue();
      }
      wechatCreateCount += 1;
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          out_trade_no: 'WXMOCK-PLAYWRIGHT-1',
          code_url: 'weixin://wxpay/bizpayurl?pr=WXMOCK-PLAYWRIGHT-1',
          mode: 'mock',
          amount_yuan: 10,
          amount_points: 1000,
        }),
      });
    });

    await page.route('**/billing/accounts/recharge_wechat_status/**', async (route) => {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'pending', out_trade_no: 'WXMOCK-PLAYWRIGHT-1' }),
      });
    });

    // 避免外网二维码图片加载失败影响断言
    await page.route('**/api.qrserver.com/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'image/png',
        body: Buffer.from(
          'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==',
          'base64',
        ),
      });
    });

    const rechargeUrl = `/tenant/${tenantId}/billing/recharge/`;
    await page.goto(rechargeUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);

    await expect(page.getByText(/已绑定手机/)).toBeVisible({ timeout: 15000 });
    await expect(page.getByRole('button', { name: '微信支付', exact: true })).toBeVisible();

    const payBtn = page.getByRole('button', { name: /微信支付 \d+/ });
    await expect(payBtn).toBeVisible();
    await expect(payBtn).toBeEnabled();

    await Promise.all([
      page.waitForRequest(
        (r) => r.url().includes('recharge_wechat_create') && r.method() === 'POST',
        { timeout: 20000 },
      ),
      payBtn.click(),
    ]);

    expect(wechatCreateCount).toBe(1);
    await expect(page.getByText('微信扫码支付')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/WXMOCK-PLAYWRIGHT-1/)).toBeVisible();
  });
});
