// @ts-check
/**
 * WorkPanel 头部「自动调度安排」入口（OPT-20260824-067）：
 * 「工作空间的排队调度」入口自租户控制台侧栏移至工作面板头部，
 * 位于任务搜索框与工作空间选择器之间，命名为「自动调度安排」。
 *
 * 覆盖：
 * - 头部 header-toolbar-row 中 auto-schedule-link 可见且文案为「自动调度安排」
 * - 链接 href 指向 /tenant/:tenant/queue-schedule/ 且携带当前 workspace_id
 * - 位置约束：链接位于搜索框（work-panel-task-search）与工作空间选择器
 *   （header-workspace-switcher-row）之间（同一行，x 坐标递增）；标题区与摘要区 y 中心接近
 * - 侧栏（aside.tenant-console-sidebar）不再渲染「工作空间的排队调度」导航项
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

const PROGRESS_COLUMNS = [
  { id: '0', name: '待处理', order: 0 },
  { id: '1', name: '进行中', order: 1 },
  { id: '2', name: '已完成', order: 2 },
];

async function setupMocks(page) {
  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-auto-schedule', url: BASE_URL },
    { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
  ]);

  // 兜底 catch-all 必须最先注册：Playwright route 后注册先匹配（LIFO），
  // 若放在最后会抢占 /me/、workspaces 等所有具体 mock（实测验证），
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

  // 用户信息（Sidebar 租户上下文）
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

  // 工作空间列表（含 is_current → WorkspaceSwitcher 加载后 emit workspace-switched）
  await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json',
      body: JSON.stringify([{ id: WORKSPACE_ID, name: 'E2E WS', is_current: true, is_default: true }]),
    });
  });

  await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: PROGRESS_COLUMNS }) });
  });

  await page.route(`**/manage-deliverable-system/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ current_deliverable_objs: [] }) });
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

  // 任务列表（空看板即可，本测试只关心头部）
  await page.route(`**/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}**`, async (r) => {
    if (r.request().method() !== 'GET') { await r.continue(); return; }
    await r.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
  });

  // EventSource mock：WorkPanel SSE 连接不挂起、不产生真实请求
  await page.addInitScript(() => {
    class MockEventSource {
      static CONNECTING = 0; static OPEN = 1; static CLOSED = 2;
      constructor(url) { this.url = String(url || ''); this.readyState = MockEventSource.OPEN; this.withCredentials = true; this.onmessage = null; this.onerror = null; this.onclose = null; this._listeners = { open: [] }; setTimeout(() => this._emitOpen(), 20); }
      addEventListener(type, cb) { if (!this._listeners[type]) this._listeners[type] = []; this._listeners[type].push(cb); }
      close() { this.readyState = MockEventSource.CLOSED; if (typeof this.onclose === 'function') this.onclose(); }
      _emitOpen() { const ev = { type: 'open' }; for (const cb of this._listeners.open || []) { try { cb(ev); } catch (_) {} } }
    }
    window.EventSource = MockEventSource;
  });
}

test.describe('WorkPanel 头部自动调度安排入口', () => {
  test('链接位于搜索框与工作空间选择器之间，侧栏不再渲染排队调度项', async ({ page }) => {
    test.setTimeout(60_000);
    await setupMocks(page);
    await page.setViewportSize({ width: 1440, height: 900 });

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel?workspace_id=${WORKSPACE_ID}`, { waitUntil: 'domcontentloaded' });

    // 1. 头部链接可见 + 文案 + href
    const link = page.getByTestId('auto-schedule-link');
    await expect(link).toBeVisible({ timeout: 15_000 });
    await expect(link).toHaveText('自动调度安排');

    const href = await link.getAttribute('href');
    expect(href).toContain(`/tenant/${TENANT_ID}/queue-schedule/`);
    expect(href).toContain(`workspace_id=${WORKSPACE_ID}`);

    // 2. 位置约束：搜索框（左）→ 链接（中）→ 工作空间选择器（右），同一行 x 坐标递增
    const searchBox = page.locator('[data-alias="work-panel-task-search"]');
    const switcherRow = page.locator('[data-alias="header-workspace-switcher-row"]');
    const toolbar = page.locator('[data-alias="header-toolbar-row"]');
    const titleRow = page.locator('[data-alias="header-title-row"]');
    const summaryRow = page.locator('[data-alias="header-summary-row"]');
    await expect(searchBox).toBeVisible({ timeout: 10_000 });
    await expect(switcherRow).toBeVisible({ timeout: 10_000 });
    await expect(toolbar).toBeVisible();
    await expect(titleRow).toBeVisible();
    await expect(summaryRow).toBeVisible();

    const searchBoxBox = await searchBox.boundingBox();
    const linkBox = await link.boundingBox();
    const switcherBox = await switcherRow.boundingBox();
    const titleBox = await titleRow.boundingBox();
    const summaryBox = await summaryRow.boundingBox();
    expect(searchBoxBox).not.toBeNull();
    expect(linkBox).not.toBeNull();
    expect(switcherBox).not.toBeNull();
    expect(titleBox).not.toBeNull();
    expect(summaryBox).not.toBeNull();
    // 同一行（y 中心点接近）
    expect(Math.abs(linkBox.y + linkBox.height / 2 - (searchBoxBox.y + searchBoxBox.height / 2))).toBeLessThan(40);
    expect(Math.abs(titleBox.y + titleBox.height / 2 - (summaryBox.y + summaryBox.height / 2))).toBeLessThan(40);
    // x 顺序：搜索框 < 链接 < 工作空间选择器
    expect(linkBox.x).toBeGreaterThan(searchBoxBox.x);
    expect(switcherBox.x).toBeGreaterThan(linkBox.x);

    // 3. 侧栏不再渲染「工作空间的排队调度」导航项
    const sidebarNav = page.locator('aside.tenant-console-sidebar nav');
    await expect(sidebarNav).toBeVisible({ timeout: 10_000 });
    await expect(sidebarNav.getByText('工作空间的排队调度')).toHaveCount(0);
    // 头部链接不在侧栏内
    await expect(sidebarNav.getByTestId('auto-schedule-link')).toHaveCount(0);
  });
});
