// @ts-check
/**
 * 轻量回归：不依赖 Docker/真实账号，纯前端 route mock 验证 task-detail zTree 执行日志区：
 * 1) 控制台输出与实时输出重复时去冗余；
 * 2) 代理步骤右上角展示模型类别与 Token；
 * 3) 展示 tool_results.result；
 * 4) 支持复制步骤 JSON。
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

const LAYER_ID = 'layer-1';
const JOB_ID = 'job-1';
const DUPLICATE_OUTPUT = 'DUPLICATE LOG LINE';

test.describe('TaskDetail zTree 代理步骤 UI（mock）', () => {
  test('显示模型+token、tool_results.result、复制 JSON，且执行日志去冗余', async ({ page }) => {
    await page.addInitScript(
      ({ tenantId, workspaceId, taskId, jobId, duplicateOutput }) => {
        window.__copiedTexts = [];
        Object.defineProperty(navigator, 'clipboard', {
          configurable: true,
          value: {
            writeText: async (text) => {
              window.__copiedTexts.push(String(text || ''));
            },
          },
        });

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
              this._emitMessage({
                status: 'container_job_stream',
                job_id: jobId,
                phase: 'chunk',
                message: duplicateOutput,
              });
            }, 60);
          }

          addEventListener(type, cb) {
            if (!this._listeners[type]) this._listeners[type] = [];
            this._listeners[type].push(cb);
          }

          close() {
            this.readyState = MockEventSource.CLOSED;
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

          _emitMessage(payload) {
            if (typeof this.onmessage !== 'function') return;
            try {
              this.onmessage({ data: JSON.stringify(payload) });
            } catch (_) {}
          }
        }

        window.EventSource = MockEventSource;
      },
      {
        tenantId: TENANT_ID,
        workspaceId: WORKSPACE_ID,
        taskId: TASK_ID,
        jobId: JOB_ID,
        duplicateOutput: DUPLICATE_OUTPUT,
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
            description: 'Mock task detail',
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
            bootstrap_layer_id: LAYER_ID,
            layers: [
              {
                layer_id: LAYER_ID,
                parent_layer_id: null,
                created_at: '2026-01-01T00:00:00Z',
                command: 'mock command',
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
                command: 'echo mock',
                created_at: '2026-01-01T00:00:01Z',
                output: DUPLICATE_OUTPUT,
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
          body: JSON.stringify({ layer_id: LAYER_ID, text: 'clone ok' }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-job-execution-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            job: {
              id: JOB_ID,
              status: 'completed',
              command: 'echo mock',
              output: DUPLICATE_OUTPUT,
            },
            steps: {
              note: 'mock steps note',
              steps: [
                {
                  step_number: 1,
                  state: 'completed',
                  timestamp: '2026-01-01T00:00:02Z',
                  llm_response: {
                    model: 'claude-sonnet-4-20250514',
                    usage: { input_tokens: 12, output_tokens: 34 },
                    content_type: 'text/html',
                    content:
                      '<p>请选择</p><div><span class="opt-radio" data-value="v-mock">选项 Mock</span></div>',
                  },
                  tool_calls: [
                    {
                      call_id: 'call_1',
                      name: 'ReadFile',
                      arguments: { path: '/tmp/demo.txt' },
                    },
                  ],
                  tool_results: [
                    {
                      call_id: 'call_1',
                      success: true,
                      result: 'file content from mock',
                      error: null,
                    },
                  ],
                },
              ],
            },
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

    const execPanel = page.getByTestId('comment-layer-ztree-exec-log-panel');
    await expect(execPanel).toBeVisible({ timeout: 15000 });
    await expect(execPanel.getByText('控制台输出已在实时输出中展示')).toBeVisible({ timeout: 15000 });

    const stepCards = page.getByTestId('layer-agent-steps-cards');
    await expect(stepCards).toBeVisible({ timeout: 15000 });
    await expect(stepCards.getByTestId('layer-agent-steps-accordion')).toBeVisible({ timeout: 15000 });
    const firstStep = stepCards.getByTestId('layer-agent-step-card').first();
    await expect(firstStep).toBeVisible({ timeout: 15000 });
    // 默认展开最近一步；summary 可见，正文可折叠
    const stepSummary = firstStep.getByTestId('layer-agent-step-card-summary');
    await expect(stepSummary).toBeVisible({ timeout: 15000 });
    await expect(firstStep).toHaveAttribute('open', '');
    await expect(firstStep.getByText('Anthropic')).toBeVisible({ timeout: 15000 });
    await expect(firstStep.getByText('Token I/O 12/34')).toBeVisible({ timeout: 15000 });
    await expect(firstStep.getByText('tool_results.result')).toBeVisible({ timeout: 15000 });
    await expect(firstStep).toContainText('file content from mock', { timeout: 15000 });

    const richFrame = firstStep.getByTestId('agent-step-rich-frame');
    await expect(richFrame).toBeVisible({ timeout: 15000 });
    const fr = page.frameLocator('[data-testid="agent-step-rich-frame"] iframe');
    const opt = fr.locator('span.opt-radio', { hasText: '选项 Mock' });
    await expect(opt).toBeVisible({ timeout: 5000 });
    await opt.evaluate((el) => el.click());
    const cmdInput = page.locator('#layer-graph-command-input');
    await expect(cmdInput).toHaveValue(/选项 Mock|v-mock/, { timeout: 5000 });

    const copyBtn = firstStep.getByRole('button', { name: /复制JSON|已复制JSON/ });
    await expect(copyBtn).toBeVisible({ timeout: 15000 });
    await copyBtn.click();
    await expect(firstStep.getByRole('button', { name: '已复制JSON' })).toBeVisible({ timeout: 5000 });
    await expect(firstStep).toHaveAttribute('open', '');
    await expect
      .poll(() => page.evaluate(() => (window.__copiedTexts || []).length), { timeout: 5000 })
      .toBeGreaterThan(0);

    // 手风琴：点击 summary 可收起/再展开
    await stepSummary.click();
    await expect(firstStep).not.toHaveAttribute('open', '');
    await expect(firstStep.getByTestId('agent-step-rich-frame')).toBeHidden({ timeout: 5000 });
    await stepSummary.click();
    await expect(firstStep).toHaveAttribute('open', '');
    await expect(firstStep.getByTestId('agent-step-rich-frame')).toBeVisible({ timeout: 5000 });
  });

  test('多步代理手风琴互斥：展开步骤 B 时步骤 A 自动收起', async ({ page }) => {
    await page.addInitScript(
      ({ tenantId, workspaceId, taskId }) => {
        window.__copiedTexts = [];
        Object.defineProperty(navigator, 'clipboard', {
          configurable: true,
          value: {
            writeText: async (text) => {
              window.__copiedTexts.push(String(text || ''));
            },
          },
        });

        class MockEventSource {
          static CONNECTING = 0; static OPEN = 1; static CLOSED = 2;
          constructor(url) {
            this.url = String(url || '');
            this.readyState = MockEventSource.OPEN;
            this.withCredentials = true;
            this.onmessage = null;
            this.onerror = null;
            this.onclose = null;
            this._listeners = { open: [] };
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
            for (const cb of this._listeners.open || []) { try { cb(ev); } catch (_) {} }
          }
          _emitMessage(payload) {
            if (typeof this.onmessage !== 'function') return;
            try { this.onmessage({ data: JSON.stringify(payload) }); } catch (_) {}
          }
        }
        window.EventSource = MockEventSource;
      },
      { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID },
    );

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    const step1 = { step_number: 1, state: 'completed', timestamp: '2026-01-01T00:00:02Z', llm_response: { model: 'claude-sonnet-4-20250514', usage: { input_tokens: 12, output_tokens: 34 }, content_type: 'text/html', content: '<p>Step A content</p>' }, tool_calls: [{ call_id: 'call_a', name: 'ReadFile', arguments: { path: '/a.txt' } }], tool_results: [{ call_id: 'call_a', success: true, result: 'step a result', error: null }] };
    const step2 = { step_number: 2, state: 'completed', timestamp: '2026-01-01T00:00:03Z', llm_response: { model: 'claude-sonnet-4-20250514', usage: { input_tokens: 5, output_tokens: 10 }, content_type: 'text/html', content: '<p>Step B content</p>' }, tool_calls: [{ call_id: 'call_b', name: 'WriteFile', arguments: { path: '/b.txt' } }], tool_results: [{ call_id: 'call_b', success: true, result: 'step b result', error: null }] };

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: TASK_ID, title: 'Mock Accordion Task', description: '', created_at: '2026-01-01T00:00:00Z', created_by: { username: 'mock-user' }, comments: [], ai_comments: [], assignees: [], workspace_id: WORKSPACE_ID }) });
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
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ layers_root: '/workspace/layers', bootstrap_layer_id: 'layer-1', layers: [{ layer_id: 'layer-1', parent_layer_id: null, created_at: '2026-01-01T00:00:00Z', command: 'mock', job_status: 'completed', git_worktree_dirty: false, mind_state: 'idle_done', queue_depth: 0 }], jobs: [{ id: 'job-1', layer_id: 'layer-1', status: 'completed', command_kind: 'trae', command: 'echo', created_at: '2026-01-01T00:00:01Z', output: 'log' }] }) });
        return;
      }
      if (url.includes('/cloud/compute/container-job-execution-log/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ job: { id: 'job-1', status: 'completed', command: 'echo', output: 'log' }, steps: { note: 'multi-step accordion test', steps: [step1, step2] } }) });
        return;
      }
      if (url.includes('/cloud/compute/container-clone-log/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ layer_id: 'layer-1', text: 'clone ok' }) });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    await expect(ztreePanel).toBeVisible({ timeout: 30000 });

    const nodeButtons = ztreePanel.locator('button.break-words.flex-1.min-w-0');
    await expect(nodeButtons.first()).toBeVisible({ timeout: 10000 });
    let count = await nodeButtons.count();
    for (let i = 0; i < count; i++) {
      const label = ((await nodeButtons.nth(i).innerText()) || '').trim();
      if (label === '可写层' || /^可写层（/.test(label)) continue;
      await nodeButtons.nth(i).click();
      break;
    }

    const execPanel = page.getByTestId('comment-layer-ztree-exec-log-panel');
    await expect(execPanel).toBeVisible({ timeout: 15000 });

    const stepCards = page.getByTestId('layer-agent-steps-cards');
    await expect(stepCards).toBeVisible({ timeout: 15000 });

    const allSteps = stepCards.getByTestId('layer-agent-step-card');
    await expect(allSteps).toHaveCount(2, { timeout: 15000 });

    const stepASummary = allSteps.nth(0).getByTestId('layer-agent-step-card-summary');
    const stepBSummary = allSteps.nth(1).getByTestId('layer-agent-step-card-summary');

    // 点击步骤 A 展开
    await stepASummary.click();
    await expect(allSteps.nth(0)).toHaveAttribute('open', '', { timeout: 5000 });
    await expect(allSteps.nth(1)).not.toHaveAttribute('open', '');

    // 点击步骤 B → 步骤 A 应自动收起（互斥）
    await stepBSummary.click();
    await expect(allSteps.nth(1)).toHaveAttribute('open', '', { timeout: 5000 });
    await expect(allSteps.nth(0)).not.toHaveAttribute('open', '');
  });

