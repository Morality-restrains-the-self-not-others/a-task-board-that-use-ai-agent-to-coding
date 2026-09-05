/**
 * 单次登录探测（凭据来自环境变量，勿写入仓库）。
 * DAYDAYMONEY_LOGIN_URL DAYDAYMONEY_LOGIN_EMAIL DAYDAYMONEY_LOGIN_PASSWORD
 */
import { chromium } from '@playwright/test';

const LOGIN_URL = process.env.DAYDAYMONEY_LOGIN_URL || 'http://www.daydaymoney.com/auth/login/';
const email = process.env.DAYDAYMONEY_LOGIN_EMAIL || '';
const password = process.env.DAYDAYMONEY_LOGIN_PASSWORD || '';

async function main() {
  if (!email || !password) {
    console.error('请设置 DAYDAYMONEY_LOGIN_EMAIL 与 DAYDAYMONEY_LOGIN_PASSWORD');
    process.exit(2);
  }

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  const consoleMsgs = [];
  page.on('console', (msg) => consoleMsgs.push(`[${msg.type()}] ${msg.text()}`));

  const privacyWait = page
    .waitForResponse((r) => r.url().includes('/api/privacy-policy/') && r.request().method() === 'GET', {
      timeout: 45000,
    })
    .catch(() => null);
  const licenseWait = page
    .waitForResponse((r) => r.url().includes('/api/license-agreement/') && r.request().method() === 'GET', {
      timeout: 45000,
    })
    .catch(() => null);

  await page.goto(LOGIN_URL, { waitUntil: 'domcontentloaded', timeout: 90000 });
  await Promise.all([privacyWait, licenseWait]);
  // SSH 转发到本地 Vite 时 SPA 挂载较慢，须等待登录表单渲染（勿仅用 domcontentloaded）
  await page.waitForSelector('[data-testid="login-privacy-accept"], #email', {
    state: 'visible',
    timeout: 120000,
  });

  const emailPwdTab = page.getByRole('button', { name: /邮箱\/密码|邮箱.*密码/ }).first();
  if (await emailPwdTab.isVisible().catch(() => false)) {
    await emailPwdTab.click();
    await page.waitForTimeout(400);
  }

  // 与 ./tests/playwrightLogin.js 一致：须两项条款均可见后再勾选
  const privacy = page.getByTestId('login-privacy-accept');
  const license = page.getByTestId('login-license-accept');
  await privacy.waitFor({ state: 'visible', timeout: 60000 });
  await license.waitFor({ state: 'visible', timeout: 60000 });
  await privacy.check();
  await license.check();

  await page
    .waitForFunction(
      () => {
        const btn = document.querySelector('form button[type="submit"]');
        return btn && !btn.disabled;
      },
      null,
      { timeout: 25000 }
    )
    .catch(() => null);

  const form = page.locator('form').first();
  await form.locator('#email').fill(email);
  await form.locator('#password').fill(password);

  let authStatus = 0;
  let authUrl = '';
  const authRespWait = page
    .waitForResponse((r) => r.url().includes('/api/auth') && r.request().method() === 'POST', { timeout: 45000 })
    .then((r) => {
      authStatus = r.status();
      authUrl = r.url();
      return r.text();
    })
    .catch(() => null);

  await form.getByRole('button', { name: /^登录$/ }).click();
  const authBody = await authRespWait;

  await page.waitForTimeout(1500);
  const finalUrl = page.url();
  const stillLogin = finalUrl.includes('/auth/login');

  console.log('POST /api/auth status:', authStatus || '(no response)');
  console.log('POST /api/auth URL:', authUrl || '(n/a)');
  if (authBody && authBody.length < 2000) console.log('POST /api/auth body:', authBody.slice(0, 1500));
  console.log('Final URL:', finalUrl);
  console.log('Still on login page:', stillLogin);
  const apiBase = await page.evaluate(() => (window.config && window.config.API_BASE_URL) || '');
  console.log('window.config.API_BASE_URL:', JSON.stringify(apiBase));

  if (!stillLogin && authStatus >= 200 && authStatus < 300) {
    console.log('RESULT: OK (redirected away from login)');
  } else if (!stillLogin) {
    console.log('RESULT: OK (left login page)');
  } else if (authStatus === 401 || authStatus === 400) {
    console.log('RESULT: FAIL (credentials or server rejected)');
  } else if (!authStatus) {
    console.log('RESULT: FAIL (no /api/auth response — button disabled or request blocked)');
  } else {
    console.log('RESULT: UNCLEAR');
  }

  if (stillLogin && consoleMsgs.length) {
    console.log('Last console lines:', consoleMsgs.slice(-20).join('\n'));
  }

  await browser.close();
  process.exit(stillLogin ? 1 : 0);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
