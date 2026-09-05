// @ts-check
/**
 * 回归：配置了 API 基址时，任务详情页建立 server-startup-status SSE 须使用与 apiFetch 同源的前缀，
 * 避免 EventSource 误指向前端域名导致连接失败与控制台报错。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662912';
const MOCK_API_ORIGIN = 'http://127.0.0.1:18081';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

const LAYER_ID = 'layer-1';
const JOB_ID = 'job-1';

test('server-startup-status SSE 使用 getApiUrl（含 API_BASE_URL 前缀）', async ({ page }) => {
  await page.addInitScript(
    ({ mockOrigin, tenantId, workspaceId, taskId, jobId }) => {
      window.__TASK2APP_API_BASE_URL__ = mockOrigin;
      window.__task2appCapturedSseUrls = [];

      class TracingMockEventSource {
        static CONNECTING = 0;
        static OPEN = 1;
        static CLOSED = 2;

        constructor(url) {
          try {
            window.__task2appCapturedSseUrls.push(String(url));
          } catch (_) {}
          this.url = String(url || '');
          this.readyState = TracingMockEventSource.OPEN;
          this.withCredentials = true;
          this.onmessage = null;
          this.onerror = null;
          this.onclose = null;
          this._listeners = { open: [] };

          setTimeout(() => {
            this._emitOpen();
            const needle = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`;
            if (!this.url.includes(needle)) return;
          }, 30);
        }

        addEventListener(type, cb) {
          if (!this._listeners[type]) this._listeners[type] = [];
          this._listeners[type].push(cb);
        }

        close() {
          this.readyState = TracingMockEventSource.CLOSED;
          if (typeof this.onclose === 'function') {
            this.onclose();
          }
        }

        _emitOpen() {
          const ev = { type: 'open' };
          for (const cb of this._listeners.open || []) {
            try {
              cb(ev);
            } catch (_) {}
          }
        }
      }

      window.EventSource = TracingMockEventSource;
    },
    {
      mockOrigin: MOCK_API_ORIGIN,
      tenantId: TENANT_ID,
      workspaceId: WORKSPACE_ID,
      taskId: TASK_ID,
      jobId: JOB_ID,
    },
  );

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
          title: 'Mock Task',
          description: 'Mock',
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

    if (url.includes('/mock-trae-online-log') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ log: 'x', in_progress: false }),
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
          container_endpoint_registered: false,
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
          bootstrap_layer_id: LAYER_ID,
          layers: [],
          jobs: [],
        }),
      });
      return;
    }

    if (url.includes('/cloud/compute/container-clone-log/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ layer_id: LAYER_ID, text: '' }),
      });
      return;
    }

    if (url.includes('/cloud/compute/container-job-execution-log/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          job: { id: JOB_ID, status: 'completed', command: 'echo', output: '' },
          steps: { steps: [] },
        }),
      });
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: '{}',
    });
  });

  await page.goto(TASK_DETAIL_PATH);
  await page.waitForLoadState('domcontentloaded');

  await expect
    .poll(
      async () => (await page.evaluate(() => (window.__task2appCapturedSseUrls || []).length)) > 0,
      { timeout: 20000 },
    )
    .toBe(true);

  const urls = await page.evaluate(() => window.__task2appCapturedSseUrls || []);
  const sse = urls.find((u) => u.includes('server-startup-status-sse'));
  expect(sse).toBeTruthy();
  expect(String(sse).startsWith(MOCK_API_ORIGIN)).toBe(true);
});
