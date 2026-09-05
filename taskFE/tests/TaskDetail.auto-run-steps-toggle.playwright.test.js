// @ts-check
/**
 * 回归：AutoRunStepsPreview 默认收起 + 展开/收起交互。
 * - auto_run=true 时 toggle 存在且 body 隐藏
 * - 点击 toggle 展开/收起 body
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

test.describe('TaskDetail AutoRunSteps 默认收起', () => {
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

    const taskDetail = {
      id: TASK_ID, title: 'AutoRun Test', description: '', created_at: '2026-01-01T00:00:00Z',
      created_by: { username: 'mock-user' }, comments: [], ai_comments: [], assignees: [],
      workspace_id: WORKSPACE_ID,
      auto_run: true,
      auto_run_steps_md: '# 自动运行步骤\n1. 拉取代码\n2. 构建镜像',
      auto_run_steps_extract_status: 'ok',
    };

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

  test('auto_run=true 时 toggle 为「查看自动运行说明」且 body 隐藏', async ({ page }) => {
    await setupPageMocks(page);

    const toggle = page.getByTestId('auto-run-steps-toggle');
    const body = page.getByTestId('auto-run-steps-body');

    await expect(toggle).toBeVisible({ timeout: 30000 });
    await expect(toggle).toContainText('查看自动运行说明');
    await expect(body).toBeHidden();
  });

  test('点击 toggle 展开 body，再点击收起', async ({ page }) => {
    await setupPageMocks(page);

    const toggle = page.getByTestId('auto-run-steps-toggle');
    const body = page.getByTestId('auto-run-steps-body');

    await expect(toggle).toBeVisible({ timeout: 30000 });
    await expect(body).toBeHidden();

    // 展开
    await toggle.click();
    await expect(body).toBeVisible({ timeout: 5000 });
    await expect(toggle).toContainText('收起自动运行说明');

    // 收起
    await toggle.click();
    await expect(body).toBeHidden({ timeout: 5000 });
    await expect(toggle).toContainText('查看自动运行说明');
  });
});
