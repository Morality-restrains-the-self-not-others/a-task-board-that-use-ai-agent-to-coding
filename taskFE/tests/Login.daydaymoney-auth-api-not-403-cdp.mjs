/**
 * CDP 核验：公网登录页 POST /api/auth/ 不得 403（deny-internal）。
 * 由 Login.daydaymoney-auth-api-not-403.playwright.test.sh 调用。
 */
import { chromium } from 'playwright';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const CDP_URL = (process.env.CDP_URL || process.env.PW_CDP_URL || 'http://127.0.0.1:9223').replace(
  /\/$/,
  '',
);
const LOGIN_URL =
  process.env.DAYDAYMONEY_LOGIN_URL ||
  'https://www.daydaymoney.com/auth/login/?next=https%3A%2F%2Fapi.daydaymoney.com%2Fapi%2Foidc%2Fauthorize%3Fclient_id%3Dgitlab-git-service%26nonce%3D3e7c9c34fecff20ab0c350b28009c3d8%26redirect_uri%3Dhttps%253A%252F%252Fgitlab.daydaymoney.com%252Fusers%252Fauth%252Fopenid_connect%252Fcallback%26response_type%3Dcode%26scope%3Dopenid%2520profile%2520email%26state%3Dffa7a645248bf637f1bd46b3c4bc776e';
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD ;

const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
const context = browser.contexts()[0] || (await browser.newContext());
const page = await context.newPage();

/** @type {{ status: number } | null} */
let authMeta = null;
page.on('response', (resp) => {
  try {
    const u = new URL(resp.url());
    if (u.pathname.replace(/\/$/, '') === '/api/auth' && resp.request().method() === 'POST') {
      authMeta = { status: resp.status() };
      console.log('[auth status]', resp.status());
    }
  } catch {
    /* ignore */
  }
});

try {
  await page.goto(LOGIN_URL, { waitUntil: 'domcontentloaded', timeout: 90000 });
  await playwrightLoginWithLegalAccept(page, { email: EMAIL, password: PASSWORD, skipGoto: true });
  console.log('final url', page.url());
  console.log('authMeta', authMeta);
  if (!authMeta || authMeta.status < 200 || authMeta.status >= 300) {
    throw new Error(`POST /api/auth/ 非 2xx: ${JSON.stringify(authMeta)}`);
  }
  if (page.url().includes('/auth/login')) {
    throw new Error(`仍停留在登录页: ${page.url()}`);
  }
  console.log('PASS');
  process.exit(0);
} catch (err) {
  console.error('FAIL', err?.message || err);
  process.exit(1);
} finally {
  await page.close().catch(() => {});
}
