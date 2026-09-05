// @ts-check
/**
 * 核验：积分流水页工作空间下拉走 /workspaces/?mine=1（与后端权限视图一致，不再请求 search_workspaces）。
 */
import { test, expect } from '@playwright/test';

test.describe('Billing 交易流水 工作空间本地搜索', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('工作空间筛选请求 workspaces?mine=1 且不调用 search_workspaces', async ({ page, baseURL }) => {
    test.setTimeout(90_000);
    const origin = baseURL || 'http://127.0.0.1:4000';
    const userId = 'playwright-e2e';
    const tenantId = '850256677331562496';
    const workspaceId = '861623708318031872';
    const workspaceName = '研发工作空间-TXNWS';

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

    let searchWorkspacesHits = 0;
    let workspacesListHits = 0;
    /** @type {string[]} */
    const workspaceUrls = [];
    await page.route('**/billing/transactions/search_workspaces/**', async (route) => {
      searchWorkspacesHits += 1;
      return route.fulfill({ status: 404, body: 'gone' });
    });
    await page.route('**/api/projects/workspaces/tenant_id/*/**', async (route) => {
      workspacesListHits += 1;
      workspaceUrls.push(route.request().url());
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: workspaceId, name: workspaceName }]),
      });
    });
    await page.route('**/api/projects/tenant_id/*', async (route) => {
      return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
    });
    await page.route('**/api/tenant/*/accounts/members/company_members/**', async (route) => {
      return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
    });
    await page.route('**/api/tenant/*/billing/units/**', async (route) => {
      return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
    });

    /** @type {string[]} */
    const txnUrls = [];
    await page.route('**/api/billing/transactions/list_filtered/tenant_id/*/**', async (route) => {
      txnUrls.push(route.request().url());
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ results: [], total: 0, page: 1, page_size: 20 }),
      });
    });

    await page.goto(`/tenant/${tenantId}/billing/transactions/`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: /交易流水|积分交易/ })).toBeVisible({ timeout: 20000 });

    const search = page.locator('[data-alias="BillingTransactionsWorkspaceSearch"]');
    await expect(search).toBeVisible();
    await search.click();
    await expect(page.locator('[data-alias="BillingTransactionsWorkspaceDropdown"]')).toBeVisible();
    await page.locator('[data-alias="BillingTransactionsWorkspaceDropdown"]').getByText(workspaceName).click();
    await page.getByRole('button', { name: '应用过滤' }).click();

    await expect.poll(() => txnUrls.some((u) => new URL(u).searchParams.get('workspace_id') === workspaceId)).toBe(true);
    expect(searchWorkspacesHits).toBe(0);
    expect(workspacesListHits).toBeGreaterThan(0);
    expect(workspaceUrls.some((u) => new URL(u).searchParams.get('mine') === '1')).toBe(true);
  });
});
