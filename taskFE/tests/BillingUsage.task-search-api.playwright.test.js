// @ts-check
/**
 * 核验：使用记录页任务下拉走 /tasks/search/，不再 fan-out workspace todos。
 */
import { test, expect } from '@playwright/test';

test.describe('Billing 使用记录 任务统一搜索', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('任务筛选请求 tasks/search 且不拉取 workspace todos', async ({ page, baseURL }) => {
    test.setTimeout(90_000);
    const origin = baseURL || 'http://127.0.0.1:4000';
    const userId = 'playwright-e2e';
    const tenantId = '850256677331562496';
    const taskId = 'task_e2e_search_1';
    const taskTitle = '计费任务搜索-E2E';

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
    await page.route('**/api/tenant/*/billing/usages/**', async (route) => {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ results: [], total: 0, page: 1, page_size: 20 }),
      });
    });

    let todosHits = 0;
    let searchHits = 0;
    /** @type {string[]} */
    const searchUrls = [];
    await page.route('**/api/tasks/todos/tenant_id/*/workspace_id/*/**', async (route) => {
      todosHits += 1;
      return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
    });
    await page.route('**/api/tasks/search/tenant_id/*/**', async (route) => {
      searchHits += 1;
      searchUrls.push(route.request().url());
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          results: [{ id: taskId, title: taskTitle, workspace_id: 'ws1' }],
        }),
      });
    });

    await page.goto(`/tenant/${tenantId}/billing/usage/`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: /使用记录|积分使用/ })).toBeVisible({ timeout: 20000 });

    const search = page.locator('[data-alias="BillingUsageTaskSearch"]');
    await expect(search).toBeVisible();
    await search.click();
    await expect(page.locator('[data-alias="BillingUsageTaskDropdown"]')).toBeVisible();
    await page.locator('[data-alias="BillingUsageTaskDropdown"]').getByText(taskTitle).click();
    await page.getByRole('button', { name: '应用过滤' }).click();

    expect(searchHits).toBeGreaterThan(0);
    expect(todosHits).toBe(0);
    expect(searchUrls.some((u) => u.includes('/tasks/search/'))).toBe(true);
  });
});
