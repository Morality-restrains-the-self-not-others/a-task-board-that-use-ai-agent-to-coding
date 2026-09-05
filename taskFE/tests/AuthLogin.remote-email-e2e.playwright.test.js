// @ts-check
/**
 * 远程环境邮箱密码登录 E2E（需已启动 vue + task-auth + saas-backend + 网关）。
 *
 * 运行示例（勿将密码提交到 git）：
 * ```bash
 * cd taskFE
 * LOGIN_URL='http://10.2.150.89:4000' \
 * LOGIN_EMAIL='your@email.com' \
 * LOGIN_PASSWORD='your-password' \
 * npx playwright test -c playwright.verify.config.js \
 *   tests/AuthLogin.remote-email-e2e.playwright.test.js --project=chromium
 * ```
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import {
  installApisixCorsWorkaround,
  projectsUrlPattern,
  trackConsoleErrorsAfterLogin,
  waitForLoginLegalPolicies,
} from './helpers/remoteLoginE2e.js';

const LOGIN_EMAIL = process.env.LOGIN_EMAIL || process.env.E2E_EMAIL || '';
const LOGIN_PASSWORD = process.env.LOGIN_PASSWORD || process.env.E2E_PASSWORD || '';
const LOGIN_URL = (process.env.LOGIN_URL || process.env.PLAYWRIGHT_BASE_URL || '').replace(/\/$/, '');

test.describe('远程邮箱登录 E2E @10.2.150.89:4000', () => {
  test.beforeEach(async ({ page }) => {
    await installApisixCorsWorkaround(page);
  });

  test('登录 API 成功、跳转 /projects/、登录后控制台无 error', async ({ page, baseURL }) => {
    test.setTimeout(120000);
    test.skip(!LOGIN_EMAIL || !LOGIN_PASSWORD, '请设置 LOGIN_EMAIL 与 LOGIN_PASSWORD 环境变量');

    const root = (LOGIN_URL || baseURL || '').replace(/\/$/, '');
    const loginPath = `${root}/auth/login/`;

    const { errors: consoleErrors, markLoginDone } = trackConsoleErrorsAfterLogin(page);

    let authPayload = null;
    let authStatus = 0;

    page.on('response', async (response) => {
      const req = response.request();
      if (req.method() !== 'POST' || !/\/api\/auth\/?$/.test(new URL(req.url()).pathname)) return;
      authStatus = response.status();
      const text = await response.text().catch(() => '');
      try {
        authPayload = JSON.parse(text);
      } catch {
        authPayload = { _raw: text.slice(0, 500) };
      }
    });

    await waitForLoginLegalPolicies(page, loginPath);

    await playwrightLoginWithLegalAccept(page, {
      email: LOGIN_EMAIL,
      password: LOGIN_PASSWORD,
      baseURL: root,
      skipGoto: true,
    });

    markLoginDone();

    expect(authStatus, '登录 API 应返回 2xx').toBeGreaterThanOrEqual(200);
    expect(authStatus, '登录 API 不应为 4xx/5xx').toBeLessThan(400);

    expect(authPayload?.redirect_url, 'redirect_url 应为 /projects/').toMatch(/\/projects\/?$/);
    expect(
      Array.isArray(authPayload?.user?.companies) ? authPayload.user.companies.length : 0,
      'user.companies 应非空（已有企业成员）',
    ).toBeGreaterThan(0);

    await expect(page).toHaveURL(projectsUrlPattern(root), { timeout: 30000 });

    await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {});
    await page.waitForTimeout(2000);

    expect(
      consoleErrors,
      `登录后控制台不应有 error：\n${consoleErrors.join('\n')}`,
    ).toEqual([]);

    test.info().annotations.push({
      type: 'auth-response',
      description: JSON.stringify({
        status: authStatus,
        redirect_url: authPayload?.redirect_url,
        companies: authPayload?.user?.companies,
        final_url: page.url(),
      }),
    });
  });
});
