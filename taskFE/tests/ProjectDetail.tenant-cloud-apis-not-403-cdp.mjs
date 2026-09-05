/**
 * CDP 核验：项目详情页 installed-images / cloud regions 不得因 deny-internal 误拦成 403。
 * 用法：由 ProjectDetail.tenant-cloud-apis-not-403.playwright.test.sh 调用。
 */
import { chromium } from 'playwright';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const CDP_URL = (process.env.CDP_URL || process.env.PW_CDP_URL || 'http://127.0.0.1:9223').replace(
  /\/$/,
  '',
);
const BASE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'https://www.daydaymoney.com').replace(/\/$/, '');
const TENANT_ID = process.env.TEST_TENANT_ID || '850256677331562496';
const PROJECT_ID = process.env.TEST_PROJECT_ID || 'proj_-5510157156476369751';
const TARGET = `${BASE}/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD ;

const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
const context = browser.contexts()[0] || (await browser.newContext());
const page = await context.newPage();

/** @type {{status:number,url:string}[]} */
const installed = [];
/** @type {{status:number,url:string}[]} */
const regions = [];
/** @type {{status:number,url:string}[]} */
const fails = [];

page.on('response', (resp) => {
  try {
    const url = resp.url();
    if (!url.includes('/api/')) return;
    const status = resp.status();
    if (url.includes('/installed-images')) installed.push({ status, url });
    if (url.includes('/cloud/regions')) regions.push({ status, url });
    if (status >= 400) fails.push({ status, url });
  } catch {
    /* ignore */
  }
});

try {
  await page.goto(`${BASE}/auth/login/`, { waitUntil: 'domcontentloaded', timeout: 90000 });
  if (page.url().includes('/auth/login')) {
    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      skipGoto: true,
      baseURL: BASE,
    });
  }
  await page.goto(TARGET, { waitUntil: 'domcontentloaded', timeout: 90000 });
  await page.waitForTimeout(10000);

  const imgOk = installed.some((x) => x.status >= 200 && x.status < 300);
  const regOk = regions.some((x) => x.status >= 200 && x.status < 300);
  const targetFail = fails.filter(
    (f) => f.url.includes('/installed-images') || f.url.includes('/cloud/regions'),
  );

  console.log('installed-images', installed.map((x) => x.status));
  console.log('regions', regions.map((x) => x.status));
  console.log('targetFails', targetFail);

  if (!imgOk || !regOk || targetFail.length > 0) {
    throw new Error(
      `project detail cloud APIs failed: imgOk=${imgOk} regOk=${regOk} fails=${JSON.stringify(targetFail)}`,
    );
  }
  console.log('PASS');
  process.exit(0);
} catch (err) {
  console.error('FAIL', err?.message || err);
  process.exit(1);
} finally {
  await page.close().catch(() => {});
}
