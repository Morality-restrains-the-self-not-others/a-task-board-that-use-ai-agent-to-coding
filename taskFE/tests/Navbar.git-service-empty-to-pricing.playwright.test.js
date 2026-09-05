// @ts-check
/**
 * E2E: 「代码仓库」导航空列表/失败状态机回归（OPT-20260829-001）
 *
 * 组件测已覆盖 NavbarGitServiceNav 的 ready→/pricing/ 与 fail-open（error→当前页），
 * 但公网 SPA 上带 accessCode 的真实导航链路（登录态 → 真实 <a href> 点击 →
 * 推荐码拦截器透传 accessCode）尚无 Playwright 断言。本用例补上：
 *   1. 已登录 + gitlab-resources 返回空数组（200 ready）→ 点击 nav-git-service
 *      → URL 变为 /pricing/?accessCode=<code>（保留 accessCode）
 *   2. 已登录 + gitlab-resources 返回 500（error）→ 点击 nav-git-service
 *      → 仍在当前 path（fail-open，不跳价格页）
 *
 * 纯 mock，无真实账号依赖：mock /me/ + membership + gitlab-resources +
 * referral-codes/status，预置 localStorage/cookie userId。
 *
 * 运行：
 *   npx playwright test --config=playwright.verify.config.js tests/Navbar.git-service-empty-to-pricing.playwright.test.js
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
const TENANT_ID = process.env.PW_TENANT_ID || '874599492341493760';
const ACCESS = 'u-opt-20260829-001';
const NICKNAME = '代码仓库回归用户';

/**
 * 预置已登录会话并 mock 账号/计费 API。
 *
 * @param {import('@playwright/test').Page} page
 * @param {'empty'|'error'} gitlabMode gitlab-resources 返回空数组(ready) 或 500(error)
 */
async function setupLoggedInNavbar(page, gitlabMode) {
  // 统一 mock 全部 /api/：登录/计费/推荐码按需返回，其余空对象快速响应，
  // 避免真实网关 401 触发全局跳登录。
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
          email: 'e2e-opt-001@example.com',
          companies: [{ id: TENANT_ID, name: '回归测试公司' }],
          current_company: { id: TENANT_ID, name: '回归测试公司' },
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
          email: 'e2e-opt-001@example.com',
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
    if (url.includes('/billing/gitlab-resources/')) {
      if (gitlabMode === 'empty') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ resources: [] }) });
      } else {
        await route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: 'internal' }) });
      }
      return;
    }
    if (url.includes('/api/accounts/users/referral-codes/status/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ access_code: ACCESS }) });
      return;
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  // 预置认证态（localStorage.currentUserId + userId cookie）与推荐码缓存。
  await page.addInitScript(
    ({ uid, tid, access }) => {
      try {
        localStorage.setItem('currentUserId', uid);
        localStorage.setItem('lastActiveTenantId', tid);
        localStorage.setItem('referralCode', access);
        sessionStorage.setItem('referral_access_code', access);
      } catch (_) {}
      document.cookie = `userId=${uid}; path=/`;
    },
    { uid: USER_ID, tid: TENANT_ID, access: ACCESS },
  );
}

test.describe('「代码仓库」导航 — gitlab-resources 空列表/失败', () => {
  test('ready 空列表：点击跳 /pricing/ 并保留 accessCode', async ({ page }) => {
    await setupLoggedInNavbar(page, 'empty');

    await page.goto(`/?accessCode=${ACCESS}`, { waitUntil: 'domcontentloaded' });

    const nav = page.locator('nav[data-alias="cmp-navbar-main"]');
    await nav.waitFor({ timeout: 15000 });
    await expect(nav).toContainText(NICKNAME, { timeout: 10000 });

    // 列表 ready 且为空 → href 指向带 accessCode 的价格页
    const gitNav = page.getByTestId('nav-git-service');
    await gitNav.waitFor({ timeout: 15000 });
    await expect(gitNav).toHaveAttribute('href', `/pricing/?accessCode=${ACCESS}`, { timeout: 10000 });

    await gitNav.click();

    await expect(page).toHaveURL(new RegExp(`/pricing/\\?accessCode=${ACCESS}`), { timeout: 15000 });
    await expect(page.locator('[data-alias="view-pricing-page"]')).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('heading', { name: '资源收费标准' })).toBeVisible({ timeout: 10000 });
  });

  test('error：点击留在当前 path（fail-open）', async ({ page }) => {
    await setupLoggedInNavbar(page, 'error');

    await page.goto(`/?accessCode=${ACCESS}`, { waitUntil: 'domcontentloaded' });

    const nav = page.locator('nav[data-alias="cmp-navbar-main"]');
    await nav.waitFor({ timeout: 15000 });

    // 拉取失败(status=error) → href 保持当前页，不跳价格页
    const gitNav = page.getByTestId('nav-git-service');
    await gitNav.waitFor({ timeout: 15000 });
    await expect(gitNav).toHaveAttribute('href', `/?accessCode=${ACCESS}`, { timeout: 10000 });

    const before = new URL(page.url());
    await gitNav.click();

    // 同 URL 刷新或原地不动：path 不变、仍在首页、accessCode 保留
    try {
      await page.waitForLoadState('domcontentloaded', { timeout: 5000 });
    } catch (_) {
      /* 原地不动时没有新导航 */
    }
    const after = new URL(page.url());
    expect(after.pathname).toBe(before.pathname);
    expect(after.pathname).toBe('/');
    expect(after.searchParams.get('accessCode')).toBe(ACCESS);
  });
});
