// @ts-check
/**
 * E2E：工作面板机器节点摘要条 + 已启动/闲置点击过滤
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

test('work-panel: machine summary click filters tasks', async ({ page }) => {
  test.setTimeout(120000);

  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-machine-summary', url: BASE_URL },
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

  await page.route('**/cloud/compute/workspace-machine-summary/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        started_count: 2,
        idle_count: 1,
        busy_count: 1,
        idle_recycle_minutes: 30,
        prefer_idle_reuse: true,
      }),
    });
  });

  await page.route('**/cloud/compute/workspace-runtime-indicators/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        indicators: [
          { task_id: 'task-busy', machine_running: true, container_running: true },
          { task_id: 'task-idle', machine_running: true, container_running: false },
          { task_id: 'task-off', machine_running: false, container_running: false },
        ],
      }),
    });
  });

  await page.route(`**/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        {
          id: 'task-busy',
          title: 'Busy Machine Task',
          description: 'has container',
          priority: 1,
          progress_column_id: 1,
          task_type: { id: TYPE_ID, name: 'Feature' },
          created_by: { username: 'e2e' },
        },
        {
          id: 'task-idle',
          title: 'Idle Machine Task',
          description: 'machine only',
          priority: 1,
          progress_column_id: 1,
          task_type: { id: TYPE_ID, name: 'Feature' },
          created_by: { username: 'e2e' },
        },
        {
          id: 'task-off',
          title: 'Off Machine Task',
          description: 'no machine',
          priority: 1,
          progress_column_id: 1,
          task_type: { id: TYPE_ID, name: 'Feature' },
          created_by: { username: 'e2e' },
        },
      ]),
    });
  });

  // progress-system 失败时前端回退默认列；仍 mock 成功以稳定看板
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
        current_deliverable_objs: [{ id: TYPE_ID, name: 'Feature', order: 0 }],
      }),
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

  await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`);
  await page.waitForLoadState('domcontentloaded');
  await page.waitForTimeout(3000);

  const summaryEl = page.locator('[data-alias="workspace-machine-summary"]');
  await expect(summaryEl).toBeVisible({ timeout: 45000 });
  await expect(summaryEl.locator('[data-testid="machine-summary-started-count"]')).toHaveText('2');
  await expect(summaryEl.locator('[data-testid="machine-summary-idle-count"]')).toHaveText('1');

  const infoBtn = page.locator('[data-alias="workspace-machine-summary-info"]');
  await infoBtn.hover();
  const tooltip = page.locator('[data-testid="workspace-machine-summary-tooltip"]');
  await expect(tooltip).toBeVisible();
  await expect(tooltip).toContainText(/闲置回收：30\s*分钟/);

  // E1: 可点击控件
  const startedBtn = page.locator('[data-alias="machine-filter-started"]');
  const idleBtn = page.locator('[data-alias="machine-filter-idle"]');
  await expect(startedBtn).toBeVisible();
  await expect(idleBtn).toBeVisible();

  // E2: 点击闲置 → 仅闲置任务
  await idleBtn.click();
  const chip = page.locator('[data-alias="machine-status-filter-chip"]');
  await expect(chip).toBeVisible();
  await expect(chip).toContainText('过滤：闲置');
  await expect(page.locator('[data-task-id="task-idle"]')).toBeVisible({ timeout: 10000 });
  await expect(page.locator('[data-task-id="task-busy"]')).toHaveCount(0);
  await expect(page.locator('[data-task-id="task-off"]')).toHaveCount(0);

  // 点击已启动 → 忙+闲
  await startedBtn.click();
  await expect(chip).toContainText('过滤：已启动');
  await expect(page.locator('[data-task-id="task-busy"]')).toBeVisible();
  await expect(page.locator('[data-task-id="task-idle"]')).toBeVisible();
  await expect(page.locator('[data-task-id="task-off"]')).toHaveCount(0);

  // E3: 清除 chip
  await chip.click();
  await expect(chip).toHaveCount(0);
  await expect(page.locator('[data-task-id="task-off"]')).toBeVisible();
});
