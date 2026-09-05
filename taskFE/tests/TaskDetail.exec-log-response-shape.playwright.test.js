// @ts-check
/**
 * 回归：container-job-execution-log 必须返回 { job, steps, layer_changes } 包装体。
 * 若网关把 job 字段扁平化到顶层，前端 jobExecutionPayload?.job 为 undefined →「暂无任务日志」，
 * 而容器页直连 /api/jobs 仍能看到日志。
 *
 * 纯前端 route mock，不依赖真实容器。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662914';
const LAYER_ID = 'layer-execlog-shape-1';
const JOB_ID = 'job-execlog-shape-1';
const MARKER = 'HELLO_WORLD_EXEC_LOG_MARKER_42';

const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936&relayToTrae=true`,
);

/**
 * @param {import('@playwright/test').Page} page
 * @param {'wrapped' | 'flat'} shape
 */
async function installMocks(page, shape) {
  await page.addInitScript(
    ({ tenantId, workspaceId, taskId, jobId, layerId }) => {
      /** @type {Array<(payload: object) => void>} */
      const sseSinks = [];
      window.__sseSinks = sseSinks;

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
            this._emitOpen();
            const expected = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`;
            if (!this.url.includes(expected)) return;
            const emit = (payload) => this._emitMessage(payload);
            sseSinks.push(emit);
            emit({
              status: 'container_task_ui_context',
              container_endpoint_registered: true,
              container_page_url: 'http://127.0.0.1:18080/ui/mock-token',
            });
            emit({
              status: 'container_layer_graph',
              layers_root: '/workspace/layers',
              bootstrap_layer_id: layerId,
              layers: [
                {
                  layer_id: layerId,
                  parent_layer_id: null,
                  created_at: '2026-01-01T00:00:00Z',
                  command: 'mock',
                  job_status: 'succeeded',
                  git_worktree_dirty: false,
                  mind_state: 'idle',
                  queue_depth: 0,
                },
              ],
              jobs: [
                {
                  id: jobId,
                  layer_id: layerId,
                  status: 'succeeded',
                  command_kind: 'trae',
                  command: '在 somanyad 中用 lisp 写一个 hello world 程序',
                  created_at: '2026-01-01T00:00:01Z',
                },
              ],
            });
          }, 0);
        }

        addEventListener(type, fn) {
          if (type === 'open') this._listeners.open.push(fn);
        }

        close() {
          this.readyState = MockEventSource.CLOSED;
        }

        _emitOpen() {
          for (const fn of this._listeners.open) fn({});
          if (typeof this.onopen === 'function') this.onopen({});
        }

        _emitMessage(payload) {
          const data = JSON.stringify(payload);
          const ev = { data };
          if (typeof this.onmessage === 'function') this.onmessage(ev);
        }
      }

      window.EventSource = MockEventSource;
    },
    {
      tenantId: TENANT_ID,
      workspaceId: WORKSPACE_ID,
      taskId: TASK_ID,
      jobId: JOB_ID,
      layerId: LAYER_ID,
    },
  );

  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    if (url.includes('/todos/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: TASK_ID,
          title: 'execlog shape',
          description: '',
          status: 'in_progress',
          workspace: WORKSPACE_ID,
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
          layers: [
            {
              layer_id: LAYER_ID,
              parent_layer_id: null,
              created_at: '2026-01-01T00:00:00Z',
              command: 'mock',
              job_status: 'succeeded',
              git_worktree_dirty: false,
              mind_state: 'idle',
              queue_depth: 0,
            },
          ],
          jobs: [
            {
              id: JOB_ID,
              layer_id: LAYER_ID,
              status: 'succeeded',
              command_kind: 'trae',
              command: '在 somanyad 中用 lisp 写一个 hello world 程序',
              created_at: '2026-01-01T00:00:01Z',
            },
          ],
        }),
      });
      return;
    }

    if (url.includes('/cloud/compute/container-clone-log/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ layer_id: LAYER_ID, text: 'clone ok\n' }),
      });
      return;
    }

    if (url.includes('/cloud/compute/container-job-execution-log/') && method === 'GET') {
      if (shape === 'flat') {
        // 旧网关错误形状：扁平 job + steps 数组
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: JOB_ID,
            status: 'succeeded',
            command: '在 somanyad 中用 lisp 写一个 hello world 程序',
            output: MARKER,
            steps: [{ i: 1, kind: 'assistant', content: 'done' }],
          }),
        });
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          job: {
            id: JOB_ID,
            status: 'succeeded',
            command: '在 somanyad 中用 lisp 写一个 hello world 程序',
            output: MARKER,
          },
          steps: {
            note: null,
            steps: [
              {
                step_number: 1,
                state: 'completed',
                timestamp: '2026-01-01T00:00:02Z',
                llm_response: {
                  model: 'mock-model',
                  content: 'Wrote hello.lisp',
                },
              },
            ],
          },
          layer_changes: {
            layer_id: LAYER_ID,
            change_count: 1,
            changes: [{ path: 'hello.lisp', kind: 'added' }],
          },
        }),
      });
      return;
    }

    if (url.includes('/projects/workspace-access/') || url.includes('/progress-system/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: '{}',
    });
  });
}

/**
 * @param {import('@playwright/test').Page} page
 */
async function selectFirstRealLayer(page) {
  const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
  await expect(ztreePanel).toBeVisible({ timeout: 30000 });
  const nodeButtons = ztreePanel.locator('button.break-words.flex-1.min-w-0');
  await expect(nodeButtons.first()).toBeVisible({ timeout: 10000 });
  const count = await nodeButtons.count();
  let picked = false;
  for (let i = 0; i < count; i++) {
    const label = ((await nodeButtons.nth(i).innerText()) || '').trim();
    if (label === '可写层' || /^可写层（/.test(label)) continue;
    await nodeButtons.nth(i).click();
    picked = true;
    break;
  }
  expect(picked).toBe(true);
}

test.describe('TaskDetail 执行日志 API 响应形状（mock）', () => {
  test('包装体 {job,steps} 时执行日志可见任务输出', async ({ page }) => {
    await installMocks(page, 'wrapped');
    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await selectFirstRealLayer(page);

    const execPanel = page.getByTestId('comment-layer-ztree-exec-log-panel');
    await expect(execPanel).toBeVisible({ timeout: 15000 });
    await expect(execPanel.getByText('暂无任务日志')).toHaveCount(0);
    await expect(execPanel.getByText(MARKER)).toBeVisible({ timeout: 15000 });
  });

  test('扁平化响应经前端归一化后仍可见任务输出（防御旧网关）', async ({ page }) => {
    await installMocks(page, 'flat');
    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await selectFirstRealLayer(page);

    const execPanel = page.getByTestId('comment-layer-ztree-exec-log-panel');
    await expect(execPanel).toBeVisible({ timeout: 15000 });
    await expect(execPanel.getByText('暂无任务日志')).toHaveCount(0);
    await expect(execPanel.getByText(MARKER)).toBeVisible({ timeout: 15000 });
  });
});
