// @ts-check
/**
 * 回归：发送指令后执行日志滚动窗口不得因 DOM 拆毁重建 / sticky 恢复过晚而持续抖动。
 * 纯前端 route + SSE mock，不依赖真实容器。
 *
 * 验证：
 * 1) 高频 container_job_stream chunk 下 layer-live-output-scroll 元素引用保持不变；
 * 2) 贴底跟随期间，rAF 采样不应反复出现 scrollTop===0（旧实现 nextTick 后再 rAF 会先画出一帧 0）；
 * 3) 执行日志刷新时 step_number 从缺失→有值，代理步骤卡片宿主不因 key 变化而拆毁重建。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662913';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936&relayToTrae=true`
);

const LAYER_ID = 'layer-jitter-1';
const JOB_ID = 'job-jitter-1';

test.describe('TaskDetail 执行日志滚动抖动（mock）', () => {
  test('高频 SSE 追加 + 步骤刷新时滚动容器不拆毁、scrollTop 不闪 0', async ({ page }) => {
    await page.addInitScript(
      ({ tenantId, workspaceId, taskId, jobId }) => {
        /** @type {Array<(payload: object) => void>} */
        const sseSinks = [];
        window.__sseSinks = sseSinks;
        window.__emitJobChunk = (message) => {
          for (const emit of sseSinks) {
            emit({
              status: 'container_job_stream',
              job_id: jobId,
              phase: 'chunk',
              message,
            });
          }
        };
        window.__emitLayerGraph = (status) => {
          for (const emit of sseSinks) {
            emit({
              status: 'container_layer_graph',
              layers_root: '/workspace/layers',
              bootstrap_layer_id: 'layer-jitter-1',
              layers: [
                {
                  layer_id: 'layer-jitter-1',
                  parent_layer_id: null,
                  created_at: '2026-01-01T00:00:00Z',
                  command: 'mock',
                  job_status: status,
                  git_worktree_dirty: false,
                  mind_state: 'busy',
                  queue_depth: 0,
                },
              ],
              jobs: [
                {
                  id: jobId,
                  layer_id: 'layer-jitter-1',
                  status,
                  command_kind: 'trae',
                  command: 'echo jitter',
                  created_at: '2026-01-01T00:00:01Z',
                },
                // 故意打乱顺序的无关 job，旧实现未排序时会误触发刷新
                {
                  id: 'job-other',
                  layer_id: 'layer-jitter-1',
                  status: 'completed',
                  command_kind: 'shell',
                  command: 'true',
                  created_at: '2026-01-01T00:00:00Z',
                },
              ],
            });
          }
        };

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
                bootstrap_layer_id: 'layer-jitter-1',
                layers: [
                  {
                    layer_id: 'layer-jitter-1',
                    parent_layer_id: null,
                    created_at: '2026-01-01T00:00:00Z',
                    command: 'mock',
                    job_status: 'running',
                    git_worktree_dirty: false,
                    mind_state: 'busy',
                    queue_depth: 0,
                  },
                ],
                jobs: [
                  {
                    id: jobId,
                    layer_id: 'layer-jitter-1',
                    status: 'running',
                    command_kind: 'trae',
                    command: 'echo jitter',
                    created_at: '2026-01-01T00:00:01Z',
                  },
                ],
              });
            }, 40);
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
      },
    );

    let execLogFetchCount = 0;
    /** @type {'provisional' | 'numbered'} */
    let stepPhase = 'provisional';

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
            title: 'Jitter Mock Task',
            description: 'exec log scroll jitter',
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
                command: 'mock',
                job_status: 'running',
                git_worktree_dirty: false,
                mind_state: 'busy',
                queue_depth: 0,
              },
            ],
            jobs: [
              {
                id: JOB_ID,
                layer_id: LAYER_ID,
                status: 'running',
                command_kind: 'trae',
                command: 'echo jitter',
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
        execLogFetchCount += 1;
        const step =
          stepPhase === 'provisional'
            ? {
                state: 'thinking',
                trajectory_provisional: true,
                timestamp: '2026-01-01T00:00:02Z',
                llm_response: {
                  model: 'claude-sonnet-4-20250514',
                  usage: { input_tokens: 1, output_tokens: 2 },
                  content: 'thinking…',
                },
              }
            : {
                step_number: 1,
                state: 'thinking',
                trajectory_provisional: true,
                timestamp: '2026-01-01T00:00:02Z',
                llm_response: {
                  model: 'claude-sonnet-4-20250514',
                  usage: { input_tokens: 1, output_tokens: 2 },
                  content: 'thinking with number…',
                },
              };
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            job: {
              id: JOB_ID,
              status: 'running',
              command: 'echo jitter',
              output: '',
            },
            steps: { note: 'mock', steps: [step] },
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

    // 先灌入足够多行，使 live 输出可滚动
    await page.evaluate(() => {
      const lines = Array.from({ length: 40 }, (_, i) => `seed-line-${i}\n`).join('');
      window.__emitJobChunk(lines);
    });
    const liveScroll = execPanel.getByTestId('layer-live-output-scroll');
    await expect(liveScroll).toBeVisible({ timeout: 15000 });

    // 安装观测：视口身份 + 虚拟 body 长度 + rAF 采样 scrollTop===0
    await page.evaluate(() => {
      const panel = document.querySelector('[data-testid="comment-layer-ztree-exec-log-panel"]');
      const viewport = panel?.querySelector('[data-testid="layer-live-output-scroll"]');
      window.__jitterProbe = {
        initialEl: viewport,
        identityFlips: 0,
        zeroScrollFrames: 0,
        samples: 0,
        stepCardEl: null,
        stepCardFlips: 0,
        maxBodyLen: 0,
        stopped: false,
      };
      const probe = window.__jitterProbe;
      const cardsHost = panel?.querySelector('[data-testid="layer-agent-steps-cards"]');
      probe.stepCardEl = cardsHost?.querySelector('div.rounded-lg.border.border-gray-200') || null;

      const tick = () => {
        if (probe.stopped) return;
        const cur = panel?.querySelector('[data-testid="layer-live-output-scroll"]');
        if (cur && probe.initialEl && cur !== probe.initialEl) {
          probe.identityFlips += 1;
          probe.initialEl = cur;
        }
        const body = cur?.querySelector('[data-testid="virtual-log-body"]');
        if (body) {
          probe.maxBodyLen = Math.max(probe.maxBodyLen, (body.textContent || '').length);
        }
        if (cur && cur.scrollHeight > cur.clientHeight + 20) {
          probe.samples += 1;
          if (cur.scrollTop === 0) probe.zeroScrollFrames += 1;
        }
        const card = cardsHost?.querySelector('div.rounded-lg.border.border-gray-200') || null;
        if (card && probe.stepCardEl && card !== probe.stepCardEl) {
          probe.stepCardFlips += 1;
          probe.stepCardEl = card;
        } else if (card && !probe.stepCardEl) {
          probe.stepCardEl = card;
        }
        requestAnimationFrame(tick);
      };
      requestAnimationFrame(tick);
    });

    // 高频追加实时输出（模拟发送给 AI 后的流式日志）
    for (let i = 0; i < 30; i++) {
      await page.evaluate((n) => {
        window.__emitJobChunk(`stream-chunk-${n} ${'y'.repeat(80)}\n`);
      }, i);
      await page.waitForTimeout(16);
    }

    // 触发执行日志刷新：step_number 从无到有（旧 key 会变）
    stepPhase = 'numbered';
    await page.evaluate(() => {
      window.__emitLayerGraph('running');
    });
    await page.waitForTimeout(200);
    // 再推一次乱序 jobs（status 不变）——不应导致额外抖动性拆毁
    await page.evaluate(() => {
      window.__emitLayerGraph('running');
    });

    for (let i = 30; i < 45; i++) {
      await page.evaluate((n) => {
        window.__emitJobChunk(`stream-chunk-${n} ${'z'.repeat(80)}\n`);
      }, i);
      await page.waitForTimeout(16);
    }

    await page.waitForTimeout(200);
    const result = await page.evaluate(() => {
      const probe = window.__jitterProbe;
      probe.stopped = true;
      const viewport = document.querySelector('[data-testid="layer-live-output-scroll"]');
      const body = viewport?.querySelector('[data-testid="virtual-log-body"]');
      const topPad = viewport?.querySelector('[data-testid="virtual-log-top-pad"]');
      const bottomPad = viewport?.querySelector('[data-testid="virtual-log-bottom-pad"]');
      return {
        identityFlips: probe.identityFlips,
        zeroScrollFrames: probe.zeroScrollFrames,
        samples: probe.samples,
        stepCardFlips: probe.stepCardFlips,
        maxBodyLen: probe.maxBodyLen,
        bodyLen: (body?.textContent || '').length,
        hasPads: !!(topPad && bottomPad),
        finalTop: viewport?.scrollTop ?? null,
        finalSh: viewport?.scrollHeight ?? null,
        finalCh: viewport?.clientHeight ?? null,
      };
    });

    expect(result.samples, `应采集到可滚动采样: ${JSON.stringify(result)}`).toBeGreaterThan(10);
    expect(result.identityFlips, `live-output 元素被拆毁重建: ${JSON.stringify(result)}`).toBe(0);
    expect(result.hasPads, `虚拟列表缺少 pad: ${JSON.stringify(result)}`).toBe(true);
    // 可见 body 远小于累计流式文本（约 40*seed + 45*chunk）
    expect(result.maxBodyLen, `虚拟窗口未裁剪 DOM 文本: ${JSON.stringify(result)}`).toBeLessThan(8000);
    expect(
      result.zeroScrollFrames,
      `贴底跟随期间 scrollTop 闪 0 过多: ${JSON.stringify(result)}`,
    ).toBeLessThanOrEqual(2);
    expect(result.stepCardFlips, `步骤卡片因 key 变化被重建: ${JSON.stringify(result)}`).toBe(0);
    expect(execLogFetchCount).toBeGreaterThan(0);
  });
});
