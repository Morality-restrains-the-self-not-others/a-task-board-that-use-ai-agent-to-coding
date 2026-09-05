// @ts-check
/**
 * 服务器生命周期 E2E 测试（纯 mock，不依赖真实阿里云）
 *
 * OPT-20260719-012: 硬件卡不再绑定运行态；停机后亦无 start-server-disabled-reason / #start-server-btn
 *   — mock runtime_status=Stopped, 验证硬件卡无启停控件
 *
 * OPT-20260721-003: 停机后服务器启动状态不再显示「已启动」
 *   — mock SSE 停止进度事件后 runtime_status 变为 Stopped, 验证生命周期状态显示「已停止」
 *
 * OPT-20260723-030: runtime_status 为 null + 「该任务尚未创建云实例」时生命周期状态
 *   — 先通过 SSE 注入启动成功状态, 再 mock runtime_status=null, 验证生命周期变为「已停止」
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

/* ==================== 共享 SSE 事件负载 ==================== */

/** 启动成功 SSE：用于设置本地 isServerRunning=true */
const START_SUCCESS_EVENT = {
  status: 'success',
  message: 'CLOUD_SERVER_STARTED',
  progress: 100,
  event_name: 'server_status_update',
};

/** 停止进度 SSE：模拟后端正在停止服务器 */
const STOP_PROGRESS_EVENT = {
  status: 'processing',
  message: '正在调用aliyunAPI停止服务器...',
  progress: 50,
  event_name: 'server_status_update',
};

/** SSE close 事件：通知前端停止完成 */
const STOP_CLOSE_EVENT = {
  type: 'close',
  event_name: 'server_status_update',
};

/* ==================== 共享 runtime-status 响应 ==================== */

/** 运行中 */
const RUNNING_RUNTIME = {
  status: 'success',
  runtime_status: 'Running',
  instance_id: 'i-e2e-lifecycle',
  platform: 'aliyun',
  region: 'cn-hangzhou',
  instance_attribute: {
    body: {
      Status: 'Running',
      InstanceId: 'i-e2e-lifecycle',
    },
  },
  public_ip: '203.0.113.10',
};

/** 已停止 */
const STOPPED_RUNTIME = {
  status: 'success',
  runtime_status: 'Stopped',
  instance_id: 'i-e2e-lifecycle',
  platform: 'aliyun',
  region: 'cn-hangzhou',
  instance_attribute: {
    body: {
      Status: 'Stopped',
      InstanceId: 'i-e2e-lifecycle',
    },
  },
  public_ip: '',
};

/** 无实例（runtime_status 为 null） */
const NULL_RUNTIME = {
  status: 'success',
  runtime_status: null,
  message: '该任务尚未创建云实例',
  instance_id: '',
  platform: '',
  region: '',
  instance_attribute: null,
};

/* ==================== 基础设置工具函数 ==================== */

/**
 * 安装 MockEventSource，支持通过 window.__emitStartupSse 推送 SSE 事件
 */
async function installMockSse(page) {
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
          try { cb(ev); } catch (_) {}
        }
      }

      /** 向 onmessage 推送 SSE 数据 */
      emit(payload) {
        if (this._closed) return;
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
}

/**
 * 设置测试所需的 cookie
 */
async function setupCookies(page) {
  await page.context().addCookies([
    { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
    { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
  ]);
}

/**
 * 构建公共的任务详情 mock 响应
 */
function taskDetailMock() {
  return {
    id: TASK_ID,
    title: 'Server Lifecycle E2E',
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    created_by: { username: 'mock-user' },
    comments: [],
    ai_comments: [],
    assignees: [],
    workspace_id: WORKSPACE_ID,
  };
}

/**
 * 设置 API route 拦截，通过 runtimeResponses 数组控制 server-runtime-status 的多次返回值
 * @param {import('@playwright/test').Page} page
 * @param {object[]} runtimeResponses — 每次 GET server-runtime-status 依次返回的对象
 */
async function setupApiRoutes(page, runtimeResponses) {
  let runtimeCallIndex = 0;

  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    // 任务详情
    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(taskDetailMock()) });
      return;
    }

    // 协作者列表
    if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    // Progress system
    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: [] }) });
      return;
    }

    // 容器 UI 上下文
    if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', container_endpoint_registered: false, container_page_url: '' }),
      });
      return;
    }

    // 容器层图
    if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ layers_root: '/', bootstrap_layer_id: 'l1', layers: [], jobs: [] }),
      });
      return;
    }

    // 前次服务器配置
    if (url.includes('/cloud/compute/previous-server-config/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', server_config: null }),
      });
      return;
    }

    // 模拟日志（mock-trae-online-log）
    if (url.includes('/mock-trae-online-log/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ log: '', in_progress: false }),
      });
      return;
    }

    // ===== 核心：server-runtime-status =====
    if (url.includes('/cloud/compute/server-runtime-status/') && method === 'GET') {
      const idx = Math.min(runtimeCallIndex, runtimeResponses.length - 1);
      runtimeCallIndex++;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(runtimeResponses[idx]),
      });
      return;
    }

    // 其他 API 返回空对象
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

