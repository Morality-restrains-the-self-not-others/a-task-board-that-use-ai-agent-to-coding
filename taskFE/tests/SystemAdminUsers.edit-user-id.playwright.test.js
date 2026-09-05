// @ts-check
/**
 * 回归：系统管理用户页「编辑用户」弹窗必须展示只读用户ID（OPT-20260823-022）。
 * - 以系统管理员进入 /system-admin/users/，用户行可见
 * - 点击行内「编辑」按钮打开编辑弹窗
 * - 断言 `#edit-user-id` 可见、readonly、值等于该行用户 ID
 *
 * 纯 mock，无真实账号依赖；参考 WorkspaceSettings.archive-modal.playwright.test.js。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const ADMIN_USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488';
// 目标用户行 ID：编辑弹窗必须回显该值
const TARGET_USER_ID = process.env.PLAYWRIGHT_TARGET_USER_ID || '837920218451263488';
const TARGET_USER_EMAIL = 'e2e-target@example.com';

test('system-admin/users: 编辑用户弹窗展示只读用户ID', async ({ page }) => {
  test.setTimeout(120000);

  await page.context().addCookies([
    { name: 'userId', value: ADMIN_USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-edit-user-id', url: BASE_URL },
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
          id: ADMIN_USER_ID,
          username: 'e2e-admin',
          is_superuser: true,
          has_phone: true,
          current_company: { id: TENANT_ID, name: 'E2E Tenant' },
          companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
          current_workspace: { id: 'ws-e2e', name: '默认工作空间' },
        }),
      });
      return;
    }

    // 平台角色：super_admin（供 canSetSuperuser / impersonate 按钮判定）
    if (url.includes('/api/auth/user-roles/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          roles: [{ role: 'super_admin', level: 1, company_id: null }],
        }),
      });
      return;
    }

    // 平台/租户权限
    if (url.includes('/api/auth/user-permissions/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ tenant_perms: {} }),
      });
      return;
    }

    // 用户列表（UserListRow 渲染 + 点击「编辑」回填 editUserForm / editUserId）
    if (url.includes('/api/system-admin/users/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          users: [
            {
              id: TARGET_USER_ID,
              username: 'e2e-target',
              email: TARGET_USER_EMAIL,
              phone: '',
              is_active: true,
              is_superuser: false,
              is_staff: false,
              is_tenant: false,
              is_tester: false,
              is_archived: false,
              date_joined: '2026-07-01T00:00:00Z',
              last_login: '2026-08-01T00:00:00Z',
              login_methods: [{ method_type: 'email', is_verified: true }],
              referrer_code: 'E2E001',
              tenant_companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
            },
          ],
          total: 1,
        }),
      });
      return;
    }

    // 其余 API 一律返回中性 200（防止真实后端 401 → redirect_url 跳登录）
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await page.goto(`${BASE_URL}/system-admin/users/`);
  await page.waitForLoadState('domcontentloaded');

  // 等待目标用户行渲染
  const row = page.locator('tr', { hasText: TARGET_USER_EMAIL });
  await row.waitFor({ state: 'visible', timeout: 45000 });

  // 点击行内「编辑」打开编辑用户弹窗
  await row.getByRole('button', { name: '编辑' }).click();

  const idField = page.locator('#edit-user-id');
  await idField.waitFor({ state: 'visible', timeout: 15000 });

  // 断言用户 ID 展示：可见、readonly、值等于该行 ID
  await expect(idField).toBeVisible();
  await expect(idField).toHaveAttribute('readonly', '');
  await expect(idField).toHaveValue(TARGET_USER_ID);
});
