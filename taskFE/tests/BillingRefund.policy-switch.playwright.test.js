// @ts-check
/**
 * OPT-20260722-013: 超管关闭「开启退款申请」后订单列表页无「申请退款」按钮；重新开启后按钮恢复。
 *
 * 测试策略：
 * 1. mock balance API 返回 refund_enabled: true → 断言「申请退款」按钮可见
 * 2. mock balance API 返回 refund_enabled: false → 断言按钮消失
 * 3. mock balance API 返回 refund_enabled: true → 断言按钮恢复
 *
 * 同时校验 admin 页面的 refund-policy-toggle 开关与 policy panel。
 * 使用 data-testid 选择器断言。
 */
import { test, expect } from '@playwright/test';

const ORIGIN = 'http://localhost:4000';
const USER_ID = 'playwright-e2e-refund-policy';
const TENANT_ID = '834421734734905344';

/**
 * 设置 userId cookie 并 mock /me/ 接口，通过 AuthSessionGuard。
 */
async function setupAuth(page) {
  await page.context().addCookies([{ name: 'userId', value: USER_ID, url: ORIGIN }]);

  await page.route(`**/api/accounts/users/me/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: USER_ID,
        username: 'playwright-e2e-refund-policy',
        is_superuser: false,
        current_company: { id: TENANT_ID, name: 'E2E Tenant' },
        companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
      }),
    });
  });
}

/**
 * 设置 billing 页面必需的 mock API（balance 除外，由测试用例动态指定）。
 * 按钮已移至订单列表页，需额外 mock orders API。
 */
async function setupBillingMocks(page, mockBalanceBody) {
  await page.route(`**/api/tenant/${TENANT_ID}/billing/accounts/balance/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockBalanceBody),
    });
  });

  await page.route(`**/api/tenant/${TENANT_ID}/billing/refund-applications/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ results: [] }),
    });
  });

  // Mock orders list with a paid, non-gift order (退订按钮仅对非赠送已支付订单可见)
  await page.route(`**/api/tenant/${TENANT_ID}/billing/orders/**`, async (route) => {
    const url = route.request().url();
    if (/\/orders\/\d+\/??$/.test(url) && route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: [] }),
      });
      return;
    }
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

  await page.route(`**/api/tenant/${TENANT_ID}/billing/transactions/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ results: [] }),
    });
  });

  await page.route(`**/api/tenant/${TENANT_ID}/billing/accounts/tenant_pricing_view/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'success', current_package: null, switchable_packages: [] }),
    });
  });

  await page.route(`**/api/tenant/${TENANT_ID}/billing/statistics/**`, async (route) => {
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
}

/**
 * 设置 system-admin 退款策略 API mock（GET / PUT）。
 * GET 返回当前策略状态；PUT 更新并返回新状态。
 */
async function setupAdminPolicyMock(page, initialEnabled) {
  let policyEnabled = initialEnabled;

  await page.route(`**/api/system-admin/refund-policy/**`, async (route) => {
    const req = route.request();
    if (req.method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ enabled: policyEnabled }),
      });
      return;
    }
    if (req.method() === 'PUT') {
      const body = JSON.parse(req.postData() || '{}');
      policyEnabled = body.enabled === true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ enabled: policyEnabled }),
      });
      return;
    }
    await route.continue();
  });
}

async function setupAdminMocks(page) {
  // system-admin 用户列表页需要的 mock
  await page.route(`**/api/system-admin/users/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        results: [],
        total: 0,
      }),
    });
  });

  // 退款申请列表 mock（页面 onMounted 时会加载）
  await page.route(`**/api/system-admin/refund-applications/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ results: [] }),
    });
  });
}

