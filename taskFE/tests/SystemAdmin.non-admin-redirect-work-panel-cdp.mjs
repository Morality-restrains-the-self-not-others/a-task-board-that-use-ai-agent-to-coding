/**
 * CDP 核验：非超管访问 /system-admin/ → /tenant/{id}/work-panel/
 * 由 SystemAdmin.non-admin-redirect-work-panel.playwright.test.sh 调用。
 */
import { chromium } from 'playwright';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { ensureE2eNonAdminAccount } from './ensureE2eNonAdminAccount.mjs';

const CDP_URL = (process.env.CDP_URL || process.env.PW_CDP_URL || 'http://127.0.0.1:9223').replace(
  /\/$/,
  '',
);
const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'https://www.daydaymoney.com').replace(/\/$/, '');

const { email, password } = await ensureE2eNonAdminAccount();

const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
const context = browser.contexts()[0] || (await browser.newContext());
const page = await context.newPage();

try {
  await context.clearCookies();
  await page.goto(`${SITE}/auth/login/`, { waitUntil: 'domcontentloaded', timeout: 90000 });
  await playwrightLoginWithLegalAccept(page, {
    email,
    password,
    skipGoto: true,
  });
  console.log('[nonadmin] after login', page.url());

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
      companyId: j.current_company?.id || j.companies?.[0]?.id || null,
    };
  });
  console.log('[nonadmin] me', JSON.stringify(me));

  if (me.status !== 200) throw new Error(`/me/ status ${me.status}`);
  if (me.is_superuser === true || me.is_superuser === 'True') {
    throw new Error(`账号是超管，不能用于本用例: is_superuser=${me.is_superuser}`);
  }
  if (!me.companyId) throw new Error('无租户 companyId，无法断言 work-panel 跳转');

  await page.goto(`${SITE}/system-admin/`, { waitUntil: 'domcontentloaded', timeout: 90000 });
  await page.waitForURL(new RegExp(`/tenant/${me.companyId}/work-panel/?`), { timeout: 30000 });
  const finalUrl = page.url();
  console.log('[nonadmin] final url', finalUrl);
  if (finalUrl.includes('/system-admin/')) {
    throw new Error(`仍停留在 system-admin: ${finalUrl}`);
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
