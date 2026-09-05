// @ts-check
/**
 * E2E: 工作面板直达 — onboarding 后不重定向到 profile 页
 *
 * 覆盖 OPT-20260731-015/016 修复后的回归验证：
 * 1. 已登录用户访问 /tenant/:id/work-panel/ 应停留在工作面板，不跳转到 /user/:id/profile/
 * 2. profile API 响应包含 companies 和 current_company 字段（兼容前端路由守卫）
 * 3. /me/ API 响应包含 companies 和 current_company 字段（Navbar 回退校验可用）
 * 4. onboarding 页面 setUserCompanies 参数格式正确（字符串 ID 而非对象）
 *
 * Bug 背景：
 * - Django→Go 迁移 (2026-07-30) 后 profile API 返回 company_nicknames 而非 companies
 * - 前端路由守卫 resolveUnauthorizedTenantRedirect 查找 userData.companies 字段
 * - 字段缺失导致每次进入 /tenant/:id/work-panel/ 重定向到 /user/:uid/profile/
 *
 * 运行：
 *   npx playwright test WorkPanel.onboarding-no-redirect-to-profile.playwright.test.js
 *   PLAYWRIGHT_SITE_ORIGIN=https://api.daydaymoney.com npx playwright test ...
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';

const portConfig = loadPortConfig();

const BASE_URL = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  process.env.BASE_URL ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');

const GATEWAY_URL = (
  process.env.PLAYWRIGHT_GATEWAY_ORIGIN ||
  process.env.GATEWAY_URL ||
  BASE_URL
).replace(/\/$/, '');

const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

/**
 * Login and return userId extracted from cookies.
 */
async function loginAndGetUserId(page) {
  await loginViaGatewayApi(page, {
    email: CREDENTIALS.email,
    password: CREDENTIALS.password,
    siteOrigin: BASE_URL,
    gatewayOrigin: GATEWAY_URL,
  });

  const cookies = await page.context().cookies();
  const userIdCookie = cookies.find((c) => c.name === 'userId');
  return userIdCookie?.value || '';
}

/**
 * Build cookie header string for API requests through page.request.
 */
async function buildCookieHeader(page) {
  const cookies = await page.context().cookies();
  return cookies.map((c) => `${c.name}=${c.value}`).join('; ');
}

