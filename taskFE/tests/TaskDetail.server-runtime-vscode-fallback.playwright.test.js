// @ts-check
/**
 * 回归：container_vscode_url 为空但实例 Running 且已有公网 IP 时，
 * 「服务器运行状态」仍显示「打开服务器 VS Code」（回退为 IP:8888）。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '837978129569890304';
const TASK_DETAIL_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`;

const MOCK_PUBLIC_IP = '203.0.113.9';
const STALE_VSCODE_IP = '198.51.100.88';
const EXPECT_VSCODE_HREF = `http://${MOCK_PUBLIC_IP}:8888/`;

test.describe('任务详情 — 服务器运行状态 VS Code 回退链接', () => {
  test('无 container_vscode_url 时仍显示宿主机 VS Code 默认端口链接', async ({ page, context }) => {
    await page.addInitScript(({ tenantId, workspaceId, taskId }) => {
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
          window.__mockEventSource = this;
          setTimeout(() => this._emitOpen(), 20);
        }

        addEventListener(type, cb) {
          if (!this._listeners[type]) this._listeners[type] = [];
          this._listeners[type].push(cb);
        }

        close() {
          this.readyState = MockEventSource.CLOSED;
          if (typeof this.onclose === 'function') this.onclose();
        }

        _emitOpen() {
          const ev = { type: 'open' };
          for (const cb of this._listeners.open || []) {
            try {
              cb(ev);
            } catch (_) {}
          }
        }

        emit(payload) {
          const expected = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`;
          if (!this.url.includes(expected)) return;
          if (typeof this.onmessage === 'function') {
            this.onmessage({ data: JSON.stringify(payload) });
          }
        }
      }

      window.EventSource = MockEventSource;
    }, { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID });

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
            title: 'Mock runtime vscode fallback',
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

      if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            has_server_config: true,
            container_endpoint_registered: true,
            container_page_url: '',
            container_vscode_url: '',
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
            instance_id: 'i-e2e-mock',
            platform: 'aliyun',
            region: 'cn-hangzhou',
            instance_attribute: {
              body: {
                Status: 'Running',
                PublicIpAddress: { IpAddress: [MOCK_PUBLIC_IP] },
              },
            },
          }),
        });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/cloud/platforms/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            platforms: [],
          }),
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

    const vscodeBtn = page.locator('#open-server-runtime-vscode-btn');
    await expect(vscodeBtn).toBeVisible({ timeout: 45000 });
    await expect(vscodeBtn).toHaveAttribute('href', EXPECT_VSCODE_HREF);
    await expect(vscodeBtn).toHaveAttribute('target', '_blank');
  });

  test('container_vscode_url 与实例公网 IP 不一致时，链接主机名与实例详情公网 IP 对齐', async ({ page, context }) => {
    await page.addInitScript(({ tenantId, workspaceId, taskId }) => {
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
          window.__mockEventSource = this;
          setTimeout(() => this._emitOpen(), 20);
        }

        addEventListener(type, cb) {
          if (!this._listeners[type]) this._listeners[type] = [];
          this._listeners[type].push(cb);
        }

        close() {
          this.readyState = MockEventSource.CLOSED;
          if (typeof this.onclose === 'function') this.onclose();
        }

        _emitOpen() {
          const ev = { type: 'open' };
          for (const cb of this._listeners.open || []) {
            try {
              cb(ev);
            } catch (_) {}
          }
        }

        emit(payload) {
          const expected = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`;
          if (!this.url.includes(expected)) return;
          if (typeof this.onmessage === 'function') {
            this.onmessage({ data: JSON.stringify(payload) });
          }
        }
      }

      window.EventSource = MockEventSource;
    }, { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID });

    await context.addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    const staleVscodeUrl = `http://${STALE_VSCODE_IP}:9999/`;
    const expectedAlignedHref = `http://${MOCK_PUBLIC_IP}:9999/`;

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
            title: 'Mock vscode host align',
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

      if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            has_server_config: true,
            container_endpoint_registered: true,
            container_page_url: '',
            container_vscode_url: staleVscodeUrl,
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
            instance_id: 'i-e2e-mock',
            platform: 'aliyun',
            region: 'cn-hangzhou',
            instance_attribute: {
              body: {
                Status: 'Running',
                PublicIpAddress: { IpAddress: [MOCK_PUBLIC_IP] },
              },
            },
          }),
        });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/cloud/platforms/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            platforms: [],
          }),
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

    const vscodeBtn = page.locator('#open-server-runtime-vscode-btn');
    await expect(vscodeBtn).toBeVisible({ timeout: 45000 });
    await expect(vscodeBtn).toHaveAttribute('href', expectedAlignedHref);
  });
});
