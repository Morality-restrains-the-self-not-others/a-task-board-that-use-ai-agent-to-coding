// @ts-check
/**
 * 回归：WorkPanel 看板卡片 task-card-id 编号展示与布局。
 * - 每个卡片展示人读 #序号（workspace_seq），无序号时不渲染徽章
 * - 编号与标题同行且 DOM 顺序在前
 * - 点击编号复制到剪贴板
 *
 * 纯 mock，无真实账号依赖。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || '827923618602258432';
const USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488';

const TYPE_ID = '2000000000000000201';
const TASK_ID = '830423831930662912';
const WORKSPACE_SEQ = 7;
const EXPECTED_DISPLAY_NO = '#7';

let cachedIndexHtml = '';

test.describe('WorkPanel 看板卡片编号', () => {
  async function setupWorkPanelMocks(page) {
    await page.context().addCookies([
      { name: 'userId', value: USER_ID, url: BASE_URL },
      { name: 'sessionid', value: 'e2e-session-card-id', url: BASE_URL },
      { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
    ]);

    if (!cachedIndexHtml) {
      const resp = await page.request.get(`${BASE_URL}/`);
      cachedIndexHtml = await resp.text();
    }

    // 绕过服务端 forward-auth 302：/tenant/ 文档导航直接返回 SPA 入口（与 companySwitcher 一致）
    await page.route('**/tenant/**', async (route) => {
      if (route.request().resourceType() === 'document') {
        await route.fulfill({ status: 200, contentType: 'text/html', body: cachedIndexHtml });
      } else {
        await route.continue();
      }
    });

    // 兜底 mock 全部 /api/：未显式覆盖的业务 API 以空对象快速响应，避免真实网关 401 触发跳登录
    await page.route('**/api/**', async (r) => {
      await r.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    // 用户信息
    await page.route(`**/api/accounts/users/me/**`, async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({
          id: USER_ID, username: 'e2e',
          current_company: { id: TENANT_ID, name: 'E2E Tenant' },
          companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
          current_workspace: { id: WORKSPACE_ID, name: 'E2E WS' },
        }),
      });
    });

    // 任务列表
    await page.route(`**/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}**`, async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify([{
          id: TASK_ID, title: 'Test Task Card ID',
          description: 'card id badge test', priority: 1,
          progress_column_id: 1,
          workspace_seq: WORKSPACE_SEQ,
          task_type: { id: TYPE_ID, name: 'Feature' },
          created_by: { username: 'e2e' },
        }]),
      });
    });

    // 进度系统
    await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/**`, async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({
          columns: [
            { id: 0, name: '待处理', order: 0 },
            { id: 1, name: '进行中', order: 1 },
            { id: 2, name: '已完成', order: 2 },
          ],
        }),
      });
    });

    // 交付系统
    await page.route(`**/manage-deliverable-system/**`, async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({
          current_deliverable_objs: [{ id: TYPE_ID, name: 'Feature', order: 0 }],
        }),
      });
    });

    // 工作区列表
    await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}**`, async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200, contentType: 'application/json',
          body: JSON.stringify([{ id: WORKSPACE_ID, name: 'E2E WS', is_current: true, is_default: true }]),
        });
        return;
      }
      await route.continue();
    });

    // 运行时指标（空）
    await page.route('**/cloud/compute/workspace-runtime-indicators/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', indicators: [] }) });
    });

    // 机器摘要（空）
    await page.route('**/cloud/compute/workspace-machine-summary/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', started_count: 0, idle_count: 0, busy_count: 0 }) });
    });

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);
  }

  test('看板卡片展示 #序号', async ({ page }) => {
    await setupWorkPanelMocks(page);

    const badge = page.getByTestId('task-card-id');
    await expect(badge.first()).toBeVisible({ timeout: 45000 });
    await expect(badge.first()).toContainText(EXPECTED_DISPLAY_NO);
  });

  test('编号在卡片内且 DOM 顺序在标题之前', async ({ page }) => {
    await setupWorkPanelMocks(page);

    const badge = page.getByTestId('task-card-id').first();
    await expect(badge).toBeVisible({ timeout: 45000 });

    // 编号在元信息行内，标题在其后的行；断言 DOM 顺序：badge 先于标题 h4
    const card = page.locator('[data-task-id]').first();
    const h4 = card.locator('h4[data-testid="task-card-title"]').first();
    await expect(h4).toBeVisible();

    const order = await card.evaluate((el) => {
      const badgeEl = el.querySelector('[data-testid="task-card-id"]');
      const h4El = el.querySelector('h4[data-testid="task-card-title"]');
      if (!badgeEl || !h4El) return 0;
      return badgeEl.compareDocumentPosition(h4El) & Node.DOCUMENT_POSITION_FOLLOWING ? 1 : -1;
    });
    expect(order).toBe(1);
  });

  test('点击编号复制到剪贴板', async ({ page }) => {
    // 须在 goto 前注入，否则 addInitScript 对已加载页无效
    await page.addInitScript(() => {
      window.__copiedTexts = [];
      Object.defineProperty(navigator, 'clipboard', {
        value: { writeText: async (text) => { window.__copiedTexts.push(text); } },
        configurable: true,
      });
    });
    await setupWorkPanelMocks(page);

    const badge = page.getByTestId('task-card-id').first();
    await expect(badge).toBeVisible({ timeout: 45000 });
    await badge.click();

    await expect.poll(async () => page.evaluate(() => window.__copiedTexts)).toContain(EXPECTED_DISPLAY_NO);
  });
});
