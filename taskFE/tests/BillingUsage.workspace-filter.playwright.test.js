// @ts-check
/**
 * 核验：使用记录页可按工作空间筛选，请求 usages 时带 workspace_id。
 */
import { test, expect } from '@playwright/test';

test.describe('Billing 使用记录 工作空间筛选', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('选择工作空间后 usages 请求携带 workspace_id', async ({ page, baseURL }) => {
    test.setTimeout(90_000);
    const origin = baseURL || 'http://127.0.0.1:4000';
    const userId = 'playwright-e2e';
    const tenantId = '850256677331562496';
    const workspaceId = '861623708318031872';
    const workspaceName = '默认工作空间-WSFILTER';

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
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    /** @type {string[]} */
    const workspaceUrls = [];
    await page.route('**/api/projects/workspaces/tenant_id/*/**', async (route) => {
      workspaceUrls.push(route.request().url());
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: workspaceId, name: workspaceName }]),
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

    /** @type {string[]} */
    const usageUrls = [];
    await page.route('**/api/tenant/*/billing/usages/**', async (route) => {
      if (route.request().method() !== 'GET') {
        return route.continue();
      }
      usageUrls.push(route.request().url());
      const url = new URL(route.request().url());
      const filteredWs = url.searchParams.get('workspace_id');
      const pageNum = Number(url.searchParams.get('page') || '1');
      const pageSize = Number(url.searchParams.get('page_size') || '20');
      const row = {
        id: 'usage-ws-1',
        usage_time: new Date().toISOString(),
        billing_unit: { name: '任务服务器启动服务费', unit: '次' },
        amount: '1',
        project_id: 'mock-proj-a',
        project_name: '项目甲',
        user_id: 'u1',
        user_name: '测试用户',
        workspace_id: workspaceId,
        workspace_name: workspaceName,
        task_id: 'task-mock-1',
        task_name: '示例任务',
        description: '任务服务器启动 1 次',
        account: 'acc1',
      };
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          results: filteredWs && filteredWs !== workspaceId ? [] : [row],
          total: filteredWs && filteredWs !== workspaceId ? 0 : 1,
          page: pageNum,
          page_size: pageSize,
        }),
      });
    });

    await page.goto(`/tenant/${tenantId}/billing/usage/`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: /使用记录/ })).toBeVisible({ timeout: 20000 });

    const search = page.locator('[data-alias="BillingUsageWorkspaceSearch"]');
    await expect(search).toBeVisible();
    await search.click();
    await expect(page.locator('[data-alias="BillingUsageWorkspaceDropdown"]')).toBeVisible();
    await page.locator('[data-alias="BillingUsageWorkspaceDropdown"]').getByText(workspaceName).click();
    await page.getByRole('button', { name: '应用过滤' }).click();

    await expect.poll(() => usageUrls.some((u) => new URL(u).searchParams.get('workspace_id') === workspaceId), {
      timeout: 15000,
    }).toBe(true);

    expect(workspaceUrls.some((u) => new URL(u).searchParams.get('mine') === '1')).toBe(true);
    await expect(page.getByText(workspaceName).first()).toBeVisible();
  });
});
