// @ts-check
/**
 * E2E: 「工作面板」导航项的公司切换下拉行为
 *
 * 覆盖目标：将工作面板链接改造为公司切换下拉后——
 *  1. 多公司用户 → 工作面板位置渲染为下拉（button[data-testid="nav-work-panel"]），
 *     触发按钮显示当前公司名；点击展开菜单列出全部公司；
 *     选择任一公司（含当前公司）→ 真实 a[href] 全页导航到目标公司工作面板。
 *  2. 多公司时不再渲染普通 a 链接与旧独立 <select> 公司切换器。
 *  3. 单公司用户 → 仍渲染普通工作面板链接（回归，不渲染下拉）。
 *
 * 运行：
 *   npx playwright test --config=playwright.config.cdp.js Navbar.companySwitcher.playwright.test.js
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
const COMPANY_A = process.env.PW_TENANT_ID || '850256677331562496';
const COMPANY_B = process.env.PW_TENANT_ID_B || '660256677331560000';

let cachedIndexHtml = '';

/**
 * 预置认证态并 mock 账号 API：
 * - 服务端对 /tenant/ 的 forward-auth 302（无 sessionid）以 SPA index.html 直接替换，
 *   由客户端 route guard + mock /me/ 走完整 SPA 链路（测试账号凭据不可用时的等效替代）
 * - mock /me/ 与 profile：用多公司载荷驱动导航栏下拉
 */
