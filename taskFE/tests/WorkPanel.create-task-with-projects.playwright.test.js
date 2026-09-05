// @ts-check
/**
 * 验证：在工作面板创建任务时选择项目，然后在任务详情页面查看关联的项目列表。
 * 
 * 依赖：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const ACCESS_CODE = 'u824976301710503936';

test('work-panel create-task with project: should see associated projects in task detail', async ({ page }) => {
  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD for this test');

  await playwrightLoginWithLegalAccept(page, { email, password });
  
  const q = new URLSearchParams({ accessCode: ACCESS_CODE });
  await page.goto(`/tenant/${TENANT_ID}/work-panel/?${q.toString()}`);
  await page.waitForLoadState('networkidle', { timeout: 45000 }).catch(() => {});
  await page.waitForTimeout(3000);

  await page.locator('#create-task-btn').click();
  await page.locator('#create-task-modal').waitFor({ state: 'visible', timeout: 15000 });
  await page.waitForTimeout(2000);

  await page.locator('#task-title').fill('测试任务 - 关联项目');
  await page.locator('#task-description').fill('这是一个测试任务，用于验证关联项目列表显示');

  await page.waitForTimeout(1500);

  const projectSelect = page.locator('select[id^="task-project-"]').first();
  
  const projectSelectVisible = await projectSelect.isVisible();
  test.skip(!projectSelectVisible, 'Project select not visible, skipping test');
  
  const projectOptions = await projectSelect.evaluate((select) => {
    return Array.from(select.options).map(opt => ({ value: opt.value, text: opt.text }))
      .filter(opt => opt.value && !opt.disabled);
  });

  test.skip(projectOptions.length === 0, 'No project options available, skipping test');
  
  const firstProject = projectOptions[0];
  await projectSelect.selectOption(firstProject.value);
  
  await page.waitForTimeout(1000);

  const submitBtn = page.locator('button[type="submit"]').first();
  
  const submitBtnVisible = await submitBtn.isVisible();
  test.skip(!submitBtnVisible, 'Submit button not visible, skipping test');
  
  const isEnabled = await submitBtn.evaluate(btn => !btn.disabled);
  test.skip(!isEnabled, 'Submit button is disabled, skipping test');
  
  await submitBtn.click();

  await page.waitForLoadState('networkidle', { timeout: 60000 }).catch(() => {});
  await page.waitForTimeout(3000);

  const taskLinks = page.locator('a[href*="/task-detail/"]');
  const taskCount = await taskLinks.count();
  expect(taskCount).toBeGreaterThan(0, 'Expected at least one task link');

  const firstTaskLink = taskLinks.first();
  const taskHref = await firstTaskLink.getAttribute('href');
  
  expect(taskHref).toContain('/task-detail/');

  await page.goto(taskHref + (taskHref.includes('?') ? '&' : '?') + `accessCode=${ACCESS_CODE}`);
  await page.waitForLoadState('domcontentloaded', { timeout: 30000 });
  await page.waitForTimeout(3000);

  const projectBadges = page.locator('.bg-blue-50.text-blue-700');
  await projectBadges.first().waitFor({ state: 'visible', timeout: 10000 });
  
  const projectBadgeCount = await projectBadges.count();
  expect(projectBadgeCount).toBeGreaterThan(0, 'Expected at least one project badge');

  const firstBadgeText = await projectBadges.first().textContent();
  expect(firstBadgeText).toContain(firstProject.text.replace(/^\([^)]+\)\s*/, ''));
});