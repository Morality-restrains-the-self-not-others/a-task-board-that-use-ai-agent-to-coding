// @ts-check
/**
 * 订单列表页：mock orders + refund-applications，在已支付订单展开区点击「申请退款」并确认提交。
 * 依赖：前端已启动（playwright.verify.config.js）。
 */
import { test, expect } from '@playwright/test';

test.describe('Billing 退款申请 mock', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('已支付订单可提交退款申请', async ({ page, baseURL }) => {
    test.setTimeout(60_000);
    const origin = baseURL || 'http://localhost:4000';
    const userId = 'playwright-e2e-refund';
    const tenantId = '834421734734905344';
    await page.context().addCookies([{ name: 'userId', value: userId, url: origin }]);

    await page.route(`**/api/accounts/users/me/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: userId,
          username: 'playwright-e2e-refund',
          is_superuser: false,
          current_company: { id: tenantId, name: 'E2E Tenant' },
          companies: [{ id: tenantId, name: 'E2E Tenant' }],
        }),
      });
    });

    let refundPostCount = 0;

    await page.route(`**/api/tenant/${tenantId}/billing/accounts/balance/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          points: 120,
          available_balance: 120,
          frozen_balance: 0,
        }),
      });
    });

    // Mock orders list with a paid, non-gift order
    await page.route(`**/api/tenant/${tenantId}/billing/orders/**`, async (route) => {
      const url = route.request().url();
      // Order detail (expanded row) endpoint
      if (/\/orders\/\d+\/??$/.test(url) && route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            items: [{ id: 'item-1', resource_type: 'task_post', quantity: 10, unit_price_yuan: '10.00', subtotal_yuan: '100.00' }],
          }),
        });
        return;
      }
      // Orders list endpoint
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          orders: [{
            id: '900001',
            order_number: 'ORD-20260701-0001',
            status: 'paid',
            total_yuan: '100.00',
            total_yuan_cents: 10000,
            payment_method: 'wechat',
            created_at: '2026-07-01T00:00:00Z',
            paid_at: '2026-07-01T01:00:00Z',
          }],
          total: 1,
        }),
      });
    });

    await page.route(`**/api/tenant/${tenantId}/billing/refund-applications/**`, async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ results: [] }),
        });
        return;
      }
      if (route.request().method() === 'POST') {
        refundPostCount += 1;
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: '900001',
            tenant_id: tenantId,
            frozen_points: 120,
            status: 'pending',
            reason: 'E2E mock',
            order_id: '900001',
          }),
        });
        return;
      }
      await route.continue();
    });

    await page.route(`**/api/tenant/${tenantId}/billing/accounts/tenant_pricing_view/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', current_package: null, switchable_packages: [] }),
      });
    });

    await page.route(`**/api/tenant/${tenantId}/billing/transactions/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ results: [] }),
      });
    });

    await page.route(`**/api/tenant/${tenantId}/billing/statistics/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          monthly_consumption_points: 0,
          total_consumption_points: 0,
          user_recharge_points: 0,
        }),
      });
    });

    // Navigate to orders page (退款按钮已移至订单详情)
    const ordersUrl = `/tenant/${tenantId}/billing/orders/`;
    await page.goto(ordersUrl);
    await page.waitForLoadState('networkidle');

    // Click on the order row to expand it (shows detail + refund button)
    const orderRow = page.locator('tr', { hasText: 'ORD-20260701-0001' }).first();
    await expect(orderRow).toBeVisible({ timeout: 15000 });
    await orderRow.click();

    // The refund button should be visible after expanding
    const applyBtn = page.getByTestId('billing-refund-apply-btn');
    await expect(applyBtn).toBeVisible({ timeout: 15000 });
    await expect(applyBtn).toBeEnabled();
    await applyBtn.click();

    // Fill in the required refund reason
    await expect(page.getByTestId('billing-refund-confirm-modal')).toBeVisible();
    const reasonTextarea = page.locator('[data-testid="billing-refund-confirm-modal"] textarea');
    await reasonTextarea.fill('E2E test refund reason');
    await page.getByTestId('billing-refund-confirm-btn').click();

    await expect.poll(() => refundPostCount).toBe(1);
  });
});