async function setupAuthenticatedPage(page, companies, currentCompanyId) {
  if (!cachedIndexHtml) {
    const resp = await page.request.get(`${BASE_URL}/`);
    cachedIndexHtml = await resp.text();
  }

  // 绕过服务端 302：/tenant/ 文档导航直接返回 SPA 入口
  await page.route('**/tenant/**', async (route) => {
    if (route.request().resourceType() === 'document') {
      await route.fulfill({ status: 200, contentType: 'text/html', body: cachedIndexHtml });
    } else {
      await route.continue();
    }
  });

  // 统一 mock 全部 /api/：无 sessionid 时真实网关会 401（forward-auth 会话失效文案）
  // 触发全局跳转登录（requestErrorDisplay.handleForwardAuthSessionExpired）或偶发挂起，
  // 快速空数据响应让页面组件稳定渲染（仅导航栏行为是本测试关注点）
  await page.route('**/api/**', async (route) => {
    const url = route.request().url();
    if (url.includes('/api/accounts/users/me/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: USER_ID,
          username: 'company-switcher-user',
          is_superuser: false,
          current_company: companies.find((c) => c.id === currentCompanyId) || null,
          companies,
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
          personal_nickname: 'Company Switcher',
          email: 'cs@example.com',
        }),
      });
      return;
    }
    if (url.includes('/accounts/companies/current/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: currentCompanyId, name: '公司A', member_is_admin: true }),
      });
      return;
    }
    if (url.includes('/api/projects/workspaces')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ workspaces: [] }) });
      return;
    }
    if (url.includes('/billing/membership/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ membership: { tier: 'normal' } }) });
      return;
    }
    if (url.includes('/user-roles/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ roles: [] }) });
      return;
    }
    // 其余业务 API：空对象快速响应
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  // 预置认证态（userId cookie + 上次活跃租户）
  await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
  await page.evaluate(({ uid, tid }) => {
    document.cookie = `userId=${uid}; path=/`;
    localStorage.setItem('lastActiveTenantId', tid);
  }, { uid: USER_ID, tid: currentCompanyId });
}

test.describe('Navbar 公司切换下拉（工作面板链接改造）', () => {
  test('多公司：渲染下拉触发按钮显示当前公司名，菜单列出全部公司，选择后导航到目标公司', async ({ page }) => {
    await setupAuthenticatedPage(
      page,
      [
        { id: COMPANY_A, name: '公司A' },
        { id: COMPANY_B, name: '公司B' },
      ],
      COMPANY_A,
    );

    await page.goto(`${BASE_URL}/tenant/${COMPANY_A}/work-panel/`, { waitUntil: 'domcontentloaded' });

    const btn = page.locator('button[data-testid="nav-work-panel"]');
    await btn.waitFor({ timeout: 15000 });
    await expect(btn).toContainText('公司A');
    await expect(btn).toHaveAttribute('aria-haspopup', 'menu');

    // 多公司时不应再有普通 <a> 工作面板链接与旧独立 <select>
    expect(await page.locator('a[data-testid="nav-work-panel"]').count()).toBe(0);
    expect(await page.locator('nav select').count()).toBe(0);

    // 展开菜单
    await btn.click();
    const menu = page.locator('[data-testid="nav-company-menu"]');
    await expect(menu).toBeVisible({ timeout: 5000 });
    await expect(menu).toContainText('公司A');
    await expect(menu).toContainText('公司B');

    // 当前公司高亮且为真实链接（任务详情等页点击当前公司须能进工作面板）
    const currentItem = menu.locator('[role="menuitem"][data-current="true"]');
    await expect(currentItem).toContainText('公司A');
    await expect(currentItem).toHaveAttribute('href', `/tenant/${COMPANY_A}/work-panel/`);

    // 选择公司 B → 全页导航到目标公司工作面板（不保留旧路径/workspace_id）
    await Promise.all([
      page.waitForURL(
        (url) => url.pathname === `/tenant/${COMPANY_B}/work-panel/` || url.pathname === `/tenant/${COMPANY_B}/work-panel`,
        { timeout: 15000 },
      ),
      menu.locator('[role="menuitem"]').nth(1).click(),
    ]);
  });

  test('多公司：从非工作面板页切换公司 → 仍落到目标公司工作面板', async ({ page }) => {
    await setupAuthenticatedPage(
      page,
      [
        { id: COMPANY_A, name: '公司A' },
        { id: COMPANY_B, name: '公司B' },
      ],
      COMPANY_A,
    );

    await page.goto(`${BASE_URL}/tenant/${COMPANY_A}/people/manage/`, { waitUntil: 'domcontentloaded' });

    const btn = page.locator('button[data-testid="nav-work-panel"]');
    await btn.waitFor({ timeout: 15000 });
    await btn.click();
    const menu = page.locator('[data-testid="nav-company-menu"]');
    await expect(menu).toBeVisible({ timeout: 5000 });

    await Promise.all([
      page.waitForURL(
        (url) => url.pathname.startsWith(`/tenant/${COMPANY_B}/work-panel`),
        { timeout: 15000 },
      ),
      menu.locator('[role="menuitem"]').nth(1).click(),
    ]);
  });

  test('多公司：任务详情页点击当前公司 → 进入该公司工作面板', async ({ page }) => {
    await setupAuthenticatedPage(
      page,
      [
        { id: COMPANY_A, name: '我的公司' },
        { id: COMPANY_B, name: '公司B' },
      ],
      COMPANY_A,
    );

    await page.goto(
      `${BASE_URL}/tenant/${COMPANY_A}/workspace/ws_-2309487803472456748/task-detail/task_878932440129761280/?accessCode=DR2AKvP9J9`,
      { waitUntil: 'domcontentloaded' },
    );

    const btn = page.locator('button[data-testid="nav-work-panel"]');
    await btn.waitFor({ timeout: 15000 });
    await btn.click();
    const currentItem = page.locator('[data-testid="nav-company-menu"] [role="menuitem"][data-current="true"]');
    await expect(currentItem).toBeVisible({ timeout: 5000 });
    await expect(currentItem).toContainText('我的公司');
    await expect(currentItem).toHaveAttribute(
      'href',
      `/tenant/${COMPANY_A}/work-panel/?accessCode=DR2AKvP9J9`,
    );

    await Promise.all([
      page.waitForURL(
        (url) => url.pathname.startsWith(`/tenant/${COMPANY_A}/work-panel`),
        { timeout: 15000 },
      ),
      currentItem.click(),
    ]);
  });

  test('多公司：点击遮罩关闭菜单', async ({ page }) => {
    await setupAuthenticatedPage(
      page,
      [
        { id: COMPANY_A, name: '公司A' },
        { id: COMPANY_B, name: '公司B' },
      ],
      COMPANY_A,
    );

    await page.goto(`${BASE_URL}/tenant/${COMPANY_A}/work-panel/`, { waitUntil: 'domcontentloaded' });

    const btn = page.locator('button[data-testid="nav-work-panel"]');
    await btn.waitFor({ timeout: 15000 });
    await btn.click();
    await expect(page.locator('[data-testid="nav-company-menu"]')).toBeVisible({ timeout: 5000 });

    // 点击遮罩（fixed inset-0）关闭
    await page.locator('[data-testid="nav-company-menu-overlay"]').click({ position: { x: 5, y: 5 } });
    await expect(page.locator('[data-testid="nav-company-menu"]')).toHaveCount(0);
  });

  test('单公司：仍渲染普通工作面板链接，不渲染下拉（回归）', async ({ page }) => {
    await setupAuthenticatedPage(page, [{ id: COMPANY_A, name: '公司A' }], COMPANY_A);

    await page.goto(`${BASE_URL}/tenant/${COMPANY_A}/work-panel/`, { waitUntil: 'domcontentloaded' });

    const link = page.locator('a[data-testid="nav-work-panel"]');
    await link.waitFor({ timeout: 15000 });
    await expect(link).toHaveAttribute('href', `/tenant/${COMPANY_A}/work-panel/`);
    expect(await page.locator('button[data-testid="nav-work-panel"]').count()).toBe(0);
    expect(await page.locator('nav select').count()).toBe(0);
  });
});
