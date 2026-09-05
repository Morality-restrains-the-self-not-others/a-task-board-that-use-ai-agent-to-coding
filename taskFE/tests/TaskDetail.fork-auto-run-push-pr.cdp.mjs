// @ts-check
/**
 * 公网验收：Fork 自动运行后工作分支带新 task id，层图出现 PR，评论有回帖。
 * 使用 chromium.connectOverCDP(http://127.0.0.1:9222)，不要把 Chrome 的
 * /json/version websocket 塞进 Playwright connectOptions（那是 Playwright server 协议，会挂死）。
 *
 * 运行：PLAYWRIGHT_INTEGRATION=1 bash tests/TaskDetail.fork-auto-run-push-pr.playwright.test.sh
 */
import { chromium } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const CDP_URL = (process.env.CDP_URL || 'http://127.0.0.1:9222').replace(/\/$/, '');
const SITE_BASE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'https://www.daydaymoney.com').replace(/\/$/, '');
const API_ORIGIN = (process.env.PLAYWRIGHT_API_ORIGIN || 'https://api.daydaymoney.com').replace(/\/$/, '');
const GATEWAY_ORIGIN = (
  process.env.PLAYWRIGHT_GATEWAY_ORIGIN ||
  process.env.GATEWAY_URL ||
  API_ORIGIN
).replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '882297276515512320';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || 'ws_882297280953085952';
const SOURCE_TASK_ID = process.env.PLAYWRIGHT_TASK_ID || 'task_882968373028220928';
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || process.env.PW_E2E_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD || process.env.PW_E2E_PASSWORD || '';
const SOURCE_URL =
  `${SITE_BASE}/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${SOURCE_TASK_ID}/`;
const REJECT_RE = /fetch first|non-fast-forward|\[rejected\]|failed to push some refs/i;
const AUTO_RUN_MS = 1_500_000;

function log(msg) {
  console.log(`[fork-push-pr ${new Date().toISOString()}] ${msg}`);
}

function fail(msg) {
  console.error(`FAIL: ${msg}`);
  process.exit(1);
}

/**
 * @param {string} url
 * @param {string} expectedTaskId  必须是本次 Fork 返回的 id；禁止匹配任意其它 task-detail 标签页
 */
function isTargetTaskUrl(url, expectedTaskId) {
  const want = String(expectedTaskId || '').trim();
  if (!want) return false;
  const m = String(url || '').match(/\/task-detail\/(task_[A-Za-z0-9]+)/);
  return Boolean(m && m[1] === want);
}

/**
 * @param {import('@playwright/test').Browser} browser
 */
function allPages(browser) {
  return browser.contexts().flatMap((c) => c.pages());
}

/**
 * @param {import('@playwright/test').Page} page
 */
