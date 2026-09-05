// @ts-check
/**
 * 跨子域会话 cookie 回归（OPT-20260808：裸域 cookie 任意子域可用）
 *
 * 契约：登录后 userId/token 以父域 cookie（Domain=.daydaymoney.com）下发，
 * 任意子域（www / 裸域 / api.*）请求自动携带 → 会话处处可用。
 * 已知缺陷（本文件用例 2/3 捕获，修复部署后转绿）：
 *  - 071 前旧登录遗留 host-only 影子 cookie 未在登录时清除 → 影子旧 token 在
 *    www 优先于域 cookie 被发送，失效后 www 永久 401；
 *  - logout / end-session / profile-401 清除不带 Domain → 域 cookie 残留，
 *    登出后他子域仍携带（失效）会话 cookie。
 *
 * 运行目标：线上 https://www.daydaymoney.com / https://daydaymoney.com。
 * 测试账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 */
import { test, expect } from '@playwright/test';

const WWW = 'https://www.daydaymoney.com';
const BARE = 'https://daydaymoney.com';
const EMAIL = 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');
const PROFILE_PATH = '/api/accounts/users/profile/';

async function login(page, origin) {
  await page.goto(`${origin}/auth/login/`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);
  const emailTab = page.locator('button:has-text("邮箱/密码")').first();
  if (await emailTab.isVisible().catch(() => false)) {
    await emailTab.click();
    await page.waitForTimeout(400);
  }
  await page.locator('#email').fill(EMAIL);
  await page.locator('#password').fill(PASSWORD);
  const agreeBoxes = await page.locator('input[type=checkbox]').all();
  for (const box of agreeBoxes) {
    const checked = await box.isChecked().catch(() => false);
    if (!checked) await box.check({ force: true }).catch(() => {});
  }
  await page.getByRole('button', { name: '登录', exact: true }).first().click();
  await page.waitForLoadState('networkidle').catch(() => {});
  await page.waitForTimeout(3000);
}

/** 导航后最近一次 profile 请求的状态码（Navbar 加载时必然触发；监听须在 goto 前挂载）。 */
async function lastProfileStatusAfterGoto(page, url) {
  const statuses = [];
  const listener = (res) => {
    if (res.url().includes(PROFILE_PATH)) statuses.push(res.status());
  };
  page.on('response', listener);
  await page.goto(url, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500); // 等 Navbar 鉴权请求发出并返回
  page.off('response', listener);
  return statuses.at(-1);
}

async function authCookies(context) {
  const all = await context.cookies();
  return all.filter((c) => c.name === 'userId' || c.name === 'token');
}

/** Playwright 返回的 domain 可能带前导点（.daydaymoney.com），归一去点比较。 */
function bareDomain(domain) {
  return String(domain || '').replace(/^\./, '');
}

test.describe('跨子域会话 cookie（裸域 ↔ 任意子域）', () => {
  test('www 登录 → 裸域可用（域 cookie 生效，无跳登录）', async ({ page, context }) => {
    test.setTimeout(120000);
    await login(page, WWW);

    // 域断言：userId/token 必须是父域 cookie（Playwright 返回去点形态 daydaymoney.com）
    const cookies = await authCookies(context);
    for (const name of ['userId', 'token']) {
      const c = cookies.find((x) => x.name === name);
      expect(c, `缺少 ${name} cookie`).toBeTruthy();
      expect(bareDomain(c.domain)).toBe('daydaymoney.com');
    }

    // 裸域可用性：打开裸域首页，Navbar 鉴权必须 200（域 cookie 随请求发送），
    // 且不得被会话失效收口跳转 /auth/login/
    expect(await lastProfileStatusAfterGoto(page, BARE + '/')).toBe(200);
    expect(page.url()).not.toContain('/auth/login/');
  });

  test('裸域登录 → www 子域可用（登录落在裸域，cookie 全子域生效）', async ({ page, context }) => {
    test.setTimeout(120000);
    await login(page, BARE);

    const cookies = await authCookies(context);
    for (const name of ['userId', 'token']) {
      const c = cookies.find((x) => x.name === name);
      expect(c, `缺少 ${name} cookie`).toBeTruthy();
      expect(bareDomain(c.domain)).toBe('daydaymoney.com');
    }

    expect(await lastProfileStatusAfterGoto(page, WWW + '/')).toBe(200);
    expect(page.url()).not.toContain('/auth/login/');
  });

  test('旧 host-only 影子 cookie 登录后被服务端清除（域 cookie 在 www 生效）', async ({ page, context }) => {
    test.setTimeout(120000);
    // 模拟 071 修复前的旧登录形态：host-only www cookie（旧 token 已失效）。
    // HttpOnly 只能服务端清 → activate-session 登录响应必须清除影子。
    await context.addCookies([
      { name: 'userId', value: 'stale.0.deadbeef', domain: 'www.daydaymoney.com', path: '/', httpOnly: true, secure: true, sameSite: 'Lax' },
      { name: 'token', value: 'stale-invalid-token', domain: 'www.daydaymoney.com', path: '/', httpOnly: true, secure: true, sameSite: 'Lax' },
    ]);

    await login(page, WWW);

    // 影子必须已清除：www host-only 变体不存在，父域变体存在
    const cookies = await authCookies(context);
    const hostOnly = cookies.filter((c) => c.domain === 'www.daydaymoney.com');
    expect(hostOnly, '旧 host-only 影子 cookie 未在登录时清除（修复前 www 被旧 token 劫持）').toEqual([]);
    for (const name of ['userId', 'token']) {
      expect(bareDomain(cookies.find((x) => x.name === name)?.domain)).toBe('daydaymoney.com');
    }

    // www 与裸域均可认证（影子不除时 www 用旧 token → 401 → 跳登录）
    expect(await lastProfileStatusAfterGoto(page, WWW + '/')).toBe(200);
    expect(page.url()).not.toContain('/auth/login/');
    expect(await lastProfileStatusAfterGoto(page, BARE + '/')).toBe(200);
    expect(page.url()).not.toContain('/auth/login/');
  });

  test('logout 清除域 cookie，裸域同步登出（一次登出全子域生效）', async ({ page, context }) => {
    test.setTimeout(120000);
    await login(page, WWW);
    expect((await authCookies(context)).length).toBe(2);

    // 经真实登出接口（与 Navbar.logic.vue 同构：Authorization + credentials）
    const token = (await context.cookies()).find((c) => c.name === 'token')?.value;
    const logoutStatus = await page.evaluate(async (tok) => {
      const r = await fetch('/api/accounts/users/logout/', {
        method: 'POST',
        credentials: 'include',
        headers: tok ? { Authorization: `Token ${tok}` } : {},
      });
      return r.status;
    }, token);
    expect(logoutStatus).toBe(204);

    // 服务端必须同步清除父域 cookie（修复前仅删服务端 token，30 天 HttpOnly
    // cookie 残留各子域 → 断言此处捕获残留）
    const cookies = await authCookies(context);
    expect(cookies, '登出后 userId/token cookie 未清除（域变体残留）').toEqual([]);

    // 裸域同步登出：profile 401（无 cookie → 网关 forward-auth 拒绝）
    await page.goto(BARE + '/', { waitUntil: 'networkidle' });
    const status = await page.evaluate(async () => {
      const r = await fetch('/api/accounts/users/profile/', { credentials: 'include' });
      return r.status;
    });
    expect(status).toBe(401);
  });
});
