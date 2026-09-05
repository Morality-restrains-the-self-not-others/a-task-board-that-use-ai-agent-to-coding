// @ts-check
/**
 * 核验：使用记录表格「项目」列可展示多项目名称（用「、」拼接），与多项目任务一致。
 * 通过 route 模拟 billing/usages 返回；不依赖库内是否已有该任务数据。
 * 需与 playwright.config.js 的 webServer（Django + Vite）一致，或本机已起服务且 reuseExistingServer。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

test.describe('Billing 使用记录 多项目列', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('项目列展示多个项目名称', async ({ page, baseURL }) => {
    test.setTimeout(90_000);
    const origin = baseURL || 'http://127.0.0.1:4000';
    const userId = 'playwright-e2e';
    const tenantId = '827923618468040704';
    const p1 = '项目甲-MPROJ';
    const p2 = '项目乙-MPROJ';
    const taskTitle = '05071851 把 somanyad 运行起来';

    await page.context().addCookies([{ name: 'userId', value: userId, url: origin }]);

    // AuthSessionGuard 会校验 /me/，仅设 userId cookie 不够（否则会落到登录页）
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

    await page.route('**/api/tenant/*/billing/usages/**', async (route) => {
      if (route.request().method() !== 'GET') {
        return route.continue();
      }
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'usage-mock-1',
            usage_time: new Date().toISOString(),
            billing_unit: { name: '任务服务器启动服务费', unit: '次' },
            amount: '1',
            project_id: 'mock-proj-a',
            project_name: `${p1}、${p2}`,
            user_id: 'u1',
            user_name: '测试用户',
            workspace_id: 'ws1',
            workspace_name: '工作空间',
            task_id: 'task-mock-1',
            task_name: taskTitle,
            description: '任务服务器启动 1 次',
            account: 'acc1',
          },
        ]),
      });
    });

    await page.goto(`/tenant/${tenantId}/billing/usage/`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: '使用记录', exact: true })).toBeVisible({
      timeout: 20000,
    });
    const expected = `${p1}、${p2}`;
    await expect(page.locator('tbody').getByText(expected, { exact: true })).toBeVisible();
    await expect(page.getByText(taskTitle).first()).toBeVisible();
  });
});
