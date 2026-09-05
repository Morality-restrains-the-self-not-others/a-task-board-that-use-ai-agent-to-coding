// @ts-check
/**
 * 回归：TaskDetail 服务器信息区折叠/展开交互。
 * - 点击折叠钮隐藏内容区，再点恢复
 * - 折叠时点击「服务器内容」Tab 自动展开（硬件已迁入镜像卡）
 *
 * 纯 mock，无真实账号依赖。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662912';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`
);

test.describe('TaskDetail 服务器信息区折叠', () => {
  async function setupPageMocks(page) {
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

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    const taskDetail = { id: TASK_ID, title: 'Server Collapse Test', description: '', created_at: '2026-01-01T00:00:00Z', created_by: { username: 'mock-user' }, comments: [], ai_comments: [], assignees: [], workspace_id: WORKSPACE_ID };

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(taskDetail) });
        return;
      }
      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }
      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: [] }) });
        return;
      }
      if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', container_endpoint_registered: true, container_page_url: 'http://127.0.0.1:18080/ui/mock-token' }) });
        return;
      }
      if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ layers_root: '/workspace/layers', bootstrap_layer_id: 'layer-1', layers: [], jobs: [] }) });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
  }

  test('点击折叠钮收起，再点展开', async ({ page }) => {
    await setupPageMocks(page);

    const toggle = page.getByTestId('server-section-collapse-toggle');
    const body = page.getByTestId('server-section-body');

    await expect(toggle).toBeVisible({ timeout: 30000 });
    await expect(body).toBeVisible();

    // 折叠
    await toggle.click();
    await expect(body).toBeHidden({ timeout: 5000 });

    // 展开
    await toggle.click();
    await expect(body).toBeVisible({ timeout: 5000 });
  });

  test('折叠状态下点击「服务器内容」Tab 自动展开内容区', async ({ page }) => {
    await setupPageMocks(page);

    const toggle = page.getByTestId('server-section-collapse-toggle');
    const body = page.getByTestId('server-section-body');
    const contentTab = page.getByRole('button', { name: '服务器内容' });

    await expect(toggle).toBeVisible({ timeout: 30000 });
    await expect(body).toBeVisible();

    // 先折叠
    await toggle.click();
    await expect(body).toBeHidden({ timeout: 5000 });

    // 点击服务器内容 Tab → 应自动展开（硬件已迁入镜像卡，不再有硬件 Tab）
    await contentTab.click();
    await expect(body).toBeVisible({ timeout: 5000 });
  });
});
