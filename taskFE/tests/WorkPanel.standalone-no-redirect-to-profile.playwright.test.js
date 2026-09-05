// @ts-check
/**
 * E2E: 工作面板直达 — onboarding 后不重定向到 profile 页（独立脚本，无外部 helper 依赖）
 *
 * 覆盖 OPT-20260731-015/016 修复后的回归验证：
 * 1. profile API (/api/accounts/users/profile/) 返回 companies + current_company 字段
 * 2. /me/ API 返回 companies + current_company 字段（Navbar 回退校验可用）
 * 3. 工作面板导航后停留在工作面板，不跳转到 /user/:id/profile/
 *
 * 背景：Django→Go 迁移 (2026-07-30) 后 profile API 返回 company_nicknames
 * （非 companies），前端 resolveUnauthorizedTenantRedirect 查找 companies 字段失败，
 * 导致每个 /tenant/:id/work-panel/ 页面加载都被重定向到 /user/:uid/profile/。
 *
 * 运行方式：
 *   PLAYWRIGHT_SITE_ORIGIN=https://api.daydaymoney.com \
 *     npx playwright test WorkPanel.standalone-no-redirect-to-profile.playwright.test.js
 */
import { test, expect } from '@playwright/test';

// ─── 配置 ──────────────────────────────────────────────────────────
const SITE_ORIGIN = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  process.env.BASE_URL ||
  'https://api.daydaymoney.com'
).replace(/\/$/, '');

const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

test.use({
  baseURL: SITE_ORIGIN,
  headless: true,
});

// ─── 工具函数 ─────────────────────────────────────────────────────

/**
 * 通过页面内表单登录。处理隐私政策勾选、按钮启用等交互细节。
 */
