// @ts-check
/**
 * 公网忘记密码：用 CDP 9222 打开重置页，填写邮箱并点「发送重置链接」，
 * 捕获 API 状态 / X-Trace-Id / 成功弹窗或 data-traceId 错误。
 * 不改密码。PRE_COMMIT 跳过。
 */
import { test, expect, chromium } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const CDP_URL = process.env.PW_CDP_URL || 'http://127.0.0.1:9222';
const TARGET =
  process.env.PW_RESET_REQUEST_URL ||
  'https://www.daydaymoney.com/auth/reset-password-request/?accessCode=DR2AKvP9J9';
const RESET_EMAIL = process.env.PW_RESET_EMAIL || 'author@example.com';
const EVIDENCE_DIR = process.env.PW_RESET_EVIDENCE_DIR || '/tmp/pw-reset-evidence';

test.describe('公网发送密码重置链接', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不打公网 SMTP 发信');

  test('填写邮箱并点击发送，捕获 API 与 UI 结果', async () => {
    test.setTimeout(120000);
    fs.mkdirSync(EVIDENCE_DIR, { recursive: true });

    const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
    const context = browser.contexts()[0] || (await browser.newContext());
    const page = await context.newPage();

    /** @type {{url:string,status:number,headers:Record<string,string>,body:string}|null} */
    let apiCapture = null;
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
    await page.waitForLoadState('domcontentloaded');
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
    const headers = {};
    for (const [k, v] of Object.entries(resp.headers())) {
      headers[k.toLowerCase()] = v;
    }
    const bodyText = await resp.text().catch(() => '');
    apiCapture = {
      url: resp.url(),
      status: resp.status(),
      headers: {
        'x-trace-id': headers['x-trace-id'] || '',
        'content-type': headers['content-type'] || '',
      },
      body: bodyText.slice(0, 2000),
    };

    await page.waitForTimeout(1500);
    await page.screenshot({ path: path.join(EVIDENCE_DIR, '02-after-submit.png'), fullPage: true });

    const errorText = await page.locator('[data-alias="cmp-reset-password-request"] .text-red-700').textContent().catch(() => '');
    const errorTraceId = await page.locator('[data-traceId]').first().getAttribute('data-traceid').catch(() => '');

    const evidence = {
      target: TARGET,
      email: RESET_EMAIL,
      cdp: CDP_URL,
      pageUrl: page.url(),
      api: apiCapture,
      dialogs,
      errorText: (errorText || '').trim(),
      errorTraceId: errorTraceId || '',
    };
    const evidencePath = path.join(EVIDENCE_DIR, 'result.json');
    fs.writeFileSync(evidencePath, `${JSON.stringify(evidence, null, 2)}\n`);

    expect(apiCapture, '应发出 send_password_reset_link').not.toBeNull();
    expect(apiCapture.status, `API body=${apiCapture.body}`).toBeLessThan(500);
    if (apiCapture.status === 200) {
      expect(apiCapture.body).toMatch(/已发送|sent/i);
      expect(dialogs.some((d) => /邮箱|已发送/.test(d.message))).toBeTruthy();
    }

    // 不断开共享 CDP Chrome
  });
});
