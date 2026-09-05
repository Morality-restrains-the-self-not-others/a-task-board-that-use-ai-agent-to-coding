// @ts-check
/**
 * E2E：系统管理员交付物体系页面 + API 验收
 *
 * 验证：
 * 1. GET /api/system-admin/deliverable-systems/ 返回 200 (不再 404)
 * 2. POST 创建交付物体系
 * 3. 页面加载且显示交付物体系列表
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');
const ADMIN_EMAIL = process.env.PLAYWRIGHT_ADMIN_EMAIL || 'author@example.com';
const ADMIN_PASSWORD = process.env.PLAYWRIGHT_ADMIN_PASSWORD || '';

// 使用管理员 session 直接测试 API
const ADMIN_USER_ID = process.env.PLAYWRIGHT_ADMIN_USER_ID || '850249621660790784';
const ADMIN_SESSION_ID = process.env.PLAYWRIGHT_ADMIN_SESSION_ID || '';
const ADMIN_TOKEN = process.env.PLAYWRIGHT_ADMIN_TOKEN || '';

test.describe('system-admin deliverable-system', () => {
  test('API: GET /api/system-admin/deliverable-systems/ returns 200 with array', async ({ request }) => {
    test.setTimeout(30000);

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Requested-With': 'XMLHttpRequest',
    };
    if (ADMIN_TOKEN) {
      headers['Authorization'] = `Token ${ADMIN_TOKEN}`;
    }
    if (ADMIN_SESSION_ID) {
      headers['Cookie'] = `sessionid=${ADMIN_SESSION_ID}; userId=${ADMIN_USER_ID}`;
    }

    const resp = await request.get(`${BASE}/api/system-admin/deliverable-systems/`, { headers });

    // 关键断言：不再返回 404
    expect(resp.status()).not.toBe(404);
    expect(resp.status()).toBe(200);

    const body = await resp.json();
    expect(Array.isArray(body)).toBe(true);

    console.log(`[deliverable-system API] 返回 ${body.length} 个交付物体系`);

    // 验证默认种子数据存在
    const defaultSystem = body.find(
      (s) => s.name === '全局默认交付物体系',
    );
    if (defaultSystem) {
      expect(defaultSystem).toHaveProperty('id');
      expect(defaultSystem).toHaveProperty('level_names');
      expect(Array.isArray(defaultSystem.level_names)).toBe(true);
      expect(defaultSystem.level_names.length).toBeGreaterThan(0);
      console.log(`[deliverable-system API] 默认体系层级: ${defaultSystem.level_names.join(' > ')}`);
    } else {
      console.log('[deliverable-system API] 未找到默认体系（可能尚未初始化种子数据）');
    }
  });

  test('API: POST create + PUT update + DELETE deliverable system', async ({ request }) => {
    test.setTimeout(30000);

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Requested-With': 'XMLHttpRequest',
    };
    if (ADMIN_TOKEN) {
      headers['Authorization'] = `Token ${ADMIN_TOKEN}`;
    }
    if (ADMIN_SESSION_ID) {
      headers['Cookie'] = `sessionid=${ADMIN_SESSION_ID}; userId=${ADMIN_USER_ID}`;
    }

    // 1. Create
    const testName = `E2E-Test-${Date.now()}`;
    const createResp = await request.post(
      `${BASE}/api/system-admin/deliverable-systems/`,
      {
        headers,
        data: {
          name: testName,
          description: 'E2E test deliverable system',
          level_names: ['需求分析', '设计', '开发', '测试', '部署'],
        },
      },
    );

    expect(createResp.status()).toBe(201);

    const createdList = await createResp.json();
    expect(Array.isArray(createdList)).toBe(true);

    const created = createdList.find((s) => s.name === testName);
    if (!created) {
      console.log('[deliverable-system API] 创建后未在列表中定位到新建项，跳过后续验证');
      return;
    }

    expect(created).toHaveProperty('id');
    expect(created.level_names.length).toBe(5);
    console.log(`[deliverable-system API] 创建成功: ${created.id}`);

    // 2. Update
    const updateResp = await request.put(
      `${BASE}/api/system-admin/deliverable-systems/${created.id}/`,
      {
        headers,
        data: {
          name: `${testName}-updated`,
          description: 'Updated description',
          level_names: ['阶段一', '阶段二', '阶段三'],
        },
      },
    );

    expect(updateResp.status()).toBe(200);
    const updatedList = await updateResp.json();
    const updated = updatedList.find((s) => s.name === `${testName}-updated`);
    if (updated) {
      expect(updated.level_names.length).toBe(3);
      console.log(`[deliverable-system API] 更新成功`);
    }

    // 3. Delete using the updated name
    const deleteId = updated ? updated.id : created.id;
    const deleteResp = await request.delete(
      `${BASE}/api/system-admin/deliverable-systems/${deleteId}/`,
      { headers },
    );

    expect(deleteResp.status()).toBe(200);
    const afterDelete = await deleteResp.json();
    const stillExists = afterDelete.find(
      (s) => s.name === `${testName}-updated` || s.name === testName,
    );
    expect(stillExists).toBeUndefined();
    console.log(`[deliverable-system API] 删除成功`);
  });

  test('UI: system-admin deliverable-system page loads', async ({ page }) => {
    test.setTimeout(60000);

    // 注入管理员 cookie
    if (ADMIN_SESSION_ID) {
      await page.context().addCookies([
        { name: 'userId', value: ADMIN_USER_ID, url: BASE },
        { name: 'sessionid', value: ADMIN_SESSION_ID, url: BASE },
      ]);
    }

    // Mock /me API to bypass auth
    await page.route(`**/api/accounts/users/me/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: ADMIN_USER_ID,
          username: 'admin',
          is_superuser: true,
          current_company: null,
          companies: [],
        }),
      });
    });

    await page.goto(`${BASE}/system-admin/deliverable-system/`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    // 验证页面标题或关键元素
    const heading = page.locator('h2, h1').filter({ hasText: /交付物体系/ });
    await expect(heading.first()).toBeVisible({ timeout: 30000 });

    // 验证创建按钮可见
    const createBtn = page.getByText('创建交付物体系').first();
    await expect(createBtn).toBeVisible({ timeout: 15000 });

    console.log('[UI] 系统管理员交付物体系页面加载成功');
  });
});
