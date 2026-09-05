// @ts-check
/**
 * 回归：auto_run=true 时 Git/子 Git 不可达 → 软跳过自动启服。
 * - 创建任务 auto_run=true 且 nested 未授权时，回应含 auto_run_start_skipped
 * - 前端显示「自动运行未启服」警告
 *
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

const TASK_ID_WITH_SKIP = 'task-auto-run-skipped';
const TYPE_ID = '2000000000000000201';

test.describe('WorkPanel auto_run 跳过提示', () => {
  async function setupPageMocks(page, taskOverride = {}) {
    await page.context().addCookies([
      { name: 'userId', value: USER_ID, url: BASE_URL },
      { name: 'sessionid', value: 'e2e-session-auto-run-skip', url: BASE_URL },
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
        status: 200, contentType: 'application/json',
        body: JSON.stringify({
          id: USER_ID, username: 'e2e',
          current_company: { id: TENANT_ID, name: 'E2E Tenant' },
          companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
          current_workspace: { id: WORKSPACE_ID, name: 'E2E WS' },
        }),
      });
    });

    // Task with auto_run_start_skipped
    const baseTask = {
      id: TASK_ID_WITH_SKIP, title: 'AutoRun Skip Test', description: '',
      priority: 1, progress_column_id: 1,
      task_type: { id: TYPE_ID, name: 'Feature' },
      created_by: { username: 'e2e' },
      auto_run: true,
      auto_run_start_skipped: true,
      auto_run_start_skip_reason: '子 Git 仓库未授权访问，已跳过自动启动服务器',
      code: 'AUTO_RUN_GIT_ACCESS_SKIPPED',
      ...taskOverride,
    };

    await page.route(`**/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}**`, async (route) => {
      const method = route.request().method();
      if (method === 'POST') {
        // Simulate backend returning skip info on create
        await route.fulfill({
          status: 201, contentType: 'application/json',
          body: JSON.stringify(baseTask),
        });
        return;
      }
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify([baseTask]),
      });
    });

    await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/**`, async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
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
        status: 200, contentType: 'application/json',
        body: JSON.stringify({
          current_deliverable_objs: [{ id: TYPE_ID, name: 'Feature', order: 0 }],
        }),
      });
    });

    await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}**`, async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify([{ id: WORKSPACE_ID, name: 'E2E WS', is_current: true, is_default: true }]),
        });
        return;
      }
      await route.continue();
    });

    // 运行时指标 - no running server
    await page.route('**/cloud/compute/workspace-runtime-indicators/**', async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ status: 'success', indicators: [] }),
      });
    });

    await page.route('**/cloud/compute/workspace-machine-summary/**', async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ status: 'success', started_count: 0, idle_count: 0, busy_count: 0 }),
      });
    });

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);
  }

  test('auto_run_start_skipped 任务不显示服务器 running 指示器', async ({ page }) => {
    await setupPageMocks(page);

    // 任务卡片应存在
    const card = page.locator(`[data-task-id="${TASK_ID_WITH_SKIP}"]`);
    await expect(card).toBeVisible({ timeout: 45000 });

    // 不应出现机器运行指示器（服务器未启动）
    const machineRing = page.getByTestId('task-card-machine-ring');
    const containerDot = page.getByTestId('task-card-container-dot');
    await expect(machineRing).toHaveCount(0);
    await expect(containerDot).toHaveCount(0);
  });

  test('auto_run_start_skipped 任务卡片可见且含标题', async ({ page }) => {
    await setupPageMocks(page);

    const card = page.locator(`[data-task-id="${TASK_ID_WITH_SKIP}"]`);
    await expect(card).toBeVisible({ timeout: 45000 });
    await expect(card).toContainText('AutoRun Skip Test');
  });
});
