// @ts-check
import { test, expect } from '@playwright/test';

const LOGIN_URL =
  'http://localhost:4000/auth/login/?accessCode=u824976301710503936&mockStart=true';

const EMAIL = process.env.PLAYWRIGHT_LOGIN_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_LOGIN_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

test.describe('Login /api/auth latency', () => {
  test('should finish auth API request within 10s', async ({ page }) => {
    let authRequestAt = -1;
    let authResponseAt = -1;
    let authResponseStatus = -1;

    page.on('request', (req) => {
      if (req.url().includes('/api/auth/') && req.method() === 'POST') {
        authRequestAt = Date.now();
      }
    });

    page.on('response', (resp) => {
      if (resp.url().includes('/api/auth/') && resp.request().method() === 'POST') {
        authResponseAt = Date.now();
        authResponseStatus = resp.status();
      }
    });

    await page.goto(LOGIN_URL, { waitUntil: 'domcontentloaded' });
    await page.locator('#email').fill(EMAIL);
    await page.locator('#password').fill(PASSWORD);

    const acceptAll = page.getByTestId('login-accept-all');
    if (await acceptAll.isVisible().catch(() => false)) {
      await acceptAll.check();
    }

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/api/auth/') && resp.request().method() === 'POST',
      { timeout: 15000 },
    );

    await page.locator('form').first().evaluate((form) => form.requestSubmit());
    await responsePromise;

    expect(authRequestAt).toBeGreaterThan(0);
    expect(authResponseAt).toBeGreaterThan(authRequestAt);
    expect(authResponseAt - authRequestAt).toBeLessThan(10000);
    expect(authResponseStatus).toBeGreaterThanOrEqual(200);
  });
});
