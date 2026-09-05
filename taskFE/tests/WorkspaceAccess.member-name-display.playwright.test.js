// @ts-check
/**
 * 回归：AccessManagementModal「工作空间访问成员」列表应展示 member_name，
 * 且不出现纯 user_id 前 8 位文案（如 "82792361"）作为成员标识。
 *
 * 触发路径：settings/task-panel → 点击「访问管理」→ AccessManagementModal 打开
 * → 调用 GET workspace-permissions API → 渲染成员行。
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
const WORKSPACE_ID = 'ws-e2e-member-name';

// 模拟返回含 member_name 的 workspace-permissions 响应
const MOCK_PERMISSIONS = [
  {
    id: '1001',
    workspace: WORKSPACE_ID,
    user: '827923618468040704',
    group: null,
    role: 'admin',
    user_info: {
      id: '827923618468040704',
      company_member_id: 'cm-001',
      member_name: '张三',
      username: 'zhangsan',
      email: 'zhangsan@example.com',
      is_tenant: false,
    },
    group_info: null,
    created_at: '2026-07-19T00:00:00Z',
  },
  {
    id: '1002',
    workspace: WORKSPACE_ID,
    user: '827923618602258433',
    group: null,
    role: 'view',
    user_info: {
      id: '827923618602258433',
      company_member_id: 'cm-002',
      member_name: '李四的账号',
      username: 'lisi',
      email: 'lisi@example.com',
      is_tenant: false,
    },
    group_info: null,
    created_at: '2026-07-19T00:00:00Z',
  },
];

test.describe('AccessManagementModal 成员名称展示', () => {
  test('成员行显示 member_name，不出现纯 user_id 前 8 位文案', async ({ page }) => {
    test.setTimeout(120000);

    await page.context().addCookies([
      { name: 'userId', value: USER_ID, url: BASE_URL },
      { name: 'sessionid', value: 'e2e-session-member-name', url: BASE_URL },
      { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
    ]);

    // Mock 用户信息 API
    await page.route(`**/api/accounts/users/me/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: USER_ID,
          username: 'e2e',
          current_company: { id: TENANT_ID, name: 'E2E Tenant' },
          companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
          current_workspace: { id: WORKSPACE_ID, name: '测试空间' },
        }),
      });
    });

    // Mock 工作空间列表
    await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}**`, async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { id: WORKSPACE_ID, name: '测试空间', is_current: true, is_default: true, text: '测试空间', value: WORKSPACE_ID },
          ]),
        });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    // Mock 机器策略
    await page.route('**/cloud/compute/workspace-machine-policy/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', idle_recycle_minutes: 30, prefer_idle_reuse: true, enabled_authorization_ids: [] }) });
    });

    // Mock workspace-permissions API — 核心 mock：返回含 member_name 的成员列表
    await page.route(`**/api/projects/workspace-access/workspace-permissions/tenant_id/${TENANT_ID}/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_PERMISSIONS),
      });
    });

    // 兜底：其他 API 返回空对象
    await page.route('**/api/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    // 导航到工作空间设置页
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/settings/task-panel/`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    // 点击「访问管理」按钮打开 AccessManagementModal
    const accessBtn = page.locator('button').filter({ hasText: '访问管理' }).first();
    await expect(accessBtn).toBeVisible({ timeout: 10000 });
    await accessBtn.click();
    await page.waitForTimeout(1500);

    // 断言：成员行显示 member_name
    // AccessMemberList 渲染 member.member_name 在 p.font-medium 元素中
    const memberName张三 = page.getByText('张三', { exact: true });
    await expect(memberName张三).toBeVisible({ timeout: 10000 });

    const memberName李四 = page.getByText('李四的账号', { exact: true });
    await expect(memberName李四).toBeVisible({ timeout: 10000 });

    // 断言：不出现纯 user_id 前 8 位文案（如 "82792361"）
    // 当 member_name 缺失时，代码回退显示 '未知用户'，而非截断的 user_id
    const truncatedId = page.locator('div.space-y-3').filter({ hasText: '82792361' });
    await expect(truncatedId).toHaveCount(0);

    // 也验证邮箱等额外信息正常展示
    await expect(page.getByText('zhangsan@example.com')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('lisi@example.com')).toBeVisible({ timeout: 5000 });
  });
});
