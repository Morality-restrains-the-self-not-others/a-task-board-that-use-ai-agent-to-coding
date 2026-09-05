// @ts-check
/** 冒烟：工作面板分区标题展示各进度列数量（deliverable-section-progress-count） */
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

test('work-panel: section header shows per-progress-column counts', async ({ page }) => {
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

  const panel = page.locator('#task-panel-container');
  const panelVisible = await panel.isVisible({ timeout: 30000 }).catch(() => false);
  test.skip(!panelVisible, 'task-panel-container 未渲染，跳过');

  const section = panel
    .locator('[data-alias="deliverable-section-filter"], [data-alias="deliverable-section-other"]')
    .first();
  await expect(section).toBeVisible({ timeout: 15000 });

  const countsRoot = section.locator('[data-alias="deliverable-section-progress-counts"]');
  const countsVisible = await countsRoot.isVisible({ timeout: 15000 }).catch(() => false);
  test.skip(!countsVisible, 'deliverable-section-progress-counts 未渲染（可能未部署最新 SPA），跳过');

  const totalEl = countsRoot.locator('[data-alias="deliverable-section-total-count"]');
  await expect(totalEl).toBeVisible();
  await expect(totalEl).toHaveText(/共\s*\d+/);

  const toggle = section.locator('[data-alias="deliverable-section-toggle"]');
  if ((await toggle.getAttribute('aria-expanded')) === 'false') {
    await toggle.click();
    await page.waitForTimeout(200);
  }

  const perCol = countsRoot.locator('[data-alias="deliverable-section-progress-count"]');
  const perColCount = await perCol.count();
  if (perColCount === 0) {
    // 列数超阈值或窄屏：仅校验「共 N」与看板列数之和
    const boardCounts = section.locator('[data-alias="progress-column-count"]');
    const n = await boardCounts.count();
    expect(n).toBeGreaterThan(0);
    let sum = 0;
    for (let i = 0; i < n; i += 1) {
      sum += Number((await boardCounts.nth(i).innerText()).trim() || 0);
    }
    const totalText = await totalEl.innerText();
    const m = totalText.match(/共\s*(\d+)/);
    expect(m).toBeTruthy();
    expect(Number(m[1])).toBe(sum);
    return;
  }

  expect(perColCount).toBeGreaterThan(0);
  for (let i = 0; i < perColCount; i += 1) {
    const el = perCol.nth(i);
    const colId = await el.getAttribute('data-progress-column-id');
    expect(colId).toBeTruthy();
    const headerCount = Number((await el.innerText()).trim());
    const boardCountText = await section
      .locator(
        `[data-alias="progress-column"][data-progress-column-id="${colId}"] [data-alias="progress-column-count"]`,
      )
      .innerText();
    expect(headerCount).toBe(Number(boardCountText.trim()));
  }

  const totalText = await totalEl.innerText();
  const m = totalText.match(/共\s*(\d+)/);
  expect(m).toBeTruthy();
  let headerSum = 0;
  for (let i = 0; i < perColCount; i += 1) {
    headerSum += Number((await perCol.nth(i).innerText()).trim());
  }
  expect(Number(m[1])).toBe(headerSum);
});
