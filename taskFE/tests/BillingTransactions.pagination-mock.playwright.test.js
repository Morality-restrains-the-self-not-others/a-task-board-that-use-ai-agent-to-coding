// @ts-check
/**
 * 核验：交易流水页在分页响应体下展示「共 N 条记录」与「下一页」。
 * 通过 route 模拟 billing/transactions/list_filtered 分页体；与用量页对齐。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;

test.describe('Billing 交易流水 分页 mock', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('分页体展示总数与下一页', async ({ page, baseURL }) => {
    test.setTimeout(90_000);
    const origin = baseURL || 'http://127.0.0.1:4000';
    const userId = 'playwright-e2e';
    const tenantId = TENANT_ID || '827923618468040704';
    const totalCount = 25;
    const desc = 'mock-txn-pagination-row';

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

    await page.route('**/api/tenant/*/billing/units/**', async (route) => {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: '1000000000000000003',
            unit_type: 'server_start',
            name: '智能体任务',
            unit: '次',
            is_active: true,
          },
        ]),
      });
    });

    await page.route('**/api/billing/transactions/list_filtered/tenant_id/*/**', async (route) => {
      if (route.request().method() !== 'GET') {
        return route.continue();
      }
      const url = new URL(route.request().url());
      const pageNum = Number(url.searchParams.get('page') || '1');
      const pageSize = Number(url.searchParams.get('page_size') || '20');
      const row = {
        id: 'txn-mock-1',
        created_at: new Date().toISOString(),
        transaction_type: 'consumption',
        amount_points: 10,
        balance_after_points: 90,
        billing_unit: { name: '智能体任务' },
        project_id: 'mock-proj-a',
        project_name: '项目甲',
        user_id: 'u1',
        user_name: '测试用户',
        description: desc,
      };
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          results: pageNum === 1 ? [row] : [],
          total: totalCount,
          page: pageNum,
          page_size: pageSize,
        }),
      });
    });

    await page.goto(`/tenant/${tenantId}/billing/transactions/`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: '积分交易流水', exact: true })).toBeVisible({
      timeout: 20000,
    });
    await expect(page.getByText(desc).first()).toBeVisible();
    await expect(page.getByText(`共 ${totalCount} 条记录`)).toBeVisible();
    await expect(page.getByRole('button', { name: '下一页' })).toBeVisible();
  });
});
