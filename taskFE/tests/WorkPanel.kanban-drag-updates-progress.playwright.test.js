// @ts-check
/**
 * 核验：工作面板看板跨进度纵轴拖拽时，PATCH 携带 progress_column_id。
 * 纵轴=进度列，每列仅一个进度。
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const TENANT_ID =
  process.env.PLAYWRIGHT_TENANT_ID ||
  process.env.PW_TENANT_ID ||
  '850256677331562496';
const SITE_ORIGIN = (process.env.PLAYWRIGHT_SITE_ORIGIN || '').trim().replace(/\/+$/, '');

function loginOpts(email, password) {
  const creds = { email, password };
  if (SITE_ORIGIN) creds.baseURL = SITE_ORIGIN;
  return creds;
}

test.describe('工作面板 看板拖拽更新进度列', () => {
  test('拖拽后 PATCH 含 progress_column_id，且每纵轴仅一 progress-lane', async ({ page }) => {
    test.setTimeout(120000);

    await playwrightLoginWithLegalAccept(page, loginOpts(EMAIL, PASSWORD));
    await page.setViewportSize({ width: 2560, height: 1080 });
    await page.goto(`/tenant/${TENANT_ID}/work-panel/`);
    await page.waitForLoadState('networkidle').catch(() => {});
    await page.waitForTimeout(1500);

    const progressColumns = page.locator('[data-alias="progress-column"]');
    await expect(progressColumns.first()).toBeVisible({ timeout: 30000 });
    const colCount = await progressColumns.count();
    expect(colCount).toBeGreaterThanOrEqual(2);
    for (let i = 0; i < Math.min(colCount, 4); i++) {
      await expect(progressColumns.nth(i).locator('.progress-lane')).toHaveCount(1);
    }

    let sourceColIdx = -1;
    for (let i = 0; i < colCount; i++) {
      if ((await progressColumns.nth(i).locator('.task-card').count()) > 0) {
        sourceColIdx = i;
        break;
      }
    }
    if (sourceColIdx < 0) {
      test.skip(true, '无任务卡片可拖拽');
      return;
    }
    const targetColIdx = sourceColIdx === 0 ? 1 : 0;

    const firstCard = progressColumns.nth(sourceColIdx).locator('.task-card').first();
    const targetContainer = progressColumns
      .nth(targetColIdx)
      .locator('.task-cards-container');

    await firstCard.scrollIntoViewIfNeeded();
    await targetContainer.scrollIntoViewIfNeeded();

    const patchWithProgress = page.waitForRequest(
      (req) => {
        if (req.method() !== 'PATCH') return false;
        if (!req.url().includes('/todos/')) return false;
        const data = req.postData();
        if (!data) return false;
        try {
          const body = JSON.parse(data);
          return Object.prototype.hasOwnProperty.call(body, 'progress_column_id');
        } catch {
          return false;
        }
      },
      { timeout: 60000 },
    );

    // SortableJS 对逐步 mouse move 不稳定；优先 locator.dragTo，失败再走坐标拖拽
    try {
      await firstCard.dragTo(targetContainer, { force: true });
    } catch {
      const fromBox = await firstCard.boundingBox();
      const toBox = await targetContainer.boundingBox();
      if (!fromBox || !toBox) {
        throw new Error('无法获取拖拽坐标');
      }
      await page.mouse.move(fromBox.x + fromBox.width / 2, fromBox.y + fromBox.height / 2);
      await page.mouse.down();
      await page.mouse.move(
        Math.max(8, toBox.x + Math.min(toBox.width / 2, 120)),
        Math.max(8, toBox.y + Math.min(toBox.height / 2, 80)),
        { steps: 24 },
      );
      await page.mouse.up();
    }

    const req = await patchWithProgress;
    expect(req.url()).toContain(`/api/tenant/${TENANT_ID}`);
    expect(req.url()).toMatch(/\/todos\/[^/]+\/?$/);
    const body = JSON.parse(req.postData() || '{}');
    expect(body).toHaveProperty('progress_column_id');
  });
});
