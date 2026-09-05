// @ts-check
/**
 * 回归：WorkPanel 交付物过滤栏下拉展示任务编号 #N。
 * 纯 mock，无真实账号依赖。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || '827923618602258432';
const USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488';
const TYPE_ID = '2000000000000000201';

test.describe('WorkPanel 过滤栏下拉任务编号', () => {
  test('同标题任务 option 以 #序号 区分', async ({ page }) => {
    await page.context().addCookies([
      { name: 'userId', value: USER_ID, url: BASE_URL },
      { name: 'sessionid', value: 'e2e-session-filter-no', url: BASE_URL },
    ]);

    // 兜底 catch-all 必须最先注册（OPT-20260824-068：Playwright route 为 LIFO，
    // 后注册先匹配；放最后会抢占 /me/ 等具体 mock 致守卫跳 /onboarding/）。
    await page.route('**/api/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });
    await page.route(`**/api/auth/user-permissions/**`, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json',
        body: JSON.stringify({
          tenant_perms: {
            [TENANT_ID]: ['project:view', 'project:manage', 'task:view', 'task:manage', 'cloud:view', 'cloud:manage'],
          },
        }),
      });
    });

    await page.route(`**/api/accounts/users/me/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: USER_ID,
          username: 'e2e',
          current_company: { id: TENANT_ID, name: 'E2E Tenant' },
          companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
          current_workspace: { id: WORKSPACE_ID, name: 'E2E WS' },
        }),
      });
    });

    await page.route(`**/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'task-a',
            title: '写一个 hello world程序',
            workspace_seq: 3,
            progress_column_id: 1,
            deliverable_obj_id: TYPE_ID,
            task_type: { id: TYPE_ID, name: '价值流' },
          },
          {
            id: 'task-b',
            title: '写一个 hello world程序',
            workspace_seq: 8,
            progress_column_id: 1,
            deliverable_obj_id: TYPE_ID,
            task_type: { id: TYPE_ID, name: '价值流' },
          },
        ]),
      });
    });

    await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          columns: [
            { id: 0, name: '待处理', order: 0 },
            { id: 1, name: '进行中', order: 1 },
            { id: 2, name: '已完成', order: 2 },
          ],
        }),
      });
    });

    await page.route(`**/manage-deliverable-system/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          current_deliverable_objs: [{ id: TYPE_ID, name: '价值流', color: '#111', order: 1 }],
        }),
      });
    });

    await page.route(`**/work-panel-filters/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ deliverable_filter_bars: [] }),
      });
    });

    await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}**`, async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: WORKSPACE_ID, name: 'E2E WS', is_current: true, is_default: true }]),
        });
        return;
      }
      await route.continue();
    });

    await page.route('**/cloud/compute/workspace-runtime-indicators/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', indicators: [] }),
      });
    });
    await page.route('**/cloud/compute/workspace-machine-summary/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', started_count: 0, idle_count: 0, busy_count: 0 }),
      });
    });

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`);
    await page.waitForLoadState('domcontentloaded');

    const select = page.locator('[data-alias="deliverable-content-select"]').first();
    await expect(select).toBeVisible({ timeout: 45000 });
    const optionTexts = await select.locator('option').allTextContents();
    expect(optionTexts.map((t) => t.trim())).toEqual(
      expect.arrayContaining(['#3 写一个 hello world程序', '#8 写一个 hello world程序']),
    );
  });
});
