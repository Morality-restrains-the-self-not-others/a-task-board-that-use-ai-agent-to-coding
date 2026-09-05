// @ts-check
/**
 * relayToTrae 全流程：直接启动 → zTree →「在 somanyad 中用 js 写 hello world」→ 完成 → 提交 → 推送。
 *
 * 推送链路（点击 zTree「推送」后）：
 * 1. 浏览器 POST …/cloud/compute/container-layer-git-push/（SaaS Django）
 * 2. Django 按任务仓库 provider 换 GitLab/GitHub OAuth token，组装 oauth_auth_by_repo
 * 3. Django POST {container}/api/layers/{layer_id}/git/oauth-access-push（非 prefer_container_remote）
 * 4. onlineServiceJS 对层内各 git 根目录执行 HTTPS git push（localhost GitLab 依赖 oauth_auth_by_repo canonical key）
 * 5. Django 可选 follow-up：GitHub PR（GitLab 仓跳过）
 *
 * 环境：localhost:4000 前端、relayToTrae、onlineServiceJS、GitLab :8012、gitOauth。
 * 凭据：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD（勿写入仓库）。
 *
 * 调试推送卡住：设 PLAYWRIGHT_TRACE_PUSH=1 在控制台打印上述各步请求；设 PLAYWRIGHT_MOCK_PUSH=1 仅 mock 第 1 步响应以隔离容器。
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = process.env.PLAYWRIGHT_RELAY_TASK_ID || '846269443533955072';
const ACCESS_CODE = process.env.PLAYWRIGHT_ACCESS_CODE || 'u824976301710503936';
const TASK_DETAIL_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?relayToTrae=true&accessCode=${ACCESS_CODE}`;

const SITE_BASE = (process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:4000').replace(/\/$/, '');
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const PROMPT =
  '在 somanyad 中，用 js 写一个 hello world。请在仓库根目录创建 hello.js，执行 node hello.js 时在控制台输出 hello world。';

const QUICK_CREATE_VALUE = '__quick_create__';

/** @typedef {{ method: string, url: string, status?: number, phase: string }} PushTraceEntry */

/**
 * @param {import('@playwright/test').Page} page
 * @returns {PushTraceEntry[]}
 */
function attachPushFlowTracer(page) {
  /** @type {PushTraceEntry[]} */
  const trace = [];
  const tracePush = process.env.PLAYWRIGHT_TRACE_PUSH === '1';

  page.on('request', (req) => {
    const url = req.url();
    const method = req.method();
    if (url.includes('container-layer-git-push') && method === 'POST') {
      trace.push({ method, url, phase: 'saas_push_start' });
      if (tracePush) console.log('[push-trace] →', method, url);
    }
    if (url.includes('/git/oauth-access-push') || (url.includes('/api/layers/') && url.includes('/git/push'))) {
      trace.push({ method, url, phase: 'container_git_push' });
      if (tracePush) console.log('[push-trace] container', method, url);
    }
  });

  page.on('response', async (resp) => {
    const url = resp.url();
    const method = resp.request().method();
    if (url.includes('container-layer-git-push') && method === 'POST') {
      const entry = { method, url, status: resp.status(), phase: 'saas_push_done' };
      trace.push(entry);
      if (tracePush) {
        const body = (await resp.text().catch(() => '')).slice(0, 500);
        console.log('[push-trace] ←', resp.status(), body);
      }
    }
  });

  return trace;
}

async function ensureGitIdentitySynced(page) {
  const sel = page.locator('#layer-git-identity-select');
  await expect(sel).toBeVisible({ timeout: 120000 });

  const optionValues = await sel.locator('option').evaluateAll((opts) =>
    opts.map((o) => /** @type {HTMLOptionElement} */ (o).value).filter(Boolean),
  );
  const realId = optionValues.find((v) => v !== QUICK_CREATE_VALUE);

  if (realId) {
    await sel.selectOption(realId);
  } else {
    await sel.selectOption(QUICK_CREATE_VALUE);
    await page.getByPlaceholder('Git 用户名').fill('e2e-playwright');
    await page.getByPlaceholder('Git 邮箱').fill('e2e-playwright@example.com');
    await page.getByRole('button', { name: '创建并同步' }).click();
    await expect(page.getByText('当前身份已同步至容器，可进行提交/推送。')).toBeVisible({ timeout: 120000 });
    return;
  }

  await page.getByRole('button', { name: '将各仓库所选身份同步到容器' }).click();
  await expect(page.getByText('当前身份已同步至容器，可进行提交/推送。')).toBeVisible({ timeout: 120000 });
}

