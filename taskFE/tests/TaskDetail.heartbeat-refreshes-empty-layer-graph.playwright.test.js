// @ts-check
/**
 * 回归：层图为空或仍有运行中 job 时，container_heartbeat ok 应节流触发 container-layer-graph 补拉
 * （克隆进度 SSE 未触发 refresh 时评论区 zTree 依赖此路径恢复）。
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

test.describe('TaskDetail 心跳补拉 layer-graph', () => {
  test('container_heartbeat ok 在层图仍空时隔 4s+ 再次触发 GET container-layer-graph', async ({ page }) => {
    let layerGraphGetCount = 0;

    await page.addInitScript(() => {
      // vite 构建可能把 API 指到 api.daydaymoney.com，Playwright route 对跨域仍应生效；
      // 强制同源可避免联调环境 DNS/证书差异导致首屏未拉到 container-task-ui-context。
      window.__TASK2APP_API_BASE_URL__ = '';
    });

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
            title: 'Mock Task heartbeat layer-graph',
            description: 'x',
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

      if (url.includes('/task-detail/') && url.includes('/mock-trae-online-stream/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', message: 'mock stream started' }),
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
        layerGraphGetCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layers_root: '/workspace/layers',
            bootstrap_layer_id: 'layer-1',
            layers: [],
            jobs: [{ id: 'job-1', layer_id: 'layer-1', status: 'running', command_kind: 'trae' }],
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', mock: true }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    await expect
      .poll(() => layerGraphGetCount, { timeout: 90000, intervals: [400, 1200, 3000] })
      .toBeGreaterThanOrEqual(1);
    const afterLoad = layerGraphGetCount;

    await page.evaluate(() => {
      window.__emitStartupSse?.({
        event_name: 'container_heartbeat',
        status: 'ok',
        bidirectional_ok: true,
        uplink_ok: true,
        downlink_ok: true,
        message: 'hb1',
      });
    });
    await page.waitForTimeout(200);
    const afterHb1 = layerGraphGetCount;
    expect(afterHb1).toBeGreaterThanOrEqual(afterLoad);

    await page.evaluate(() => {
      window.__emitStartupSse?.({
        event_name: 'container_heartbeat',
        status: 'ok',
        bidirectional_ok: true,
        uplink_ok: true,
        downlink_ok: true,
        message: 'hb2',
      });
    });
    await page.waitForTimeout(200);
    expect(layerGraphGetCount).toBe(afterHb1);

    await page.waitForTimeout(4200);
    await page.evaluate(() => {
      window.__emitStartupSse?.({
        event_name: 'container_heartbeat',
        status: 'ok',
        bidirectional_ok: true,
        uplink_ok: true,
        downlink_ok: true,
        message: 'hb3',
      });
    });
    await page.waitForTimeout(500);
    expect(layerGraphGetCount).toBeGreaterThan(afterHb1);
  });
});
