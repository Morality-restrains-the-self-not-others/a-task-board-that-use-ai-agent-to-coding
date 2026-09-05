// @ts-check
/**
 * 回归：settings/task-panel 工作空间删除按钮显隐。
 * - 带「默认」badge 的工作空间不应有删除按钮
 * - 非默认工作空间应有删除按钮
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
const USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488';

test('settings/task-panel: 默认工作空间无删除按钮，非默认有删除按钮', async ({ page }) => {
  test.setTimeout(120000);

  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-ws-delete', url: BASE_URL },
    { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
  ]);

  const defaultWs = { id: 'ws-default', name: '默认工作空间', is_current: true, is_default: true, text: '默认工作空间', value: 'ws-default' };
  const nonDefaultWs = { id: 'ws-extra', name: '额外空间', is_current: false, is_default: false, text: '额外空间', value: 'ws-extra' };

  await page.route(`**/api/accounts/users/me/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: USER_ID,
        username: 'e2e',
        current_company: { id: TENANT_ID, name: 'E2E Tenant' },
        companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
        current_workspace: { id: 'ws-default', name: '默认工作空间' },
      }),
    });
  });

  await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}**`, async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([defaultWs, nonDefaultWs]),
      });
      return;
    }
    // 其它方法（如 POST 创建工作空间）返回空
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await page.route('**/cloud/compute/workspace-machine-policy/**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', idle_recycle_minutes: 30, prefer_idle_reuse: true, enabled_authorization_ids: [] }) });
      return;
    }
    await route.continue();
  });

  await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/settings/task-panel/`);
  await page.waitForLoadState('domcontentloaded');
  await page.waitForTimeout(2500);

  // 至少有一个 workspace 行
  const wsRows = page.locator('.border.border-gray-200.rounded-lg');
  const rowCount = await wsRows.count();
  expect(rowCount).toBeGreaterThanOrEqual(2);

  // 找出包含 "默认" 文字的 workspace 行
  let defaultRow = null;
  let nonDefaultRow = null;
  // 默认 badge 是文本 "默认"，在 v-if="workspace.is_current" 条件下显示
  for (let i = 0; i < rowCount; i++) {
    const row = wsRows.nth(i);
    const text = await row.innerText();
    // 默认按钮在 workspace.is_current=true 的行显示"默认"badge
    if (text.includes('默认')) {
      // 注意：该行的 "默认" 可能是 space name（默认工作空间）或 badge
      defaultRow = row;
    } else {
      nonDefaultRow = row;
    }
  }
  // 若仅通过文字无法区分，使用 data 属性
  if (!defaultRow || !nonDefaultRow) {
    // fallback: 按 "默认工作空间" 标题文字定位
    defaultRow = page.locator('.border.border-gray-200.rounded-lg').filter({ hasText: '默认工作空间' }).first();
    nonDefaultRow = page.locator('.border.border-gray-200.rounded-lg').filter({ hasText: '额外空间' }).first();
  }

  // 默认工作空间行不应有删除按钮
  const defaultDeleteBtn = defaultRow.locator('[data-testid="workspace-delete"]');
  await expect(defaultDeleteBtn).toHaveCount(0);

  // 非默认工作空间行应有删除按钮
  const nonDefaultDeleteBtn = nonDefaultRow.locator('[data-testid="workspace-delete"]');
  await expect(nonDefaultDeleteBtn).toBeVisible({ timeout: 5000 });
});