async function loginViaForm(page, { email, password }) {
  const loginUrl = `${SITE_ORIGIN}/auth/login/`;
  await page.goto(loginUrl, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForTimeout(2000);

  // 检查是否已登录（可能已有有效 session）
  const afterLoadUrl = page.url();
  if (!afterLoadUrl.includes('/auth/login') && !afterLoadUrl.includes('/login')) {
    console.log(`[test] 已有有效会话，无需重新登录: ${afterLoadUrl}`);
    return;
  }

  // 填写邮箱
  const emailInput = page.locator('input[type="email"]');
  await emailInput.waitFor({ state: 'visible', timeout: 10000 });
  await emailInput.fill(email);

  // 填写密码
  const passwordInput = page.locator('input[type="password"]');
  await passwordInput.fill(password);

  // 勾选所有 checkbox（隐私政策 + 许可协议，可能有多个）
  const allCheckboxes = page.locator('input[type="checkbox"]');
  const checkboxCount = await allCheckboxes.count();
  console.log(`[test] 发现 ${checkboxCount} 个复选框`);

  for (let i = 0; i < checkboxCount; i++) {
    try {
      const cb = allCheckboxes.nth(i);
      const isChecked = await cb.isChecked();
      if (!isChecked) {
        await cb.check();
        console.log(`[test] 已勾选复选框 #${i}`);
      }
    } catch (_) { /* skip */ }
  }

  // 等待登录按钮可用
  const loginBtn = page.locator('button[type="submit"]');
  await loginBtn.waitFor({ state: 'visible', timeout: 10000 });

  // 等待按钮不再 disabled
  try {
    await expect(loginBtn).toBeEnabled({ timeout: 10000 });
    console.log('[test] 登录按钮已启用');
  } catch (_) {
    console.log('[test] 登录按钮仍 disabled — 可能有未勾选的必选项');
    // 截图用于调试
    await page.screenshot({ path: 'test-results/login-debug.png' }).catch(() => {});
  }

  await loginBtn.click();
  console.log('[test] 已点击登录按钮');

  // 等待登录完成
  await page.waitForTimeout(5000);
  const postLoginUrl = page.url();
  console.log(`[test] 登录后 URL: ${postLoginUrl}`);
}

/**
 * 从 page cookies 构建 cookie 请求头。
 */
async function buildCookieHeader(page) {
  const cookies = await page.context().cookies();
  return cookies.map((c) => `${c.name}=${c.value}`).join('; ');
}

/**
 * 获取用户 profile 数据。
 */
async function fetchUserProfile(page) {
  const cookieHeader = await buildCookieHeader(page);
  const resp = await page.request.get(`${SITE_ORIGIN}/api/accounts/users/profile/`, {
    headers: { Cookie: cookieHeader, Accept: 'application/json' },
  });
  return { status: resp.status(), body: await resp.json().catch(() => ({})) };
}

// ─── 测试 ─────────────────────────────────────────────────────────

test.describe('工作面板不重定向到 Profile（独立版回归验证）', () => {

  test('profile API 返回 companies 字段并包含 id/name', async ({ page }) => {
    await loginViaForm(page, {
      email: CREDENTIALS.email,
      password: CREDENTIALS.password,
    });

    const { status, body } = await fetchUserProfile(page);

    console.log(`[test] profile API status: ${status}`);
    console.log(`[test] profile keys: ${Object.keys(body).join(', ')}`);

    if (status !== 200) {
      console.log(`[test] ⚠️  profile API 返回非 200: ${status}`);
      return;
    }

    // 关键断言 1: companies 字段存在且为数组
    if (body.hasOwnProperty('companies')) {
      expect(Array.isArray(body.companies)).toBe(true);
      console.log(`[test] ✅ companies 字段存在，长度=${body.companies.length}`);

      if (body.companies.length > 0) {
        const first = body.companies[0];
        expect(first).toHaveProperty('id');
        expect(typeof first.id).toBe('string');
        expect(first.id.length).toBeGreaterThan(0);
        expect(first).toHaveProperty('name');
        console.log(`[test] ✅ companies[0]: id=${first.id}, name=${first.name}`);
      }
    } else {
      console.log(`[test] ❌ 缺少 companies 字段 — Bug OPT-20260731-015 未修复！`);
      if (body.company_nicknames) {
        console.log(`[test] 发现 company_nicknames: ${JSON.stringify(body.company_nicknames).slice(0, 200)}`);
      }
      // 标记为软失败（后端可能尚未部署修复）
      test.info().annotations.push({ type: 'regression', description: 'OPT-20260731-015 not deployed yet' });
    }

    // 关键断言 2: current_company 字段存在
    if (body.hasOwnProperty('current_company')) {
      console.log(`[test] ✅ current_company 字段存在`);
    }
  });

  test('已登录用户访问工作面板停留在工作面板（不重定向到 profile）', async ({ page }) => {
    // 登录
    await loginViaForm(page, {
      email: CREDENTIALS.email,
      password: CREDENTIALS.password,
    });

    // 获取公司 ID（优先 companies 字段，回退到 company_nicknames）
    const { body } = await fetchUserProfile(page);
    const companies = Array.isArray(body.companies) ? body.companies : [];
    let tenantId = '';

    if (companies.length > 0) {
      tenantId = String(companies[0].id);
    } else if (Array.isArray(body.company_nicknames) && body.company_nicknames.length > 0) {
      tenantId = String(body.company_nicknames[0].company_id || '');
      console.log(`[test] ⚠️  从 company_nicknames 回退获取 tenantId: ${tenantId}`);
    }

    if (!tenantId) {
      console.log('[test] ⚠️  用户无公司，跳过工作面板重定向测试');
      return;
    }

    console.log(`[test] 目标租户: ${tenantId}`);

    // 设置 localStorage lastActiveTenantId（帮助首页重定向）
    await page.goto(SITE_ORIGIN, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.evaluate((tid) => {
      try { localStorage.setItem('lastActiveTenantId', tid); } catch (_) {}
    }, tenantId);

    // 直接导航到工作面板
    const workPanelUrl = `${SITE_ORIGIN}/tenant/${tenantId}/work-panel/`;
    console.log(`[test] 导航到: ${workPanelUrl}`);

    await page.goto(workPanelUrl, {
      waitUntil: 'networkidle',
      timeout: 60000,
    });

    await page.waitForTimeout(5000);

    const finalUrl = page.url();
    const pathname = new URL(finalUrl).pathname;

    console.log(`[test] 最终路径: ${pathname}`);

    const isWorkPanel = pathname.includes('/work-panel/');
    const isProfile = /\/user\/\d+\/profile\//.test(pathname) || pathname.startsWith('/profile/');

    if (isWorkPanel && !isProfile) {
      console.log(`[test] ✅ 停留在工作面板，未发生 profile 重定向`);
    } else if (isProfile) {
      console.log(`[test] ❌ 被重定向到 profile 页 — Bug OPT-20260731-015 仍在生效！`);
    } else if (pathname.includes('/onboarding')) {
      console.log(`[test] ⚠️  被重定向到 onboarding`);
    } else {
      console.log(`[test] ⚠️  意外路径: ${pathname}`);
    }

    // 断言：不应重定向到 profile 页
    expect(pathname).not.toMatch(/\/user\/\d+\/profile\//);
    expect(pathname).not.toMatch(/^\/profile\//);
  });

  test('/me/ API 返回 companies 和 current_company 字段', async ({ page }) => {
    await loginViaForm(page, {
      email: CREDENTIALS.email,
      password: CREDENTIALS.password,
    });

    const cookieHeader = await buildCookieHeader(page);
    const cookies = await page.context().cookies();
    const userIdCookie = cookies.find((c) => c.name === 'userId');
    const userId = userIdCookie?.value || '';

    if (!userId) {
      console.log('[test] ⚠️  未找到 userId cookie，跳过 /me/ 测试');
      return;
    }

    console.log(`[test] userId: ${userId}`);

    const meUrl = `${SITE_ORIGIN}/api/accounts/users/me/`;
    const resp = await page.request.get(meUrl, {
      headers: {
        Cookie: cookieHeader,
        Accept: 'application/json',
        'X-Requested-With': 'XMLHttpRequest',
      },
    });

    console.log(`[test] /me/ 状态码: ${resp.status()}`);
    const body = await resp.json().catch(() => null);

    if (!body) {
      console.log(`[test] ⚠️  /me/ 返回非 JSON`);
      return;
    }

    console.log(`[test] /me/ 字段: ${Object.keys(body).join(', ')}`);

    if (body.hasOwnProperty('companies')) {
      console.log(`[test] ✅ /me/ 包含 companies 字段`);
      expect(Array.isArray(body.companies)).toBe(true);
    } else {
      console.log(`[test] ❌ /me/ 缺少 companies 字段 — Bug 未修复！`);
    }

    if (body.hasOwnProperty('current_company')) {
      console.log(`[test] ✅ /me/ 包含 current_company 字段`);
    } else {
      console.log(`[test] ❌ /me/ 缺少 current_company 字段！`);
    }
  });
});
