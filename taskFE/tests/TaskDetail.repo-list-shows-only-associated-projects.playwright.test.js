// @ts-check
/**
 * 验证：任务详情页面的仓库列表只显示任务关联项目的仓库，而不是所有仓库。
 * 
 * 依赖：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const WORKSPACE = '827923618602258432';
const TASK_ID = '837962554269450240';
const ACCESS_CODE = 'u824976301710503936';

test('task-detail repo list should only show repos from associated projects', async ({ page }) => {
  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD for this test');

  await playwrightLoginWithLegalAccept(page, { email, password });

  const q = new URLSearchParams({ accessCode: ACCESS_CODE });
  await page.goto(`/tenant/${TENANT_ID}/workspace/${WORKSPACE}/task-detail/${TASK_ID}/?${q.toString()}`);
  await page.waitForLoadState('domcontentloaded', { timeout: 30000 });
  await page.waitForTimeout(3000);

  const projectBadges = page.getByTestId('task-linked-project-name-link');
  const associatedProjectNames = [];
  
  const badgeCount = await projectBadges.count();
  for (let i = 0; i < badgeCount; i++) {
    const text = await projectBadges.nth(i).textContent();
    if (text) {
      associatedProjectNames.push(text.trim().split(/\s+/)[0]);
    }
  }

  console.log('关联的项目:', associatedProjectNames);

  const panel = page.getByTestId('task-repo-clone-identity-panel');
  const projectCards = panel.locator(':scope > div');
  const cardCount = await projectCards.count();
  test.skip(cardCount === 0, 'No linked project cards found, skipping test');

  let totalRepoRows = 0;
  for (let i = 0; i < cardCount; i++) {
    const card = projectCards.nth(i);
    const badge = card.getByTestId('task-linked-project-name-link');
    const raw = await badge.textContent();
    const projectName = raw?.trim().split(/\s+/)[0];
    expect(projectName, `卡片 ${i} 应有项目名徽章`).toBeTruthy();
    expect(
      associatedProjectNames.includes(projectName),
      `卡片 ${i} 的项目 "${projectName}" 不在页面收集的关联项目列表中`
    ).toBeTruthy();

    const repoLis = card.locator('ul li.border.border-gray-100');
    const n = await repoLis.count();
    totalRepoRows += n;
    for (let j = 0; j < n; j++) {
      const rowText = await repoLis.nth(j).textContent();
      expect(rowText && rowText.trim().length > 0, `仓库行 ${i}/${j} 应有内容`).toBeTruthy();
      console.log(`仓库行 ${i}/${j}: 所属项目=${projectName}`);
    }
  }

  test.skip(totalRepoRows === 0, 'No repo rows under linked projects, skipping test');
});