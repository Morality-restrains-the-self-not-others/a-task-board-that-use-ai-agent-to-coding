// @ts-check
/**
 * 回归：relayToTrae=true 时「直接启动」面板登记 relay 任务、经 SSE 收状态（不再轮询 /v1/status），
 * 运行中显示「停止」并调用 /v1/stop。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '843742455533076480';
const TASK_DETAIL_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?relayToTrae=true&accessCode=u824976301710503936`;

const RELAY_UI_URL = 'http://127.0.0.1:8765/ui/e2e-access-token';

test.describe('TaskDetail relayToTrae direct start', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑 relay 全栈 E2E');

  test('进入直接启动时查询状态，运行中可停止 onlineServiceJS', async ({ page }) => {
    let healthRequestCount = 0;
    let registerRequestCount = 0;
    let statusRequestCount = 0;
    let stopRequestCount = 0;
    let startRequestCount = 0;
    let tokenInitRequestCount = 0;
    let precheckRequestCount = 0;
    /** @type {null | Record<string, unknown>} */
    let startRequestBody = null;
    let onlineServiceRunning = true;

    await page.addInitScript(
      ({ tenantId, workspaceId, taskId }) => {
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
            window.__mockEventSources = window.__mockEventSources || [];
            window.__mockEventSources.push(this);
            setTimeout(() => {
              this._emitOpen();
              const expected = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`;
              if (!this.url.includes(expected)) return;
            }, 20);
          }

          addEventListener(type, cb) {
            if (!this._listeners[type]) this._listeners[type] = [];
            this._listeners[type].push(cb);
          }

          emit(data) {
            if (typeof this.onmessage === 'function') {
              this.onmessage({ data: JSON.stringify(data) });
            }
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
        }

        // @ts-ignore
        window.EventSource = MockEventSource;
      },
      { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID },
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
            title: 'Relay Task',
            description: 'Mock',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            container_image_id: 'img-relay-1',
          }),
        });
        return;
      }

      if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'img-relay-1', name: 'Relay Image', version: '1.0' }]),
        });
        return;
      }

      if (url.includes('/relay-to-trae/env-prepare/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            env: {
              TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
              BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
              ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
            },
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/health/') && method === 'GET') {
        healthRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/register/') && method === 'POST') {
        registerRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', task_id: TASK_ID }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/stop/') && method === 'POST') {
        stopRequestCount += 1;
        onlineServiceRunning = false;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', killed_pids: [] }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/start/') && method === 'POST') {
        startRequestCount += 1;
        startRequestBody = req.postDataJSON();
        onlineServiceRunning = true;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'ok',
            port: 8765,
            ui_url: RELAY_UI_URL,
            pid: 5151,
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/token-init/') && method === 'POST') {
        tokenInitRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'ok',
            token_initialized: true,
            task_id: TASK_ID,
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/repo-credentials-precheck/') && method === 'POST') {
        precheckRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'ok',
            message: '仓库凭证预检通过',
            repo_count: 1,
          }),
        });
        return;
      }

      if (url.includes('/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', runtime_status: 'Stopped' }),
        });
        return;
      }

      if (url.includes('/server-content/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', content: '', http_status: 200 }),
        });
        return;
      }

      if (url.includes('/cloud/compute/server-start-history/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', records: [] }),
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
          body: JSON.stringify({ status: 'success', container_endpoint_registered: false }),
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

    const relayTab = page.getByTestId('server-config-relay-direct-tab');
    await expect(relayTab).toBeVisible({ timeout: 30000 });
    await relayTab.click();

    const statusRow = page.getByTestId('relay-to-trae-status-row');
    await expect(statusRow).toBeVisible();
    await expect(statusRow.getByText('relayToTrae 服务：在线')).toBeVisible({ timeout: 15000 });

    await expect
      .poll(() => healthRequestCount, { timeout: 15000, message: '应请求 /api/.../relay-to-trae/health/' })
      .toBeGreaterThan(0);
    expect(registerRequestCount, '页面初始化阶段最多触发一次 register').toBeLessThanOrEqual(1);
    expect(statusRequestCount, '不应再轮询 relay /v1/status').toBe(0);

    await expect(page.getByTestId('relay-to-trae-access-token-masked')).toBeVisible();
    await expect(page.getByTestId('relay-to-trae-access-token-masked')).toHaveValue('由服务端签发（不展示）');

    await page.evaluate((uiUrl) => {
      const payload = {
        status: 'relay_to_trae_status',
        event_name: 'server_status_update',
        relay_payload: {
          running: true,
          online_service_up: true,
          port_listening: true,
          orphan_port: false,
          ui_url: uiUrl,
          logs: ['[onlineServiceJS] server listening'],
        },
      };
      for (const es of window.__mockEventSources || []) {
        es.emit(payload);
      }
    }, RELAY_UI_URL);

    await expect(statusRow.getByText('onlineServiceJS：运行中')).toBeVisible({ timeout: 10000 });

    const stopBtn = page.getByTestId('relay-to-trae-stop-btn');
    await expect(stopBtn).toBeVisible();
    await expect(stopBtn).toBeEnabled();
    await stopBtn.click();

    await expect
      .poll(() => stopRequestCount, { timeout: 10000, message: '应调用 /api/.../relay-to-trae/stop/' })
      .toBeGreaterThan(0);

    await expect(statusRow.getByText('onlineServiceJS：未启动')).toBeVisible({ timeout: 10000 });
    await expect(stopBtn).toBeHidden();

    await page.getByTestId('relay-to-trae-status-refresh').click();

    onlineServiceRunning = false;
    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    await expect(startBtn).toBeEnabled();
    await startBtn.click();

    await expect
      .poll(() => startRequestCount, { timeout: 15000, message: '应调用 /api/.../relay-to-trae/start/' })
      .toBeGreaterThan(0);
    await expect
      .poll(() => tokenInitRequestCount, { timeout: 15000, message: '应先调用 token-init' })
      .toBeGreaterThan(0);
    await expect
      .poll(() => precheckRequestCount, { timeout: 15000, message: '应调用 repo-credentials-precheck' })
      .toBeGreaterThan(0);

    expect(startRequestBody?.env?.ACCESS_TOKEN).toBe('__TASK2APP_ACCESS_TOKEN__');
    expect(startRequestBody?.env?.DEBUG_AGENT).toBe('True');
    expect(startRequestBody?.task_id).toBe(TASK_ID);
    expect(startRequestBody?.installed_image_id).toBe('img-relay-1');

    await page.evaluate((uiUrl) => {
      const payload = {
        status: 'relay_to_trae_status',
        event_name: 'server_status_update',
        relay_payload: {
          running: true,
          online_service_up: true,
          port_listening: true,
          ui_url: uiUrl,
        },
      };
      for (const es of window.__mockEventSources || []) {
        es.emit(payload);
      }
    }, RELAY_UI_URL);

    await expect(stopBtn).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('link', { name: /打开容器页面|打开控制台/ })).toHaveAttribute('href', RELAY_UI_URL);
  });

  test('预检失败时应阻断 start 并展示缺失仓库提示', async ({ page }) => {
    let startRequestCount = 0;

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
            title: 'Relay Task',
            workspace_id: WORKSPACE_ID,
            container_image_id: 'img-relay-1',
            comments: [],
            ai_comments: [],
            assignees: [],
          }),
        });
        return;
      }
      if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'img-relay-1', name: 'Relay Image', version: '1.0' }]),
        });
        return;
      }
      if (url.includes('/relay-to-trae/env-prepare/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            env: {
              TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
              BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
              ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
            },
          }),
        });
        return;
      }
      if (url.includes('/relay-to-trae/health/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }
      if (url.includes('/relay-to-trae/register/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', task_id: TASK_ID }),
        });
        return;
      }
      if (url.includes('/relay-to-trae/token-init/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', token_initialized: true, task_id: TASK_ID }),
        });
        return;
      }
      if (url.includes('/relay-to-trae/repo-credentials-precheck/') && method === 'POST') {
        await route.fulfill({
          status: 409,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'error',
            error_code: 'REPO_CLONE_CREDENTIALS_INCOMPLETE',
            message: '任务仓库克隆凭证不完整',
            missing_repo_credentials: ['http://localhost:8012/demo/repo-a.git'],
          }),
        });
        return;
      }
      if (url.includes('/relay-to-trae/start/') && method === 'POST') {
        startRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'accepted', request_id: 'should-not-happen' }),
        });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.getByTestId('server-config-relay-direct-tab').click();
    await page.getByTestId('relay-to-trae-start-btn').click();

    await expect(page.getByText(/缺失仓库\(1\)/)).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('relay-to-trae-repo-credential-guide')).toBeVisible({ timeout: 10000 });
    expect(startRequestCount, '预检失败后不应调用 start').toBe(0);
  });

  test('relay 未连接时显示未连接提示', async ({ page }) => {
    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes('/relay-to-trae/health/') && method === 'GET') {
        await route.abort('connectionrefused');
        return;
      }

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Relay Task',
            workspace_id: WORKSPACE_ID,
            container_image_id: 'img-relay-1',
            comments: [],
            ai_comments: [],
            assignees: [],
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/env-prepare/')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            env: {
              TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
              BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
              ACCESS_TOKEN: 'e2e-access-token',
            },
          }),
        });
        return;
      }

      if (url.includes('/installed-images/')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      if (url.includes('/server-runtime-status/')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', runtime_status: 'Stopped' }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    await page.getByTestId('server-config-relay-direct-tab').click();
    const statusRow = page.getByTestId('relay-to-trae-status-row');
    await expect(statusRow.getByText('relayToTrae 服务：未连接')).toBeVisible({ timeout: 15000 });
    await expect(page.getByText(/无法连接本机 relayToTrae/)).toBeVisible();
    await expect(page.getByTestId('relay-to-trae-stop-btn')).toBeHidden();
  });

  test('刷新状态时 health 短暂失败不应误判 relay 未连接', async ({ page }) => {
    let healthRequestCount = 0;
    let statusRequestCount = 0;

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes('/relay-to-trae/health/') && method === 'GET') {
        healthRequestCount += 1;
        if (healthRequestCount === 1) {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ status: 'ok' }),
          });
          return;
        }
        await route.abort('timedout');
        return;
      }

      if (url.includes('/relay-to-trae/clear-logs/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'ok',
            cleared: true,
            tenant_id: TENANT_ID,
            workspace_id: WORKSPACE_ID,
            task_id: TASK_ID,
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/status/') && method === 'GET') {
        statusRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            running: false,
            online_service_up: false,
            orphan_port: false,
            logs: ['status-probe-ok'],
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/register/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', task_id: TASK_ID }),
        });
        return;
      }

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Relay Task',
            workspace_id: WORKSPACE_ID,
            container_image_id: 'img-relay-1',
            comments: [],
            ai_comments: [],
            assignees: [],
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/env-prepare/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            env: {
              TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
              BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
              ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
            },
          }),
        });
        return;
      }

      if (url.includes('/installed-images/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'img-relay-1', name: 'Relay Image', version: '1.0' }]),
        });
        return;
      }

      if (url.includes('/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', runtime_status: 'Stopped' }),
        });
        return;
      }

      if (url.includes('/server-content/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', content: '', http_status: 200 }),
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
          body: JSON.stringify({ status: 'success', container_endpoint_registered: false }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.getByTestId('server-config-relay-direct-tab').click();

    const statusRow = page.getByTestId('relay-to-trae-status-row');
    await expect(statusRow.getByText('relayToTrae 服务：在线')).toBeVisible({ timeout: 15000 });
    await expect
      .poll(() => statusRequestCount, { timeout: 10000, message: '初次进入应请求 status' })
      .toBeGreaterThan(0);

    await page.getByTestId('relay-to-trae-status-refresh').click();

    await expect
      .poll(() => healthRequestCount, { timeout: 10000, message: '刷新应触发第二次 health（失败）' })
      .toBeGreaterThan(1);
    await expect
      .poll(() => statusRequestCount, { timeout: 10000, message: 'health 失败后应回退请求 status' })
      .toBeGreaterThan(1);
    await expect(statusRow.getByText('relayToTrae 服务：在线')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/无法连接本机 relayToTrae/)).toHaveCount(0);
  });

  test('停止后刷新状态时 health 失败仍应保持 relay 在线且 onlineService 未启动', async ({ page }) => {
    let healthRequestCount = 0;
    let stopRequestCount = 0;

    await page.addInitScript(
      ({ tenantId, workspaceId, taskId }) => {
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
            window.__mockEventSources = window.__mockEventSources || [];
            window.__mockEventSources.push(this);
            setTimeout(() => {
              this._emitOpen();
            }, 20);
          }

          addEventListener(type, cb) {
            if (!this._listeners[type]) this._listeners[type] = [];
            this._listeners[type].push(cb);
          }

          emit(data) {
            if (typeof this.onmessage === 'function') {
              this.onmessage({ data: JSON.stringify(data) });
            }
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
        }

        // @ts-ignore
        window.EventSource = MockEventSource;
      },
      { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID },
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
            title: 'Relay Task',
            workspace_id: WORKSPACE_ID,
            container_image_id: 'img-relay-1',
            comments: [],
            ai_comments: [],
            assignees: [],
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/health/') && method === 'GET') {
        healthRequestCount += 1;
        if (healthRequestCount <= 1) {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ status: 'ok' }),
          });
          return;
        }
        await route.abort('timedout');
        return;
      }

      if (url.includes('/relay-to-trae/register/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', task_id: TASK_ID }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/clear-logs/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'ok',
            cleared: true,
            tenant_id: TENANT_ID,
            workspace_id: WORKSPACE_ID,
            task_id: TASK_ID,
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            running: false,
            online_service_up: false,
            orphan_port: false,
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/stop/') && method === 'POST') {
        stopRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/env-prepare/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            env: {
              TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
              BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
              ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
            },
          }),
        });
        return;
      }

      if (url.includes('/installed-images/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'img-relay-1', name: 'Relay Image', version: '1.0' }]),
        });
        return;
      }

      if (url.includes('/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', runtime_status: 'Stopped' }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.getByTestId('server-config-relay-direct-tab').click();

    const statusRow = page.getByTestId('relay-to-trae-status-row');
    await expect(statusRow.getByText('relayToTrae 服务：在线')).toBeVisible({ timeout: 15000 });

    await page.evaluate((uiUrl) => {
      const payload = {
        status: 'relay_to_trae_status',
        event_name: 'server_status_update',
        relay_payload: {
          running: true,
          online_service_up: true,
          port_listening: true,
          ui_url: uiUrl,
        },
      };
      for (const es of window.__mockEventSources || []) {
        es.emit(payload);
      }
    }, RELAY_UI_URL);

    await expect(statusRow.getByText('onlineServiceJS：运行中')).toBeVisible({ timeout: 10000 });

    const stopBtn = page.getByTestId('relay-to-trae-stop-btn');
    await stopBtn.click();
    await expect
      .poll(() => stopRequestCount, { timeout: 10000, message: '应调用 stop' })
      .toBeGreaterThan(0);
    await expect(statusRow.getByText('onlineServiceJS：未启动')).toBeVisible({ timeout: 10000 });

    await page.getByTestId('relay-to-trae-status-refresh').click();

    await expect
      .poll(() => healthRequestCount, { timeout: 10000, message: '刷新应再次触发 health' })
      .toBeGreaterThan(1);
    await expect(statusRow.getByText('relayToTrae 服务：在线')).toBeVisible({ timeout: 10000 });
    await expect(statusRow.getByText('onlineServiceJS：未启动')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/无法连接本机 relayToTrae/)).toHaveCount(0);
  });

  test('停止后刷新时 health 与 status 均失败仍应保持 relay 在线', async ({ page }) => {
    let healthRequestCount = 0;
    let statusRequestCount = 0;
    let stopRequestCount = 0;

    await page.addInitScript(
      ({ tenantId, workspaceId, taskId }) => {
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
            window.__mockEventSources = window.__mockEventSources || [];
            window.__mockEventSources.push(this);
            setTimeout(() => {
              this._emitOpen();
            }, 20);
          }

          addEventListener(type, cb) {
            if (!this._listeners[type]) this._listeners[type] = [];
            this._listeners[type].push(cb);
          }

          emit(data) {
            if (typeof this.onmessage === 'function') {
              this.onmessage({ data: JSON.stringify(data) });
            }
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
        }

        // @ts-ignore
        window.EventSource = MockEventSource;
      },
      { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID },
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
            title: 'Relay Task',
            workspace_id: WORKSPACE_ID,
            container_image_id: 'img-relay-1',
            comments: [],
            ai_comments: [],
            assignees: [],
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/health/') && method === 'GET') {
        healthRequestCount += 1;
        if (healthRequestCount <= 1) {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ status: 'ok' }),
          });
          return;
        }
        await route.abort('timedout');
        return;
      }

      if (url.includes('/relay-to-trae/register/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', task_id: TASK_ID }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/clear-logs/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'ok',
            cleared: true,
            tenant_id: TENANT_ID,
            workspace_id: WORKSPACE_ID,
            task_id: TASK_ID,
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/status/') && method === 'GET') {
        statusRequestCount += 1;
        if (statusRequestCount <= 1) {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
              running: false,
              online_service_up: false,
              orphan_port: false,
            }),
          });
          return;
        }
        await route.fulfill({
          status: 502,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'error', message: 'gateway' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/stop/') && method === 'POST') {
        stopRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/env-prepare/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            env: {
              TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
              BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
              ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
            },
          }),
        });
        return;
      }

      if (url.includes('/installed-images/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'img-relay-1', name: 'Relay Image', version: '1.0' }]),
        });
        return;
      }

      if (url.includes('/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', runtime_status: 'Stopped' }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.getByTestId('server-config-relay-direct-tab').click();

    const statusRow = page.getByTestId('relay-to-trae-status-row');
    await expect(statusRow.getByText('relayToTrae 服务：在线')).toBeVisible({ timeout: 15000 });

    await page.evaluate((uiUrl) => {
      const payload = {
        status: 'relay_to_trae_status',
        event_name: 'server_status_update',
        relay_payload: {
          running: true,
          online_service_up: true,
          port_listening: true,
          ui_url: uiUrl,
        },
      };
      for (const es of window.__mockEventSources || []) {
        es.emit(payload);
      }
    }, RELAY_UI_URL);

    await expect(statusRow.getByText('onlineServiceJS：运行中')).toBeVisible({ timeout: 10000 });

    await page.getByTestId('relay-to-trae-stop-btn').click();
    await expect
      .poll(() => stopRequestCount, { timeout: 10000, message: '应调用 stop' })
      .toBeGreaterThan(0);
    await expect(statusRow.getByText('onlineServiceJS：未启动')).toBeVisible({ timeout: 10000 });

    await page.getByTestId('relay-to-trae-status-refresh').click();

    await expect
      .poll(() => healthRequestCount, { timeout: 10000, message: '刷新应再次触发 health' })
      .toBeGreaterThan(1);
    await expect
      .poll(() => statusRequestCount, { timeout: 10000, message: 'health 失败后应回退 status' })
      .toBeGreaterThan(1);
    await expect(statusRow.getByText('relayToTrae 服务：在线')).toBeVisible({ timeout: 10000 });
    await expect(statusRow.getByText('onlineServiceJS：未启动')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/无法连接本机 relayToTrae/)).toHaveCount(0);
  });

  test('清理日志后刷新状态不应回填历史日志', async ({ page }) => {
    let statusRequestCount = 0;
    let clearLogsRequestCount = 0;
    /** @type {string[]} */
    const clearLogsPaths = [];
    const replayLogs = ['legacy-log-line-1', 'legacy-log-line-2'];

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
            title: 'Relay Task',
            workspace_id: WORKSPACE_ID,
            container_image_id: 'img-relay-1',
            comments: [],
            ai_comments: [],
            assignees: [],
          }),
        });
        return;
      }

      if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'img-relay-1', name: 'Relay Image', version: '1.0' }]),
        });
        return;
      }

      if (url.includes('/relay-to-trae/env-prepare/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            env: {
              TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
              BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
              ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
            },
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/health/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/register/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/clear-logs/') && method === 'POST') {
        clearLogsRequestCount += 1;
        clearLogsPaths.push(new URL(url).pathname);
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'ok',
            cleared: true,
            tenant_id: TENANT_ID,
            workspace_id: WORKSPACE_ID,
            task_id: TASK_ID,
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/status/') && method === 'GET') {
        statusRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            running: true,
            online_service_up: true,
            active_task_id: TASK_ID,
            log_task_id: TASK_ID,
            logs: replayLogs,
          }),
        });
        return;
      }

      if (url.includes('/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', runtime_status: 'Stopped' }),
        });
        return;
      }

      if (url.includes('/server-content/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', content: '', http_status: 200 }),
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
          body: JSON.stringify({ status: 'success', container_endpoint_registered: false }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.getByTestId('server-config-relay-direct-tab').click();

    await expect
      .poll(() => statusRequestCount, { timeout: 10000, message: '初次进入应请求 relay status' })
      .toBeGreaterThan(0);
    await expect(page.getByText('legacy-log-line-1')).toBeVisible({ timeout: 10000 });

    await page.getByTestId('relay-to-trae-logs-clear').click();
    await expect(page.getByText('legacy-log-line-1')).toHaveCount(0);
    await expect
      .poll(() => clearLogsRequestCount, { timeout: 10000, message: '清理日志应调用 clear-logs' })
      .toBe(1);
    expect(clearLogsPaths[0]).toContain(
      `/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/task_id/${TASK_ID}relay-to-trae/clear-logs`,
    );

    await page.getByTestId('relay-to-trae-status-refresh').click();
    await expect
      .poll(() => statusRequestCount, { timeout: 10000, message: '刷新状态应再次请求 relay status' })
      .toBeGreaterThan(1);
    await expect(page.getByText('legacy-log-line-1')).toHaveCount(0);
    await expect(page.getByText('legacy-log-line-2')).toHaveCount(0);
  });

  test('清理日志后再次启动不应回填历史日志', async ({ page }) => {
    let startRequestCount = 0;
    const replayLogs = ['legacy-log-line-1', 'legacy-log-line-2'];

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
            title: 'Relay Task',
            workspace_id: WORKSPACE_ID,
            container_image_id: 'img-relay-1',
            comments: [],
            ai_comments: [],
            assignees: [],
          }),
        });
        return;
      }

      if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'img-relay-1', name: 'Relay Image', version: '1.0' }]),
        });
        return;
      }

      if (url.includes('/relay-to-trae/env-prepare/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            env: {
              TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
              BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
              ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
            },
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/health/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/register/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/token-init/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', token_initialized: true, task_id: TASK_ID }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/repo-credentials-precheck/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', message: '仓库凭证预检通过', repo_count: 1 }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/start/') && method === 'POST') {
        startRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'accepted', request_id: `start-${startRequestCount}` }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/clear-logs/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'ok',
            cleared: true,
            tenant_id: TENANT_ID,
            workspace_id: WORKSPACE_ID,
            task_id: TASK_ID,
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            running: false,
            online_service_up: false,
            active_task_id: TASK_ID,
            log_task_id: TASK_ID,
            logs: replayLogs,
          }),
        });
        return;
      }

      if (url.includes('/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', runtime_status: 'Stopped' }),
        });
        return;
      }

      if (url.includes('/server-content/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', content: '', http_status: 200 }),
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
          body: JSON.stringify({ status: 'success', container_endpoint_registered: false }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.getByTestId('server-config-relay-direct-tab').click();

    await expect(page.getByText('legacy-log-line-1')).toBeVisible({ timeout: 10000 });

    await page.getByTestId('relay-to-trae-logs-clear').click();
    await expect(page.getByText('legacy-log-line-1')).toHaveCount(0);

    await page.getByTestId('relay-to-trae-start-btn').click();
    await expect
      .poll(() => startRequestCount, { timeout: 15000, message: '清理后启动应调用 start' })
      .toBe(1);

    await expect(page.getByText('legacy-log-line-1')).toHaveCount(0);
    await expect(page.getByText('legacy-log-line-2')).toHaveCount(0);
  });

  test('启动后清理日志再刷新状态不应回填历史日志', async ({ page }) => {
    let startRequestCount = 0;
    let statusRequestCount = 0;
    const replayLogs = ['legacy-log-line-1', 'legacy-log-line-2'];

    await page.addInitScript(
      ({ tenantId, workspaceId, taskId }) => {
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
            window.__mockEventSources = window.__mockEventSources || [];
            window.__mockEventSources.push(this);
            setTimeout(() => this._emitOpen(), 20);
          }

          addEventListener(type, cb) {
            if (!this._listeners[type]) this._listeners[type] = [];
            this._listeners[type].push(cb);
          }

          emit(data) {
            if (typeof this.onmessage === 'function') {
              this.onmessage({ data: JSON.stringify(data) });
            }
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
        }

        // @ts-ignore
        window.EventSource = MockEventSource;
      },
      { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID },
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
            title: 'Relay Task',
            workspace_id: WORKSPACE_ID,
            container_image_id: 'img-relay-1',
            comments: [],
            ai_comments: [],
            assignees: [],
          }),
        });
        return;
      }

      if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'img-relay-1', name: 'Relay Image', version: '1.0' }]),
        });
        return;
      }

      if (url.includes('/relay-to-trae/env-prepare/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            env: {
              TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
              BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
              ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
            },
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/health/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/register/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/token-init/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', token_initialized: true, task_id: TASK_ID }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/repo-credentials-precheck/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', message: '仓库凭证预检通过', repo_count: 1 }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/start/') && method === 'POST') {
        startRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'accepted', request_id: `start-${startRequestCount}` }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/clear-logs/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'ok',
            cleared: true,
            tenant_id: TENANT_ID,
            workspace_id: WORKSPACE_ID,
            task_id: TASK_ID,
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/status/') && method === 'GET') {
        statusRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            running: true,
            port_listening: true,
            online_service_up: true,
            active_task_id: TASK_ID,
            logs: replayLogs,
          }),
        });
        return;
      }

      if (url.includes('/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', runtime_status: 'Stopped' }),
        });
        return;
      }

      if (url.includes('/server-content/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', content: '', http_status: 200 }),
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
          body: JSON.stringify({ status: 'success', container_endpoint_registered: false }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.getByTestId('server-config-relay-direct-tab').click();

    await page.getByTestId('relay-to-trae-start-btn').click();
    await expect
      .poll(() => startRequestCount, { timeout: 15000, message: '应调用 start' })
      .toBe(1);

    await page.evaluate(
      ({ logs, taskId }) => {
        const payload = {
          status: 'relay_to_trae_status',
          event_name: 'server_status_update',
          relay_payload: {
            running: true,
            port_listening: true,
            online_service_up: true,
            active_task_id: taskId,
            logs,
          },
        };
        for (const es of window.__mockEventSources || []) {
          es.emit(payload);
        }
      },
      { logs: replayLogs, taskId: TASK_ID },
    );

    await expect(page.getByText('legacy-log-line-1')).toBeVisible({ timeout: 10000 });

    await page.getByTestId('relay-to-trae-logs-clear').click();
    await expect(page.getByText('legacy-log-line-1')).toHaveCount(0);

    await page.getByTestId('relay-to-trae-status-refresh').click();
    await expect
      .poll(() => statusRequestCount, { timeout: 10000, message: '刷新状态应请求 relay status' })
      .toBeGreaterThan(0);

    await page.evaluate(
      ({ logs, taskId }) => {
        const payload = {
          status: 'relay_to_trae_status',
          event_name: 'server_status_update',
          relay_payload: {
            running: true,
            port_listening: true,
            online_service_up: true,
            active_task_id: taskId,
            logs,
          },
        };
        for (const es of window.__mockEventSources || []) {
          es.emit(payload);
        }
      },
      { logs: replayLogs, taskId: TASK_ID },
    );

    await expect(page.getByText('legacy-log-line-1')).toHaveCount(0);
    await expect(page.getByText('legacy-log-line-2')).toHaveCount(0);
  });

  test('启动后日志尚未出现时清理日志仍应抑制刷新回填', async ({ page }) => {
    let startRequestCount = 0;
    const replayLogs = ['legacy-log-line-1', 'legacy-log-line-2'];

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
            title: 'Relay Task',
            workspace_id: WORKSPACE_ID,
            container_image_id: 'img-relay-1',
            comments: [],
            ai_comments: [],
            assignees: [],
          }),
        });
        return;
      }

      if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'img-relay-1', name: 'Relay Image', version: '1.0' }]),
        });
        return;
      }

      if (url.includes('/relay-to-trae/env-prepare/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            env: {
              TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
              BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
              ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
            },
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/health/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/register/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/token-init/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', token_initialized: true, task_id: TASK_ID }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/repo-credentials-precheck/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', message: '仓库凭证预检通过', repo_count: 1 }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/start/') && method === 'POST') {
        startRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'accepted', request_id: `start-${startRequestCount}` }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/clear-logs/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'ok',
            cleared: true,
            tenant_id: TENANT_ID,
            workspace_id: WORKSPACE_ID,
            task_id: TASK_ID,
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            running: true,
            port_listening: true,
            online_service_up: true,
            active_task_id: TASK_ID,
            logs: replayLogs,
          }),
        });
        return;
      }

      if (url.includes('/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', runtime_status: 'Stopped' }),
        });
        return;
      }

      if (url.includes('/server-content/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', content: '', http_status: 200 }),
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
          body: JSON.stringify({ status: 'success', container_endpoint_registered: false }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.getByTestId('server-config-relay-direct-tab').click();

    await page.getByTestId('relay-to-trae-start-btn').click();
    await expect
      .poll(() => startRequestCount, { timeout: 15000, message: '应调用 start' })
      .toBe(1);

    const clearBtn = page.getByTestId('relay-to-trae-logs-clear');
    await expect(clearBtn).toBeEnabled();
    await clearBtn.click();

    await page.getByTestId('relay-to-trae-status-refresh').click();
    await expect(page.getByText('legacy-log-line-1')).toHaveCount(0);
    await expect(page.getByText('legacy-log-line-2')).toHaveCount(0);
  });
});
