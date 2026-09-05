// @ts-check
/**
 * E2E: 已登录用户访问 /auth/login/ 时，导航栏仍展示昵称/退出，不含「登录」主按钮。
 *
 * 回归目标：Navbar.logic 曾因 AUTH_ROUTE_NAMES 在 auth 路由强制 setLoggedOutUser，
 * 导致已登录用户打开登录页看到「登录/注册」而非自己的昵称（假登录态）。
 * 修复后 auth 路由不再提前 setLoggedOutUser，正常走 /me/ 拉会话。
 *
 * 纯 mock，无真实账号依赖：mock /me/ 返回已登录载荷，预置 localStorage/cookie userId。
 *
 * 运行：
 *   npx playwright test --config=playwright.config.headless.js Navbar.authLogin.logged-in.playwright.test.js
 *   # 或 CDP 9222（01_core_testing_rules.md）：--config=playwright.config.cdp.js
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();

const BASE_URL = (
  process.env.PW_BASE_URL ||
  process.env.BASE_URL ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');

const USER_ID = process.env.PW_USER_ID || '870705390352887808';
const NEXT_PATH = `/tenant/${process.env.PW_TENANT_ID || '874599492341493760'}/work-panel/`;
const NICKNAME = '已登录昵称';

let cachedIndexHtml = '';

/**
 * 预置已登录会话并 mock 账号 API：
 * - localStorage.currentUserId + userId cookie → getStoredUserId 返回 USER_ID
 * - mock /me/ 返回已登录载荷（昵称 + 公司），Navbar 走完整 SPA 链路
 * - 其余 /api/ 快速空响应，避免真实网关 401 触发全局跳登录
 */
async function setupLoggedInNavbar(page) {
  if (!cachedIndexHtml) {
    const resp = await page.request.get(`${BASE_URL}/`);
    cachedIndexHtml = await resp.text();
  }

  // 文档导航直接回 SPA 入口（/auth/login/ 与 /tenant/ 均无真实网关前依赖）
  await page.route('**/auth/login/**', async (route) => {
    if (route.request().resourceType() === 'document') {
      await route.fulfill({ status: 200, contentType: 'text/html', body: cachedIndexHtml });
    } else {
      await route.continue();
    }
  });

  // 统一 mock 全部 /api/
  await page.route('**/api/**', async (route) => {
    const url = route.request().url();
    if (url.includes('/api/accounts/users/me/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: USER_ID,
          username: NICKNAME,
          is_superuser: false,
          is_active: true,
          email: 'logged-in@example.com',
          companies: [{ id: '874599492341493760', name: '已登录公司' }],
          current_company: { id: '874599492341493760', name: '已登录公司' },
          login_methods: [],
          platform_roles: [],
        }),
      });
      return;
    }
    if (url.includes('/api/accounts/users/profile/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user_id: USER_ID,
          personal_nickname: NICKNAME,
          email: 'logged-in@example.com',
        }),
      });
      return;
    }
    if (url.includes('/api/auth/user-roles/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ roles: [] }) });
      return;
    }
    if (url.includes('/billing/membership/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ membership: { tier: 'normal' } }) });
      return;
    }
    // 其余业务 API：空对象快速响应
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  // 预置认证态（localStorage.currentUserId + userId cookie）
  await page.addInitScript(({ uid }) => {
    try {
      localStorage.setItem('currentUserId', uid);
      localStorage.setItem('lastActiveTenantId', '874599492341493760');
    } catch (_) {}
    document.cookie = `userId=${uid}; path=/`;
  }, { uid: USER_ID });
}

test.describe('Navbar — 已登录访问登录页', () => {
  test('auth_login + 有效会话：导航栏展示昵称与退出，不含「登录」主按钮', async ({ page }) => {
    await setupLoggedInNavbar(page);

    await page.goto(`${BASE_URL}/auth/login/?next=${encodeURIComponent(NEXT_PATH)}`, {
      waitUntil: 'domcontentloaded',
    });

    const nav = page.locator('nav[data-alias="cmp-navbar-main"]');
    await nav.waitFor({ timeout: 15000 });

    // 已登录导航：展示昵称
    await expect(nav).toContainText(NICKNAME, { timeout: 10000 });

    // 已登录导航：含「退出」快捷入口
    await expect(nav.getByRole('button', { name: '退出' })).toBeVisible();

    // 已登录导航：不得渲染「登录」主按钮（<a href="/auth/login/">）。
    // exact 匹配：name 默认子串匹配会误命中「已登录昵称」等链接（含「登录」子串）
    expect(await nav.getByRole('link', { name: '登录', exact: true }).count()).toBe(0);
    // 注册主按钮同样不出现
    expect(await nav.getByRole('link', { name: '注册', exact: true }).count()).toBe(0);
  });
});
