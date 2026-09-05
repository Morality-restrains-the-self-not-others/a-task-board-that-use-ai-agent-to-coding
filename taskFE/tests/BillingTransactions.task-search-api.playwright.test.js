// @ts-check
/**
 * 核验：交易流水页任务下拉走 /tasks/search/，并暴露与用量页对齐的 data-alias。
 */
import { test, expect } from '@playwright/test';

test.describe('Billing 交易流水 任务统一搜索', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('任务筛选请求 tasks/search 且不拉取 workspace todos', async ({ page, baseURL }) => {
    test.setTimeout(90_000);
    const origin = baseURL || 'http://127.0.0.1:4000';
    const userId = 'playwright-e2e';
    const tenantId = '850256677331562496';
    const taskId = 'task_e2e_txn_search_1';
    const taskTitle = '流水任务搜索-E2E';

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

    await page.route('**/api/projects/tenant_id/*', async (route) => {
      return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
    });
    await page.route('**/api/projects/workspaces/tenant_id/*/**', async (route) => {
      return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
    });
    await page.route('**/api/tenant/*/accounts/members/company_members/**', async (route) => {
      return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
    });
    await page.route('**/api/tenant/*/billing/units/**', async (route) => {
      return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
    });
    await page.route('**/api/billing/transactions/list_filtered/tenant_id/*/**', async (route) => {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ results: [], total: 0, page: 1, page_size: 20 }),
      });
    });

    let todosHits = 0;
    let searchHits = 0;
    await page.route('**/api/tasks/todos/tenant_id/*/workspace_id/*/**', async (route) => {
      todosHits += 1;
      return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
    });
    await page.route('**/api/tasks/search/tenant_id/*/**', async (route) => {
      searchHits += 1;
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          results: [{ id: taskId, title: taskTitle, workspace_id: 'ws1' }],
        }),
      });
    });

    await page.goto(`/tenant/${tenantId}/billing/transactions/`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: /交易流水|积分交易/ })).toBeVisible({ timeout: 20000 });

    const search = page.locator('[data-alias="BillingTransactionsTaskSearch"]');
    await expect(search).toBeVisible();
    await search.click();
    await expect(page.locator('[data-alias="BillingTransactionsTaskDropdown"]')).toBeVisible();
    await page.locator('[data-alias="BillingTransactionsTaskDropdown"]').getByText(taskTitle).click();
    await page.getByRole('button', { name: '应用过滤' }).click();

    expect(searchHits).toBeGreaterThan(0);
    expect(todosHits).toBe(0);
  });
});
