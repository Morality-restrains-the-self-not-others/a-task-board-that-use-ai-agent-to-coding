// @ts-check
/**
 * 验证：任务详情「关联项目」中的项目名称可点击并跳转到项目详情页。
 *
 * 依赖：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const WORKSPACE = '827923618602258432';
const TASK_ID = '846269443533955072';

test('task-detail linked project name navigates to project detail', async ({ page }) => {
  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD for this test');

  await playwrightLoginWithLegalAccept(page, { email, password });

  await page.goto(`/tenant/${TENANT_ID}/workspace/${WORKSPACE}/task-detail/${TASK_ID}/?relayToTrae=true`);
  await page.waitForLoadState('domcontentloaded', { timeout: 30000 });

  const panel = page.getByTestId('task-repo-clone-identity-panel');
  await expect(panel).toBeVisible({ timeout: 30000 });

  const projectLink = panel.getByTestId('task-linked-project-name-link').first();
  await expect(projectLink).toBeVisible({ timeout: 15000 });

  const href = await projectLink.getAttribute('href');
  expect(href, '项目名链接应指向项目详情路径').toMatch(
    new RegExp(`^/tenant/${TENANT_ID}/projects/\\d+/$`),
  );

  const projectIdMatch = href?.match(/\/projects\/(\d+)\//);
  expect(projectIdMatch?.[1], '链接应包含 project id').toBeTruthy();
  const projectId = projectIdMatch[1];

  await projectLink.click();
  await page.waitForURL(new RegExp(`/tenant/${TENANT_ID}/projects/${projectId}/`), { timeout: 30000 });
  expect(page.url()).toContain(`/tenant/${TENANT_ID}/projects/${projectId}/`);
});
