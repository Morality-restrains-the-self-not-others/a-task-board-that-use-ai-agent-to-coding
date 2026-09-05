// @ts-check
/**
 * 核验：工作面板任务卡片修改「进度状态」下拉后，卡片自动移入目标进度泳道。
 *
 * 凭据：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 * 路径：PLAYWRIGHT_WORK_PANEL_PATH（可选）
 */
import { test, expect } from '@playwright/test';
import { submitLoginWithEmailPassword } from './helpers/e2eLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const WORK_PANEL_PATH =
  process.env.PLAYWRIGHT_WORK_PANEL_PATH ||
  `/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`;

async function login(page) {
  await page.goto('/auth/login/');
  await page.waitForLoadState('domcontentloaded');

  const emailPasswordTab = page.locator('text=邮箱').or(page.locator('button:has-text("邮箱")')).first();
  if (await emailPasswordTab.isVisible()) {
    await emailPasswordTab.click();
    await page.waitForTimeout(300);
  }

  await submitLoginWithEmailPassword(page, EMAIL, PASSWORD);
  await page.waitForLoadState('domcontentloaded');
  await page.waitForTimeout(2000);
}

test.describe('工作面板 进度下拉自动移卡', () => {
  test('修改进度状态后卡片进入目标 progress-lane', async ({ page }) => {
    test.setTimeout(120000);

    await login(page);
    await page.setViewportSize({ width: 2560, height: 1080 });
    await page.goto(WORK_PANEL_PATH);
    await page.waitForLoadState('networkidle').catch(() => {});
    await page.waitForTimeout(1500);

    const firstCol = page.locator('.deliverable-column').first();
    await expect(firstCol).toBeVisible({ timeout: 30000 });

    const lanes = firstCol.locator('.progress-lane');
    const laneCount = await lanes.count();
    if (laneCount < 2) {
      test.skip(true, '需要至少两个进度泳道');
      return;
    }

    const sourceLane = lanes.nth(0);
    const targetLane = lanes.nth(1);
    const sourceCard = sourceLane.locator('.task-card').first();
    if ((await sourceCard.count()) === 0) {
      test.skip(true, '第一进度泳道无任务卡片');
      return;
    }

    const taskId = await sourceCard.getAttribute('data-task-id');
    expect(taskId).toBeTruthy();

    const targetProgressId = await targetLane.getAttribute('data-progress-column-id');
    expect(targetProgressId).toBeTruthy();

    const select = sourceCard.locator('select.task-progress-select');
    await expect(select).toBeVisible({ timeout: 10000 });

    const patchReq = page.waitForRequest(
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
    await patchReq;

    await expect(
      targetLane.locator(`.task-card[data-task-id="${taskId}"]`),
    ).toBeVisible({ timeout: 20000 });
    await expect(sourceLane.locator(`.task-card[data-task-id="${taskId}"]`)).toHaveCount(0);
  });
});
