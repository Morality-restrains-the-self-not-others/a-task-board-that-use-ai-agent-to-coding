// @ts-check
/**
 * E2E: 租户控制台侧边栏工作面板保留 + 人员管理菜单按公司租户状态条件渲染
 *
 * 覆盖目标（需求：公司租户可见「人员管理」菜单；工作面板所有租户保留）——
 *  1. 公司租户（member_is_active=true，非管理员）→ 工作面板保留（位置 2）+ 人员管理菜单可见
 *  2. 公司管理员（member_is_admin=true）→ 同样工作面板保留 + 人员管理可见
 *  3. 非公司租户（member_is_active=false）→ 工作面板可见，人员管理隐藏
 *  4. 公司租户点击人员管理 → 子菜单展开（邀请人/管理人员/管理分组）
 *  5. /people/manage/ 路由下人员管理入口高亮且子菜单自动展开
 *  6. 真实登录态（仅 localStorage currentUserId，HttpOnly 签名 cookie 场景）→
 *     人员管理可见且 companies/current 被调用（2026-08-09 生产回归：JS 读不到
 *     HttpOnly userId cookie 时 getCookie('userId') 早退导致身份判定缺失）
 *
 * 运行：
 *   npx playwright test --config=playwright.config.cdp.js Sidebar.companyTenantMenu.playwright.test.js
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
const TENANT_ID = process.env.PW_TENANT_ID || '850256677331562496';

let cachedIndexHtml = '';

/**
 * 预置认证态并 mock 账号 API（复用 Navbar.companySwitcher 的等效替代模式）：
 * - 服务端对 /tenant/ 的 forward-auth 302 以 SPA index.html 直接替换，走完整 SPA 链路
 * - mock /me/ 与 companies/current：用 memberFlags 驱动侧边栏 isCompanyTenant 判定
 */
/** 统一 mock 全部 /api/：无 sessionid 时真实网关 401 会触发登录跳转，
 * 快速空数据响应让页面组件稳定渲染（仅侧边栏行为是本测试关注点） */
async function mockApis(page, memberFlags) {
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

  await page.route('**/api/**', async (route) => {
    const url = route.request().url();
    if (url.includes('/api/accounts/users/me/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: USER_ID,
          username: 'sidebar-tenant-user',
          is_superuser: false,
          current_company: { id: TENANT_ID, name: '验收公司' },
          companies: [{ id: TENANT_ID, name: '验收公司' }],
        }),
      });
      return;
    }
    if (url.includes('/accounts/companies/current/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ name: '验收公司', ...memberFlags }),
      });
      return;
    }
    // 其余业务 API：空对象快速响应
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

async function setupAuthenticatedPage(page, memberFlags) {
  await mockApis(page, memberFlags);

  // 预置认证态（userId cookie + 上次活跃租户）
  await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
  await page.evaluate(({ uid, tid }) => {
    document.cookie = `userId=${uid}; path=/`;
    localStorage.setItem('lastActiveTenantId', tid);
  }, { uid: USER_ID, tid: TENANT_ID });
}

/**
 * 真实登录态场景：activate-session 落的是 HttpOnly 签名 userId cookie（JS 不可读），
 * 前端仅靠 localStorage currentUserId 判定用户身份（OPT-20260807-004 语义）。
 * 不注入 JS 可写 userId cookie → 旧实现 getCookie('userId') 早退的回归在此被捕获
 * （2026-08-09 生产复现：公司租户判定永不执行，人员管理不显示）。
 */
async function setupAuthenticatedPageLSOnly(page, memberFlags) {
  await mockApis(page, memberFlags);

  await page.goto(`${BASE_URL}/`, { waitUntil: 'domcontentloaded' });
  await page.evaluate(({ uid, tid }) => {
    localStorage.setItem('currentUserId', uid);
    localStorage.setItem('lastActiveTenantId', tid);
  }, { uid: USER_ID, tid: TENANT_ID });
}

const sidebar = (page) => page.locator('aside.tenant-console-sidebar');
const navItems = (page) => sidebar(page).locator('nav > *');
const peopleEntry = (page) =>
  navItems(page).filter({ hasText: '人员管理' }).first();
const workPanelLink = (page) => sidebar(page).locator('a[href*="/work-panel"]');

