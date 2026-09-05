// @ts-check
/**
 * 工作面板「其他」看板撑满视口剩余高度，横向滚动条贴浏览器底部。
 * Mock 驱动，无需真实凭据。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID =
  process.env.PW_WORKSPACE_ID ||
  process.env.PLAYWRIGHT_WORKSPACE_ID ||
  PW_WORKSPACE_ID;
const USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488';

async function setupMocks(page) {
  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-kanban-fill', url: BASE_URL },
    { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
  ]);

  // 宽匹配：避免遗漏接口导致看板不渲染（test.skip）
  await page.route('**/api/**', async (r) => {
    const u = r.request().url();
    if (u.includes('/users/me')) {
      await r.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: USER_ID,
          username: 'e2e',
          current_company: { id: TENANT_ID, name: 'E2E' },
          companies: [{ id: TENANT_ID, name: 'E2E' }],
          current_workspace: { id: WORKSPACE_ID, name: 'E2E WS' },
        }),
      });
      return;
    }
    if (u.includes('progress-system')) {
      await r.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          columns: [
            { id: '0', name: '待处理', order: 0 },
            { id: '1', name: '进行中', order: 1 },
            { id: '2', name: '已完成', order: 2 },
          ],
        }),
      });
      return;
    }
    if (u.includes('manage-deliverable-system')) {
      await r.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ current_deliverable_objs: [{ id: '2001', name: 'Feature', order: 0 }] }),
      });
      return;
    }
    if (u.includes('work-panel-filters')) {
      await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ bars: [] }) });
      return;
    }
    if (u.includes('/todos/')) {
      if (r.request().method() !== 'GET') {
        await r.continue();
        return;
      }
      await r.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: '830423831930662901',
            title: '任务 A',
            description: '',
            priority: 1,
            progress_column_id: '0',
            task_type: { id: '2001', name: 'Feature' },
            created_by: { username: 'e2e' },
          },
        ]),
      });
      return;
    }
    if (u.includes('/workspaces/')) {
      await r.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: WORKSPACE_ID, name: 'E2E WS', is_current: true, is_default: true }]),
      });
      return;
    }
    if (u.includes('runtime-indicators') || u.includes('machine-summary')) {
      await r.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', indicators: [], started_count: 0, idle_count: 0, busy_count: 0 }),
      });
      return;
    }
    await r.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
  });
}

test.describe('WorkPanel kanban fill viewport', () => {
  test('#task-panel-board-other 底边贴近视口底部', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await setupMocks(page);
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`);
    await page.waitForLoadState('domcontentloaded');

    const board = page.locator('#task-panel-board-other');
    await expect(board).toBeVisible({ timeout: 20000 });

    const section = page.locator('[data-alias="deliverable-section-other"]');
    const toggle = section.locator('[data-alias="deliverable-section-toggle"]');
    if ((await toggle.getAttribute('aria-expanded')) === 'false') {
      await toggle.click();
      await page.waitForTimeout(200);
    }

    const metrics = await page.evaluate(() => {
      const el = document.querySelector('#task-panel-board-other');
      if (!el) return null;
      const rect = el.getBoundingClientRect();
      const col = el.querySelector('[data-alias="progress-column"]');
      const colRect = col ? col.getBoundingClientRect() : null;
      return {
        boardHeight: rect.height,
        gapToViewportBottom: window.innerHeight - rect.bottom,
        colHeight: colRect ? colRect.height : 0,
      };
    });

    expect(metrics).toBeTruthy();
    expect(metrics.boardHeight).toBeGreaterThan(360);
    expect(metrics.gapToViewportBottom).toBeGreaterThanOrEqual(0);
    expect(metrics.gapToViewportBottom).toBeLessThanOrEqual(16);
    expect(metrics.colHeight).toBeGreaterThan(300);
    expect(metrics.colHeight).toBeGreaterThanOrEqual(metrics.boardHeight - 24);
  });
});
