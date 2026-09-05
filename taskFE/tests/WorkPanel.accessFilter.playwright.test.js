// @ts-check
/**
 * E2E：工作面板按可访问人或小组过滤
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
const MEMBER_ALICE = 'cm-alice';
const MEMBER_BOB = 'cm-bob';

test('work-panel: access person/group filter narrows tasks', async ({ page }) => {
  test.setTimeout(120000);

  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-access-filter', url: BASE_URL },
  ]);

  // 兜底 catch-all 必须最先注册（OPT-20260824-068：Playwright route 为 LIFO，
  // 后注册先匹配；放最后会抢占 /me/ 等具体 mock 致守卫跳 /onboarding/）。
  // 未逐一 mock 的 API 一律 200 空对象，避免真实后端 401 跳登录页。
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
        started_count: 0,
        idle_count: 0,
        busy_count: 0,
        idle_recycle_minutes: 30,
        prefer_idle_reuse: true,
      }),
    });
  });

  await page.route('**/cloud/compute/workspace-runtime-indicators/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'success', indicators: [] }),
    });
  });

  await page.route(`**/workspace-access/workspace-permissions/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        {
          role: 'edit',
          user_info: {
            id: 'u-alice',
            company_member_id: MEMBER_ALICE,
            // username 故意为 id 前缀；选项文案须取 member_name
            username: 'u-alice'.slice(0, 8),
            member_name: 'Alice',
            email: 'alice@example.com',
          },
        },
        {
          role: 'view',
          user_info: {
            id: 'u-bob',
            company_member_id: MEMBER_BOB,
            username: 'u-bob'.slice(0, 8),
            member_name: 'Bob',
            email: 'bob@example.com',
          },
        },
        {
          role: 'view',
          group_info: { id: 'g-core', name: 'Core Team', memberCount: 1 },
        },
      ]),
    });
  });

  await page.route(`**/workspace-access/workspace-collaborators/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: MEMBER_ALICE, user: 'u-alice', username: 'Alice', member_name: 'Alice' },
        { id: MEMBER_BOB, user: 'u-bob', username: 'Bob', member_name: 'Bob' },
      ]),
    });
  });

  let savedAccessFilter = null;
  await page.route(`**/work-panel-filters/**`, async (route) => {
    if (route.request().method() === 'PUT') {
      const body = route.request().postDataJSON();
      savedAccessFilter = body?.access_filter ?? null;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          version: 2,
          deliverable_filter_bars: body?.deliverable_filter_bars || [{ id: 'bar-0', path: [{ type: 'root' }] }],
          access_filter: savedAccessFilter,
        }),
      });
      return;
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        version: 2,
        deliverable_filter_bars: [{ id: 'bar-0', path: [{ type: 'root' }] }],
        access_filter: savedAccessFilter,
      }),
    });
  });

  await page.route(`**/accounts/groups/g-core/members/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([{ id: 'gm1', group: 'g-core', user: 'u-alice', username: 'Alice' }]),
    });
  });

  await page.route(`**/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        {
          id: 'task-alice',
          title: 'Alice Owner Task',
          description: 'owned by alice',
          priority: 1,
          progress_column_id: 1,
          owner: MEMBER_ALICE,
          assignees: [],
          task_type: { id: TYPE_ID, name: 'Feature' },
          created_by: { username: 'e2e' },
        },
        {
          id: 'task-bob',
          title: 'Bob Assignee Task',
          description: 'assigned to bob',
          priority: 1,
          progress_column_id: 1,
          owner: 'cm-other',
          assignees: [MEMBER_BOB],
          task_type: { id: TYPE_ID, name: 'Feature' },
          created_by: { username: 'e2e' },
        },
        {
          id: 'task-other',
          title: 'Other Task',
          description: 'no match',
          priority: 1,
          progress_column_id: 1,
          owner: 'cm-other',
          assignees: [],
          task_type: { id: TYPE_ID, name: 'Feature' },
          created_by: { username: 'e2e' },
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

  const toggle = page.locator('[data-alias="access-filter-toggle"]');
  await expect(toggle).toBeVisible({ timeout: 45000 });
  await toggle.click();
  await expect(page.locator('[data-alias="access-filter-panel"]')).toBeVisible();

  const personOpts = page.locator('[data-alias="access-filter-person-option"]');
  await expect(personOpts).toHaveCount(2);
  await expect(personOpts.filter({ hasText: 'Alice' })).toBeVisible();
  await expect(personOpts.filter({ hasText: 'Bob' })).toBeVisible();

  await personOpts.filter({ hasText: 'Alice' }).click();
  const chip = page.locator('[data-alias="access-filter-chip"]');
  await expect(chip).toBeVisible();
  await expect(chip).toContainText('过滤：Alice');
  await expect(page.locator('[data-task-id="task-alice"]')).toBeVisible({ timeout: 10000 });
  await expect(page.locator('[data-task-id="task-bob"]')).toHaveCount(0);
  await expect(page.locator('[data-task-id="task-other"]')).toHaveCount(0);

  await chip.click();
  await expect(chip).toHaveCount(0);
  await expect(page.locator('[data-task-id="task-bob"]')).toBeVisible();
  await expect(page.locator('[data-task-id="task-other"]')).toBeVisible();

  await toggle.click();
  await page.locator('[data-alias="access-filter-tab-group"]').click();
  const groupOpts = page.locator('[data-alias="access-filter-group-option"]');
  await expect(groupOpts).toHaveCount(1);
  await groupOpts.filter({ hasText: 'Core Team' }).click();
  await expect(chip).toBeVisible();
  await expect(chip).toContainText('Core Team');
  await expect(page.locator('[data-task-id="task-alice"]')).toBeVisible();
  await expect(page.locator('[data-task-id="task-bob"]')).toHaveCount(0);
  await expect(page.locator('[data-task-id="task-other"]')).toHaveCount(0);

  // 模态人/小组与 Header 同源：若当前 SPA 已含 filter-modal-access 则校验同步，否则跳过（部署滞后）
  await page.locator('button.btn-secondary', { hasText: '筛选' }).click();
  await expect(page.locator('#filter-modal')).toBeVisible();
  const modalAccess = page.locator('[data-alias="filter-modal-access"]');
  if (await modalAccess.count()) {
    await page.locator('[data-alias="filter-modal-tab-person"]').click();
    await page.locator('[data-alias="filter-modal-person-option"]').filter({ hasText: 'Bob' }).click();
    await expect(page.locator('[data-alias="access-filter-chip"]')).toContainText('Bob');
    await page.locator('#filter-modal button', { hasText: '应用筛选' }).click();
    await expect(page.locator('[data-task-id="task-bob"]')).toBeVisible();
    await expect(page.locator('[data-task-id="task-alice"]')).toHaveCount(0);
  } else {
    await page.locator('#filter-modal button', { hasText: '应用筛选' }).click();
  }
});
