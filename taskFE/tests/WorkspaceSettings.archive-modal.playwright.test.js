// @ts-check
/**
 * 回归：settings/task-panel「套餐设置」点击必须打开任务存档档位模态（OPT-20260827-047）。
 * - 进入 task-panel 页面后工作空间行可见
 * - 点击 `[data-testid=open-task-archive-settings]` 打开模态
 * - 断言模态标题「套餐设置 · 任务存档时间」与档位下拉/保存按钮存在
 *
 * 纯 mock，无真实账号依赖；参考 WorkspaceSettings.priority-field-toggle.playwright.test.js。
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
const WS_ID = 'ws-archive-modal';

test('settings/task-panel: 套餐设置点击打开任务存档模态', async ({ page }) => {
  test.setTimeout(120000);

  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-archive-modal', url: BASE_URL },
    { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
  ]);

  await page.route(/\/api\/.*/, async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    // 用户信息（Navbar 依赖；has_phone=true 避免登录后手机号门禁弹窗）
    if (url.includes(`/api/accounts/users/me/`)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: USER_ID,
          username: 'e2e',
          has_phone: true,
          current_company: { id: TENANT_ID, name: 'E2E Tenant' },
          companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
          current_workspace: { id: WS_ID, name: '默认工作空间' },
        }),
      });
      return;
    }

    // 租户页面访问判定（usePermissions → useTenantPageAccess）：授权 settings.task_panel
    if (url.includes('/api/auth/user-permissions/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          tenant_perms: {
            [TENANT_ID]: [
              'page:settings.task_panel',
              'page:settings.task_panel:view',
              'page:settings.task_panel:operate',
            ],
          },
        }),
      });
      return;
    }
    if (url.includes('/api/auth/user-roles/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ roles: [] }) });
      return;
    }

    // 工作空间列表（渲染每行 + 套餐设置按钮）
    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { id: WS_ID, name: '默认工作空间', is_current: true, is_default: true, text: '默认工作空间', value: WS_ID, task_archive_tier: '7d' },
          { id: 'ws-extra', name: '额外空间', is_current: false, is_default: false, text: '额外空间', value: 'ws-extra', task_archive_tier: '30d' },
        ]),
      });
      return;
    }

    // 其余 API 一律返回中性 200（防止真实后端 401 → redirect_url 跳登录）
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/settings/task-panel/`);
  await page.waitForLoadState('domcontentloaded');

  // 等待工作空间行渲染（含非默认行的套餐设置按钮）
  const archiveBtn = page.locator('[data-testid="open-task-archive-settings"]').first();
  await archiveBtn.waitFor({ state: 'visible', timeout: 45000 });

  // 点击「套餐设置」打开任务存档模态
  await archiveBtn.click();
  const modal = page.locator('[data-testid="task-archive-settings-modal"]');
  await modal.waitFor({ state: 'visible', timeout: 15000 });

  // 断言模态标题与档位下拉/保存按钮存在
  await expect(modal.locator('h4')).toHaveText('套餐设置 · 任务存档时间');
  await expect(page.locator('[data-testid="task-archive-tier-select"]')).toBeVisible();
  await expect(page.locator('[data-testid="task-archive-tier-save"]')).toBeVisible();
});
