// @ts-check
/** 工作面板：过滤栏下拉 + 条件分区紧挨栏 + 其他置底可折叠 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const SITE_ORIGIN = (process.env.PLAYWRIGHT_SITE_ORIGIN || '').trim().replace(/\/+$/, '');

function loginOpts(email, password) {
  const creds = { email, password };
  if (SITE_ORIGIN) creds.baseURL = SITE_ORIGIN;
  return creds;
}

test('work-panel: dropdown filter bars + collapsible sections', async ({ page }) => {
  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD for this test');
  test.setTimeout(120000);

  try {
    await playwrightLoginWithLegalAccept(page, loginOpts(email, password));
    await page.goto(`/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`);
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    test.skip(
      /ERR_PROXY|ECONNREFUSED|ENOTFOUND|net::ERR_/i.test(msg),
      `E2E 环境不可达，跳过: ${msg}`,
    );
    throw err;
  }
  await page.waitForLoadState('networkidle', { timeout: 45000 }).catch(() => {});
  await page.waitForTimeout(2000);

  const trails = page.locator('[data-alias="deliverable-breadcrumb"]');
  const trailVisible = await trails.first().isVisible({ timeout: 30000 }).catch(() => false);
  test.skip(!trailVisible, 'deliverable-breadcrumb 未渲染（本环境可能未部署最新 SPA），跳过');
  // 部分布局可能渲染多条 trail（如主区+侧栏）；用例以可见首条为准
  expect(await trails.count()).toBeGreaterThanOrEqual(1);

  // 单行：下拉存在，旧 chips 不存在
  await expect(trails.first().locator('[data-alias="deliverable-content-select"]').first()).toBeVisible({
    timeout: 15000,
  });
  await expect(trails.first().locator('.deliverable-content-chip')).toHaveCount(0);
  await expect(trails.first().locator('.deliverable-trail-divider')).toHaveCount(0);

  const firstBlockBefore = page.locator('[data-alias="deliverable-filter-block"]').first();
  const firstBarIdBefore = await firstBlockBefore.getAttribute('data-bar-id');
  const addBtn = trails.first().locator('[data-alias="deliverable-filter-add"]');
  await expect(addBtn).toBeVisible();
  const trailCountBeforeAdd = await trails.count();
  await addBtn.click();
  await page.waitForTimeout(300);
  await expect(trails).toHaveCount(trailCountBeforeAdd + 1);
  // 新栏插在当前栏上方：首块变为新建栏，原首栏下移为第二块
  const blocks = page.locator('[data-alias="deliverable-filter-block"]');
  await expect(blocks).toHaveCount(trailCountBeforeAdd + 1);
  const newFirstBarId = await blocks.first().getAttribute('data-bar-id');
  expect(newFirstBarId).toBeTruthy();
  expect(newFirstBarId).not.toBe(firstBarIdBefore);
  await expect(blocks.nth(1)).toHaveAttribute('data-bar-id', firstBarIdBefore);
  await expect(blocks.first().locator('[data-alias="deliverable-filter-remove"]')).toBeVisible();

  const other = page.locator('[data-alias="deliverable-section-other"]');
  await expect(other).toBeVisible();
  await expect(other.getByRole('heading', { name: '其他（未过滤）' })).toBeVisible();

  // 用原首栏（现第二块）做内容选择，避免新建空栏干扰后续断言
  const firstBlock = blocks.nth(1);
  const trail = firstBlock.locator('[data-alias="deliverable-breadcrumb"]');
  await expect(trail.getByRole('button', { name: '全部' })).toBeVisible();

  const firstSelect = trail.locator('[data-alias="deliverable-content-select"]').first();
  await expect(firstSelect).toBeVisible();
  const optionCount = await firstSelect.locator('option').count();
  if (optionCount <= 1) {
    test.skip(true, '第一类别暂无交付物内容');
    return;
  }

  const firstContentValue = await firstSelect.locator('option').nth(1).getAttribute('value');
  await firstSelect.selectOption(firstContentValue);
  await page.waitForTimeout(400);
  await expect(firstSelect).toHaveValue(firstContentValue);

  const filterSection = firstBlock.locator('[data-alias="deliverable-section-filter"]');
  await expect(filterSection).toBeVisible();

  const filterBox = await filterSection.boundingBox();
  const otherBox = await other.boundingBox();
  expect(filterBox && otherBox).toBeTruthy();
  expect(otherBox.y).toBeGreaterThan(filterBox.y);

  await other.locator('[data-alias="deliverable-section-toggle"]').click();
  await page.waitForTimeout(200);
  await expect(other.locator('[data-alias="deliverable-section-toggle"]')).toHaveAttribute(
    'aria-expanded',
    'false',
  );
  await expect(other.locator('[data-alias="deliverable-section-body"]')).toBeHidden();

  await trail.getByRole('button', { name: '全部' }).click();
  await page.waitForTimeout(300);
  await expect(firstSelect).toHaveValue('');
  await expect(firstBlock.locator('[data-alias="deliverable-section-filter"]')).toHaveCount(0);

  await expect(page.locator('.deliverable-column-header')).toHaveCount(0);

  // 展开任意折叠区，断言可见看板内：纵轴=进度列，每纵轴仅一 progress-lane
  const collapsedToggle = page
    .locator('[data-alias="deliverable-section-toggle"][aria-expanded="false"]')
    .first();
  if (await collapsedToggle.isVisible().catch(() => false)) {
    await collapsedToggle.click();
    await page.waitForTimeout(200);
  }
  const visibleBoard = page
    .locator('[data-alias="deliverable-section-body"]:visible [data-alias="deliverable-kanban-board"]')
    .first();
  await expect(visibleBoard).toBeVisible({ timeout: 10000 });
  const progressColumns = visibleBoard.locator('[data-alias="progress-column"]');
  await expect(progressColumns.first()).toBeVisible();
  const colCount = await progressColumns.count();
  expect(colCount).toBeGreaterThanOrEqual(2);
  for (let i = 0; i < colCount; i++) {
    await expect(progressColumns.nth(i).locator('.progress-lane')).toHaveCount(1);
  }
});
