// @ts-check
/**
 * 经 CDP 连接已登录 Chrome，核验 relay 任务页 zTree 推送（本地应较快）。
 *
 * 前置：Chrome --remote-debugging-port=9222；前端 :4000、Django :8001、relay、onlineServiceJS 已起。
 *
 *   cd taskFE
 *   node tests/TaskDetail.relay-push-verify-cdp.mjs
 *
 * 可选：PLAYWRIGHT_PUSH_ONLY=1 — 仅当 zTree 已就绪且「推送」可点时测推送（跳过启动/Agent）
 */
import { chromium } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const SITE = (process.env.SITE_BASE || 'http://localhost:4000').replace(/\/$/, '');
const TASK_URL =
  process.env.PLAYWRIGHT_TASK_URL ||
  `${SITE}/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/846269443533955072/?relayToTrae=true&accessCode=u824976301710503936`;
const CDP_URL = (process.env.CDP_URL || 'http://127.0.0.1:9222').replace(/\/$/, '');
const PROMPT =
  '在 somanyad 中，用 js 写一个 hello world。请在仓库根目录创建 hello.js，执行 node hello.js 时在控制台输出 hello world。';
const PUSH_ONLY = process.env.PLAYWRIGHT_PUSH_ONLY === '1';

/** @type {{ phase: string, method?: string, url?: string, status?: number, ms?: number }[]} */
const trace = [];

async function firstEnabledZtreeButton(panel, name) {
  const buttons = panel.getByRole('button', { name });
  const n = await buttons.count();
  for (let i = 0; i < n; i++) {
    if (await buttons.nth(i).isEnabled().catch(() => false)) return buttons.nth(i);
  }
  throw new Error(`zTree 无可用「${name}」按钮`);
}

function logTrace(entry) {
  trace.push(entry);
  console.log('[verify]', JSON.stringify(entry));
}

async function ensureRelayRunning(page) {
  const relayTab = page.getByTestId('server-config-relay-direct-tab');
  if (await relayTab.isVisible().catch(() => false)) {
    await relayTab.click();
  }
  const statusRow = page.getByTestId('relay-to-trae-status-row');
  await statusRow.waitFor({ state: 'visible', timeout: 60000 });
  const running = await statusRow.getByText('onlineServiceJS：运行中').isVisible().catch(() => false);
  if (!running) {
    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    if (await startBtn.isVisible().catch(() => false) && (await startBtn.isEnabled().catch(() => false))) {
      logTrace({ phase: 'relay_start_click' });
      await startBtn.click();
      await statusRow.getByText('onlineServiceJS：运行中').waitFor({ timeout: 300000 });
    }
  }
  logTrace({ phase: 'relay_running' });
}

async function waitZtree(page) {
  await page.getByTestId('comment-layer-ztree-panel').waitFor({ state: 'visible', timeout: 300000 });
  logTrace({ phase: 'ztree_visible' });
}

async function runAgentAndCommit(page) {
  const ztree = page.getByTestId('comment-layer-ztree-panel');
  const nameBtns = ztree.locator('button.break-words.flex-1.min-w-0');
  const n = await nameBtns.count();
  for (let i = 0; i < n; i++) {
    const label = ((await nameBtns.nth(i).innerText()) || '').trim();
    if (label === '可写层' || /^可写层（/.test(label)) continue;
    if (/idle_done|idle_interrupt/i.test(label)) continue;
    await nameBtns.nth(i).click();
    break;
  }
  await page.locator('#layer-graph-command-input').fill(PROMPT);
  const sendBtn = page.locator('#layer-graph-command-send-btn');
  await sendBtn.waitFor({ state: 'visible', timeout: 60000 });
  await sendBtn.click();
  const execPanel = page.getByTestId('comment-layer-ztree-exec-log-panel');
  await execPanel.getByText(/completed|克隆完成|hello/i).first().waitFor({ timeout: 600000 });
  logTrace({ phase: 'agent_done' });

  const submitBtn = await firstEnabledZtreeButton(ztree, '提交');
  await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('container-layer-git-commit') && r.request().method() === 'POST',
      { timeout: 180000 },
    ),
    submitBtn.click(),
  ]);
  logTrace({ phase: 'commit_done' });
}

