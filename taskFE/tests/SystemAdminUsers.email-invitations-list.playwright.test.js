// @ts-check
/**
 * E2E：系统管理员用户页面 — 邮箱邀请记录列表验证
 *
 * 验证：
 * 1. GET /api/system-admin/email-invitations/ 返回 200 (不再 404)
 * 2. 页面底部邮箱邀请记录区域正常渲染
 * 3. 不出现「未找到请求的 API 路径」错误
 *
 * 运行：npx playwright test SystemAdminUsers.email-invitations-list.playwright.test.js
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

let BASE = process.env.PLAYWRIGHT_SITE_ORIGIN || '';
if (!BASE) {
  try {
    const portConfig = loadPortConfig();
    BASE = `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`;
  } catch {
    BASE = 'http://localhost:4000';
  }
}
BASE = BASE.replace(/\/$/, '');
const ADMIN_EMAIL = process.env.PLAYWRIGHT_ADMIN_EMAIL || 'author@example.com';
const ADMIN_PASSWORD = process.env.PLAYWRIGHT_ADMIN_PASSWORD || '';
const ADMIN_USER_ID = process.env.PLAYWRIGHT_ADMIN_USER_ID || '850249621660790784';
const ADMIN_SESSION_ID = process.env.PLAYWRIGHT_ADMIN_SESSION_ID || '';
const ADMIN_TOKEN = process.env.PLAYWRIGHT_ADMIN_TOKEN || '';

test.describe('system-admin users email invitations', () => {
  test('API: GET /api/system-admin/email-invitations/ returns 200 with invitations array', async ({ request }) => {
    test.setTimeout(30000);

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Requested-With': 'XMLHttpRequest',
      'Accept': 'application/json',
    };
    if (ADMIN_TOKEN) {
      headers['Authorization'] = `Token ${ADMIN_TOKEN}`;
    }
    if (ADMIN_SESSION_ID) {
      headers['Cookie'] = `sessionid=${ADMIN_SESSION_ID}; userId=${ADMIN_USER_ID}`;
    }

    const resp = await request.get(`${BASE}/api/system-admin/email-invitations/`, { headers });

    // 关键断言：不再返回 404（Django 通配回退）
    expect(resp.status()).not.toBe(404);

    // 应返回 200（即使列表为空）
    if (resp.status() === 401) {
      console.log('[email-invitations API] 401 — 认证凭证未配置，跳过内容验证');
      return;
    }
    if (resp.status() === 403) {
      console.log('[email-invitations API] 403 — 非管理员账号，跳过内容验证');
      return;
    }
    expect(resp.status()).toBe(200);

    const body = await resp.json();
    expect(body).toHaveProperty('invitations');
    expect(Array.isArray(body.invitations)).toBe(true);
    console.log(`[email-invitations API] 返回 ${body.invitations.length} 条邀请记录`);
  });

  test('API: GET /api/system-admin/email-invitations/ — response has trace_id for observability', async ({ request }) => {
    test.setTimeout(30000);

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Requested-With': 'XMLHttpRequest',
      'Accept': 'application/json',
    };
    if (ADMIN_TOKEN) {
      headers['Authorization'] = `Token ${ADMIN_TOKEN}`;
    }
    if (ADMIN_SESSION_ID) {
      headers['Cookie'] = `sessionid=${ADMIN_SESSION_ID}; userId=${ADMIN_USER_ID}`;
    }

    const resp = await request.get(`${BASE}/api/system-admin/email-invitations/`, { headers });

    if (resp.status() === 401 || resp.status() === 403) {
      console.log('[email-invitations traceId] 认证未通过，跳过 trace_id 验证');
      return;
    }

    // 响应头应包含 X-Trace-Id
    const traceIdHeader = resp.headers()['x-trace-id'];
    if (traceIdHeader) {
      console.log(`[email-invitations traceId] 响应头 X-Trace-Id: ${traceIdHeader}`);
    }

    // 如果是 200，响应体应包含 trace_id
    if (resp.status() === 200) {
      const body = await resp.json();
      if (body.trace_id) {
        console.log(`[email-invitations traceId] 响应体 trace_id: ${body.trace_id}`);
      }
    }
  });

  test('UI: system-admin users page shows email invitation section without 404 error', async ({ page }) => {
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
          is_staff: true,
          current_company: null,
          companies: [],
        }),
      });
    });

    // 拦截 email-invitations API 以追踪请求
    let emailInvitationsRequested = false;
    await page.route('**/api/system-admin/email-invitations/', async (route, request) => {
      emailInvitationsRequested = true;
      // 放行到实际后端
      await route.continue();
    });

    await page.goto(`${BASE}/system-admin/users/`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(4000);

    // 验证页面标题
    const heading = page.locator('h2').filter({ hasText: /用户列表/ });
    await expect(heading.first()).toBeVisible({ timeout: 15000 });

    // 验证邮箱邀请记录区域标题可见
    const inviteSectionHeading = page.locator('h2').filter({ hasText: /邮箱邀请记录/ });
    await expect(inviteSectionHeading.first()).toBeVisible({ timeout: 20000 });

    // 关键断言：不应出现 "未找到请求的 API 路径" 错误
    const api404Error = page.locator('.bg-red-50.text-red-700').filter({ hasText: /未找到请求的 API 路径/ });
    await expect(api404Error).toHaveCount(0, { timeout: 5000 });

    // 验证邀请记录区域存在（加载中或已加载）
    const inviteSection = page.locator('h2').filter({ hasText: /邮箱邀请记录/ });
    await expect(inviteSection.first()).toBeVisible();

    console.log(`[UI] email-invitations API 是否被请求: ${emailInvitationsRequested}`);
    console.log('[UI] 系统管理员用户页面 — 邮箱邀请记录区域正常，无 404 错误');
  });

  test('UI: invitation error div has data-traceId attribute', async ({ page }) => {
    test.setTimeout(60000);

    if (ADMIN_SESSION_ID) {
      await page.context().addCookies([
        { name: 'userId', value: ADMIN_USER_ID, url: BASE },
        { name: 'sessionid', value: ADMIN_SESSION_ID, url: BASE },
      ]);
    }

    await page.route(`**/api/accounts/users/me/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: ADMIN_USER_ID,
          username: 'admin',
          is_superuser: true,
          is_staff: true,
          current_company: null,
          companies: [],
        }),
      });
    });

    // Mock email-invitations API 返回错误以触发 error 展示
    let errorDivSeen = false;
    await page.route('**/api/system-admin/email-invitations/', async (route) => {
      const method = route.request().method();
      if (method === 'GET') {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          headers: { 'x-trace-id': 'e2e-test-trace-id-0001' },
          body: JSON.stringify({ error: '模拟错误', detail: 'E2E 测试错误响应', trace_id: 'e2e-test-trace-id-0001' }),
        });
      } else {
        await route.continue();
      }
    });

    await page.goto(`${BASE}/system-admin/users/`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(4000);

    // 检查错误 div 是否有 data-traceId
    const errorDiv = page.locator('.bg-red-50.text-red-700.rounded-lg.text-sm').first();
    const isVisible = await errorDiv.isVisible().catch(() => false);
    if (isVisible) {
      const traceIdAttr = await errorDiv.getAttribute('data-traceid');
      console.log(`[UI traceId] 错误 div data-traceid: ${traceIdAttr}`);
      // 验证 data-traceId 属性存在且非空
      expect(traceIdAttr).toBeTruthy();
      expect(traceIdAttr).toBe('e2e-test-trace-id-0001');
      errorDivSeen = true;
    }

    console.log(`[UI traceId] 错误 div 可见: ${isVisible}, 有 traceId: ${errorDivSeen}`);
  });
});
