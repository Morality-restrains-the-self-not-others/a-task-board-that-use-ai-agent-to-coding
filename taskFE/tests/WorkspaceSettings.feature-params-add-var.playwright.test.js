// @ts-check
/**
 * 功能参数页：点击「添加变量」应出现可编辑空行（回归：空 key 被 filter + watch 回写导致无反应）。
 *
 * 依赖：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;

test.describe('WorkspaceSettings 功能参数 · 添加变量', () => {
  test('点击添加变量后出现空输入行，填写后行仍保留', async ({ page }) => {
    const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
    const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
    test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD');

    await playwrightLoginWithLegalAccept(page, { email, password });

    await page.goto(`/tenant/${TENANT_ID}/settings/feature-params/`);
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByRole('heading', { name: '环境变量设置' })).toBeVisible({ timeout: 30000 });

    const addBtn = page.getByRole('button', { name: '+ 添加变量' });
    await expect(addBtn).toBeVisible();

    const keyInputBefore = page.locator('input[placeholder="MY_VAR"]');
    const beforeCount = await keyInputBefore.count();

    await addBtn.click();

    const keyInputs = page.locator('input[placeholder="MY_VAR"]');
    await expect(keyInputs).toHaveCount(beforeCount + 1, { timeout: 5000 });

    const newKey = keyInputs.nth(beforeCount);
    await newKey.fill('CUSTOM_ENDPOINT');
    await expect(newKey).toHaveValue('CUSTOM_ENDPOINT');
    await expect(keyInputs).toHaveCount(beforeCount + 1);

    await addBtn.click();
    await expect(keyInputs).toHaveCount(beforeCount + 2, { timeout: 5000 });
  });
});
