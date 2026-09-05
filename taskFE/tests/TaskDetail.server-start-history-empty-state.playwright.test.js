// @ts-check
/**
 * 意图 / OPT-20260811-054: 历史服务器启动记录空态仅一条文案
 *
 * 场景：fetch 空成功（records=[]，无 message 之外干扰）时，
 * 「历史服务器启动记录」面板只渲染 1 条「暂无历史服务器启动记录」，
 * 不再出现 message 与空态双重复制。
 *
 * 纯 mock（假 Cookie + page.route 拦截全部 /api/ + MockEventSource），
 * 不依赖真实登录与后端。
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

function taskDetailMock() {
  return {
    id: TASK_ID,
    title: 'Server Start History Empty E2E',
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    created_by: { username: 'mock-user' },
    comments: [],
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

async function setupApiRoutes(page, { taskPayload = taskDetailMock(), historyUrls } = {}) {
  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(taskPayload) });
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

    // 核心：server-start-history 空成功（无 message 干扰）
    if (url.includes('/cloud/compute/server-start-history/') && method === 'GET') {
      if (Array.isArray(historyUrls)) historyUrls.push(url);
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', records: [] }),
      });
      return;
    }

    // 运行状态：无实例，避免其它面板干扰
    if (url.includes('/cloud/compute/server-runtime-status/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', runtime_status: null, message: '该任务尚未创建云实例' }),
      });
      return;
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

test.describe('任务详情 — 历史服务器启动记录空态', () => {
  test('空成功仅渲染一条「暂无历史服务器启动记录」且不是缺少任务ID', async ({ page }) => {
    const historyUrls = [];
    await installMockSse(page);
    await setupCookies(page);
    await setupApiRoutes(page, { historyUrls });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 点击「历史服务器启动记录」Tab（自动展开内容区并触发 fetch）
    const historyTab = page.locator('button:has-text("历史服务器启动记录")').first();
    await expect(historyTab).toBeVisible({ timeout: 30000 });
    await historyTab.click();

    await expect.poll(() => historyUrls.length, { timeout: 15000 }).toBeGreaterThan(0);
    expect(historyUrls.some((u) => u.includes(`task_id=${TASK_ID}`))).toBe(true);

    // 空成功不写 message：message 节点不应出现，空态由面板「v-else-if」单条渲染
    await expect(page.getByTestId('server-start-history-message')).toHaveCount(0, { timeout: 15000 });
    await expect(page.getByText('缺少任务ID')).toHaveCount(0);

    await expect(page.getByText('暂无历史服务器启动记录')).toHaveCount(1, { timeout: 15000 });

    // 无空态卡片
    await expect(page.getByTestId('server-start-history-card')).toHaveCount(0);
  });
});
