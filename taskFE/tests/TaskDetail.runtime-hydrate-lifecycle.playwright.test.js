// @ts-check
/**
 * 意图 029 E1：机器已 Running 时打开/刷新任务详情，
 * 「服务器运行状态」为运行中，且「服务器启动状态」为「已启动」（不依赖本会话 SSE success）。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '861623708318031999';
const TASK_DETAIL_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`;

test.describe('任务详情 — runtime hydrate 对齐服务器启动状态', () => {
  test('Running 冷打开后 lifecycle 显示已启动', async ({ page, context }) => {
    await page.addInitScript(() => {
      class MockEventSource {
        static CONNECTING = 0;
        static OPEN = 1;
        static CLOSED = 2;

        constructor(url) {
          this.url = String(url || '');
          this.readyState = MockEventSource.OPEN;
          this.withCredentials = true;
          this.onmessage = null;
          this.onerror = null;
          this.onclose = null;
          this._listeners = { open: [] };
          setTimeout(() => {
            const ev = { type: 'open' };
            for (const cb of this._listeners.open || []) {
              try {
                cb(ev);
              } catch (_) {}
            }
          }, 20);
        }

        addEventListener(type, cb) {
          if (!this._listeners[type]) this._listeners[type] = [];
          this._listeners[type].push(cb);
        }

        close() {
          this.readyState = MockEventSource.CLOSED;
        }
      }

      window.EventSource = MockEventSource;
    });

    await context.addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock runtime hydrate lifecycle',
            description: 'e2e',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            runtime_status: 'Running',
            instance_id: 'i-e2e-hydrate',
            platform: 'aliyun',
            region: 'cn-hangzhou',
            instance_attribute: {
              body: {
                Status: 'Running',
              },
            },
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            has_server_config: true,
            container_endpoint_registered: false,
            container_page_url: '',
            container_vscode_url: '',
          }),
        });
        return;
      }

      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ columns: [] }),
        });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/cloud/platforms/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', platforms: [] }),
        });
        return;
      }

      if (url.includes('/cloud/compute/previous-server-config/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', server_config: null }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    await expect(page.getByText('运行中').first()).toBeVisible({ timeout: 45000 });
    await expect(page.getByTestId('server-lifecycle-status')).toHaveText('已启动', { timeout: 45000 });
  });
});
