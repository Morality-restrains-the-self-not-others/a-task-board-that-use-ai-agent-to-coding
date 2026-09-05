// @ts-check
/**
 * 回归：公网登录页 POST /api/auth/ 不得因 taskAuth→Django enrich-login
 * 误走 APISIX deny-internal 而返回 403 {"detail":"forbidden"}。
 *
 * 运行：
 *   bash taskFE/tests/Login.daydaymoney-auth-api-not-403.playwright.test.sh
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const LOGIN_URL =
  process.env.DAYDAYMONEY_LOGIN_URL ||
  'https://www.daydaymoney.com/auth/login/?next=https%3A%2F%2Fapi.daydaymoney.com%2Fapi%2Foidc%2Fauthorize%3Fclient_id%3Dgitlab-git-service%26nonce%3D3e7c9c34fecff20ab0c350b28009c3d8%26redirect_uri%3Dhttps%253A%252F%252Fgitlab.daydaymoney.com%252Fusers%252Fauth%252Fopenid_connect%252Fcallback%26response_type%3Dcode%26scope%3Dopenid%2520profile%2520email%26state%3Dffa7a645248bf637f1bd46b3c4bc776e';

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

test.describe('daydaymoney 登录 /api/auth/ 不因 deny-internal 403', () => {
  test('邮箱密码登录 POST /api/auth/ 返回 2xx 并离开登录页', async ({ page }) => {
    /** @type {import('@playwright/test').Response | null} */
    let authResponse = null;
    page.on('response', (resp) => {
      try {
        const u = new URL(resp.url());
        if (u.pathname.replace(/\/$/, '') === '/api/auth' && resp.request().method() === 'POST') {
          authResponse = resp;
        }
      } catch {
        /* ignore */
      }
    });

    await page.goto(LOGIN_URL, { waitUntil: 'domcontentloaded', timeout: 90000 });
    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      skipGoto: true,
    });

    expect(authResponse, '应捕获到 POST /api/auth/ 响应').toBeTruthy();
    const status = authResponse.status();
    // 登录成功后常会立刻导航，勿再读 response body（CDP 可能已释放）
    expect(
      status,
      `POST /api/auth/ 不应为 403（常见于 djangoInternalApiBase 指向公网触发 deny-internal）`,
    ).toBeGreaterThanOrEqual(200);
    expect(status).toBeLessThan(300);

    await expect(page).not.toHaveURL(/\/auth\/login/, { timeout: 30000 });
  });
});
