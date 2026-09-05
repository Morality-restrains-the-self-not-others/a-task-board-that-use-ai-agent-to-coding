// @ts-check
/**
 * 核验任务详情「服务器运行状态」中停止服务器：发起 POST stop-vm，并在云平台拒绝时展示错误文案。
 * 与手工 URL 对齐；接口 route mock，不依赖真实阿里云。
 *
 * 依赖本地 Vue（默认 localhost:4000）：使用 playwright.verify.config.js。
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

/** @param {import('@playwright/test').Page} page */
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
    }

    window.EventSource = MockEventSource;
    window.__emitStartupSse = (payload) => {
      window.__mockEventSource?.emit?.(payload);
    };
  }, { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID });
}

/** @param {import('@playwright/test').Page} page */
async function commonCookies(page) {
  await page.context().addCookies([
    { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
    { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
  ]);
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ runtimeBody: Record<string, unknown>, stopHandler?: (route: import('@playwright/test').Route) => Promise<void> }} opts
 */
async function routeApiMocks(page, opts) {
  let stopPostCount = 0;

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
          title: 'stop-vm e2e',
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

    if (url.includes('/cloud/compute/server-runtime-status/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(opts.runtimeBody),
      });
      return;
    }

    if (url.includes('/cloud/compute/stop-vm/') && method === 'POST') {
      stopPostCount += 1;
      if (opts.stopHandler) {
        await opts.stopHandler(route);
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', message: 'mock stop accepted' }),
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
          container_page_url: '',
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

  return {
    getStopPostCount: () => stopPostCount,
  };
}

test.describe('TaskDetail 服务器运行状态 — 停止服务器（mock API）', () => {
  test('运行中：点击「停止服务器」应 POST stop-vm（200）', async ({ page }) => {
    await installMockSse(page);
    await commonCookies(page);

    const runningPayload = {
      status: 'success',
      runtime_status: 'Running',
      message: '',
      instance_attribute: {
        body: {
          Status: 'Running',
          InstanceId: 'i-mock-stop-e2e',
          PublicIpAddress: { IpAddress: ['203.0.113.10'] },
        },
      },
    };

    const { getStopPostCount } = await routeApiMocks(page, { runtimeBody: runningPayload });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    await page.getByRole('button', { name: '服务器运行状态' }).click();

    const stopBtn = page.getByRole('button', { name: '停止服务器' });
    await expect(stopBtn).toBeVisible({ timeout: 30000 });

    await stopBtn.click();

    await expect.poll(() => getStopPostCount(), { timeout: 15000 }).toBe(1);
  });

  test('云平台 Initializing：403 时应写入启动状态区的错误文案', async ({ page }) => {
    await installMockSse(page);
    await commonCookies(page);

    const runningPayload = {
      status: 'success',
      runtime_status: 'Running',
      message: '',
      instance_attribute: {
        body: {
          Status: 'Running',
          InstanceId: 'i-mock',
          PublicIpAddress: { IpAddress: ['203.0.113.11'] },
        },
      },
    };

    const errMsg =
      '停止虚拟机失败: Error: IncorrectInstanceStatus.Initializing code: 403, The specified instance status does not support this operation. request id: 48C9ABB9-FC7C-3747-9E2F-2DA68288D5DE';

    await routeApiMocks(page, {
      runtimeBody: runningPayload,
      stopHandler: async (route) => {
        await route.fulfill({
          status: 403,
          contentType: 'application/json',
          body: JSON.stringify({ message: errMsg }),
        });
      },
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    await page.getByRole('button', { name: '服务器运行状态' }).click();

    await page.getByRole('button', { name: '停止服务器' }).click();

    await expect(page.getByRole('heading', { name: '服务器启动状态' })).toBeVisible({ timeout: 15000 });
    await expect(page.getByText(/IncorrectInstanceStatus\.Initializing/).first()).toBeVisible({
      timeout: 15000,
    });
  });

  test('实例为 Initializing：不应展示「停止服务器」按钮', async ({ page }) => {
    await installMockSse(page);
    await commonCookies(page);

    const initPayload = {
      status: 'success',
      runtime_status: 'Initializing',
      message: '',
      instance_attribute: {
        body: {
          Status: 'Initializing',
          InstanceId: 'i-mock-init',
          PublicIpAddress: { IpAddress: [] },
        },
      },
    };

    await routeApiMocks(page, { runtimeBody: initPayload });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    await page.getByRole('button', { name: '服务器运行状态' }).click();

    await expect(page.getByText('初始化中', { exact: true }).first()).toBeVisible({ timeout: 30000 });
    await expect(page.getByRole('button', { name: '停止服务器' })).toHaveCount(0);
  });
});