async function verifyPush(page) {
  page.on('dialog', (d) => void d.accept());
  page.on('request', (req) => {
    const u = req.url();
    if (u.includes('container-layer-git-push') && req.method() === 'POST') {
      logTrace({ phase: 'saas_push_request', method: 'POST', url: u });
    }
  });

  const ztree = page.getByTestId('comment-layer-ztree-panel');
  const pushBtn = await firstEnabledZtreeButton(ztree, '推送');
  const t0 = Date.now();
  const [resp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('container-layer-git-push') && r.request().method() === 'POST',
      { timeout: 120000 },
    ),
    pushBtn.click(),
  ]);
  const ms = Date.now() - t0;
  const body = (await resp.text()).slice(0, 600);
  const post = resp.request().postData() || '';
  logTrace({ phase: 'saas_push_response', status: resp.status(), ms, url: resp.url() });
  console.log('[verify] push body:', body);
  console.log('[verify] push post:', post.slice(0, 300));

  if (resp.status() !== 200) {
    throw new Error(`推送 HTTP ${resp.status()}: ${body}`);
  }
  const postJson = post ? JSON.parse(post) : {};
  if (postJson.prefer_container_remote === true) {
    throw new Error('单仓 relay 不应 prefer_container_remote=true');
  }
  let json = {};
  try {
    json = JSON.parse(body);
  } catch {
    /* */
  }
  if (!(json.ok === true || json.github_oauth_multirepo)) {
    throw new Error(`推送响应异常: ${body}`);
  }
  const repos = json.github_oauth_multirepo?.repos;
  if (Array.isArray(repos)) {
    const row = repos.find((r) => r?.provider === 'gitlab' || /somanyad|8012/i.test(String(r?.github_slug || '')));
    if (row?.push_ok) {
      logTrace({ phase: 'gitlab_push_ok', ms });
    } else if (row && !row.push_ok) {
      console.warn('[verify] gitlab row:', JSON.stringify(row));
    }
  }
  await pushBtn.waitFor({ state: 'visible', timeout: 30000 });
  const stillBusy = await pushBtn.innerText().then((t) => t.includes('…')).catch(() => false);
  if (stillBusy) {
    throw new Error('推送完成后按钮仍显示 busy（…）');
  }
  logTrace({ phase: 'push_ui_idle', ms });
  console.log(`\n✅ 推送核验通过（${ms}ms）`);
}

async function main() {
  let browser;
  try {
    browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
  } catch (e) {
    console.error(`无法连接 CDP ${CDP_URL}，请用 --remote-debugging-port=9222 启动 Chrome:`, e);
    process.exit(1);
  }
  const ctx = browser.contexts()[0] || (await browser.newContext());
  const page = await ctx.newPage();
  page.on('dialog', (d) => void d.accept());

  console.log('[verify] open', TASK_URL);
  await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  await page.waitForTimeout(2000);
  if (page.url().includes('/auth/login')) {
    console.error('未登录：请在 CDP Chrome 中先登录 localhost:4000');
    process.exit(1);
  }

  if (PUSH_ONLY) {
    await waitZtree(page);
  } else {
    await ensureRelayRunning(page);
    await waitZtree(page);
    const pushReady = await page
      .getByTestId('comment-layer-ztree-panel')
      .getByRole('button', { name: '推送' })
      .first()
      .isEnabled()
      .catch(() => false);
    if (!pushReady) {
      await runAgentAndCommit(page);
    }
  }

  await verifyPush(page);
  console.log('\n--- push trace ---');
  for (const e of trace) console.log(e);
  await page.close().catch(() => {});
  process.exit(0);
}

main().catch((e) => {
  console.error('[verify] FAILED:', e);
  process.exit(1);
});
