// One-off diagnostic: capture console + failed API on work-panel
import { chromium } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const BASE_URL = process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://183.250.1.132:4000';
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD ;

const errors = [];
const failedResponses = [];

const browser = await chromium.connectOverCDP('http://127.0.0.1:9222');
const context = browser.contexts()[0] || (await browser.newContext());
const page = context.pages()[0] || (await context.newPage());

page.on('console', (msg) => {
  if (msg.type() === 'error') errors.push(`[console.error] ${msg.text()}`);
});
page.on('pageerror', (err) => errors.push(`[pageerror] ${err.message}`));
page.on('response', (response) => {
  const url = response.url();
  const status = response.status();
  if (status >= 400 && (url.includes('/api/') || url.includes(BASE_URL))) {
    failedResponses.push(`${status} ${response.request().method()} ${url}`);
  }
});

await playwrightLoginWithLegalAccept(page, { email: EMAIL, password: PASSWORD, baseURL: BASE_URL });
await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel`, { waitUntil: 'domcontentloaded' });
await page.waitForLoadState('networkidle', { timeout: 45000 }).catch(() => {});
await page.waitForTimeout(3000);

console.log('=== CONSOLE ERRORS ===');
for (const e of errors.filter((t) => !t.includes('favicon'))) console.log(e);
console.log('=== FAILED RESPONSES ===');
for (const r of failedResponses) console.log(r);
console.log(`Total console errors: ${errors.length}, failed responses: ${failedResponses.length}`);

await browser.close();
