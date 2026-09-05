// @ts-check
/**
 * 回归：API 返回 container_vscode_url 时，「打开容器开发页面」链接可见，
 * 点击后以新窗口打开配置 URL，且目标页可加载（对 127.0.0.1:code-server 端口做 route mock）。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '837978129569890304';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

const CONTAINER_PAGE_URL = 'http://127.0.0.1:18080/ui/mock-token';
const VSCODE_PAGE_URL = 'http://127.0.0.1:18888/';

test.describe('TaskDetail 打开容器开发页面', () => {
  test('链接可见且点击后新页可打开 code-server URL', async ({ page, context }) => {
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
      window.__emitStartupSse = (payload) => {
        window.__mockEventSource?.emit(payload);
      };
    }, { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID });

    await context.addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    // 新标签请求的 VS Code Web 地址无真实服务时由 mock 响应，避免用例因连接被拒而失败
    await context.route(
      (url) => url.hostname === '127.0.0.1' && url.port === '18888',
      async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'text/html; charset=utf-8',
          body: '<!DOCTYPE html><html><head><title>mock-code-server</title></head><body data-e2e="vscode-mock">e2e-code-server-ok</body></html>',
        });
      },
    );

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
            title: 'Mock Task VSCode link',
            description: 'e2e container vscode button',
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
            has_server_config: true,
            container_endpoint_registered: true,
            container_page_url: CONTAINER_PAGE_URL,
            container_vscode_url: VSCODE_PAGE_URL,
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
            layers: [
              {
                layer_id: 'layer-1',
                created_at: '2026-01-01T00:00:00Z',
                job_status: 'completed',
              },
            ],
            jobs: [],
          }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 须挂在任务关联区域（与「打开容器页面」同排），不在「添加评论」区
    const associationPanel = page.locator('[data-testid="comment-layer-ztree-panel"]');
    await expect(associationPanel).toBeVisible({ timeout: 30000 });
    const vscodeBtn = associationPanel.locator('#open-container-vscode-btn');
    await expect(vscodeBtn).toBeVisible({ timeout: 30000 });
    await expect(vscodeBtn).toHaveAttribute('href', VSCODE_PAGE_URL);
    await expect(vscodeBtn).toHaveAttribute('target', '_blank');
    await expect(page.locator('h4', { hasText: '添加评论' }).locator('..').locator('#open-container-vscode-btn')).toHaveCount(0);

    await vscodeBtn.scrollIntoViewIfNeeded();

    const popupPromise = page.waitForEvent('popup');
    await vscodeBtn.click();
    const popup = await popupPromise;

    await expect(popup).not.toBeNull();
    await expect
      .poll(() => {
        try {
          return popup.url();
        } catch {
          return '';
        }
      }, { timeout: 10000 })
      .toContain('127.0.0.1:18888');

    await popup.waitForLoadState('domcontentloaded');
    await expect(popup.locator('[data-e2e="vscode-mock"]')).toHaveText('e2e-code-server-ok');

    await popup.close();
  });
});
