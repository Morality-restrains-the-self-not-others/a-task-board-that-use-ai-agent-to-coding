// Quick verify: work-panel console errors after fixes
import { chromium } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const BASE = process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://183.250.1.132:4000';
const TENANT = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const ws = process.env.PW_WS_ENDPOINT;
if (!ws) {
  console.error('Set PW_WS_ENDPOINT');
  process.exit(2);
}

const errors = [];
const failed = [];

const browser = await chromium.connectOverCDP('http://127.0.0.1:9222');
const ctx = await browser.newContext();
const page = await ctx.newPage();

page.on('console', (msg) => {
  if (msg.type() === 'error') errors.push(msg.text());
});
page.on('pageerror', (err) => errors.push(err.message));
page.on('response', (r) => {
  const u = r.url();
  if (r.status() >= 400 && u.includes('/api/')) failed.push(`${r.status()} ${u}`);
});
page.on('requestfailed', (req) => {
  failed.push(`FAIL ${req.failure()?.errorText || '?'} ${req.url()}`);
});

await playwrightLoginWithLegalAccept(page, {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD ,
  baseURL: BASE,
});

await page.goto(`${BASE}/tenant/${TENANT}/work-panel`, { waitUntil: 'domcontentloaded', timeout: 60000 });
await page.waitForTimeout(4000);

const filtered = errors.filter((t) => !t.includes('favicon'));
console.log('CONSOLE_ERRORS', JSON.stringify(filtered, null, 2));
console.log('FAILED_API', JSON.stringify(failed, null, 2));
console.log('SUMMARY', { consoleErrors: filtered.length, failedApi: failed.length });

await page.close();
await ctx.close();
await browser.close();
process.exit(filtered.length ? 1 : 0);