test.describe('Billing 退款政策开关 — 订单列表页按钮可见性', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  /**
   * 辅助：导航到订单列表页并展开第一行订单
   */
  async function gotoOrdersAndExpand(page, origin) {
    await page.goto(`${origin}/tenant/${TENANT_ID}/billing/orders/`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);
    // 展开订单行以显示退款按钮
    const orderRow = page.locator('tr', { hasText: 'ORD-20260701-0001' }).first();
    await orderRow.click();
    await page.waitForTimeout(300);
  }

  test('refund_enabled 控制「申请退款」按钮显示/隐藏', async ({ page, baseURL }) => {
    test.setTimeout(60_000);
    const origin = baseURL || ORIGIN;

    await setupAuth(page);

    // === 场景 1: refund_enabled = true，按钮可见 ===
    await setupBillingMocks(page, {
      points: 200,
      available_balance: 200,
      frozen_balance: 0,
      refund_enabled: true,
    });

    await gotoOrdersAndExpand(page, origin);

    const applyBtn = page.getByTestId('billing-refund-apply-btn');
    await expect(applyBtn).toBeVisible({ timeout: 15000 });
    await expect(applyBtn).toBeEnabled();
    console.log('[PASS] refund_enabled=true 时「申请退款」按钮可见且可用');

    // === 场景 2: refund_enabled = false，按钮消失 ===
    await page.unroute(`**/api/tenant/${TENANT_ID}/billing/accounts/balance/**`);
    await page.route(`**/api/tenant/${TENANT_ID}/billing/accounts/balance/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          points: 200,
          available_balance: 200,
          frozen_balance: 0,
          refund_enabled: false,
        }),
      });
    });

    await gotoOrdersAndExpand(page, origin);

    await expect(page.getByTestId('billing-refund-apply-btn')).not.toBeVisible({ timeout: 15000 });
    console.log('[PASS] refund_enabled=false 时「申请退款」按钮不可见');

    // === 场景 3: refund_enabled = true 恢复，按钮恢复可见 ===
    await page.unroute(`**/api/tenant/${TENANT_ID}/billing/accounts/balance/**`);
    await page.route(`**/api/tenant/${TENANT_ID}/billing/accounts/balance/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          points: 200,
          available_balance: 200,
          frozen_balance: 0,
          refund_enabled: true,
        }),
      });
    });

    await gotoOrdersAndExpand(page, origin);

    await expect(page.getByTestId('billing-refund-apply-btn')).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('billing-refund-apply-btn')).toBeEnabled();
    console.log('[PASS] refund_enabled=true 恢复后「申请退款」按钮重新可见且可用');
  });

  test('超管页面 refund-policy 开关可切换', async ({ page, baseURL }) => {
    test.setTimeout(60_000);
    const origin = baseURL || ORIGIN;

    // 以超管身份 mock
    await page.context().addCookies([{ name: 'userId', value: 'admin-e2e', url: origin }]);
    await page.route(`**/api/user/admin-e2e/accounts/users/me/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'admin-e2e',
          username: 'superadmin',
          is_superuser: true,
          current_company: { id: TENANT_ID, name: 'E2E Tenant' },
          companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
        }),
      });
    });

    // 初始：开启状态
    await setupAdminPolicyMock(page, true);
    await setupAdminMocks(page);

    // 导航到系统管理-订单与退款（退款 Tab）；旧 /refund-applications/ 会重定向至此
    await page.goto(`${origin}/system-admin/order-records/?tab=refund`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);

    // 校验 refund-policy-panel 可见
    const policyPanel = page.getByTestId('refund-policy-panel');
    await expect(policyPanel).toBeVisible({ timeout: 15000 });

    const policySwitch = page.getByTestId('enable-refund-applications-switch');
    await expect(policySwitch).toBeVisible();
    await expect(policySwitch).toBeChecked();
    await expect(page.getByText('已开启')).toBeVisible();
    console.log('[PASS] 初始状态：退款申请已开启');

    // 点击开关 → 关闭（须二次确认）
    await policySwitch.click();
    await page.getByRole('button', { name: '确定' }).click();
    await page.waitForTimeout(500);

    // PUT 请求后状态变为关闭
    await expect(policySwitch).not.toBeChecked();
    await expect(page.getByText('已关闭')).toBeVisible();
    console.log('[PASS] 关闭后开关状态已更新');

    // 再次点击 → 开启（须二次确认）
    await policySwitch.click();
    await page.getByRole('button', { name: '确定' }).click();
    await page.waitForTimeout(500);

    await expect(policySwitch).toBeChecked();
    await expect(page.getByText('已开启')).toBeVisible();
    console.log('[PASS] 重新开启后开关状态已恢复');

    // 取消确认：开关回滚，不变更
    await policySwitch.click();
    await page.getByRole('button', { name: '取消' }).click();
    await page.waitForTimeout(300);
    await expect(policySwitch).toBeChecked();
    await expect(page.getByText('已开启')).toBeVisible();
    console.log('[PASS] 取消确认后开关保持开启');
  });
});
