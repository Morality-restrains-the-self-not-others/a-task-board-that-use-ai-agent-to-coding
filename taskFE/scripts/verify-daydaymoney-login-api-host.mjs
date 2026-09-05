/**
 * 核验：登录页加载 + 尝试登录时 API 是否发往 api.daydaymoney.com。
 * 运行：在 task2app 下 `npm --prefix playwright run verify:daydaymoney-api-host`（或 `node playwright/./scripts/verify-daydaymoney-login-api-host.mjs`）
 */
import { chromium } from '@playwright/test';

const LOGIN_URL = process.env.DAYDAYMONEY_LOGIN_URL || 'http://www.daydaymoney.com/auth/login/';

async function main() {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  const requests = [];
  page.on('request', (req) => {
    try {
      const u = new URL(req.url());
      requests.push({
        method: req.method(),
        url: req.url(),
        host: u.hostname,
      });
    } catch {
      /* ignore */
    }
  });

  // 须在 goto 之前注册，否则会错过导航阶段已完成的响应并长时间空等
  const privacyWait = page
    .waitForResponse(
      (r) => r.url().includes('/api/privacy-policy/') && r.request().method() === 'GET',
      { timeout: 25000 }
    )
    .catch(() => null);
  const licenseWait = page
    .waitForResponse(
      (r) => r.url().includes('/api/license-agreement/') && r.request().method() === 'GET',
      { timeout: 25000 }
    )
    .catch(() => null);
  await page.goto(LOGIN_URL, { waitUntil: 'domcontentloaded', timeout: 90000 });
  await Promise.all([privacyWait, licenseWait]);
  await page.waitForTimeout(500);

  // 与 playwrightLogin.js 一致：邮箱/密码 Tab + 勾选条款后提交（假账号即可触发 /api/auth/ POST）
  const emailPwdTab = page.getByRole('button', { name: /邮箱\/密码|邮箱.*密码/ }).first();
  if (await emailPwdTab.isVisible().catch(() => false)) {
    await emailPwdTab.click();
    await page.waitForTimeout(300);
  }
  const acceptAll = page.getByTestId('login-accept-all');
  if (await acceptAll.isVisible().catch(() => false)) {
    await acceptAll.check();
  } else {
    const privacy = page.getByTestId('login-privacy-accept');
    const license = page.getByTestId('login-license-accept');
    if (await privacy.isVisible().catch(() => false)) await privacy.check();
    if (await license.isVisible().catch(() => false)) await license.check();
  }
  await page
    .waitForFunction(
      () => {
        const btn = document.querySelector('form button[type="submit"]');
        return btn && !btn.disabled;
      },
      null,
      { timeout: 15000 }
    )
    .catch(() => null);
  const form = page.locator('form').first();
  const email = form.locator('#email');
  if (await email.isVisible().catch(() => false)) {
    await email.fill('playwright-verify-nonexistent@example.com');
    await form.locator('#password').fill('wrong-password-verify');
    const loginBtn = form.getByRole('button', { name: /^登录$/ });
    const authWait = page
      .waitForRequest((req) => req.method() === 'POST' && req.url().includes('/api/auth'), { timeout: 20000 })
      .catch(() => null);
    await loginBtn.click({ force: true });
    await authWait;
    await page.waitForTimeout(800);
  }

  const apiBaseRuntime = await page.evaluate(() => (window.config && window.config.API_BASE_URL) || '');

  const byHost = new Map();
  for (const r of requests) {
    byHost.set(r.host, (byHost.get(r.host) || 0) + 1);
  }

  const apiAuth = requests.filter((r) => r.url.includes('/api/auth'));
  const apiSubdomain = requests.filter((r) => r.host.includes('api.daydaymoney'));

  console.log('Login URL:', LOGIN_URL);
  console.log('window.config.API_BASE_URL:', JSON.stringify(apiBaseRuntime));
  console.log('Request counts by hostname:', Object.fromEntries(byHost));
  console.log('POST /api/auth/* URLs:', apiAuth.map((r) => `${r.method} ${r.url}`));
  console.log(
    apiSubdomain.length
      ? `ISSUE: ${apiSubdomain.length} request(s) used host api.daydaymoney.com`
      : 'OK: no requests targeted api.daydaymoney.com'
  );
  if (apiSubdomain.length) {
    apiSubdomain.slice(0, 30).forEach((r) => console.log(' ', r.method, r.url));
  }

  const apiUrls = requests.filter((r) => r.url.includes('/api/')).map((r) => `${r.method} ${r.url}`);
  if (!apiAuth.length && apiUrls.length) {
    console.log('Sample /api/ requests (no /api/auth POST captured — 按钮可能仍禁用或 Tab 非邮箱):');
    apiUrls.slice(0, 25).forEach((u) => console.log(' ', u));
  }

  await browser.close();
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
