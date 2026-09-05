// @ts-check
/**
 * 验证：创建任务模态框中从 datalist 选择「目标分支」与「基准分支」后，
 * 输入框应失焦，避免建议列表再次弹出。
 *
 * 依赖：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const ACCESS_CODE = 'u824976301710503936';

/**
 * 模拟从 datalist 选中候选值：写入 input 并触发 change（与浏览器选中行为一致）。
 * @param {import('@playwright/test').Locator} input
 * @param {string} value
 */
async function pickDatalistValue(input, value) {
  await input.fill(value);
  await input.evaluate((el, picked) => {
    el.value = picked;
    el.dispatchEvent(new Event('input', { bubbles: true }));
    el.dispatchEvent(new Event('change', { bubbles: true }));
  }, value);
}

async function openCreateTaskModal(page) {
  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD for this test');

  await playwrightLoginWithLegalAccept(page, { email, password });

  const q = new URLSearchParams({ accessCode: ACCESS_CODE });
  await page.goto(`/tenant/${TENANT_ID}/work-panel/?${q.toString()}`);
  await page.waitForLoadState('networkidle', { timeout: 45000 }).catch(() => {});
  await page.waitForTimeout(2500);

  await page.locator('#create-task-btn').click();
  await page.locator('#create-task-modal').waitFor({ state: 'visible', timeout: 15000 });
  await page.waitForTimeout(2000);
}

test('work-panel create-task: merge target datalist blurs after pick', async ({ page }) => {
  await openCreateTaskModal(page);

  const mergeTargetInput = page.locator('#merge-target-name');
  await mergeTargetInput.waitFor({ state: 'visible', timeout: 10000 });
  await pickDatalistValue(mergeTargetInput, 'main');
  await page.waitForTimeout(300);

  await expect(mergeTargetInput).not.toBeFocused();
  await expect(mergeTargetInput).toHaveValue('main');
});

test('work-panel create-task: base branch datalist blurs after pick', async ({ page }) => {
  await openCreateTaskModal(page);

  const projectSelect = page.locator('select[id^="task-project-"]').first();
  await projectSelect.waitFor({ state: 'visible', timeout: 10000 });

  const projectOptions = await projectSelect.evaluate((select) =>
    Array.from(select.options)
      .map((opt) => ({ value: opt.value, text: opt.text }))
      .filter((opt) => opt.value && !opt.disabled),
  );
  test.skip(projectOptions.length === 0, 'No project options available, skipping base branch assertion');

  await projectSelect.selectOption(projectOptions[0].value);
  await page.waitForTimeout(2500);

  const baseBranchInput = page.locator('input[id^="task-base-branch-"]').first();
  await baseBranchInput.waitFor({ state: 'visible', timeout: 10000 });

  const branchCandidates = await baseBranchInput.evaluate((input) => {
    const listId = input.getAttribute('list');
    if (!listId) return [];
    const datalist = document.getElementById(listId);
    if (!datalist) return [];
    return Array.from(datalist.querySelectorAll('option'))
      .map((opt) => opt.value)
      .filter(Boolean);
  });

  test.skip(branchCandidates.length === 0, 'No branch candidates loaded, skipping base branch blur assertion');

  await pickDatalistValue(baseBranchInput, branchCandidates[0]);
  await page.waitForTimeout(300);

  await expect(baseBranchInput).not.toBeFocused();
  await expect(baseBranchInput).toHaveValue(branchCandidates[0]);
});
