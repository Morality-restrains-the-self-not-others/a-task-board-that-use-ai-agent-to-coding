// @ts-check
/**
 * 核验：充值页点击「发送验证码」时，recharge_send_sms 的 JSON 应包含 request_id
 * （云短信 SDK 的 RequestId；未走 SDK / mock 时为 null）。
 *
 * 依赖：前后端已启动（playwright.verify.config.js 不拉起 webServer）。
 *
 * 默认对 recharge_send_sms 使用 Playwright stub（多浏览器并行时同一租户连调真实发短信易触发频控 400）。
 * 要跑真实后端短信：E2E_REAL_RECHARGE_SMS=1 npx playwright test ...（建议加 --workers=1）。
 * 仍可与 E2E_STUB_RECHARGE_SMS=1 显式开启 stub（与默认等价）。
 *
 * 未绑定场景填写的手机号须来自 ../../../../conf/core/sms/config.test.yaml 的 test_phone_numbers
 * （与后端 pytest 共用白名单：≥4 个用第 4 个；3 个用第 3 个；2 个用第 2 个）。
 */
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { test, expect } from '@playwright/test';
import { submitLoginWithEmailPassword } from './helpers/e2eLogin.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

function loadPlaywrightBillingTestPhone() {
  const portConfigTestPath = path.resolve(__dirname, '../../../../conf/core/sms/config.test.yaml');
  const text = fs.readFileSync(portConfigTestPath, 'utf8');
  let list = null;
  try {
    const raw = JSON.parse(text);
    // OPT-20260806-057: django 键已清理，仅保留顶层 test_phone_numbers
    list = raw?.test_phone_numbers;
  } catch {
    const m = text.match(/test_phone_numbers:\s*\n((?:\s*-\s*["']?\d+["']?\s*\n)+)/);
    if (m) {
      list = [...m[1].matchAll(/-\s*["']?(\d+)["']?/g)].map((x) => x[1]);
    }
  }
  if (!Array.isArray(list) || list.length < 2) {
    throw new Error(
      `${portConfigTestPath} 中 test_phone_numbers 至少需 2 个号码`
    );
  }
  const idx = list.length >= 4 ? 3 : list.length >= 3 ? 2 : 1;
  return String(list[idx]).trim();
}

test.describe('Billing 充值发短信响应含 request_id', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前后端就绪的全栈 E2E');

  test('POST recharge_send_sms 返回体包含 message 与 request_id 字段', async ({ page }) => {
    const stubRechargeSendSms =
      process.env.E2E_STUB_RECHARGE_SMS === '1' || process.env.E2E_REAL_RECHARGE_SMS !== '1';
    if (stubRechargeSendSms) {
      await page.route('**/billing/accounts/recharge_send_sms/**', async (route) => {
        if (route.request().method() !== 'POST') {
          await route.continue();
          return;
        }
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            message: '验证码已发送',
            request_id: 'e2e-stub-aliyun-request-id',
            sms_provider: 'aliyun',
          }),
        });
      });
    }

    // 避免测试账号在真实环境中「已通过短信验证」时不再渲染发码按钮，导致用例与环境强耦合
    await page.route('**/billing/accounts/recharge_phone_status/**', async (route) => {
      if (route.request().method() !== 'GET') {
        await route.continue();
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          has_phone: false,
          phone_masked: '',
          sms_verified: false,
          paypal_enabled: false,
        }),
      });
    });

    const loginUrl = '/auth/login/';
    const tenantId = '821976991517573120';
    const rechargeUrl = `/tenant/${tenantId}/billing/recharge/`;

    await page.goto(loginUrl);
    await page.waitForLoadState('networkidle');

    await submitLoginWithEmailPassword(page, 'contact@daydaymoney.com', process.env.PLAYWRIGHT_TEST_PASSWORD);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1500);

    await page.goto(rechargeUrl);
    await page.waitForLoadState('networkidle');

    const phoneInput = page.getByPlaceholder('11 位手机号');
    if (await phoneInput.isVisible()) {
      await phoneInput.fill(loadPlaywrightBillingTestPhone());
    }

    const sendBtn = page.getByRole('button', { name: '发送验证码' });
    await expect(sendBtn).toBeVisible({ timeout: 15000 });

    const resPromise = page.waitForResponse(
      (r) =>
        r.url().includes('recharge_send_sms') && r.request().method() === 'POST',
      { timeout: 60000 }
    );
    await sendBtn.click();
    const res = await resPromise;

    if (res.status() !== 200) {
      const errText = await res.text();
      throw new Error(
        `recharge_send_sms 应 200，实际 ${res.status()} ${res.url()} body=${errText.slice(0, 500)}。` +
          '若因短信外网/SDK 失败，可设 SMS_PROVIDER=mock-for-tests 启动后端；本用例默认已 stub 发短信，仅当设置 E2E_REAL_RECHARGE_SMS=1 时才走后端真实短信。'
      );
    }
    const data = await res.json();
    expect(data, '响应 JSON').toMatchObject({
      message: '验证码已发送',
    });
    expect(data, '应携带 request_id 字段（真发短信时为字符串，模拟发送时为 null）').toHaveProperty('request_id');
  });
});
