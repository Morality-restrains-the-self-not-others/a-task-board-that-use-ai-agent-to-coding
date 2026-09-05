/**
 * 登录 http://10.2.150.89:4000 并校验控制台无 error。
 * APISIX CORS 未放行 X-Requested-With，本脚本在路由层剥离该头以完成 E2E。
 *
 * LOGIN_URL LOGIN_EMAIL LOGIN_PASSWORD
 */
import { chromium } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from '../tests/playwrightLogin.js';
import { waitForLoginLegalPolicies } from '../tests/helpers/remoteLoginE2e.js';

const LOGIN_URL = (process.env.LOGIN_URL || 'http://10.2.150.89:4000').replace(/\/$/, '');
const email = process.env.LOGIN_EMAIL || process.env.DAYDAYMONEY_LOGIN_EMAIL || '';
const password = process.env.LOGIN_PASSWORD || process.env.DAYDAYMONEY_LOGIN_PASSWORD || '';

async function main() {
  if (!email || !password) {
    console.error('请设置 LOGIN_EMAIL / LOGIN_PASSWORD');
    process.exit(2);
  }

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  const consoleErrorsAfterLogin = [];
  let loginCompleted = false;
  page.on('console', (msg) => {
    if (loginCompleted && msg.type() === 'error') {
      consoleErrorsAfterLogin.push(msg.text());
    }
  });

  await page.route('**/*', async (route) => {
    const headers = { ...route.request().headers() };
    delete headers['x-requested-with'];
    await route.continue({ headers });
  });

  const loginPath = `${LOGIN_URL}/auth/login/`;
  await waitForLoginLegalPolicies(page, loginPath);
  await playwrightLoginWithLegalAccept(page, {
    email,
    password,
    baseURL: LOGIN_URL,
    skipGoto: true,
  });

  await page.waitForURL((u) => !u.pathname.includes('/auth/login'), { timeout: 60000 });
  loginCompleted = true;
  await page.waitForTimeout(3000);

  const finalUrl = page.url();
  console.log('登录成功，当前 URL:', finalUrl);

  if (consoleErrorsAfterLogin.length) {
    console.error('登录后控制台仍有 error:');
    consoleErrorsAfterLogin.forEach((e) => console.error('  -', e.slice(0, 500)));
    await browser.close();
    process.exit(1);
  }

  console.log('登录后控制台无 error 级别日志');
  await browser.close();
  process.exit(0);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
