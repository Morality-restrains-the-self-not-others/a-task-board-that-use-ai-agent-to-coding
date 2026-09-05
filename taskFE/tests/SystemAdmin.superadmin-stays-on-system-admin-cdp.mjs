/**
 * CDP 核验：系统超管访问 /system-admin/ 应停留（不跳转 work-panel）。
 * 账号默认：author@example.com
 * 由 SystemAdmin.superadmin-stays-on-system-admin.playwright.test.sh 调用。
 */
import { chromium } from 'playwright';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const CDP_URL = (process.env.CDP_URL || process.env.PW_CDP_URL || 'http://127.0.0.1:9223').replace(
  /\/$/,
  '',
);
const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'https://www.daydaymoney.com').replace(/\/$/, '');
const EMAIL = process.env.E2E_SUPERADMIN_EMAIL || 'author@example.com';
const PASSWORD = process.env.E2E_SUPERADMIN_PASSWORD ;

const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
const context = browser.contexts()[0] || (await browser.newContext());
const page = await context.newPage();

try {
  await context.clearCookies();
  await page.goto(`${SITE}/auth/login/`, { waitUntil: 'domcontentloaded', timeout: 90000 });
  await playwrightLoginWithLegalAccept(page, {
    email: EMAIL,
    password: PASSWORD,
    skipGoto: true,
  });
  console.log('[superadmin] after login', page.url());

  const me = await page.evaluate(async () => {
    const m = document.cookie.match(/(?:^|; )userId=([^;]+)/);
    const uid = m ? decodeURIComponent(m[1]) : '';
    const r = await fetch(`/api/accounts/users/me/`, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    });
    const j = await r.json().catch(() => ({}));
    return {
      status: r.status,
      is_superuser: j.is_superuser,
    };
  });
  console.log('[superadmin] me', JSON.stringify(me));

  if (me.status !== 200) throw new Error(`/me/ status ${me.status}`);
  if (!(me.is_superuser === true || me.is_superuser === 'True')) {
    throw new Error(`账号不是超管，不能用于本用例: is_superuser=${me.is_superuser}`);
  }

  await page.goto(`${SITE}/system-admin/`, { waitUntil: 'domcontentloaded', timeout: 90000 });
  // 放行后路由应稳定在 system-admin（给守卫异步校验一点时间）
  await page.waitForTimeout(2000);
  const finalUrl = page.url();
  console.log('[superadmin] final url', finalUrl);

  if (!/\/system-admin\/?/.test(new URL(finalUrl).pathname)) {
    throw new Error(`超管应停留在 /system-admin/，实际: ${finalUrl}`);
  }
  if (/\/work-panel\/?/.test(new URL(finalUrl).pathname)) {
    throw new Error(`超管不应被跳转到 work-panel: ${finalUrl}`);
  }

  console.log('PASS');
  process.exit(0);
} catch (err) {
  console.error('FAIL', err?.message || err);
  console.error('url', page.url());
  process.exit(1);
} finally {
  await page.close().catch(() => {});
}
