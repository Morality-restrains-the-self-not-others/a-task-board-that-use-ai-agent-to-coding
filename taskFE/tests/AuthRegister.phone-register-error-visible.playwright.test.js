// @ts-check
/**
 * 回归：phone_register 返回 400「验证码无效或已过期」时，注册页必须展示错误文案并挂 data-traceId。
 * 复现：通用 error 曾被写入 errors.email，手机表单不渲染该字段 → 页面静默。
 */
import { test, expect } from '@playwright/test';

const TRACE = '07b8eec9-7eca-4e1c-8d57-b5cdb12ab02b';
const ERROR_TEXT = '验证码无效或已过期';

async function mockRegisterPageApis(page) {
  await page.route('**/api/public/system-feature-policy/', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: {
          enable_email_register: true,
          enable_phone_register: true,
          phone_country_options: [{ code: '+86', name: '中国大陆' }],
        },
      }),
    });
  });

  await page.route('**/api/public/registration-invite-policy/', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'success', data: { enabled: false } }),
    });
  });

  await page.route('**/api/privacy-policy/public/current/', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: '877396269985726464',
        version: 'v1',
        title: '隐私政策',
        content: 'privacy',
      }),
    });
  });

  await page.route('**/api/license-agreement/public/current/', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: '877396270002503680',
        version: 'v1',
        title: '服务协议',
        content: 'license',
      }),
    });
  });

  await page.route('**/api/accounts/users/phone_register/', async (route) => {
    await route.fulfill({
      status: 400,
      contentType: 'application/json',
      headers: { 'X-Trace-Id': TRACE },
      body: JSON.stringify({
        error: ERROR_TEXT,
        message: ERROR_TEXT,
        status: 'error',
        trace_id: TRACE,
      }),
    });
  });
}

test.describe('Register phone_register 业务错误展示', () => {
  test('400 验证码无效或已过期须出现在页面且带 data-traceId', async ({ page, baseURL }) => {
    const origin = baseURL || 'http://127.0.0.1:4000';
    await mockRegisterPageApis(page);

    await page.goto(`${origin}/auth/register/`);
    await page.waitForLoadState('domcontentloaded');

    await page.locator('#register-phone-national').fill('13900001111');
    await page.locator('#code').fill('435939');
    await page.locator('#password').fill(process.env.PLAYWRIGHT_TEST_PASSWORD);
    await page.locator('[data-testid="register-privacy-accept"]').check();
    await page.locator('[data-testid="register-license-accept"]').check();

    const submit = page.locator('[data-testid="register-submit"]');
    await expect(submit).toBeEnabled({ timeout: 15000 });
    await submit.click();

    const banner = page.locator('[data-testid="register-submit-error"]');
    await expect(banner).toBeVisible({ timeout: 10000 });
    await expect(banner).toContainText(ERROR_TEXT);
    await expect(banner).toHaveAttribute('data-traceId', TRACE);
  });
});
