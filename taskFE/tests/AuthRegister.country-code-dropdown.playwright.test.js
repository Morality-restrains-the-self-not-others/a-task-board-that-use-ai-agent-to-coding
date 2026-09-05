// @ts-check
import { test, expect } from '@playwright/test';

test.describe('注册页面区号下拉框策略过滤', () => {
  test('后端返回 limited 区号列表时，下拉框仅显示过滤后的选项', async ({ page, baseURL }) => {
    const origin = baseURL || 'http://127.0.0.1:4000';

    // Mock system-feature-policy to return restricted country code list
    await page.route('**/api/public/system-feature-policy/', async (route) => {
      const method = route.request().method();
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              phone_country_options: [
                { code: '+86', name: '中国' },
                { code: '+1', name: '美国/加拿大' },
              ],
            },
          }),
        });
        return;
      }
      await route.continue();
    });

    await page.goto('/auth/register/');
    await page.waitForLoadState('networkidle');

    // The country code <select> has aria-label="国家或地区代码"
    const countrySelect = page.locator('select[aria-label="国家或地区代码"]');
    await expect(countrySelect).toBeVisible({ timeout: 10000 });

    // Should have exactly 2 options: +86 and +1
    const options = countrySelect.locator('option');
    await expect(options).toHaveCount(2);

    // Verify the option text content
    await expect(options.nth(0)).toHaveText('+86 中国');
    await expect(options.nth(1)).toHaveText('+1 美国/加拿大');

    // Verify the option values
    await expect(options.nth(0)).toHaveValue('+86');
    await expect(options.nth(1)).toHaveValue('+1');
  });

  test('后端返回空列表时，回退到全量 COUNTRY_DIAL_OPTIONS', async ({ page, baseURL }) => {
    const origin = baseURL || 'http://127.0.0.1:4000';

    // Mock system-feature-policy with empty phone_country_options
    await page.route('**/api/public/system-feature-policy/', async (route) => {
      const method = route.request().method();
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              phone_country_options: [],
            },
          }),
        });
        return;
      }
      await route.continue();
    });

    await page.goto('/auth/register/');
    await page.waitForLoadState('networkidle');

    const countrySelect = page.locator('select[aria-label="国家或地区代码"]');
    await expect(countrySelect).toBeVisible({ timeout: 10000 });

    // Empty array = allow all → fallback to full COUNTRY_DIAL_OPTIONS
    // Should have many more than 2 options
    const options = countrySelect.locator('option');
    const count = await options.count();
    expect(count).toBeGreaterThan(10);

    // Default +86 should still be the first option
    await expect(options.nth(0)).toHaveText('+86 中国');
  });

  test('API 失败时，回退到全量 COUNTRY_DIAL_OPTIONS', async ({ page, baseURL }) => {
    const origin = baseURL || 'http://127.0.0.1:4000';

    // Mock system-feature-policy to return 500
    await page.route('**/api/public/system-feature-policy/', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ detail: 'internal error' }),
      });
    });

    await page.goto('/auth/register/');
    await page.waitForLoadState('networkidle');

    const countrySelect = page.locator('select[aria-label="国家或地区代码"]');
    await expect(countrySelect).toBeVisible({ timeout: 10000 });

    // API failure → fallback to full COUNTRY_DIAL_OPTIONS
    const options = countrySelect.locator('option');
    const count = await options.count();
    expect(count).toBeGreaterThan(10);

    // Default +86 should still be present
    await expect(options.nth(0)).toHaveText('+86 中国');
  });

  test('后端返回策略数据但无 phone_country_options 字段时，回退到全量列表', async ({ page, baseURL }) => {
    const origin = baseURL || 'http://127.0.0.1:4000';

    // Mock system-feature-policy without phone_country_options field
    await page.route('**/api/public/system-feature-policy/', async (route) => {
      const method = route.request().method();
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              enable_phone_login: true,
              // phone_country_options field absent
            },
          }),
        });
        return;
      }
      await route.continue();
    });

    await page.goto('/auth/register/');
    await page.waitForLoadState('networkidle');

    const countrySelect = page.locator('select[aria-label="国家或地区代码"]');
    await expect(countrySelect).toBeVisible({ timeout: 10000 });

    // Missing field → fallback to full COUNTRY_DIAL_OPTIONS
    const options = countrySelect.locator('option');
    const count = await options.count();
    expect(count).toBeGreaterThan(10);
  });
});