test.describe('侧边栏工作面板保留 + 人员管理按公司租户状态条件渲染', () => {
  test('公司租户（member_is_active=true，非管理员）→ 工作面板保留（位置 2）+ 人员管理菜单可见', async ({ page }) => {
    await setupAuthenticatedPage(page, {
      member_is_active: true,
      member_is_admin: false,
      member_is_creator: false,
    });

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/`, { waitUntil: 'domcontentloaded' });

    // 工作面板链接保留（位置 2，所有租户可见）
    await expect(workPanelLink(page)).toBeVisible({ timeout: 15000 });
    await expect(workPanelLink(page)).toHaveAttribute('href', `/tenant/${TENANT_ID}/work-panel`);
    // 人员管理菜单独立可见
    await expect(peopleEntry(page)).toBeVisible();
    await expect(peopleEntry(page)).toContainText('人员管理');
    // nav 顺序：项目列表、工作面板、镜像市场、人员管理…
    const labels = await navItems(page).allInnerTexts();
    const compact = labels.map((t) => t.split('\n').filter(Boolean).join(' ')).filter(Boolean);
    expect(compact[1]).toContain('工作面板');
    expect(compact).toContain('人员管理');
  });

  test('公司管理员（member_is_admin=true）→ 同样工作面板保留 + 人员管理可见', async ({ page }) => {
    await setupAuthenticatedPage(page, {
      member_is_active: true,
      member_is_admin: true,
      member_is_creator: false,
    });

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/`, { waitUntil: 'domcontentloaded' });

    await expect(workPanelLink(page)).toBeVisible({ timeout: 15000 });
    await expect(peopleEntry(page)).toBeVisible({ timeout: 15000 });
  });

  test('真实登录态（仅 localStorage currentUserId，HttpOnly 签名 cookie 场景）→ 人员管理可见', async ({ page }) => {
    await setupAuthenticatedPageLSOnly(page, {
      member_is_active: true,
      member_is_admin: true,
      member_is_creator: false,
    });

    // companies/current 必须被调用（旧实现因 JS 读不到 HttpOnly cookie 早退，此请求缺失）
    const companiesReq = page
      .waitForRequest((req) => req.url().includes('/accounts/companies/current/'), { timeout: 15000 })
      .catch(() => null);

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/`, { waitUntil: 'domcontentloaded' });

    await expect(workPanelLink(page)).toBeVisible({ timeout: 15000 });
    await expect(peopleEntry(page)).toBeVisible({ timeout: 15000 });
    expect(await companiesReq).toBeTruthy();
  });

  test('非公司租户（member_is_active=false）→ 工作面板可见，人员管理隐藏', async ({ page }) => {
    await setupAuthenticatedPage(page, {
      member_is_active: false,
      member_is_admin: false,
      member_is_creator: false,
    });

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/`, { waitUntil: 'domcontentloaded' });

    await expect(workPanelLink(page)).toBeVisible({ timeout: 15000 });
    await expect(workPanelLink(page)).toHaveAttribute('href', `/tenant/${TENANT_ID}/work-panel`);
    expect(await peopleEntry(page).count()).toBe(0);
  });

  test('公司租户点击人员管理 → 子菜单展开（邀请人/管理人员/管理分组）', async ({ page }) => {
    await setupAuthenticatedPage(page, {
      member_is_active: true,
      member_is_admin: false,
      member_is_creator: false,
    });

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/`, { waitUntil: 'domcontentloaded' });

    await peopleEntry(page).click();
    await expect(sidebar(page).locator('a[href*="/people/invite/"]')).toBeVisible({ timeout: 5000 });
    await expect(sidebar(page).locator('a[href*="/people/manage/"]')).toBeVisible();
    await expect(sidebar(page).locator('a[href*="/people/groups/"]')).toBeVisible();
  });

  test('/people/manage/ 路由下人员管理入口高亮且子菜单自动展开', async ({ page }) => {
    await setupAuthenticatedPage(page, {
      member_is_active: true,
      member_is_admin: true,
      member_is_creator: false,
    });

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/people/manage/`, { waitUntil: 'domcontentloaded' });

    // 子菜单自动展开（updateMenuState 按 /people/ 路由展开）
    await expect(sidebar(page).locator('a[href*="/people/manage/"]')).toBeVisible({ timeout: 15000 });
    // 人员管理入口高亮（入口为内层 cursor-pointer div；外层为条件渲染包装 div）
    const entry = peopleEntry(page).locator('div.cursor-pointer');
    await expect(entry).toHaveClass(/bg-primary\/10 text-primary font-medium/);
  });
});