async function firstEnabledZtreeActionButton(ztreePanel, actionLabel) {
  const buttons = ztreePanel.getByRole('button', { name: actionLabel });
  await expect(buttons.first()).toBeVisible({ timeout: 120000 });
  const n = await buttons.count();
  for (let i = 0; i < n; i++) {
    const b = buttons.nth(i);
    if (await b.isEnabled().catch(() => false)) {
      return b;
    }
  }
  throw new Error(`zTree 内没有已启用的「${actionLabel}」按钮`);
}

/** Django /api/auth/ 若被 runserver 单线程长请求占满，登录会卡在「登录中…」；推送 E2E 前须可快速响应。 */
async function assertDjangoAuthResponsive() {
  const djangoBase = (process.env.PLAYWRIGHT_DJANGO_BASE || 'http://127.0.0.1:8001').replace(/\/$/, '');
  const ping = await fetch(`${djangoBase}/api/core/test-number-serialization/`, {
    signal: AbortSignal.timeout(5000),
  }).catch(() => null);
  if (!ping?.ok) {
    throw new Error(`Django 不可达：${djangoBase}（请先启动 runserver）`);
  }
  const authProbe = await fetch(`${djangoBase}/api/auth/`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: '__e2e_probe__', password: '__e2e_probe__' }),
    signal: AbortSignal.timeout(8000),
  }).catch(() => null);
  if (!authProbe) {
    throw new Error(
      `Django POST /api/auth/ 在 8s 内无响应（runserver 可能被未结束的 container-layer-git-push 等长请求阻塞，请重启 Django 后再跑本用例）`,
    );
  }
}

