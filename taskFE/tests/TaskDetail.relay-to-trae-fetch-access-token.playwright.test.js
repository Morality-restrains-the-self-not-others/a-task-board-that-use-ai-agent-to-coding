// @ts-check
/**
 * relayToTrae 直接启动 → 打开控制台 → 拉取 AccessToken
 *
 * - mock：不依赖本机 8765，验证控制台按钮在请求结束后恢复文案
 * - 集成：PLAYWRIGHT_RELAY_FETCH_TOKEN=1 + 登录凭据 + 本机 relay/onlineServiceJS
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const SITE_ORIGIN = String(process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://localhost:4000').replace(/\/$/, '');
const MOCK_TASK_ID = '843742455533076480';
const INTEGRATION_TASK_ID = process.env.PLAYWRIGHT_RELAY_TASK_ID || '846269443533955072';
const MOCK_TASK_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${MOCK_TASK_ID}/?relayToTrae=true&accessCode=u824976301710503936`;
const INTEGRATION_TASK_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${INTEGRATION_TASK_ID}/?relayToTrae=true`;

const RELAY_UI_URL = 'http://127.0.0.1:8765/ui/e2e-fetch-token-mock';

const INTEGRATION = process.env.PLAYWRIGHT_RELAY_FETCH_TOKEN === '1';

const MOCK_CONSOLE_HTML = `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
<button type="button" data-testid="layer-git-oauth-fetch-token-files">\u62c9\u53d6AccessToken</button>
<script>
const btn = document.querySelector('[data-testid="layer-git-oauth-fetch-token-files"]');
btn.onclick = async () => {
  btn.disabled = true;
  btn.setAttribute('aria-busy', 'true');
  btn.textContent = '\u62c9\u53d6\u4e2d\u2026';
  try {
    const r = await fetch('/api/layers/mock-layer/git/oauth-fetch-token-files', { method: 'POST' });
    await r.json();
  } finally {
    btn.textContent = '\u62c9\u53d6AccessToken';
    btn.disabled = false;
    btn.removeAttribute('aria-busy');
  }
};
</script></body></html>`;

test.describe('TaskDetail relayToTrae → console → 拉取 AccessToken', () => {
  test('mock：端口就绪后才显示打开控制台；控制台拉取后按钮恢复', async ({ page, context }) => {
    test.setTimeout(90_000);

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

        window.EventSource = MockEventSource;
      },
      { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: MOCK_TASK_ID },
    );

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: `${SITE_ORIGIN}/` },
      { name: 'csrftoken', value: 'e2e-csrf', url: `${SITE_ORIGIN}/` },
    ]);

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${MOCK_TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: MOCK_TASK_ID,
            title: 'Fetch Token Mock',
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
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'ok' }) });
        return;
      }
      if (url.includes('/relay-to-trae/register/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', task_id: MOCK_TASK_ID }),
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
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await context.route('http://127.0.0.1:8765/**', async (route) => {
      const url = route.request().url();
      if (url.includes('oauth-fetch-token-files') && route.request().method() === 'POST') {
        await new Promise((r) => setTimeout(r, 300));
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ ok: true, token_files: [{ write_ok: true, github_slug: 'o/r' }] }),
        });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'text/html', body: MOCK_CONSOLE_HTML });
    });

    await page.goto(`${SITE_ORIGIN}${MOCK_TASK_PATH}`);
    await page.waitForLoadState('domcontentloaded');
    const relayTab = page.getByTestId('server-config-relay-direct-tab');
    await expect(relayTab).toBeVisible({ timeout: 30_000 });
    await relayTab.click();

    const statusRow = page.getByTestId('relay-to-trae-status-row');
    const consoleLink = page.getByRole('link', { name: '打开控制台' });

    await page.evaluate((uiUrl) => {
      const payload = {
        status: 'relay_to_trae_status',
        relay_payload: {
          running: true,
          online_service_up: true,
          port_listening: false,
          ui_url: uiUrl,
        },
      };
      for (const es of window.__mockEventSources || []) {
        es.emit(payload);
      }
    }, RELAY_UI_URL);

    await expect(consoleLink).toHaveCount(0);

    await page.evaluate((uiUrl) => {
      const payload = {
        status: 'relay_to_trae_status',
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

    await expect(statusRow.getByText('onlineServiceJS：运行中')).toBeVisible({ timeout: 10_000 });
    await expect(consoleLink).toBeVisible({ timeout: 10_000 });

    const consolePage = await context.newPage();
    await consolePage.goto(RELAY_UI_URL);
    const fetchBtn = consolePage.getByTestId('layer-git-oauth-fetch-token-files');
    await fetchBtn.click();
    await expect(fetchBtn).toHaveText('拉取AccessToken', { timeout: 15_000 });
    await expect(fetchBtn).toBeEnabled();
    await consolePage.close();
  });

  test('集成：直接启动、打开控制台、拉取 AccessToken 不长时间 pending', async ({ page, context }) => {
    test.setTimeout(180_000);
    const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
    const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
    test.skip(!INTEGRATION || !email || !password, 'Set PLAYWRIGHT_RELAY_FETCH_TOKEN=1 and login env vars');

    await playwrightLoginWithLegalAccept(page, { email, password, baseURL: SITE_ORIGIN });

    await page.goto(`${SITE_ORIGIN}${INTEGRATION_TASK_PATH}`);
    await page.waitForLoadState('domcontentloaded');

    const relayTab = page.getByTestId('server-config-relay-direct-tab');
    await expect(relayTab).toBeVisible({ timeout: 30_000 });
    await relayTab.click();

    const statusRow = page.getByTestId('relay-to-trae-status-row');
    await expect(statusRow.getByText(/relayToTrae 服务：在线/)).toBeVisible({ timeout: 30_000 });

    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    const stopBtn = page.getByTestId('relay-to-trae-stop-btn');
    const consoleLink = page.getByRole('link', { name: '打开控制台' });

    if (await stopBtn.isVisible().catch(() => false)) {
      await stopBtn.click();
      await expect(statusRow.getByText(/onlineServiceJS：未启动/)).toBeVisible({ timeout: 60_000 });
    }

    if (await startBtn.isEnabled().catch(() => false)) {
      await startBtn.click();
    }

    await expect(statusRow.getByText('onlineServiceJS：运行中')).toBeVisible({ timeout: 120_000 });
    await expect(consoleLink).toBeVisible({ timeout: 120_000 });

    const consoleHref = await consoleLink.getAttribute('href');
    expect(consoleHref, '控制台链接应指向 onlineServiceJS /ui/').toMatch(/\/ui\//);

    const consolePage = await context.newPage();
    await consolePage.goto(consoleHref, { waitUntil: 'domcontentloaded', timeout: 60_000 });
    await expect(consolePage.getByTestId('layer-branch-graph')).toBeVisible({ timeout: 60_000 });

    const layerBtn = consolePage.locator('button.layer-serial-row').first();
    await expect(layerBtn).toBeVisible({ timeout: 30_000 });
    await layerBtn.click();

    const fetchBtn = consolePage.getByTestId('layer-git-oauth-fetch-token-files');
    await expect(fetchBtn).toBeVisible({ timeout: 15_000 });
    await expect(fetchBtn).toHaveText('拉取AccessToken');

    const responsePromise = consolePage.waitForResponse(
      (resp) =>
        resp.url().includes('/git/oauth-fetch-token-files') && resp.request().method() === 'POST',
      { timeout: 90_000 },
    );

    const clickAt = Date.now();
    await fetchBtn.click();

    const response = await responsePromise;
    const elapsed = Date.now() - clickAt;
    expect(elapsed, '拉取请求耗时应小于 90s').toBeLessThan(90_000);

    const bodyText = await response.text();
    let body = {};
    try {
      body = JSON.parse(bodyText);
    } catch {
      /* ignore */
    }

    if (!response.ok()) {
      throw new Error(`oauth-fetch-token-files HTTP ${response.status()}: ${bodyText.slice(0, 500)}`);
    }
    if (body && body.ok === false) {
      throw new Error(`oauth-fetch-token-files failed: ${body.detail || bodyText.slice(0, 500)}`);
    }

    await expect(fetchBtn).toHaveText('拉取AccessToken', { timeout: 20_000 });
    await expect(fetchBtn).toBeEnabled();
    await expect(fetchBtn).not.toHaveAttribute('aria-busy', 'true');

    await consolePage.close();
  });
});