async function ensureLoggedIn(page) {
  log(`login from ${page.url()}`);
  const loginJson = await loginViaGatewayApi(page, {
    email: EMAIL,
    password: PASSWORD,
    siteOrigin: SITE_BASE,
    gatewayOrigin: GATEWAY_ORIGIN,
  });
  const token = String(loginJson?.token || '').trim();
  const userId = String(loginJson?.user?.id || loginJson?.user_id || '').trim();
  if (token) {
    const activate = await page.request.post(`${API_ORIGIN}/api/accounts/users/activate-session/`, {
      headers: {
        Authorization: `Token ${token}`,
        Origin: SITE_BASE,
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      data: userId ? { user_id: userId } : {},
      timeout: 30000,
    });
    if (!activate.ok()) fail(`activate-session HTTP ${activate.status()}`);
  }
  await page.goto(SOURCE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  if (page.url().includes('/auth/login')) {
    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      baseURL: SITE_BASE,
      skipGoto: true,
    });
    await page.goto(SOURCE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  }
  const privacyContinue = page.getByRole('button', { name: '同意并继续' });
  if (await privacyContinue.isVisible().catch(() => false)) {
    await privacyContinue.click();
    await privacyContinue.waitFor({ state: 'hidden', timeout: 30000 }).catch(() => {});
  }
  if (page.url().includes('/auth/login')) fail(`still on login: ${page.url()}`);
}

/**
 * @param {import('@playwright/test').Page} page
 */
async function confirmForkAutoRun(page) {
  const forkBtn = page.locator('#task-fork-btn');
  await forkBtn.waitFor({ state: 'visible', timeout: 60000 });
  await forkBtn.click();
  await page.getByTestId('fork-auto-run-confirm-modal').waitFor({ state: 'visible', timeout: 15000 });
  await page.getByTestId('fork-mode-auto-run').check();
  const identitySelect = page.getByTestId('create-task-repo-git-identity-select').first();
  if (await identitySelect.isVisible().catch(() => false)) {
    const current = await identitySelect.inputValue().catch(() => '');
    if (!String(current || '').trim()) {
      const n = await identitySelect.locator('option').count();
      for (let i = 0; i < n; i++) {
        const value = String((await identitySelect.locator('option').nth(i).getAttribute('value')) || '').trim();
        if (value) {
          await identitySelect.selectOption(value);
          break;
        }
      }
    }
  }
  if (await page.getByTestId('fork-auto-run-oauth-bind').isVisible().catch(() => false)) {
    fail('Fork 自动运行仍显示 OAuth 绑定入口');
  }
  const submit = page.getByTestId('fork-confirm-submit');
  await submit.waitFor({ state: 'visible', timeout: 15000 });
  const t0 = Date.now();
  while (Date.now() - t0 < 60000) {
    if (await submit.isEnabled().catch(() => false)) return submit;
    await page.waitForTimeout(500);
  }
  fail('fork-confirm-submit stayed disabled');
  return submit;
}

async function main() {
  if (process.env.PLAYWRIGHT_INTEGRATION !== '1') {
    log('skip: set PLAYWRIGHT_INTEGRATION=1');
    return;
  }
  if (!PASSWORD) fail('PLAYWRIGHT_TEST_PASSWORD required');

  log(`connectOverCDP ${CDP_URL}`);
  const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
  const waitOnlyId = String(process.env.PLAYWRIGHT_WAIT_TASK_ID || '').trim();
  if (waitOnlyId) {
    const existing = allPages(browser);
    let seed = existing[0];
    if (!seed) {
      const ctx = browser.contexts()[0] || (await browser.newContext());
      seed = ctx.pages()[0] || (await ctx.newPage());
    }
    const context = seed.context();
    context.on('page', (p) => p.on('dialog', (d) => void d.accept()));
    for (const p of context.pages()) p.on('dialog', (d) => void d.accept());
    log(`wait-only ${waitOnlyId}`);
    await waitForAutoRunPrAndReplies(browser, seed, waitOnlyId);
    return;
  }
  let page =
    allPages(browser).find((p) => String(p.url() || '').includes(`/task-detail/${SOURCE_TASK_ID}`)) ||
    null;
  const context = page?.context() || browser.contexts()[0] || (await browser.newContext());
  context.on('page', (p) => p.on('dialog', (d) => void d.accept()));
  for (const p of context.pages()) p.on('dialog', (d) => void d.accept());
  if (page) {
    await page.bringToFront().catch(() => {});
    log(`reuse tab ${page.url()}`);
  } else {
    page = context.pages()[0] || (await context.newPage());
    await ensureLoggedIn(page);
  }
  if (!String(page.url() || '').includes(`/task-detail/${SOURCE_TASK_ID}`)) {
    await page.goto(SOURCE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  }
  log(`working tab ${page.url()}`);

  const forkRespPromise = context.waitForEvent('response', {
    predicate: (r) =>
      r.request().method() === 'POST' &&
      /\/api\/tasks\/todos\/tenant_id\//.test(r.url()) &&
      !r.url().includes(`/${SOURCE_TASK_ID}/`),
    timeout: 180000,
  });
  const submit = await confirmForkAutoRun(page);
  log('submit fork');
  await submit.click();
  const forkResp = await forkRespPromise;
  const forkBody = await forkResp.json().catch(() => ({}));
  if (!forkResp.ok()) fail(`fork POST ${forkResp.status()} ${JSON.stringify(forkBody)}`);
  const workBranch = String(
    forkBody.branch_strategy?.work_branch_name || forkBody.branch_strategy?.target_branch_name || '',
  );
  const newId = String(forkBody.id || '').replace(/^task_/, '');
  log(`forked ${forkBody.id} branch=${workBranch}`);
  if (!workBranch.includes(newId)) fail(`work branch missing new task id: ${workBranch}`);
  if (workBranch.includes('882908895993950208')) fail(`work branch still ancestor id: ${workBranch}`);

  const expectedTaskId = String(forkBody.id || '').trim();
  if (!expectedTaskId.startsWith('task_')) fail(`fork body missing task id: ${JSON.stringify(forkBody)}`);
  await waitForAutoRunPrAndReplies(browser, page, expectedTaskId);
}

/**
 * @param {import('@playwright/test').Browser} browser
 * @param {import('@playwright/test').Page} page
 * @param {string} expectedTaskId
 */
async function waitForAutoRunPrAndReplies(browser, page, expectedTaskId) {
  const expectedUrl = `${SITE_BASE}/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${expectedTaskId}/`;
  const tNav = Date.now();
  let forkedPage = allPages(browser).find((p) => isTargetTaskUrl(p.url(), expectedTaskId)) || null;
  while (!forkedPage && Date.now() - tNav < 15000) {
    await page.waitForTimeout(500);
    forkedPage = allPages(browser).find((p) => isTargetTaskUrl(p.url(), expectedTaskId)) || null;
  }
  if (!forkedPage) {
    forkedPage = page;
    await forkedPage.goto(expectedUrl, { waitUntil: 'domcontentloaded', timeout: 60000 });
  }
  await forkedPage.bringToFront().catch(() => {});
  if (!isTargetTaskUrl(forkedPage.url(), expectedTaskId)) {
    await forkedPage.goto(expectedUrl, { waitUntil: 'domcontentloaded', timeout: 60000 });
  } else {
    await forkedPage.reload({ waitUntil: 'domcontentloaded', timeout: 60000 });
  }
  log(`forked tab ${forkedPage.url()}`);

  await forkedPage.locator('#comments-container').waitFor({ state: 'visible', timeout: 180000 });
  await expandCommentDetails(forkedPage);
  const tZ = Date.now();
  while (Date.now() - tZ < AUTO_RUN_MS) {
    if (await forkedPage.getByTestId('container-http-unreachable-banner').isVisible().catch(() => false)) {
      fail('容器无法连接');
    }
    await expandCommentDetails(forkedPage);
    if (await forkedPage.getByTestId('comment-layer-ztree-panel').count()) break;
    await forkedPage.waitForTimeout(8000);
  }
  if (!(await forkedPage.getByTestId('comment-layer-ztree-panel').count())) {
    fail('layer ztree never appeared');
  }
  log('ztree visible, waiting PR + replies');

  const tPr = Date.now();
  let reloadedForReply = false;
  while (Date.now() - tPr < AUTO_RUN_MS) {
    await expandCommentDetails(forkedPage);
    const labels = forkedPage.getByTestId('layer-ztree-push-error-label');
    const n = await labels.count();
    for (let i = 0; i < n; i++) {
      const title = String((await labels.nth(i).getAttribute('title')) || '');
      const text = String((await labels.nth(i).innerText()) || '');
      if (REJECT_RE.test(title) || REJECT_RE.test(text)) {
        fail(`层图仍是 fetch-first 推送失败：${text} ${title.slice(0, 400)}`);
      }
    }
    const prVisible = await forkedPage.getByTestId('layer-ztree-pr-btn').first().isVisible().catch(() => false);
    const gitPrReply = await forkedPage.getByTestId('comment-git-pr-reply').count();
    const childBubbles = await forkedPage.locator('[data-testid="comment-children"] [data-testid="comment-bubble"]').count();
    const agentBadge = await forkedPage.getByText('容器 Agent', { exact: true }).count();
    const replyOk = gitPrReply > 0 || childBubbles > 0 || agentBadge > 0;
    log(`prVisible=${prVisible} gitPrReply=${gitPrReply} childBubbles=${childBubbles} agentBadge=${agentBadge}`);
    if (prVisible && replyOk) {
      log('PASS: unique branch, PR button, comment replies');
      process.exit(0);
    }
    if (prVisible && !replyOk && !reloadedForReply) {
      reloadedForReply = true;
      log('reload once so comments pick up git_pr reply created during layer-graph GET');
      await forkedPage.reload({ waitUntil: 'domcontentloaded', timeout: 60000 });
      await forkedPage.locator('#comments-container').waitFor({ state: 'visible', timeout: 180000 });
      continue;
    }
    await forkedPage.waitForTimeout(10000);
  }
  fail('timeout waiting for PR button and comment replies');
}

/**
 * @param {import('@playwright/test').Page} page
 */
async function expandCommentDetails(page) {
  await page
    .locator('#comments-container details')
    .evaluateAll((els) => {
      for (const el of els) el.open = true;
    })
    .catch(() => {});
}

main().catch((err) => {
  fail(err?.stack || String(err));
});
