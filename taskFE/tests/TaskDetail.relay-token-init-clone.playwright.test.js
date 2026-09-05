// @ts-check
/**
 * 回归：任务详情「直接启动」点击启动后，
 * POST .../relay-to-trae/token-init/ 不得再 501（taskCloudService not yet ported），
 * 须经 taskGateway → taskContainerGateway 成功签发；并继续 precheck → start。
 *
 * 含两层断言：
 * 1) API：登录后直调 token-init（主回归，稳定）
 * 2) UI：页面点击启动，捕获同一链路网络响应
 *
 * 运行：
 * PLAYWRIGHT_SITE_ORIGIN=http://127.0.0.1:4000 \
 * PLAYWRIGHT_GATEWAY_ORIGIN=http://127.0.0.1:18081 \
 * PLAYWRIGHT_RELAY_TASK_ID=task_12675381068363715869 \
 * PLAYWRIGHT_WORKSPACE_ID=861623708318031872 \
 * bash tests/TaskDetail.relay-token-init-clone.playwright.test.js.sh
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID =
  process.env.PLAYWRIGHT_WORKSPACE_ID || process.env.PW_WORKSPACE_ID || '861623708318031872';
const TASK_ID = process.env.PLAYWRIGHT_RELAY_TASK_ID || 'task_12675381068363715869';

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const ROUTING_FAILURE = 'compute action not yet ported to taskCloudService';
const TASK_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?relayToTrae=true`;
const TOKEN_INIT_PATH = `/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/task_id/${TASK_ID}relay-to-trae/token-init/`;

test.describe('TaskDetail relay token-init → start', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑 relay 全栈 E2E');

  test('API: token-init 经 TCG 成功（非 501）', async ({ page, request }) => {
    test.setTimeout(120_000);

    const loginData = await loginViaGatewayApi(page, {
      email: EMAIL,
      password: PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });
    const token = loginData.token;
    expect(token, 'login token').toBeTruthy();

    const resp = await request.post(`${GATEWAY}${TOKEN_INIT_PATH}`, {
      headers: {
        Authorization: `Token ${token}`,
        'Content-Type': 'application/json',
        Origin: SITE,
        Accept: 'application/json',
      },
      data: {
        env: {
          ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
          TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
          BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
        },
        tenant_id: TENANT_ID,
        workspace_id: WORKSPACE_ID,
        task_id: TASK_ID,
      },
    });

    const bodyText = await resp.text();
    expect(bodyText).not.toContain(ROUTING_FAILURE);
    expect(resp.status(), `token-init body=${bodyText.slice(0, 400)}`).not.toBe(501);
    expect(resp.status()).toBe(200);

    const parsed = JSON.parse(bodyText);
    expect(parsed.status).toBe('ok');
    expect(parsed.token_initialized).toBe(true);
  });

  test('UI: 点击启动走通 token-init → precheck → start', async ({ page }) => {
    test.setTimeout(360_000);

    /** @type {{ status: number, body: Record<string, unknown> } | null} */
    let tokenInitCapture = null;
    /** @type {{ status: number, body: Record<string, unknown> } | null} */
    let precheckCapture = null;
    /** @type {{ status: number, body: Record<string, unknown> } | null} */
    let startCapture = null;

    page.on('response', async (resp) => {
      const url = resp.url();
      const method = resp.request().method();
      let body = {};
      try {
        body = await resp.json();
      } catch {
        /* ignore */
      }
      if (url.includes('/relay-to-trae/token-init/') && method === 'POST') {
        tokenInitCapture = { status: resp.status(), body };
      }
      if (url.includes('/relay-to-trae/repo-credentials-precheck/') && method === 'POST') {
        precheckCapture = { status: resp.status(), body };
      }
      if (url.includes('/relay-to-trae/start/') && method === 'POST') {
        startCapture = { status: resp.status(), body };
      }
    });

    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      baseURL: SITE,
    });

    await page.goto(`${SITE}${TASK_PATH}`);
    await page.waitForLoadState('domcontentloaded');

    const relayTab = page.getByTestId('server-config-relay-direct-tab');
    await expect(relayTab).toBeVisible({ timeout: 60_000 });
    await relayTab.click();

    const statusRow = page.getByTestId('relay-to-trae-status-row');
    await expect(statusRow.getByText(/relayToTrae 服务：在线/)).toBeVisible({ timeout: 60_000 });

    const stopBtn = page.getByTestId('relay-to-trae-stop-btn');
    if (await stopBtn.isVisible().catch(() => false)) {
      await stopBtn.click();
      await expect(statusRow.getByText(/onlineServiceJS：未启动/)).toBeVisible({ timeout: 90_000 });
    }

    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    await expect(startBtn).toBeEnabled({ timeout: 30_000 });
    await startBtn.click();

    await expect
      .poll(() => tokenInitCapture?.status, {
        timeout: 60_000,
        message: '应完成 token-init',
      })
      .toBeDefined();

    expect(tokenInitCapture?.status, JSON.stringify(tokenInitCapture?.body).slice(0, 400)).not.toBe(501);
    expect(JSON.stringify(tokenInitCapture?.body || {})).not.toContain(ROUTING_FAILURE);
    expect(tokenInitCapture?.status).toBe(200);
    expect(String(tokenInitCapture?.body?.status || '')).toBe('ok');
    expect(tokenInitCapture?.body?.token_initialized).toBe(true);

    await expect
      .poll(() => precheckCapture?.status, {
        timeout: 90_000,
        message: '应完成 repo-credentials-precheck',
      })
      .toBeDefined();
    expect(precheckCapture?.status, JSON.stringify(precheckCapture?.body).slice(0, 400)).toBe(200);
    expect(String(precheckCapture?.body?.status || '')).toBe('ok');

    await expect
      .poll(() => startCapture?.status, {
        timeout: 60_000,
        message: '应完成 start',
      })
      .toBeDefined();
    expect(startCapture?.status).toBe(202);
    expect(String(startCapture?.body?.status || '').toLowerCase()).toBe('accepted');
    expect(String(startCapture?.body?.request_id || '')).not.toBe('');

    // 引导/克隆完成：必须出现明确「任务引导完成」；status-push 不得因 csc_* pk 500
    await expect
      .poll(
        async () => {
          const bodyText = await page.locator('body').innerText();
          if (bodyText.includes('INTERNAL_DISPATCH_ERROR')) return 'dispatch-500';
          if (bodyText.includes('无效的 access_token')) return 'invalid-token';
          if (bodyText.includes('任务引导完成')) return 'bootstrap-ok';
          return 'pending';
        },
        {
          timeout: 240_000,
          message: '启动后应出现「任务引导完成」，且无 status-push 500/401 文案',
        },
      )
      .toBe('bootstrap-ok');

    const finalText = await page.locator('body').innerText();
    expect(finalText).toContain('任务引导完成');
    expect(finalText).not.toContain('INTERNAL_DISPATCH_ERROR');
    expect(finalText).not.toContain('无效的 access_token');
  });
});