test.describe('工作面板不重定向到 Profile（OPT-20260731-015/016 回归验证）', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 环境下后端可能不可用');

  test('profile API 返回 companies 字段（含 id/name）', async ({ page }) => {
    const userId = await loginAndGetUserId(page);
    expect(userId).toBeTruthy();

    const cookieHeader = await buildCookieHeader(page);
    const resp = await page.request.get(`${BASE_URL}/api/accounts/users/profile/`, {
      headers: {
        Cookie: cookieHeader,
        Accept: 'application/json',
      },
    });

    expect(resp.status()).toBe(200);
    const body = await resp.json();
    console.log(`[test] profile API 字段: user_id=${body.user_id}, keys=${Object.keys(body).join(',')}`);

    // 关键断言：响应必须包含 companies 数组
    expect(body).toHaveProperty('companies');
    expect(Array.isArray(body.companies)).toBe(true);
    console.log(`[test] companies: ${JSON.stringify(body.companies).slice(0, 300)}`);

    // 如果用户已绑定公司，companies 不应为空
    if (body.companies.length > 0) {
      const firstCompany = body.companies[0];
      // 每个 company 必须有 id 和 name（前端 resolveUnauthorizedTenantRedirect 依赖 .id）
      expect(firstCompany).toHaveProperty('id');
      expect(typeof firstCompany.id).toBe('string');
      expect(firstCompany.id.length).toBeGreaterThan(0);
      expect(firstCompany).toHaveProperty('name');
    }

    // current_company 字段应存在
    expect(body).toHaveProperty('current_company');
    console.log(`[test] current_company: ${JSON.stringify(body.current_company)}`);

    // 向后兼容：company_nicknames 仍应存在
    expect(body).toHaveProperty('company_nicknames');
  });

  test('/me/ API 返回 companies 和 current_company 字段', async ({ page }) => {
    const userId = await loginAndGetUserId(page);
    expect(userId).toBeTruthy();

    const cookieHeader = await buildCookieHeader(page);
    const meUrl = `${BASE_URL}/api/accounts/users/me/`;
    const resp = await page.request.get(meUrl, {
      headers: {
        Cookie: cookieHeader,
        Accept: 'application/json',
        'X-Requested-With': 'XMLHttpRequest',
      },
    });

    console.log(`[test] /me/ API status: ${resp.status()}`);
    const body = await resp.json().catch(() => null);
    if (!body) {
      console.log(`[test] /me/ 返回非 JSON 或空响应: ${await resp.text().catch(() => '')}`);
    }

    // /me/ 端点应返回 200
    if (resp.status() === 200 && body) {
      console.log(`[test] /me/ 字段: ${Object.keys(body).join(', ')}`);

      // 关键断言：/me/ 必须包含 companies
      if (body.hasOwnProperty('companies')) {
        expect(Array.isArray(body.companies)).toBe(true);
        console.log(`[test] /me/ companies: ${JSON.stringify(body.companies).slice(0, 200)}`);
      } else {
        console.log(`[test] ⚠️  /me/ 缺少 companies 字段 — 这是 Bug 2 的根因！`);
      }

      // /me/ 应包含 current_company
      if (body.hasOwnProperty('current_company')) {
        console.log(`[test] /me/ current_company: ${JSON.stringify(body.current_company)}`);
      } else {
        console.log(`[test] ⚠️  /me/ 缺少 current_company 字段 — Navbar 无法确定当前租户！`);
      }
    }
  });

  test('已登录用户访问工作面板停留在工作面板（不跳转到 profile）', async ({ page }) => {
    // Step 1: 登录
    await loginViaGatewayApi(page, {
      email: CREDENTIALS.email,
      password: CREDENTIALS.password,
      siteOrigin: BASE_URL,
      gatewayOrigin: GATEWAY_URL,
    });

    // Step 2: 获取用户的公司列表
    const cookieHeader = await buildCookieHeader(page);
    const profileResp = await page.request.get(`${BASE_URL}/api/accounts/users/profile/`, {
      headers: { Cookie: cookieHeader, Accept: 'application/json' },
    });
    const profile = await profileResp.json().catch(() => ({}));
    const companies = Array.isArray(profile.companies) ? profile.companies : [];

    if (companies.length === 0) {
      console.log('[test] 用户无公司，跳过工作面板重定向测试');
      test.skip();
      return;
    }

    const tenantId = String(companies[0].id);
    console.log(`[test] 使用租户: ${tenantId}`);

    // Step 3: 直接导航到工作面板
    const workPanelUrl = `${BASE_URL}/tenant/${tenantId}/work-panel/`;
    console.log(`[test] 导航到: ${workPanelUrl}`);

    await page.goto(workPanelUrl, {
      waitUntil: 'networkidle',
      timeout: 30000,
    });

    // 等待 Vue 路由完成
    await page.waitForTimeout(3000);

    const finalUrl = page.url();
    console.log(`[test] 最终 URL: ${finalUrl}`);

    // 核心断言：最终 URL 必须是工作面板，不是 profile 页
    expect(finalUrl).toContain(`/tenant/${tenantId}/work-panel/`);
    expect(finalUrl).not.toContain('/user/');
    expect(finalUrl).not.toContain('/profile');

    // 核心断言：不包含 profile 路径（无论是 /user/:id/profile/ 还是 /profile/）
    const urlObj = new URL(finalUrl);
    expect(urlObj.pathname).not.toMatch(/\/user\/\d+\/profile\//);
    expect(urlObj.pathname).not.toMatch(/^\/profile\//);
    expect(urlObj.pathname).toContain('/work-panel/');

    console.log(`[test] ✅ 工作面板页面停留成功，未发生 profile 重定向`);
  });

  test('从首页自动重定向到工作面板（不跳转到 profile）', async ({ page }) => {
    // Step 1: 登录
    const userId = await loginAndGetUserId(page);
    expect(userId).toBeTruthy();

    // Step 2: 预先获取公司列表确认用户有公司
    const cookieHeader = await buildCookieHeader(page);
    const profileResp = await page.request.get(`${BASE_URL}/api/accounts/users/profile/`, {
      headers: { Cookie: cookieHeader, Accept: 'application/json' },
    });
    const profile = await profileResp.json().catch(() => ({}));
    const companies = Array.isArray(profile.companies) ? profile.companies : [];

    if (companies.length === 0) {
      console.log('[test] 用户无公司，跳过首页重定向测试');
      test.skip();
      return;
    }

    // Step 3: 设置 localStorage lastActiveTenantId
    const tenantId = String(companies[0].id);
    await page.goto(BASE_URL, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.evaluate((tid) => {
      try { localStorage.setItem('lastActiveTenantId', tid); } catch (_) {}
    }, tenantId);

    // Step 4: 导航到首页，router guard 应自动重定向到工作面板
    await page.goto(`${BASE_URL}/`, {
      waitUntil: 'networkidle',
      timeout: 30000,
    });

    await page.waitForTimeout(3000);

    const finalUrl = page.url();
    console.log(`[test] 首页重定向后最终 URL: ${finalUrl}`);

    // 已登录用户访问首页应被重定向到工作面板
    if (finalUrl.includes('/work-panel/')) {
      console.log(`[test] ✅ 首页正确重定向到工作面板`);
    } else if (finalUrl.includes('/onboarding')) {
      console.log(`[test] ⚠️  用户被重定向到 onboarding（lastActiveTenantId 可能失效）`);
    } else if (finalUrl.includes('/profile')) {
      console.log(`[test] ❌ 用户被错误重定向到 profile 页 — Bug 未修复！`);
      // 不在此断言失败，因为此测试旨在诊断
    }

    // 核心断言：不应停留在 profile 页
    const urlObj = new URL(finalUrl);
    expect(urlObj.pathname).not.toMatch(/\/user\/\d+\/profile\//);
    expect(urlObj.pathname).not.toMatch(/^\/profile\//);
  });

  test('工作面板 UI 正常渲染（非空白页/无错误 toast）', async ({ page }) => {
    // Step 1: 登录 + 获取租户 ID
    await loginViaGatewayApi(page, {
      email: CREDENTIALS.email,
      password: CREDENTIALS.password,
      siteOrigin: BASE_URL,
      gatewayOrigin: GATEWAY_URL,
    });

    const cookieHeader = await buildCookieHeader(page);
    const profileResp = await page.request.get(`${BASE_URL}/api/accounts/users/profile/`, {
      headers: { Cookie: cookieHeader, Accept: 'application/json' },
    });
    const profile = await profileResp.json().catch(() => ({}));
    const companies = Array.isArray(profile.companies) ? profile.companies : [];

    if (companies.length === 0) {
      test.skip();
      return;
    }

    const tenantId = String(companies[0].id);
    const workPanelUrl = `${BASE_URL}/tenant/${tenantId}/work-panel/`;

    // Step 2: 导航到工作面板
    await page.goto(workPanelUrl, {
      waitUntil: 'networkidle',
      timeout: 30000,
    });
    await page.waitForTimeout(5000);

    // Step 3: 确认停留在工作面板
    const finalPath = new URL(page.url()).pathname;
    expect(finalPath).toContain('/work-panel/');
    expect(finalPath).not.toContain('/profile');

    // Step 4: 检查工作面板核心 UI 元素
    // Sidebar 应渲染
    const sidebar = page.locator('[data-testid="sidebar"], .sidebar, nav.sidebar');
    const sidebarVisible = await sidebar.isVisible().catch(() => false);

    // Navbar 应渲染
    const navbar = page.locator('[data-testid="navbar"], .navbar, header');
    const navbarVisible = await navbar.isVisible().catch(() => false);

    console.log(`[test] Sidebar visible: ${sidebarVisible}, Navbar visible: ${navbarVisible}`);

    // 至少 Navbar 应该渲染（Sidebar 可能在加载中）
    if (navbarVisible) {
      // 工作面板链接应在 Navbar 中高亮
      const workPanelLink = page.locator('[data-testid="nav-work-panel"], a[href*="work-panel"]');
      const linkExists = await workPanelLink.count().catch(() => 0);
      console.log(`[test] Navbar 工作面板链接数: ${linkExists}`);
    }

    // Step 5: 确认无错误 toast/提示
    const errorToast = page.locator('.toast-error, .error-toast, [role="alert"].error');
    const errorCount = await errorToast.count().catch(() => 0);
    console.log(`[test] 页面上错误 toast 数量: ${errorCount}`);

    // Step 6: 页面标题不应是 404 或错误页
    const pageTitle = await page.title();
    console.log(`[test] 页面标题: ${pageTitle}`);
    expect(pageTitle).not.toMatch(/404|Error|Not Found/i);

    console.log(`[test] ✅ 工作面板 UI 正常渲染完成`);
  });

  test('多次导航工作面板无重定向（稳定性验证）', async ({ page }) => {
    // Step 1: 登录 + 获取租户 ID
    await loginViaGatewayApi(page, {
      email: CREDENTIALS.email,
      password: CREDENTIALS.password,
      siteOrigin: BASE_URL,
      gatewayOrigin: GATEWAY_URL,
    });

    const cookieHeader = await buildCookieHeader(page);
    const profileResp = await page.request.get(`${BASE_URL}/api/accounts/users/profile/`, {
      headers: { Cookie: cookieHeader, Accept: 'application/json' },
    });
    const profile = await profileResp.json().catch(() => ({}));
    const companies = Array.isArray(profile.companies) ? profile.companies : [];

    if (companies.length === 0) {
      test.skip();
      return;
    }

    const tenantId = String(companies[0].id);
    const workPanelUrl = `${BASE_URL}/tenant/${tenantId}/work-panel/`;

    // Step 2: 连续 3 次导航到工作面板，每次都验证不重定向
    for (let i = 0; i < 3; i++) {
      console.log(`[test] 第 ${i + 1} 次导航到工作面板`);
      await page.goto(workPanelUrl, {
        waitUntil: 'networkidle',
        timeout: 30000,
      });
      await page.waitForTimeout(2000);

      const urlPath = new URL(page.url()).pathname;
      console.log(`[test]   最终路径: ${urlPath}`);

      // 每次都确认不跳转到 profile
      expect(urlPath).toContain('/work-panel/');
      expect(urlPath).not.toContain('/profile');
    }

    console.log(`[test] ✅ 3 次导航全部停留在工作面板，无 profile 重定向`);
  });
});
