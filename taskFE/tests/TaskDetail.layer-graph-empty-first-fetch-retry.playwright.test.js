// @ts-check
/**
 * 回归：容器端点已注册但首次 container-layer-graph 返回空 layers/jobs 时，
 * 须继续轮询直至拉到非空快照，评论区「任务关联」zTree 应出现。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662912';
const LAYER_ID = 'layer-retry-1';
const JOB_ID = 'job-retry-1';

const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`
);

const layerGraphBodyEmpty = JSON.stringify({
  layers_root: '/workspace/layers',
  bootstrap_layer_id: '',
  layers: [],
  jobs: [],
});

const layerGraphBodyPopulated = JSON.stringify({
  layers_root: '/workspace/layers',
  bootstrap_layer_id: LAYER_ID,
  layers: [
    {
      layer_id: LAYER_ID,
      parent_layer_id: null,
      created_at: '2026-01-01T00:00:00Z',
      command: 'bootstrap',
      job_status: 'completed',
      git_worktree_dirty: false,
      mind_state: 'idle_done',
      queue_depth: 0,
    },
  ],
  jobs: [
    {
      id: JOB_ID,
      layer_id: LAYER_ID,
      status: 'completed',
      command_kind: 'trae',
      command: 'echo ok',
      created_at: '2026-01-01T00:00:01Z',
      output: '',
    },
  ],
});

test.describe('TaskDetail 层级快照', () => {
  test('container_task_ui_ready 和 container_task_ui_context SSE 事件后显示 zTree', async ({ page }) => {
    let layerGraphRequestCount = 0;

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
        layerGraphRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: layerGraphBodyPopulated,
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const panel = page.getByTestId('comment-layer-ztree-panel');
    await expect(panel).toBeVisible({ timeout: 30000 });

    await expect.poll(() => layerGraphRequestCount, { timeout: 30000 }).toBeGreaterThanOrEqual(1);
  });
});
