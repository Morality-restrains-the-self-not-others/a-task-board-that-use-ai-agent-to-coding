// @ts-check
/**
 * GitSiteOAuth 回调跨子域「不劫持登录页」回归（OPT-20260807-071 线上复现修复）
 *
 * 背景：GitHub OAuth 回调按 SSOT 契约落地裸域 daydaymoney.com（state 签名自足、
 * 换票不依赖登录态），但回调页 SPA 外壳（Navbar）会并行发起鉴权 API。当用户
 * 会话凭据不随裸域传输（修复前的 host-only www cookie、或跨源 localStorage
 * 无 token）时，这些请求 401 → handleForwardAuthSessionExpired 统一收口
 * 跳 /auth/login/?next=<回调URL> → OAuth 授权中途被劫持中断。
 * 修复：handleForwardAuthSessionExpired 排除 /redirect/gitsite/* 路径。
 *
 * 本测试用真实账号登录 www，捕获授权 URL（redirect_uri + state），再模拟
 * 「会话 cookie 仅存在于 www（host-only）」的旧凭据形态访问回调：
 *   - 修复前：被劫持到 /auth/login/?next=…（断言失败，指明回归）
 *   - 修复后：换票照常执行（bogus code → github=exchange_rejected 回跳设置页）
 *
 * 运行目标：线上 https://www.daydaymoney.com（需部署含修复的前端包后转绿）。
 * 测试账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 */
import { test, expect } from '@playwright/test';

const WWW = 'https://www.daydaymoney.com';
const EMAIL = 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');
const PROFILE_PATH = '/user/873438061961179136/profile/git-site-oauth/';

async function loginOnWww(page) {
  await page.goto(`${WWW}/auth/login/`, { waitUntil: 'networkidle' });
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

/** 从 github.com 重定向 URL 中解析出真正的 authorize URL（return_to 可能为相对路径）。 */
function parseAuthorizeUrl(pageUrl) {
  const u = new URL(pageUrl);
  const returnTo = u.searchParams.get('return_to') || '';
  const authorizeUrl = returnTo ? 'https://github.com' + decodeURIComponent(returnTo) : pageUrl;
  const au = new URL(authorizeUrl);
  const redirectUri = au.searchParams.get('redirect_uri');
  const state = au.searchParams.get('state');
  if (!redirectUri || !state) {
    throw new Error(`authorize URL 缺 redirect_uri/state: ${pageUrl.slice(0, 200)}`);
  }
  return { redirectUri, state };
}

test.describe('GitSiteOAuth 回调不劫持登录页（跨子域凭据缺失场景）', () => {
  test('cookie 仅存在于 www（host-only 旧凭据）时，回调仍完成换票而非跳登录', async ({ page, context }) => {
    test.setTimeout(120000);

    // 1. 登录 www
    await loginOnWww(page);
    await expect(page).toHaveURL(/www\.daydaymoney\.com/);

    // 2. 打开 git-site-oauth 页并捕获授权 URL
    await page.goto(`${WWW}${PROFILE_PATH}`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    const navPromise = page.waitForURL(/github\.com/, { timeout: 30000 }).catch(() => null);
    await page.getByRole('button', { name: /使用\s*github/i }).first().click({ timeout: 15000 });
    await navPromise;
    const { redirectUri, state } = parseAuthorizeUrl(page.url());
    expect(redirectUri).toContain('daydaymoney.com/redirect/gitsite/github.com/oauth/callback/');

    // 3. 模拟旧凭据形态：域 cookie（.daydaymoney.com）改写成 www host-only，
    //    使 www 可认证而裸域回调页收不到会话 cookie（复现线上劫持条件）。
    const domainCookies = (await context.cookies()).filter(
      (c) => c.domain === '.daydaymoney.com' && (c.name === 'userId' || c.name === 'token'),
    );
    expect(domainCookies.length).toBeGreaterThan(0);
    for (const c of domainCookies) {
      await context.clearCookies({ domain: c.domain, name: c.name });
    }
    await context.addCookies(
      domainCookies.map((c) => ({
        name: c.name,
        value: c.value,
        domain: 'www.daydaymoney.com',
        path: c.path,
        httpOnly: c.httpOnly,
        secure: c.secure,
        sameSite: c.sameSite,
      })),
    );

    // 4. 直接访问回调 URL（裸域，bogus code —— 换票必然 exchange_rejected，
    //    但必须「完成换票流程」，绝不允许被劫持到 /auth/login/）
    const cbUrl = `${redirectUri}?code=bogus-e2e-verify-code&state=${encodeURIComponent(state)}`;
    const hijackPromise = page.waitForURL(/\/auth\/login\//, { timeout: 5000 }).catch(() => null);
    await page.goto(cbUrl, { waitUntil: 'domcontentloaded', timeout: 30000 }).catch(() => {});
    await page.waitForTimeout(6000);
    const hijacked = await hijackPromise;

    // 断言 1：绝不跳登录页（修复目标；修复前此处必然劫持）
    expect(hijacked, '回调被会话失效收口劫持到 /auth/login/ —— 线上回归（OPT-20260807-071）').toBeNull();
    expect(page.url()).not.toContain('/auth/login/');

    // 断言 2：换票流程执行完成 —— 最终回到 www 设置页并展示
    // exchange_rejected 错误 toast（bogus code 的服务端归类），或至少
    // 回到设置页（github=* 回参），证明回调未被中断。
    await page.waitForURL(/www\.daydaymoney\.com/, { timeout: 15000 }).catch(() => {});
    const finalUrl = page.url();
    expect(finalUrl).toContain('/profile/git-site-oauth/');
    await expect
      .poll(() => page.locator('body').innerText(), { timeout: 15000 })
      .toContain('授权失败');
  });

  test('authorize_url 的 redirect_uri 与 state 声明一致（换票不 redirect_uri mismatch）', async ({ page }) => {
    test.setTimeout(120000);
    await loginOnWww(page);
    await page.goto(`${WWW}${PROFILE_PATH}`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    const navPromise = page.waitForURL(/github\.com/, { timeout: 30000 }).catch(() => null);
    await page.getByRole('button', { name: /使用\s*github/i }).first().click({ timeout: 15000 });
    await navPromise;
    const { redirectUri, state } = parseAuthorizeUrl(page.url());

    // state 为签名 JWT：payload 段 base64url 解码后应含与 authorize 参数相同的 redirect_uri
    const payload = Buffer.from(state.split('.')[1].replace(/-/g, '+').replace(/_/g, '/'), 'base64').toString('utf8');
    expect(payload).toContain(`"redirect_uri":"${redirectUri}"`);
  });
});
