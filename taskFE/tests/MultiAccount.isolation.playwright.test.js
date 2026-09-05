// @ts-check
/**
 * 多账号切换隔离 E2E：A→B 后 /me/ 为 B，且 activate-session 写入 sessionid。
 *
 * 环境变量：
 * - PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD（账号 A）
 * - PLAYWRIGHT_TEST_EMAIL_B / PLAYWRIGHT_TEST_PASSWORD_B（账号 B，缺省则 skip）
 * - PLAYWRIGHT_SITE_ORIGIN / PLAYWRIGHT_GATEWAY_ORIGIN
 *
 * 运行：
 *   cd taskFE && npx playwright test \
 *     tests/MultiAccount.isolation.playwright.test.js \
 *     -c playwright.verify.config.js --project=chromium
 */
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi, readE2eOrigins } from './helpers/gatewayLoginE2e.js';

const EMAIL_A = process.env.PLAYWRIGHT_TEST_EMAIL || process.env.E2E_EMAIL || '';
const PASS_A = process.env.PLAYWRIGHT_TEST_PASSWORD || process.env.E2E_PASSWORD || '';
const EMAIL_B = process.env.PLAYWRIGHT_TEST_EMAIL_B || '';
const PASS_B = process.env.PLAYWRIGHT_TEST_PASSWORD_B || '';

test.describe('多账号切换隔离', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需网关就绪的全栈 E2E');

  test('A→B activate-session 后 me 为 B 且写入 sessionid', async ({ page }) => {
    test.skip(!EMAIL_A || !PASS_A, '需要 PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD');
    test.skip(!EMAIL_B || !PASS_B, '需要 PLAYWRIGHT_TEST_EMAIL_B / PLAYWRIGHT_TEST_PASSWORD_B');

    const { siteOrigin, gatewayOrigin } = readE2eOrigins();

    const loginB = await loginViaGatewayApi(page, {
      email: EMAIL_B,
      password: PASS_B,
      siteOrigin,
      gatewayOrigin,
    });
    const userB = loginB?.user || {};
    const idB = String(userB.id || userB.user_id || '').trim();
    const tokenB = String(loginB?.token || '').trim();
    expect(idB, '账号 B user id').toBeTruthy();
    expect(tokenB, '账号 B token').toBeTruthy();

    const loginA = await loginViaGatewayApi(page, {
      email: EMAIL_A,
      password: PASS_A,
      siteOrigin,
      gatewayOrigin,
    });
    const userA = loginA?.user || {};
    const idA = String(userA.id || userA.user_id || '').trim();
    const tokenA = String(loginA?.token || '').trim();
    expect(idA, '账号 A user id').toBeTruthy();
    expect(tokenA, '账号 A token').toBeTruthy();
    expect(idA).not.toBe(idB);

    await page.goto(`${siteOrigin}/`, { waitUntil: 'domcontentloaded', timeout: 60_000 });
    await page.evaluate(
      ({ tokenA, tokenB, idA, idB, userA, userB }) => {
        localStorage.setItem('authToken', tokenA);
        localStorage.setItem(
          'savedAccounts',
          JSON.stringify([
            {
              userId: idA,
              username: userA.username || userA.email || 'A',
              avatarUrl: null,
              token: tokenA,
              addedAt: Date.now() - 1000,
            },
            {
              userId: idB,
              username: userB.username || userB.email || 'B',
              avatarUrl: null,
              token: tokenB,
              addedAt: Date.now(),
            },
          ]),
        );
        document.cookie = `userId=${encodeURIComponent(idA)}; path=/`;
      },
      { tokenA, tokenB, idA, idB, userA, userB },
    );

    const activateRes = await page.request.post(
      `${gatewayOrigin}/api/accounts/users/activate-session/`,
      {
        headers: {
          Origin: siteOrigin,
          Accept: 'application/json',
          'Content-Type': 'application/json',
          Authorization: `Token ${tokenB}`,
        },
        data: { user_id: idB },
      },
    );
    expect(activateRes.ok(), `activate-session ${activateRes.status()}`).toBeTruthy();
    const activateBody = await activateRes.json();
    expect(activateBody.session_key, 'session_key 不得出现在公网 JSON').toBeFalsy();
    expect(String(activateBody?.user?.id || '')).toBe(idB);

    const setCookie = activateRes.headers()['set-cookie'] || '';
    expect(setCookie, '应 Set-Cookie sessionid').toMatch(/sessionid=/i);

    const sessionMatch = setCookie.match(/sessionid=([^;]+)/i);
    if (sessionMatch) {
      await page.context().addCookies([
        {
          name: 'sessionid',
          value: decodeURIComponent(sessionMatch[1]),
          url: `${siteOrigin}/`,
          httpOnly: true,
        },
      ]);
    }
    await page.evaluate(
      ({ idB, tokenB }) => {
        localStorage.setItem('authToken', tokenB);
        document.cookie = `userId=${encodeURIComponent(idB)}; path=/`;
      },
      { idB, tokenB },
    );

    const meRes = await page.request.get(
      `${gatewayOrigin}/api/accounts/users/me/`,
      {
        headers: {
          Origin: siteOrigin,
          Accept: 'application/json',
          Authorization: `Token ${tokenB}`,
        },
      },
    );
    expect(meRes.status(), await meRes.text()).toBe(200);
    const me = await meRes.json();
    expect(String(me.id || me.user_id || '')).toBe(idB);

    const leakRes = await page.request.get(
      `${gatewayOrigin}/api/accounts/users/me/`,
      {
        headers: {
          Origin: siteOrigin,
          Accept: 'application/json',
          Authorization: `Token ${tokenB}`,
        },
      },
    );
    expect(leakRes.status()).toBe(403);
  });
});
