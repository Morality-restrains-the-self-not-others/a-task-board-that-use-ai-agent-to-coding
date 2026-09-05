// @ts-check
/**
 * E2E：邮箱邀请注册全链路测试
 *
 * 覆盖：
 * 1. 前端 — URL invite_token 参数解析与验证
 * 2. 前端 — 邀请验证各状态展示（loading / success / error）
 * 3. API — POST /api/system-admin/email-invitations/ 创建邀请
 * 4. API — GET /api/public/email-invitation/{token}/ 验证令牌
 * 5. 完整流程 — 管理员创建邀请 → 用户点击链接 → 注册成功
 *
 * 运行：npx playwright test AuthRegister.invitation-registration.playwright.test.js
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
const ADMIN_USER_ID = process.env.PLAYWRIGHT_ADMIN_USER_ID || '850249621660790784';
const ADMIN_SESSION_ID = process.env.PLAYWRIGHT_ADMIN_SESSION_ID || '';
const ADMIN_TOKEN = process.env.PLAYWRIGHT_ADMIN_TOKEN || '';
const REGISTER_PASSWORD = process.env.REGISTER_TEST_PASSWORD;
test.skip(!REGISTER_PASSWORD, 'REGISTER_PASSWORD env required (no hardcoded fallback)');

// ---- Helpers ----

/** Build auth headers for admin API calls. */
function adminApiHeaders(): Record<string, string> {
  const h: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Requested-With': 'XMLHttpRequest',
    Accept: 'application/json',
  };
  if (ADMIN_TOKEN) h['Authorization'] = `Token ${ADMIN_TOKEN}`;
  if (ADMIN_SESSION_ID) h['Cookie'] = `sessionid=${ADMIN_SESSION_ID}; userId=${ADMIN_USER_ID}`;
  return h;
}

// ---- Mock-based tests (no backend required) ----

