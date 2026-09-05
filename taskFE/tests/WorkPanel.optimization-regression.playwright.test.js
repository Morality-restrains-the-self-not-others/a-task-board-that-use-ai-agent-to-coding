// @ts-check
/**
 * WorkPanel 优化回归测试集：OPT-014（deliverable-section-count）、
 * OPT-021（SSE task_created/deleted resync）、OPT-042（Fork 自动出卡）。
 * 全部基于 mock，无需真实后端或凭据。
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

// ====== Mock 数据 ======

const PROGRESS_COLUMNS = [
  { id: '0', name: '待处理', order: 0 },
  { id: '1', name: '进行中', order: 1 },
  { id: '2', name: '已完成', order: 2 },
];
const DELIVERABLE_TYPES = {
  current_deliverable_objs: [{ id: '2001', name: 'Feature', order: 0 }],
};

// 工具函数：构造 mock 任务对象
const T = (id, title, priority, colId) => ({
  id, title, description: '', priority, progress_column_id: colId,
  task_type: { id: '2001', name: 'Feature' }, created_by: { username: 'e2e' },
});

const TA = T('830423831930662901', '任务 A', 1, '0');
const TB = T('830423831930662902', '任务 B', 1, '0');
const TC = T('830423831930662903', '任务 C', 2, '1');
const TD = T('830423831930662904', '任务 D', 1, '0');
const TF1 = T('830423831930662905', 'Fork 任务 1', 1, '0');
const TF2 = T('830423831930662906', 'Fork 任务 2', 1, '1');

// OPT-014: 列0=2张, 列1=1张, 列2=0张
const INITIAL_TODOS = [TA, TB, TC];
const AFTER_CREATE_TODOS = [...INITIAL_TODOS, TD];
const AFTER_DELETE_TODOS = [TB, TC];
const FORK_INITIAL = [TA];
const FORK_AFTER_1 = [...FORK_INITIAL, TF1];
const FORK_AFTER_2 = [...FORK_AFTER_1, TF2];

// ====== Helpers ======

async function setupCommonMocks(page) {
  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-wp-opt', url: BASE_URL },
    { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
  ]);

  // 兜底 catch-all 必须最先注册：Playwright route 后注册先匹配（LIFO），
  // 若放在最后会抢占 /me/、workspaces 等所有具体 mock（OPT-20260824-068 实测验证），
  // 导致 tenantRouteGuard 拿到空 companies 跳 /onboarding/。
  // 未逐一 mock 的 API 一律返回 200 空对象，避免真实后端 401 → requestErrorDisplay 整页跳登录页。
  await page.route('**/api/**', async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  // 权限码（Sidebar usePermissions.load 请求；缺 tenant_perms 时主导航三项全隐藏，nav 空判 hidden）
  await page.route(`**/api/auth/user-permissions/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json',
      body: JSON.stringify({
        tenant_perms: {
          [TENANT_ID]: ['project:view', 'project:manage', 'task:view', 'task:manage', 'cloud:view', 'cloud:manage'],
        },
      }),
    });
  });

  await page.route(`**/api/accounts/users/me/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json',
      body: JSON.stringify({
        id: USER_ID, username: 'e2e',
        current_company: { id: TENANT_ID, name: 'E2E' },
        companies: [{ id: TENANT_ID, name: 'E2E' }],
        current_workspace: { id: WORKSPACE_ID, name: 'E2E WS' },
      }),
    });
  });

  // 工作区列表（先注册、后匹配，具体子路径路由优先）
  await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json',
      body: JSON.stringify([{ id: WORKSPACE_ID, name: 'E2E WS', is_current: true, is_default: true }]),
    });
  });

  await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: PROGRESS_COLUMNS }) });
  });

  await page.route(`**/manage-deliverable-system/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(DELIVERABLE_TYPES) });
  });

  await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/work-panel-filters/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ bars: [] }) });
  });

  await page.route('**/cloud/compute/workspace-runtime-indicators/**', async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', indicators: [] }) });
  });

  await page.route('**/cloud/compute/workspace-machine-summary/**', async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', started_count: 0, idle_count: 0, busy_count: 0 }) });
  });

  await page.route(`**/api/projects/tenant_id/${TENANT_ID}**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
  });

  await page.route(`**/api/cloud/installed-images/tenant_id/${TENANT_ID}**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
  });

  await page.route(`**/projects/workspace-access/workspace-collaborators/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
  });
}

/**
 * 注册 todos mock，支持分阶段返回数据集。单参数=始终返回相同数据；多参数=显式 advance。
 *
 * 注意：不再按「每次 fetch 自动前进」——WorkPanel 初始加载会触发多次 fetchTodos
 * （挂载 + 工作空间切换 resync 等），按调用次数前进会让初始断言取到下一个数据集
 * （OPT-20260824-068 实测：task_deleted 初始 3 卡取到 2、Fork 初始 1 卡取到 2）。
 * 改为调用方在注入 SSE 前显式 `advance()` 切换数据集，保证初始断言确定。
 * @param {import('@playwright/test').Page} page
 * @param  {...Array<object>} datasets
 * @returns {{ advance: () => void, current: () => Array<object> }}
 */
async function setupTodosMock(page, ...datasets) {
  const items = datasets.length === 0 ? [[]] : datasets;
  let idx = 0;
  const state = {
    advance: () => { idx = Math.min(idx + 1, items.length - 1); },
    current: () => items[idx],
  };
  await page.route(`**/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}**`, async (r) => {
    if (r.request().method() !== 'GET') { await r.continue(); return; }
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(items[idx]) });
  });
  return state;
}

/**
 * 注入 EventSource mock，暴露 window.__emitWorkPanelSse 用于注入 SSE 帧。
 */
async function setupEventSourceMock(page) {
  await page.addInitScript(
    ({ tenantId, workspaceId }) => {
      let _latestInstance = null;

      class MockWorkPanelEventSource {
        static CONNECTING = 0;
        static OPEN = 1;
        static CLOSED = 2;

        constructor(url) {
          this.url = String(url || '');
          this.readyState = MockWorkPanelEventSource.OPEN;
          this.withCredentials = true;
          this.onmessage = null;
          this.onerror = null;
          this.onclose = null;
          this._listeners = { open: [] };
          this._closed = false;
          _latestInstance = this;
          setTimeout(() => {
            if (typeof this.onopen === 'function') this.onopen({ type: 'open' });
            for (const cb of this._listeners.open || []) { try { cb({ type: 'open' }); } catch (_) {} }
          }, 30);
        }

        addEventListener(type, cb) {
          if (!this._listeners[type]) this._listeners[type] = [];
          this._listeners[type].push(cb);
        }

        close() {
          this.readyState = MockWorkPanelEventSource.CLOSED;
          this._closed = true;
          if (typeof this.onclose === 'function') this.onclose();
        }
      }

      window.EventSource = MockWorkPanelEventSource;

      window.__emitWorkPanelSse = (payload) => {
        const inst = _latestInstance;
        if (!inst || inst._closed) return;
        if (!inst.url.includes(`/api/sse/work-panel-events/tenant_id/${tenantId}/workspace_id/${workspaceId}`)) return;
        if (typeof inst.onmessage === 'function') inst.onmessage({ data: JSON.stringify(payload) });
      };
    },
    { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID },
  );
}

// ====== Tests ======

test.describe('OPT-20260719-014: deliverable-section-count 列计数一致性', () => {
  test('分区标题各列计数与看板 progress-column-count 一致', async ({ page }) => {
    await setupCommonMocks(page);
    await setupTodosMock(page, INITIAL_TODOS);
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    const section = page.locator('[data-alias="deliverable-section-other"]');
    await expect(section).toBeVisible({ timeout: 15000 });

    const toggle = section.locator('[data-alias="deliverable-section-toggle"]');
    if ((await toggle.getAttribute('aria-expanded')) === 'false') {
      await toggle.click();
      await page.waitForTimeout(300);
    }

    const countContainer = section.locator('[data-alias="deliverable-section-count"]');
    await expect(countContainer).toBeVisible({ timeout: 5000 });

    const colCountItems = countContainer.locator('[data-alias="deliverable-section-column-count"]');
    const itemCount = await colCountItems.count();
    expect(itemCount).toBeGreaterThanOrEqual(2);

    // 逐列比较标题计数与看板内 progress-column-count
    for (let i = 0; i < itemCount; i++) {
      const item = colCountItems.nth(i);
      const colId = await item.getAttribute('data-progress-column-id');
      expect(colId).toBeTruthy();
      const headerCount = Number((await item.innerText()).trim());
      const boardCountText = await section
        .locator(`[data-alias="progress-column"][data-progress-column-id="${colId}"] [data-alias="progress-column-count"]`)
        .innerText();
      expect(headerCount).toBe(Number(boardCountText.trim()));
    }

    // 看板列数 >= 标题列数（空列可能无卡但标题仍展示）
    expect(await section.locator('[data-alias="progress-column"]').count()).toBeGreaterThanOrEqual(itemCount);
  });
});

test.describe('OPT-20260720-021: SSE task_created/task_deleted resync', () => {
  test('task_created 后看板自动出现新卡', async ({ page }) => {
    await setupEventSourceMock(page);
    await setupCommonMocks(page);
    const todosMock = await setupTodosMock(page, INITIAL_TODOS, AFTER_CREATE_TODOS);
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    await expect(page.locator('#task-panel-container')).toBeVisible({ timeout: 15000 }).catch(() => {});

    const cardLocator = page.locator('[data-task-id]');
    await expect(cardLocator.first()).toBeVisible({ timeout: 15000 });
    expect(await cardLocator.count()).toBe(3);

    await page.waitForFunction(() => typeof window.__emitWorkPanelSse === 'function', null, { timeout: 10000 });

    // 注入 task_created → onNeedResync → fetchTodos → 4 张卡
    todosMock.advance();
    await page.evaluate(() => { window.__emitWorkPanelSse?.({ event_name: 'task_created' }); });

    await expect.poll(async () => page.locator('[data-task-id]').count(), {
      timeout: 15000, message: 'task_created SSE 后看板应出现 4 张卡片',
    }).toBe(4);
  });

  test('task_deleted 后看板自动移除对应卡', async ({ page }) => {
    await setupEventSourceMock(page);
    await setupCommonMocks(page);
    const todosMock = await setupTodosMock(page, INITIAL_TODOS, AFTER_DELETE_TODOS);
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    await expect(page.locator('#task-panel-container')).toBeVisible({ timeout: 15000 }).catch(() => {});

    const cardLocator = page.locator('[data-task-id]');
    await expect(cardLocator.first()).toBeVisible({ timeout: 15000 });
    expect(await cardLocator.count()).toBe(3);

    await page.waitForFunction(() => typeof window.__emitWorkPanelSse === 'function', null, { timeout: 10000 });

    // 注入 task_deleted → onNeedResync → fetchTodos → 2 张卡
    todosMock.advance();
    await page.evaluate(() => { window.__emitWorkPanelSse?.({ event_name: 'task_deleted' }); });

    await expect.poll(async () => page.locator('[data-task-id]').count(), {
      timeout: 15000, message: 'task_deleted SSE 后看板应只剩 2 张卡片',
    }).toBe(2);
  });
});

test.describe('OPT-20260720-042: Fork → 自动出卡（SSE 模式）', () => {
  test('模拟两次 Fork（task_created×2），看板逐步 1→2→3 张卡', async ({ page }) => {
    await setupEventSourceMock(page);
    await setupCommonMocks(page);
    const todosMock = await setupTodosMock(page, FORK_INITIAL, FORK_AFTER_1, FORK_AFTER_2);
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    await expect(page.locator('#task-panel-container')).toBeVisible({ timeout: 15000 }).catch(() => {});

    const cardLocator = page.locator('[data-task-id]');
    await expect(cardLocator.first()).toBeVisible({ timeout: 15000 });
    expect(await cardLocator.count()).toBe(1);

    await page.waitForFunction(() => typeof window.__emitWorkPanelSse === 'function', null, { timeout: 10000 });

    // Fork #1
    todosMock.advance();
    await page.evaluate(() => { window.__emitWorkPanelSse?.({ event_name: 'task_created' }); });
    await expect.poll(async () => page.locator('[data-task-id]').count(), {
      timeout: 15000, message: '第一次 Fork 后看板应出现 2 张卡片',
    }).toBe(2);
    await expect(page.locator('[data-task-id="830423831930662905"]')).toBeVisible({ timeout: 5000 });

    // Fork #2
    todosMock.advance();
    await page.evaluate(() => { window.__emitWorkPanelSse?.({ event_name: 'task_created' }); });
    await expect.poll(async () => page.locator('[data-task-id]').count(), {
      timeout: 15000, message: '第二次 Fork 后看板应出现 3 张卡片',
    }).toBe(3);
    await expect(page.locator('[data-task-id="830423831930662906"]')).toBeVisible({ timeout: 5000 });
  });
});