test.describe('TaskDetail relayToTrae somanyad hello world 提交/推送', () => {
  test('直接启动 → 指令 → completed → 提交 → 推送（记录推送链路）', async ({ page }) => {
    test.skip(process.env.PRE_COMMIT === '1', 'pre-commit 不跑全栈 relay 用例');
    test.skip(!EMAIL || !PASSWORD, '请设置 PLAYWRIGHT_TEST_EMAIL 与 PLAYWRIGHT_TEST_PASSWORD');
    test.setTimeout(3600000);

    await assertDjangoAuthResponsive();

    page.on('dialog', (d) => void d.accept());

    if (process.env.PLAYWRIGHT_MOCK_PUSH === '1') {
      await page.route('**/container-layer-git-push/**', async (route) => {
        if (route.request().method() !== 'POST') {
          await route.continue();
          return;
        }
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ ok: true, playwright_mock_push: true }),
        });
      });
    }

    const pushTrace = attachPushFlowTracer(page);

    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      baseURL: SITE_BASE,
    });
    await page.waitForURL((u) => !u.pathname.includes('/auth/login'), { timeout: 90000 });

    const taskUrl = `${SITE_BASE}${TASK_DETAIL_PATH.startsWith('/') ? TASK_DETAIL_PATH : `/${TASK_DETAIL_PATH}`}`;
    await page.goto(taskUrl, { waitUntil: 'commit', timeout: 120000 });
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1500);
    if (page.url().includes('/auth/login')) {
      throw new Error(
        '未登录：请设置 PLAYWRIGHT_TEST_EMAIL/PASSWORD，或改用 CDP 脚本 tests/TaskDetail.relay-push-verify-cdp.mjs（需 Chrome --remote-debugging-port=9222 且已登录）',
      );
    }

    const relayTab = page.getByTestId('server-config-relay-direct-tab');
    await expect(relayTab).toBeVisible({ timeout: 60000 });
    await relayTab.click();

    const statusRow = page.getByTestId('relay-to-trae-status-row');
    const alreadyRunning = await statusRow.getByText('onlineServiceJS：运行中').isVisible().catch(() => false);
    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    await expect(startBtn).toBeVisible({ timeout: 60000 });
    if (!alreadyRunning && (await startBtn.isEnabled().catch(() => false))) {
      await startBtn.click();
    }

    await expect
      .poll(
        async () => {
          const t = (await statusRow.innerText().catch(() => '')) || '';
          if (/onlineServiceJS：运行中/i.test(t)) return true;
          const logs = (await page.locator('.font-mono.whitespace-pre-wrap').last().innerText().catch(() => '')).slice(-1200);
          if (/onlineServiceJS exited|exited \(code=1\)|EADDRINUSE|address already in use/i.test(logs)) {
            throw new Error(`relay/onlineServiceJS 启动失败：\n${logs}`);
          }
          return false;
        },
        { timeout: 300000, intervals: [2000, 3000, 5000, 10000] },
      )
      .toBe(true);

    await expect
      .poll(
        async () => {
          if (
            await page.getByTestId('container-http-unreachable-banner').isVisible().catch(() => false)
          ) {
            throw new Error('容器无法连接，zTree 不会出现');
          }
          return page.getByTestId('comment-layer-ztree-panel').isVisible().catch(() => false);
        },
        { timeout: 1200000, intervals: [800, 2000, 5000, 10000] },
      )
      .toBe(true);

    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    const nameBtns = ztreePanel.locator('button.break-words.flex-1.min-w-0');
    await expect(nameBtns.first()).toBeVisible({ timeout: 120000 });
    const n = await nameBtns.count();
    let clickedLayer = false;
    for (let i = 0; i < n; i++) {
      const label = ((await nameBtns.nth(i).innerText()) || '').trim();
      if (label === '可写层' || /^可写层（/.test(label)) continue;
      if (/idle_done|idle_interrupt/i.test(label)) continue;
      await nameBtns.nth(i).click();
      clickedLayer = true;
      break;
    }
    expect(clickedLayer, '应存在可选层级节点').toBe(true);
    await page.waitForTimeout(400);

    await page.locator('#layer-graph-command-input').fill(PROMPT);
    const sendBtn = page.locator('#layer-graph-command-send-btn');
    await expect(sendBtn).toBeEnabled({ timeout: 180000 });
    await Promise.all([
      page.waitForResponse(
        (r) =>
          r.url().includes('container-layer-command') &&
          r.request().method() === 'POST' &&
          r.status() === 200,
        { timeout: 300000 },
      ),
      sendBtn.click(),
    ]);

    const execPanel = page.getByTestId('comment-layer-ztree-exec-log-panel');
    await expect
      .poll(
        async () => {
          const t = ((await execPanel.innerText().catch(() => '')) || '').trim();
          return /completed|interrupted|克隆完成|bootstrap.*完成|hello|hello\.js|node/i.test(t);
        },
        { timeout: 900000, intervals: [2000, 5000, 10000] },
      )
      .toBe(true);

    await expect
      .poll(
        async () => {
          const submits = ztreePanel.getByRole('button', { name: '提交' });
          for (let i = 0; i < (await submits.count()); i++) {
            if (await submits.nth(i).isEnabled()) return true;
          }
          return false;
        },
        { timeout: 600000, intervals: [2000, 4000] },
      )
      .toBe(true);

    await ensureGitIdentitySynced(page);

    const submitBtn = await firstEnabledZtreeActionButton(ztreePanel, '提交');
    const [commitResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('container-layer-git-commit') && r.request().method() === 'POST',
        { timeout: 300000 },
      ),
      submitBtn.click(),
    ]);
    expect(commitResp.status()).toBe(200);

    await expect
      .poll(
        async () => {
          const pushBtns = ztreePanel.getByRole('button', { name: '推送' });
          for (let i = 0; i < (await pushBtns.count()); i++) {
            if (await pushBtns.nth(i).isEnabled()) return true;
          }
          return false;
        },
        { timeout: 300000, intervals: [1000, 3000] },
      )
      .toBe(true);

    const pushBtn = await firstEnabledZtreeActionButton(ztreePanel, '推送');
    const pushStart = Date.now();
    const [pushResp] = await Promise.all([
      page.waitForResponse(
        (r) =>
          r.url().includes('container-layer-git-push') && r.request().method() === 'POST',
        { timeout: 120000 },
      ),
      pushBtn.click(),
    ]);
    const pushMs = Date.now() - pushStart;
    console.log(`[push-verify] SaaS 推送耗时 ${pushMs}ms`);
    expect(pushMs, '本地全栈推送应在 120s 内返回（修复前可能无限 busy）').toBeLessThan(120000);

    const pushBodyStr = await pushResp.text();
    expect(pushResp.status(), `推送 SaaS 响应: ${pushBodyStr.slice(0, 800)}`).toBe(200);

    const rawPushPost = pushResp.request().postData();
    const pushPost = rawPushPost ? JSON.parse(rawPushPost) : {};
    expect(pushPost?.prefer_container_remote, 'relay 单仓应走平台 OAuth 换票').not.toBe(true);
    expect(String(pushPost?.identity_id || '').trim().length > 0, '推送须携带 identity_id').toBe(true);

    expect(
      pushTrace.some((e) => e.phase === 'saas_push_start'),
      '应发起 container-layer-git-push',
    ).toBe(true);
    expect(
      pushTrace.some((e) => e.phase === 'saas_push_done' && e.status === 200),
      'SaaS 推送应返回 200',
    ).toBe(true);

    if (process.env.PLAYWRIGHT_MOCK_PUSH === '1') {
      expect(pushBodyStr).toContain('playwright_mock_push');
    } else {
      let pushJson = {};
      try {
        pushJson = JSON.parse(pushBodyStr);
      } catch {
        /* ignore */
      }
      expect(pushJson.ok === true || pushJson.github_oauth_multirepo, '容器推送应成功').toBeTruthy();
    }

    await expect(pushBtn).toBeEnabled({ timeout: 120000 });
  });
});