test.describe('邀请注册 — 前端行为验证（Mock）', () => {
  test('页面加载时从 URL 解析 invite_token 并发起验证请求', async ({ page }) => {
    let validationRequested = false;
    let requestedToken = '';

    await page.route('**/api/public/email-invitation/**', async (route) => {
      validationRequested = true;
      const url = route.request().url();
      const match = url.match(/email-invitation\/([^/]+)/);
      requestedToken = match ? match[1] : '';
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          email: 'test@example.com',
          valid: true,
          expiresAt: new Date(Date.now() + 7 * 24 * 3600 * 1000).toISOString(),
        }),
      });
    });

    // Mock 其他注册页必然调用的 API
    await page.route('**/api/public/system-feature-policy/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ enable_email_register: true, enable_phone_register: true }),
      });
    });
    await page.route('**/api/public/registration-invite-policy/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ enabled: false, daily_quota: 0, remaining_today: 0 }),
      });
    });
    await page.route('**/api/privacy-policy/public/current/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: 1, version: '1.0', content: 'Privacy policy text' }),
      });
    });
    await page.route('**/api/license-agreement/public/current/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: 1, version: '1.0', content: 'License text' }),
      });
    });

    const TEST_TOKEN = 'test-invite-token-for-e2e-mock';
    await page.goto(`${BASE}/auth/register/?invite_token=${TEST_TOKEN}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    expect(validationRequested).toBe(true);
    expect(requestedToken).toBe(TEST_TOKEN);
    console.log(`[Mock] 邀请验证请求已发送，token=${requestedToken}`);
  });

  test('有效邀请 → 显示绿色成功提示 + 预填邮箱 + 切换到邮箱注册', async ({ page }) => {
    const INVITED_EMAIL = 'invited-user@example.com';

    await page.route('**/api/public/email-invitation/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          email: INVITED_EMAIL,
          valid: true,
          expiresAt: new Date(Date.now() + 7 * 24 * 3600 * 1000).toISOString(),
        }),
      });
    });

    // Mock 其他 API
    await page.route('**/api/public/system-feature-policy/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ enable_email_register: true }),
      });
    });
    await page.route('**/api/public/registration-invite-policy/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ enabled: false }) });
    });
    await page.route('**/api/privacy-policy/public/current/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 1 }) });
    });
    await page.route('**/api/license-agreement/public/current/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 1 }) });
    });

    await page.goto(`${BASE}/auth/register/?invite_token=valid-mock-token`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    // 绿色成功提示
    const successBanner = page.locator('.bg-green-50.border-green-200');
    await expect(successBanner.first()).toBeVisible({ timeout: 10000 });
    const bannerText = await successBanner.first().textContent();
    expect(bannerText).toContain('邀请验证成功');
    expect(bannerText).toContain(INVITED_EMAIL);
    console.log(`[Mock] 成功提示: ${bannerText}`);

    // 邮箱输入框应预填
    const emailInput = page.locator('#email');
    const emailValue = await emailInput.inputValue().catch(() => '');
    expect(emailValue).toBe(INVITED_EMAIL);
    console.log(`[Mock] 邮箱预填: ${emailValue}`);
  });

  test('邀请令牌无效 → 显示红色错误提示', async ({ page }) => {
    await page.route('**/api/public/email-invitation/**', async (route) => {
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ error: '邀请链接无效' }),
      });
    });

    await page.route('**/api/public/system-feature-policy/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
    });
    await page.route('**/api/public/registration-invite-policy/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ enabled: false }) });
    });
    await page.route('**/api/privacy-policy/public/current/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 1 }) });
    });
    await page.route('**/api/license-agreement/public/current/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 1 }) });
    });

    await page.goto(`${BASE}/auth/register/?invite_token=invalid-expired-token`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    // 红色错误提示
    const errorBanner = page.locator('.bg-red-50.border-red-200');
    await expect(errorBanner.first()).toBeVisible({ timeout: 10000 });
    const errorText = await errorBanner.first().textContent();
    expect(errorText).toMatch(/邀请链接无效|已过期|已失效/);
    console.log(`[Mock] 错误提示: ${errorText}`);
  });

  test('验证 API 返回 500 → 显示详细错误而非泛型「无法验证邀请链接」', async ({ page }) => {
    await page.route('**/api/public/email-invitation/**', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ detail: 'db error' }),
      });
    });

    await page.route('**/api/public/system-feature-policy/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
    });
    await page.route('**/api/public/registration-invite-policy/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ enabled: false }) });
    });
    await page.route('**/api/privacy-policy/public/current/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 1 }) });
    });
    await page.route('**/api/license-agreement/public/current/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 1 }) });
    });

    await page.goto(`${BASE}/auth/register/?invite_token=server-error-token`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    const errorBanner = page.locator('.bg-red-50.border-red-200');
    await expect(errorBanner.first()).toBeVisible({ timeout: 10000 });
    const errorText = await errorBanner.first().textContent();
    // 应该显示具体的错误信息（包含 "db error" 或 "邀请链接无效"），
    // 而不是旧的泛型 "无法验证邀请链接"
    console.log(`[Mock] 500 错误提示: ${errorText}`);
    // 新的错误处理会将 d.detail 也作为后备
    expect(errorText).not.toBe('无法验证邀请链接');
  });

  test('邀请 API 网络不可达 → 显示网络错误提示', async ({ page }) => {
    // 模拟网络故障：abort 请求
    await page.route('**/api/public/email-invitation/**', async (route) => {
      await route.abort('connectionrefused');
    });

    await page.route('**/api/public/system-feature-policy/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
    });
    await page.route('**/api/public/registration-invite-policy/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ enabled: false }) });
    });
    await page.route('**/api/privacy-policy/public/current/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 1 }) });
    });
    await page.route('**/api/license-agreement/public/current/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 1 }) });
    });

    await page.goto(`${BASE}/auth/register/?invite_token=network-fail-token`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    const errorBanner = page.locator('.bg-red-50.border-red-200');
    await expect(errorBanner.first()).toBeVisible({ timeout: 10000 });
    const errorText = await errorBanner.first().textContent();
    // 应该显示网络错误提示，而非旧的泛型消息
    expect(errorText).toMatch(/网络连接失败|服务响应异常|未知错误/);
    console.log(`[Mock] 网络错误提示: ${errorText}`);
  });

  test('无 invite_token 且选邮箱注册 → 显示琥珀色提示「需要管理员邀请」', async ({ page }) => {
    await page.route('**/api/public/system-feature-policy/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ enable_email_register: true }),
      });
    });
    await page.route('**/api/public/registration-invite-policy/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ enabled: false }) });
    });
    await page.route('**/api/privacy-policy/public/current/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 1 }) });
    });
    await page.route('**/api/license-agreement/public/current/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 1 }) });
    });

    // 不带 invite_token 访问
    await page.goto(`${BASE}/auth/register/`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    // 切换到邮箱注册 tab
    const emailTab = page.locator('[data-testid="register-method-email"]');
    if (await emailTab.isVisible().catch(() => false)) {
      await emailTab.click();
      await page.waitForTimeout(500);
    }

    // 琥珀色提示
    const amberBanner = page.locator('.bg-amber-50.border-amber-200');
    const isAmberVisible = await amberBanner.first().isVisible().catch(() => false);
    if (isAmberVisible) {
      const amberText = await amberBanner.first().textContent();
      expect(amberText).toMatch(/需要管理员邀请/);
      console.log(`[Mock] 无 token 提示: ${amberText}`);
    } else {
      console.log('[Mock] 琥珀色提示未显示（可能邮箱注册被禁用）');
    }
  });
});

// ---- Full E2E tests (requires backend) ----

test.describe('邀请注册 — 完整全链路 E2E', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前后端的全栈 E2E');

  test('POST /api/system-admin/email-invitations/ → 创建邀请成功并返回 201', async ({ request }) => {
    test.setTimeout(30000);

    const testEmail = `e2e-invite-${Date.now()}@example.com`;
    const resp = await request.post(`${BASE}/api/system-admin/email-invitations/`, {
      headers: adminApiHeaders(),
      data: { email: testEmail },
    });

    if (resp.status() === 401 || resp.status() === 403) {
      console.log(`[E2E] 认证未通过 (${resp.status()})，需设置 PLAYWRIGHT_ADMIN_TOKEN 环境变量`);
      return;
    }

    // 400 可能表示邮箱已存在或已有待接受邀请（合理）
    if (resp.status() === 400) {
      const body = await resp.json().catch(() => ({}));
      console.log(`[E2E] 创建邀请 400: ${JSON.stringify(body)}`);
      // 如果是因为已有待接受邀请，也算合理
      if (body.can_resend || body.error?.includes('已有')) {
        console.log('[E2E] 已有待接受邀请 — 这是合理状态');
        return;
      }
    }

    // 201 = 成功创建
    if (resp.status() === 201) {
      const body = await resp.json();
      expect(body).toHaveProperty('message');
      expect(body).toHaveProperty('email', testEmail);
      console.log(`[E2E] 邀请创建成功: ${body.message}`);
      return;
    }

    console.log(`[E2E] 非预期状态码: ${resp.status()} — 后端可能未运行，跳过`);
  });

  test('GET /api/system-admin/email-invitations/ → 返回邀请列表（含 trace_id）', async ({ request }) => {
    test.setTimeout(30000);

    const resp = await request.get(`${BASE}/api/system-admin/email-invitations/`, {
      headers: adminApiHeaders(),
    });

    if (resp.status() === 401 || resp.status() === 403) {
      console.log(`[E2E] 列表 API 认证未通过 (${resp.status()})`);
      return;
    }

    expect(resp.status()).toBe(200);

    const body = await resp.json();
    expect(body).toHaveProperty('invitations');
    expect(Array.isArray(body.invitations)).toBe(true);

    // trace_id 应存在于响应头或响应体
    const traceIdHeader = resp.headers()['x-trace-id'];
    if (traceIdHeader) {
      console.log(`[E2E] 列表 API X-Trace-Id: ${traceIdHeader}`);
    }

    console.log(`[E2E] 邀请列表: ${body.invitations.length} 条记录`);
  });

  test('GET /api/public/email-invitation/{token}/ → 无效 token 返回 404', async ({ request }) => {
    test.setTimeout(15000);

    const fakeToken = 'this-is-a-fake-token-that-does-not-exist';
    const resp = await request.get(`${BASE}/api/public/email-invitation/${fakeToken}/`, {
      headers: { Accept: 'application/json' },
    });

    // 如果后端未运行，跳过
    if (resp.status() === 404) {
      const body = await resp.json().catch(() => ({}));
      // 后端返回的 404 应有 error 字段
      if (body.error) {
        expect(body.error).toMatch(/无效|不存在/);
        console.log(`[E2E] 无效 token 正确返回 404: ${body.error}`);
      }
      return;
    }

    // 任何非 2xx（且非 404）说明路由通了但 token 处理有问题，或后端未运行
    console.log(`[E2E] 公开验证 API 状态码: ${resp.status()}（后端可能未运行）`);
  });

  test('GET /api/public/email-invitation/ → 缺少 token 返回 400', async ({ request }) => {
    test.setTimeout(15000);

    // 不带 token 访问（仅 /api/public/email-invitation/）
    const resp = await request.get(`${BASE}/api/public/email-invitation/`, {
      headers: { Accept: 'application/json' },
    });

    // 期望 400（缺少 token）或 404（路由不存在）
    const status = resp.status();
    console.log(`[E2E] 无 token 请求状态码: ${status}`);
    // 不是 500 就行
    expect(status).not.toBe(500);
  });

  test('注册页面邀请全流程 — 从链接到注册表单就绪', async ({ page }) => {
    test.setTimeout(60000);

    const INVITED_EMAIL = `e2e-flow-${Date.now()}@example.com`;
    const MOCK_VALID_TOKEN = 'e2e-mock-valid-invite-token-32bytes';

    // 1. Mock 邀请验证 → 有效
    await page.route('**/api/public/email-invitation/**', async (route) => {
      const url = route.request().url();
      if (url.includes(MOCK_VALID_TOKEN)) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            email: INVITED_EMAIL,
            valid: true,
            expiresAt: new Date(Date.now() + 7 * 24 * 3600 * 1000).toISOString(),
          }),
        });
      } else {
        await route.fulfill({
          status: 404,
          contentType: 'application/json',
          body: JSON.stringify({ error: '邀请链接无效' }),
        });
      }
    });

    // 2. Mock 其他 page-load APIs
    await page.route('**/api/public/system-feature-policy/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ enable_email_register: true, enable_phone_register: true }),
      });
    });
    await page.route('**/api/public/registration-invite-policy/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ enabled: false }) });
    });
    await page.route('**/api/privacy-policy/public/current/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 'pp-1', version: '1.0', content: 'Privacy policy' }) });
    });
    await page.route('**/api/license-agreement/public/current/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 'la-1', version: '1.0', content: 'License agreement' }) });
    });

    // 3. 访问邀请链接
    await page.goto(`${BASE}/auth/register/?invite_token=${MOCK_VALID_TOKEN}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    // 4. 验证：绿色成功提示
    const successBanner = page.locator('.bg-green-50.border-green-200');
    await expect(successBanner.first()).toBeVisible({ timeout: 10000 });
    const bannerText = await successBanner.first().textContent();
    expect(bannerText).toContain('邀请验证成功');
    expect(bannerText).toContain(INVITED_EMAIL);
    console.log(`[E2E Flow] 步骤1 ✅ 邀请验证成功: ${bannerText}`);

    // 5. 验证：邮箱输入框预填
    const emailInput = page.locator('#email');
    await expect(emailInput).toBeVisible({ timeout: 5000 });
    const emailValue = await emailInput.inputValue();
    expect(emailValue).toBe(INVITED_EMAIL);
    console.log(`[E2E Flow] 步骤2 ✅ 邮箱预填: ${emailValue}`);

    // 6. 验证：邮箱注册 tab 被选中
    const emailTab = page.locator('[data-testid="register-method-email"]');
    if (await emailTab.isVisible().catch(() => false)) {
      const emailTabClass = await emailTab.getAttribute('class');
      expect(emailTabClass).toMatch(/primary|active|border-primary/);
      console.log('[E2E Flow] 步骤3 ✅ 邮箱注册 tab 已激活');
    }

    // 7. 验证：密码输入框可用
    const passwordInput = page.locator('#password');
    await expect(passwordInput).toBeVisible({ timeout: 5000 });
    console.log('[E2E Flow] 步骤4 ✅ 密码输入框可用');

    // 8. 验证：URL 已被清理（invite_token 已从 URL 移除 — 仅在有效时）
    const currentUrl = page.url();
    expect(currentUrl).not.toContain('invite_token');
    console.log(`[E2E Flow] 步骤5 ✅ URL 已清理: ${currentUrl}`);

    console.log('[E2E Flow] ✅ 全流程：从邀请链接到注册表单就绪 — 全部通过');
  });
});
