// @ts-check
/**
 * PeopleGroups 管理分组创建成功与失败路径 E2E 回归测试。
 *
 * OPT-20260719-020 (medium)：
 *   成功创建：弹窗关闭并出现新分组；
 *   失败创建：错误节点 data-traceId 为合法 UUID/web-*，且不得等于中文错误文案。
 *
 * 独立于 TaskDetail 测试，直接访问 PeopleGroups 页面路由。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const PEOPLE_GROUPS_PATH = `/tenant/${TENANT_ID}/people/groups/`;
const USER_ID = 'e2e-user-id-12345';

/**
 * 设置 PeopleGroups 页面基础 mock。
 */
async function setupPeopleGroupsPage(page) {
  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: 'http://localhost:4000/' },
    { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
  ]);

  // 用户信息
  await page.route(`**/api/accounts/users/me/`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        user_id: USER_ID,
        username: 'e2e-user',
        companies: [],
      }),
    });
  });

  // 公司成员
  await page.route(`**/api/tenant/${TENANT_ID}/accounts/members/company_members/`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ members: [] }),
    });
  });
}

test.describe('OPT-20260719-020: PeopleGroups create success/failure', () => {
  test('创建分组成功：弹窗关闭并在列表中看到新分组', async ({ page }) => {
    test.setTimeout(60_000);
    await setupPeopleGroupsPage(page);

    // 分组 API：GET 返回初始列表，POST 返回 201
    await page.route(`**/api/tenant/${TENANT_ID}/accounts/groups/`, async (route) => {
      const method = route.request().method();
      if (method === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'g-new',
            name: '新分组',
            description: '新分组描述',
          }),
        });
        return;
      }
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { id: 'g1', name: '默认分组', description: '系统默认分组', memberCount: 3 },
            { id: 'g-new', name: '新分组', description: '新分组描述', memberCount: 0 },
          ]),
        });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(PEOPLE_GROUPS_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 等待分组列表加载完成
    await expect(page.getByText('默认分组')).toBeVisible({ timeout: 15_000 });

    // 点击「创建分组」
    const createBtn = page.getByRole('button', { name: '创建分组' });
    await expect(createBtn).toBeVisible({ timeout: 5_000 });
    await createBtn.click();

    // 断言创建弹窗出现
    const createModal = page.locator('h3').filter({ hasText: '创建分组' });
    await expect(createModal).toBeVisible({ timeout: 5_000 });

    // 填写分组名称
    const nameInput = page.locator('#groupName');
    await expect(nameInput).toBeVisible();
    await nameInput.fill('新分组');

    // 点击提交
    const submitBtn = page.locator('button[type="submit"]');
    await submitBtn.click();

    // 断言弹窗关闭
    await expect(createModal).not.toBeVisible({ timeout: 5_000 });

    // 断言列表中出现了新分组
    await expect(page.getByText('新分组')).toBeVisible({ timeout: 5_000 });
  });

  test('创建分组失败：错误节点 data-traceId 为合法 UUID/web-*', async ({ page }) => {
    test.setTimeout(60_000);
    await setupPeopleGroupsPage(page);

    const FAIL_TRACE_ID = 'web-e2e-fail-trace-20260719';

    // 分组 API：GET 返回列表，POST 返回 400 + X-Trace-Id
    await page.route(`**/api/tenant/${TENANT_ID}/accounts/groups/`, async (route) => {
      const method = route.request().method();
      if (method === 'POST') {
        await route.fulfill({
          status: 400,
          contentType: 'application/json',
          headers: { 'X-Trace-Id': FAIL_TRACE_ID },
          body: JSON.stringify({
            error: '创建分组失败：分组名称已存在',
            trace_id: FAIL_TRACE_ID,
          }),
        });
        return;
      }
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { id: 'g1', name: '默认分组', description: '系统默认分组', memberCount: 3 },
          ]),
        });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(PEOPLE_GROUPS_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 等待加载完成
    await expect(page.getByText('默认分组')).toBeVisible({ timeout: 15_000 });

    // 点击「创建分组」
    const createBtn = page.getByRole('button', { name: '创建分组' });
    await expect(createBtn).toBeVisible({ timeout: 5_000 });
    await createBtn.click();

    // 填写分组名称
    const nameInput = page.locator('#groupName');
    await expect(nameInput).toBeVisible();
    await nameInput.fill('重复组名');

    // 点击提交
    const submitBtn = page.locator('button[type="submit"]');
    await submitBtn.click();

    // 等待错误弹窗出现
    const errorModal = page.locator('.fixed.inset-0.bg-black\\/50');
    await expect(errorModal).toBeVisible({ timeout: 15_000 });

    // 断言错误文案可见
    await expect(errorModal.getByText('创建分组失败：分组名称已存在')).toBeVisible({ timeout: 5_000 });

    // 获取弹窗内层容器的 data-traceId
    const modalDialog = errorModal.locator('.bg-white.rounded-lg.shadow-xl');
    await expect(modalDialog).toBeVisible();
    const traceIdAttr = await modalDialog.getAttribute('data-traceId');

    // traceId 应存在且不为空
    expect(traceIdAttr).toBeTruthy();
    expect(traceIdAttr).toBe(FAIL_TRACE_ID);

    // traceId 不得等于中文错误文案
    expect(traceIdAttr).not.toMatch(/[一-鿿]/);

    // traceId 应为合法格式
    expect(traceIdAttr).toMatch(/^[A-Za-z0-9][A-Za-z0-9._:-]{1,127}$/);
  });
});