/* ==================== 测试套件 ==================== */

test.describe('TaskDetail 服务器生命周期 — 停机后状态', () => {
  /**
   * OPT-20260719-012: 硬件卡与运行态解耦
   *
   * 场景：服务器已停止（runtime_status=Stopped），验证：
   * 1. 无 start-server-disabled-reason / #start-server-btn
   * 2. 评论提交按钮仍可见
   */
  test('OPT-012: 硬件卡无启停按钮与已在运行提示，评论区仍可提交', async ({ page }) => {
    await installMockSse(page);
    await setupCookies(page);

    await setupApiRoutes(page, [STOPPED_RUNTIME]);

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    await page.waitForTimeout(2000);

    await expect(page.getByTestId('start-server-disabled-reason')).toHaveCount(0);
    await expect(page.locator('#start-server-btn')).toHaveCount(0);
    await expect(page.getByTestId('task-detail-comment-submit')).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('task-detail-comment-submit')).toContainText('提交评论');
  });

  /**
   * OPT-20260721-003: 停机后服务器启动状态不再显示「已启动」
   *
   * 场景：先模拟运行中状态，再通过 SSE 模拟停止过程，验证生命周期状态文字变化
   * 1. 初始 runtime_status=Running → lifecycle-status 为「已启动」
   * 2. 发送 stop-progress SSE → 再发送 close
   * 3. 页面重新获取 runtime_status=Stopped → lifecycle-status 变为「已停止」
   */
  test('OPT-003: 停机后 lifecycle-status 不再显示已启动, 变为已停止', async ({ page }) => {
    await installMockSse(page);
    await setupCookies(page);

    // 第一次请求返回 Running, 后续返回 Stopped
    await setupApiRoutes(page, [RUNNING_RUNTIME, STOPPED_RUNTIME]);

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 等待页面渲染并在 Vue 建立 SSE 连接后检查初始状态
    await page.waitForFunction(
      () => window.__mockEventSource && !window.__mockEventSource._closed,
      null,
      { timeout: 15000 }
    );

    // 初始状态应为「已启动」；停止控件在评论执行细节
    const lifecycleStatus = page.getByTestId('server-lifecycle-status');
    await expect(lifecycleStatus).toHaveText('已启动', { timeout: 30000 });
    await expect(page.getByTestId('comment-runtime-stop-server-btn')).toBeVisible({ timeout: 10000 });

    // 模拟停止流程：先发进度事件，再发 close
    await page.evaluate((payload) => {
      window.__emitStartupSse(payload);
    }, STOP_PROGRESS_EVENT);

    // 等待 UI 响应进度事件
    await page.waitForTimeout(500);

    // 发 close 事件，触发前端重新获取 runtime-status
    await page.evaluate((payload) => {
      window.__emitStartupSse(payload);
    }, STOP_CLOSE_EVENT);

    // 此时 setupApiRoutes 的第二次调用返回 STOPPED_RUNTIME
    // 等待生命周期状态更新为「已停止」
    await expect(lifecycleStatus).toHaveText('已停止', { timeout: 30000 });
  });

  /**
   * OPT-20260723-030: runtime_status 为 null + 「该任务尚未创建云实例」
   *
   * 场景：本地 isServerRunning=true（通过 start-success SSE 注入），
   * 但后端返回 runtime_status=null，验证生命周期状态降级为「已停止」
   *
   * 流程：
   * 1. 初始 runtime_status=Running → lifecycle-status 为「已启动」
   * 2. 发送 start-success SSE 确保本地状态为 isServerRunning=true
   * 3. 发送 close SSE 触发页面重新获取 runtime-status
   * 4. 后续请求返回 runtime_status=null + 「该任务尚未创建云实例」
   * 5. lifecycle-status 应回退为「已停止」
   */
  test('OPT-030: runtime_status=null 该任务尚未创建云实例时 lifecycle 变为已停止', async ({ page }) => {
    await installMockSse(page);
    await setupCookies(page);

    // 第一次请求 Running, 后续返回 null
    await setupApiRoutes(page, [RUNNING_RUNTIME, NULL_RUNTIME]);

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 等待 Vue 建立 SSE 连接
    await page.waitForFunction(
      () => window.__mockEventSource && !window.__mockEventSource._closed,
      null,
      { timeout: 15000 }
    );

    // 初始状态应为「已启动」
    const lifecycleStatus = page.getByTestId('server-lifecycle-status');
    await expect(lifecycleStatus).toHaveText('已启动', { timeout: 30000 });

    // 注入 start-success SSE 确保本地 isServerRunning=true
    await page.evaluate((payload) => {
      window.__emitStartupSse(payload);
    }, START_SUCCESS_EVENT);

    await page.waitForTimeout(300);

    // 发送 close 触发重新获取 runtime-status
    await page.evaluate((payload) => {
      window.__emitStartupSse(payload);
    }, STOP_CLOSE_EVENT);

    // 后续 API 调用返回 NULL_RUNTIME（runtime_status=null）
    // 生命周期应降级为「已停止」
    await expect(lifecycleStatus).toHaveText('已停止', { timeout: 30000 });
  });
});
