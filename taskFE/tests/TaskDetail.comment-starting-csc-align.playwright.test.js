// @ts-check
/**
 * 意图 T8 / OPT-20260812-011 + OPT-20260813-019: 评论启动中 + 评论级 CSC 双面板对齐
 * 且切到「服务器运行状态」Tab 后仍可见容器名 / CSC / 启动 TraceId。
 *
 * 场景：启动中绑定仅有评论级 CSC（无 instance_id）时，
 * 「服务器运行状态」Tab 应展示「启动中」+「云实例创建中，等待分配」；
 * 容器元信息（容器名 / CSC / 启动 TraceId）由两 Tab 共享，切 Tab 后仍可见。
 *
 * 纯 mock（假 Cookie + page.route 拦截全部 /api/ + MockEventSource），
 * 不依赖真实登录与阿里云。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662912';
// OPT-20260813-019: 评论 id 对齐 STARTING_RUNTIME.comment_id，使运行时面板与容器绑定同属一条评论
const COMMENT_ID = 'comment-mock-starting';
const MOCK_CONTAINER_NAME = 'mock-container-opt019';
const MOCK_CSC_ID = 'csc-mock-opt019';
const MOCK_START_TRACE_ID = 'trace-opt019-0001';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`
);

/** 评论级 CSC 启动中：后端无 instance_id，返回 Starting + 云实例创建中 */
const STARTING_RUNTIME = {
  status: 'success',
  runtime_status: 'Starting',
  instance_id: null,
  csc_id: MOCK_CSC_ID,
  comment_id: COMMENT_ID,
  platform: 'aliyun',
  region: 'cn-hangzhou',
  message: '云实例创建中，等待分配',
};

function taskDetailMock() {
  return {
    id: TASK_ID,
    title: 'Comment Starting + CSC Align E2E',
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    created_by: { username: 'mock-user' },
    comments: [
      {
        id: COMMENT_ID,
        content: '启动评论：容器元信息回归',
        created_at: '2026-01-01T00:00:00Z',
        created_by: { username: 'mock-user' },
      },
    ],
    ai_comments: [],
    assignees: [],
    workspace_id: WORKSPACE_ID,
  };
}

async function installMockSse(page) {
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
        this._closed = false;
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
        this._closed = true;
        if (typeof this.onclose === 'function') this.onclose();
      }
    }
    window.EventSource = MockEventSource;
  });
}

async function setupCookies(page) {
  await page.context().addCookies([
    { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
    { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
  ]);
}

async function setupApiRoutes(page) {
  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(taskDetailMock()) });
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
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', container_endpoint_registered: false, container_page_url: '' }),
      });
      return;
    }

    if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ layers_root: '/', bootstrap_layer_id: 'l1', layers: [], jobs: [] }),
      });
      return;
    }

    if (url.includes('/cloud/compute/previous-server-config/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', server_config: null }) });
      return;
    }

    if (url.includes('/mock-trae-online-log/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ log: '', in_progress: false }) });
      return;
    }

    // 核心：server-runtime-status → 启动中（评论级 CSC 无 instance）
    if (url.includes('/cloud/compute/server-runtime-status/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(STARTING_RUNTIME),
      });
      return;
    }

    // OPT-20260813-019: 人类评论 Feed — fetchTaskDetail 单独拉取并覆盖 task.comments
    if (url.includes(`/api/tasks/${TASK_ID}/comments/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: COMMENT_ID,
            content: '启动评论：容器元信息回归',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
          },
        ]),
      });
      return;
    }

    // AI 评论 Feed（空）
    if (url.includes('/ai-comment/task-detail/') && url.includes('/ai-comments/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    // 容器 agent 评论 Feed（空）
    if (url.includes('/container-agent-comments/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    // OPT-20260813-019: comment-container-bindings GET → mock 评论的容器绑定
    // （容器名 / CSC / 启动 TraceId 均来自该绑定，供两 Tab 共享的容器元信息渲染）
    if (url.includes('comment-container-bindings') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          bindings: [
            {
              comment_id: COMMENT_ID,
              status: 'starting',
              container_name: MOCK_CONTAINER_NAME,
              csc_id: MOCK_CSC_ID,
              start_trace_id: MOCK_START_TRACE_ID,
              logs: [],
            },
          ],
        }),
      });
      return;
    }

    // ensure / advance POST（syncBindingsForComments 内部调用）
    if (url.includes('comment-container-bindings') && method === 'POST') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true }) });
      return;
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

test.describe('任务详情 — 评论启动中 + 评论级 CSC 双面板对齐 + 容器元信息', () => {
  test('Starting 时「服务器运行状态」Tab 展示启动中，切 Tab 后容器元信息三项仍可见', async ({ page }) => {
    test.setTimeout(120000);
    await installMockSse(page);
    await setupCookies(page);
    await setupApiRoutes(page);

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 有 mock 评论（绑定 status=starting）→ 该评论成为「当前执行」，出现「服务器运行状态」Tab
    const runtimeTab = page.getByTestId('comment-execution-tab-server-runtime');
    await expect(runtimeTab).toBeVisible({ timeout: 45000 });

    // OPT-20260813-019: 容器元信息在 Tab 面板外，两 Tab 共享 — 切 Tab 前已可见三项
    const containerMeta = page.getByTestId('comment-execution-container-meta');
    await expect(containerMeta).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('comment-execution-container-name-full')).toHaveText(MOCK_CONTAINER_NAME);
    await expect(page.getByTestId('comment-execution-csc-id')).toHaveText(MOCK_CSC_ID);
    const startTrace = page.getByTestId('comment-execution-start-trace-id');
    await expect(startTrace).toBeVisible({ timeout: 15000 });
    await expect(startTrace).toHaveAttribute('data-traceId', MOCK_START_TRACE_ID);
    await expect(page.getByTestId('comment-execution-start-trace-id-value')).toHaveText(MOCK_START_TRACE_ID);

    // 服务器启动状态 = 启动中（绑定 status=starting → 生命周期「启动中」）
    const lifecycle = page.getByTestId('server-lifecycle-status');
    await expect(lifecycle).toBeVisible({ timeout: 15000 });
    await expect(lifecycle).toHaveText('启动中');

    // 点「服务器运行状态」Tab 后容器元信息仍在（两 Tab 共享，不在 Tab 面板内卸载）
    await runtimeTab.click();
    await expect(containerMeta).toBeVisible({ timeout: 15000 });
    await expect(page.getByTestId('comment-execution-container-name-full')).toHaveText(MOCK_CONTAINER_NAME);
    await expect(page.getByTestId('comment-execution-csc-id')).toHaveText(MOCK_CSC_ID);
    await expect(page.getByTestId('comment-execution-start-trace-id-value')).toHaveText(MOCK_START_TRACE_ID);
  });
});
