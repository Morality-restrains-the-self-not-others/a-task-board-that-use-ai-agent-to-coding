// @ts-check
/**
 * 回归：停止虚拟机后端会推 type=close，旧逻辑永久关闭 EventSource；
 * 若 containerEndpointRegistered 未变回 false，页面不会再次 establishSSE，第二次启动看不到 SSE 日志。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '839037065709281280';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`
);

test.describe('TaskDetail SSE after stop close', () => {
  test('close 后应重建 EventSource，第二轮推送仍能写入启动日志', async ({ page }) => {
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
          this._closed = false;
          window.__mockEventSource = this;

          setTimeout(() => this._emitOpen(), 20);
        }

        addEventListener(type, cb) {
          if (!this._listeners[type]) this._listeners[type] = [];
          this._listeners[type].push(cb);
        }

        close() {
          this.readyState = MockEventSource.CLOSED;
          this._closed = true;
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
          if (this._closed) return;
          const expected = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`;
          if (!this.url.includes(expected)) return;
          if (typeof this.onmessage === 'function') {
            this.onmessage({ data: JSON.stringify(payload) });
          }
        }
      }

      window.EventSource = MockEventSource;
      window.__emitStartupSse = (payload) => {
        window.__mockEventSource?.emit(payload);
      };
    }, { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID });

    await page.context().addCookies([
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
            title: 'SSE reconnect test',
            description: '',
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

      if (url.includes('/task-detail/') && url.includes('/mock-trae-online-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ log: '', in_progress: false }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            container_endpoint_registered: true,
            container_page_url: 'http://127.0.0.1:18080/ui/mock-token',
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layers_root: '/workspace/layers',
            bootstrap_layer_id: 'layer-1',
            layers: [],
            jobs: [],
          }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    await page.waitForFunction(() => typeof window.__emitStartupSse === 'function', null, { timeout: 30000 });
    // TaskDetail 在 onMounted 末尾才 establishSSE；仅 domcontentloaded 时 EventSource 可能尚未创建。
    await page.waitForFunction(
      () =>
        window.__mockEventSource &&
        String(window.__mockEventSource.url || '').includes('server-startup-status-sse'),
      null,
      { timeout: 30000 }
    );

    await page.evaluate(() => {
      window.__emitStartupSse?.({
        status: 'processing',
        message: 'sse-round-one',
        progress: 10,
        event_name: 'server_status_update',
      });
    });

    // statusMessage 与 statusLogs 各出现一次，避免 strict 双匹配
    await expect(page.locator('p', { hasText: /] sse-round-one$/ })).toBeVisible({ timeout: 15000 });

    await page.evaluate(() => {
      window.__emitStartupSse?.({
        type: 'close',
        event_name: 'server_status_update',
      });
    });

    await page.waitForFunction(
      () => {
        const es = window.__mockEventSource;
        return es && es._closed === false && es.readyState === window.EventSource.OPEN;
      },
      null,
      { timeout: 15000 }
    );

    await page.evaluate(() => {
      window.__emitStartupSse?.({
        status: 'processing',
        message: 'sse-round-two-after-close',
        progress: 20,
        event_name: 'server_status_update',
      });
    });

    await expect(page.locator('p', { hasText: /] sse-round-two-after-close$/ })).toBeVisible({ timeout: 15000 });
  });
});
