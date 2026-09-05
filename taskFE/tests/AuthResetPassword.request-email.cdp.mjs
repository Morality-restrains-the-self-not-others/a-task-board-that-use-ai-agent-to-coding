#!/usr/bin/env node
// Standalone Playwright CDP driver (test runner would launch a second Chrome).
import { chromium } from '@playwright/test';
import fs from 'fs';
import path from 'path';

const CDP_URL = process.env.PW_CDP_URL || 'http://127.0.0.1:9222';
const TARGET =
  process.env.PW_RESET_REQUEST_URL ||
  'https://www.daydaymoney.com/auth/reset-password-request/?accessCode=DR2AKvP9J9';
const RESET_EMAIL = process.env.PW_RESET_EMAIL || 'author@example.com';
const EVIDENCE_DIR = process.env.PW_RESET_EVIDENCE_DIR || '/tmp/pw-reset-evidence';

async function main() {
  fs.mkdirSync(EVIDENCE_DIR, { recursive: true });
  const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
  const context = browser.contexts()[0] || (await browser.newContext());
  const page = await context.newPage();
  page.setDefaultTimeout(20000);
  const dialogs = [];
  page.on('dialog', async (dialog) => {
    dialogs.push({ type: dialog.type(), message: dialog.message() });
    try {
      await dialog.accept();
    } catch {
      // already dismissed by navigation or a previous handler
    }
  });

  await page.goto(TARGET, { waitUntil: 'domcontentloaded', timeout: 60000 });
  if (await page.getByRole('heading', { name: '欢迎登录' }).isVisible().catch(() => false)) {
    await page.getByRole('link', { name: '忘记密码？' }).click();
    await page.waitForURL(/reset-password-request/, { timeout: 15000 });
  }
  await page.getByRole('heading', { name: '忘记密码' }).waitFor({ state: 'visible', timeout: 30000 });
  await page.locator('#email').waitFor({ state: 'visible', timeout: 15000 });
  await page.screenshot({ path: path.join(EVIDENCE_DIR, '01-page.png'), fullPage: true });
  await page.locator('#email').fill(RESET_EMAIL);

  const responsePromise = page.waitForResponse(
    (r) => r.url().includes('/api/accounts/users/send_password_reset_link/') && r.request().method() === 'POST',
    { timeout: 60000 },
  );
  await page.locator('form').first().evaluate((form) => form.requestSubmit());
  const resp = await responsePromise;
  const headers = resp.headers();
  const bodyText = await resp.text().catch(() => '');
  await page.waitForTimeout(1500);
  await page.screenshot({ path: path.join(EVIDENCE_DIR, '02-after-submit.png'), fullPage: true });

  const errorText = (await page.locator('[data-alias="cmp-reset-password-request"] .text-red-700').textContent().catch(() => '')) || '';
  const errorTraceId = (await page.locator('[data-traceId]').first().getAttribute('data-traceid').catch(() => '')) || '';

  const evidence = {
    target: TARGET,
    email: RESET_EMAIL,
    cdp: CDP_URL,
    pageUrl: page.url(),
    api: {
      url: resp.url(),
      status: resp.status(),
      headers: {
        'x-trace-id': headers['x-trace-id'] || headers['X-Trace-Id'] || '',
        'content-type': headers['content-type'] || '',
      },
      body: bodyText.slice(0, 2000),
    },
    dialogs,
    errorText: errorText.trim(),
    errorTraceId,
  };
  const evidencePath = path.join(EVIDENCE_DIR, 'result.json');
  fs.writeFileSync(evidencePath, `${JSON.stringify(evidence, null, 2)}\n`);
  console.log(JSON.stringify(evidence, null, 2));
  if (evidence.api.status >= 500) {
    process.exit(1);
  }
  if (evidence.api.status === 200 && !/已发送|sent/i.test(evidence.api.body)) {
    process.exit(1);
  }
  await page.close();
  process.exit(0);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
