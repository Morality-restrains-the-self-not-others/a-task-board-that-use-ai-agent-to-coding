// @ts-check
/**
 * 核验：工作面板任务卡片修改「进度状态」后，卡片移入目标进度纵轴。
 * 纵轴=进度列，每列仅一个 .progress-lane。
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const TENANT_ID =
  process.env.PLAYWRIGHT_TENANT_ID ||
  process.env.PW_TENANT_ID ||
  '850256677331562496';
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || '';
const SITE_ORIGIN = (process.env.PLAYWRIGHT_SITE_ORIGIN || '').trim().replace(/\/+$/, '');

function loginOpts(email, password) {
  const creds = { email, password };
  if (SITE_ORIGIN) creds.baseURL = SITE_ORIGIN;
  return creds;
}

function workPanelPath() {
  const base = `/tenant/${TENANT_ID}/work-panel/`;
  if (WORKSPACE_ID && WORKSPACE_ID !== PW_WORKSPACE_ID) {
    return `${base}?workspace_id=${WORKSPACE_ID}`;
  }
  if (WORKSPACE_ID && String(TENANT_ID) === '827923618468040704') {
    return `${base}?workspace_id=${WORKSPACE_ID}`;
  }
  return base;
}

test.describe('工作面板 进度状态下拉换纵轴', () => {
  test('每个进度纵轴仅一 lane，且改进度后卡片进入目标纵轴', async ({ page }) => {
    test.setTimeout(120000);

    try {
      await playwrightLoginWithLegalAccept(page, loginOpts(EMAIL, PASSWORD));
      await page.setViewportSize({ width: 2560, height: 1080 });
      await page.goto(workPanelPath());
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      test.skip(
        /ERR_PROXY|ECONNREFUSED|ENOTFOUND|net::ERR_/i.test(msg),
        `E2E 环境不可达，跳过: ${msg}`,
      );
      throw err;
    }
    await page.waitForLoadState('networkidle').catch(() => {});
    await page.waitForTimeout(1500);

    const progressColumns = page.locator('[data-alias="progress-column"]');
    await expect(progressColumns.first()).toBeVisible({ timeout: 30000 });
    const colCount = await progressColumns.count();
    expect(colCount).toBeGreaterThanOrEqual(2);

    for (let i = 0; i < colCount; i++) {
      await expect(progressColumns.nth(i).locator('.progress-lane')).toHaveCount(1);
    }

    /** @type {import('@playwright/test').Locator | null} */
    let sourceCol = null;
    /** @type {import('@playwright/test').Locator | null} */
    let card = null;
    /** @type {import('@playwright/test').Locator | null} */
    let targetCol = null;
    /** @type {string | null} */
    let targetProgressId = null;

    for (let i = 0; i < colCount; i++) {
      const col = progressColumns.nth(i);
      const cards = col.locator('.task-card');
      if ((await cards.count()) === 0) continue;
      const otherIdx = i === 0 ? 1 : 0;
      sourceCol = col;
      card = cards.first();
      targetCol = progressColumns.nth(otherIdx);
      targetProgressId = await targetCol.getAttribute('data-progress-column-id');
      break;
    }

    if (!card || !targetCol || !targetProgressId || !sourceCol) {
      test.skip(true, '需要至少两个进度纵轴且其一有任务卡片');
      return;
    }

    const taskId = await card.getAttribute('data-task-id');
    expect(taskId).toBeTruthy();
    const sourceProgressId = await sourceCol.getAttribute('data-progress-column-id');
    expect(String(sourceProgressId)).not.toBe(String(targetProgressId));

    const select = card.locator('select.task-progress-select');
    await expect(select).toBeVisible({ timeout: 10000 });

    const patchWithProgress = page.waitForRequest(
      (req) => {
        if (req.method() !== 'PATCH') return false;
        if (!req.url().includes(`/todos/${taskId}`)) return false;
        const data = req.postData();
        if (!data) return false;
        try {
          const body = JSON.parse(data);
          return String(body.progress_column_id) === String(targetProgressId);
        } catch {
          return false;
        }
      },
      { timeout: 60000 },
    );

    await select.selectOption(String(targetProgressId));
    await patchWithProgress;

    await expect
      .poll(
        async () => {
          return page
            .locator(
              `[data-alias="progress-column"][data-progress-column-id="${targetProgressId}"] .task-card[data-task-id="${taskId}"]`,
            )
            .count();
        },
        { timeout: 30000 },
      )
      .toBe(1);

    await expect(
      page.locator(
        `[data-alias="progress-column"][data-progress-column-id="${sourceProgressId}"] .task-card[data-task-id="${taskId}"]`,
      ),
    ).toHaveCount(0);
  });
});
