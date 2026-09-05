// @ts-check
/**
 * 回归：在任务详情页评论「提交并运行」身份下拉选择时，不应触发页面整体下滚；
 * 且不得再 PATCH 任务级 repo-clone-git-identities（身份已评论级）。
 *
 * 依赖：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const WORKSPACE = '827923618602258432';
const TASK_ID = '840502615007272960';
const SITE_ORIGIN = String(process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://daydaymoney.com').replace(/\/$/, '');

test('selecting comment-run git identity should not scroll page down unexpectedly', async ({ page }) => {
  test.setTimeout(120000);
  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD for this test');

  let patchTriggered = false;
  await page.route('**/repo-clone-git-identities/', async (route) => {
    const request = route.request();
    if (request.method() === 'PATCH') {
      patchTriggered = true;
    }
    await route.continue();
  });

  await playwrightLoginWithLegalAccept(page, { email, password, baseURL: SITE_ORIGIN });
  await page.goto(`${SITE_ORIGIN}/tenant/${TENANT_ID}/workspace/${WORKSPACE}/task-detail/${TASK_ID}/`);
  await page.waitForLoadState('domcontentloaded', { timeout: 30000 });

  const mentionEditor = page.getByTestId('comment-content-editor');
  test.skip(!(await mentionEditor.isVisible().catch(() => false)), 'at-mode mention editor not visible');
  await mentionEditor.click();
  await mentionEditor.pressSequentially('@');
  const pickerItem = page.locator('[data-testid="comment-image-mention-picker"] [role="option"]').first();
  test.skip(!(await pickerItem.isVisible({ timeout: 8000 }).catch(() => false)), 'no installed image to mention');
  await pickerItem.click();

  const select = page.getByTestId('comment-repo-git-identity-select').first();
  await expect(select).toBeVisible({ timeout: 15000 });

  const optionValues = await select.locator('option').evaluateAll((options) =>
    options
      .map((option) => ({ value: option.value, disabled: option.disabled }))
      .filter((option) => !option.disabled && option.value)
      .map((option) => option.value)
  );
  test.skip(optionValues.length === 0, 'No selectable Git identity options found');

  const currentValue = await select.inputValue();
  const targetValue = optionValues.find((value) => value !== currentValue) || optionValues[0];
  test.skip(!targetValue, 'No Git identity option to select');

  await select.scrollIntoViewIfNeeded();
  const beforeScrollY = await page.evaluate(() => window.scrollY);

  await select.selectOption(targetValue);
  await page.waitForTimeout(1500);

  expect(patchTriggered, 'must not write task-level repo-clone-git-identities').toBeFalsy();

  const afterScrollY = await page.evaluate(() => window.scrollY);
  const delta = Math.abs(afterScrollY - beforeScrollY);
  expect(delta, `Page should remain stable after selection, but scrolled ${delta}px`).toBeLessThan(20);
});
