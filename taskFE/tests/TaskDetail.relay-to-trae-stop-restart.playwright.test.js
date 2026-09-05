// @ts-check
/**
 * 回归：stop → 再次 start 不应出现 onlineServiceJS exited (code=1)。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '843742455533076480';
const TASK_DETAIL_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?relayToTrae=true&accessCode=u824976301710503936`;
const RELAY_UI_URL = 'http://127.0.0.1:8765/ui/e2e-restart-token';

test('stop 后再次 start 不应显示 onlineServiceJS exited (code=1)', async ({ page }) => {
  test.setTimeout(90_000);

  let startRequestCount = 0;
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
          this.onmessage = null;
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
        }

        _emitOpen() {
          for (const cb of this._listeners.open || []) {
            try {
              cb({ type: 'open' });
            } catch (_) {}
          }
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

  const emitRunning = async () => {
    await page.evaluate((uiUrl) => {
      const payload = {
        status: 'relay_to_trae_status',
        event_name: 'server_status_update',
        relay_payload: {
          running: true,
          online_service_up: true,
          port_listening: true,
          ui_url: uiUrl,
          error: '',
        },
      };
      for (const es of window.__mockEventSources || []) {
        es.emit(payload);
      }
    }, RELAY_UI_URL);
  };

  const emitStopped = async () => {
    await page.evaluate(() => {
      const payload = {
        status: 'relay_to_trae_status',
        event_name: 'server_status_update',
        relay_payload: {
          running: false,
          online_service_up: false,
          port_listening: false,
          error: '',
        },
      };
      for (const es of window.__mockEventSources || []) {
        es.emit(payload);
      }
    });
  };

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
          title: 'Relay restart',
          comments: [],
          ai_comments: [],
          assignees: [],
          workspace_id: WORKSPACE_ID,
          container_image_id: 'img-relay-1',
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

    if (url.includes('/relay-to-trae/health/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"ok"}' });
      return;
    }

    if (url.includes('/relay-to-trae/register/') && method === 'POST') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"ok"}' });
      return;
    }

    if (url.includes('/relay-to-trae/stop/') && method === 'POST') {
      stopRequestCount += 1;
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"ok"}' });
      return;
    }

    if (url.includes('/relay-to-trae/start/') && method === 'POST') {
      startRequestCount += 1;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'accepted', request_id: `req-${startRequestCount}` }),
      });
      return;
    }

    if (url.includes('/relay-to-trae/token-init/') && method === 'POST') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"ok"}' });
      return;
    }

    if (url.includes('/relay-to-trae/repo-credentials-precheck/') && method === 'POST') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"ok"}' });
      return;
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await page.goto(TASK_DETAIL_PATH);
  await page.waitForLoadState('domcontentloaded');

  const relayTab = page.getByTestId('server-config-relay-direct-tab');
  await expect(relayTab).toBeVisible({ timeout: 30_000 });
  await relayTab.click();

  const statusRow = page.getByTestId('relay-to-trae-status-row');
  const startBtn = page.getByTestId('relay-to-trae-start-btn');
  const stopBtn = page.getByTestId('relay-to-trae-stop-btn');

  await emitRunning();
  await expect(statusRow.getByText('onlineServiceJS：运行中')).toBeVisible({ timeout: 10_000 });
  await expect(stopBtn).toBeVisible();

  await stopBtn.click();
  await expect.poll(() => stopRequestCount).toBeGreaterThan(0);
  await emitStopped();
  await expect(statusRow.getByText('onlineServiceJS：未启动')).toBeVisible({ timeout: 10_000 });

  await startBtn.click();
  await expect.poll(() => startRequestCount).toBe(1);
  await emitRunning();
  await expect(statusRow.getByText('onlineServiceJS：运行中')).toBeVisible({ timeout: 10_000 });
  await expect(page.getByText(/onlineServiceJS exited \(code=1\)/i)).toHaveCount(0);

  await stopBtn.click();
  await emitStopped();
  await startBtn.click();
  await expect.poll(() => startRequestCount).toBe(2);
  await emitRunning();
  await expect(statusRow.getByText('onlineServiceJS：运行中')).toBeVisible({ timeout: 10_000 });
  await expect(page.getByText(/exited \(code=/i)).toHaveCount(0);
});
