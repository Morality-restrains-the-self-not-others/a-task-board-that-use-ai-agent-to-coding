// @ts-check
/**
 * 公网验收：Fork 自动运行不得因共用祖先工作分支被 git push [rejected] fetch first。
 * 成功：层图无该拒绝文案、出现 PR 链接、评论区有自动运行之外的回帖。
 *
 * CDP 外接 Chrome 时 Playwright 的 page 经常是 about:blank；必须切到已有 task-detail 页。
 *
 * 源任务：task_882968373028220928
 * 运行：PLAYWRIGHT_INTEGRATION=1 bash tests/TaskDetail.fork-auto-run-push-pr.playwright.test.sh
 */
import fs from 'node:fs';
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const E2E_LOG = '/tmp/fork-push-pr-e2e.log';
function e2eLog(msg) {
  const line = `[fork-push-pr ${new Date().toISOString()}] ${msg}`;
  try {
    fs.appendFileSync(E2E_LOG, `${line}\n`);
  } catch {
    /* ignore */
  }
  console.log(line);
}

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

test.skip(!PASSWORD, 'PLAYWRIGHT_TEST_PASSWORD required (no hardcoded fallback)');

/**
 * @param {string} url
 * @param {string} sourceTaskId
 */
function isForkedTaskUrl(url, sourceTaskId) {
  const m = String(url || '').match(/\/task-detail\/(task_[A-Za-z0-9]+)/);
  return Boolean(m && m[1] && m[1] !== sourceTaskId);
}

/**
 * CDP 首个 page 常为 about:blank；优先复用已打开的源任务详情。
 * @param {import('@playwright/test').Page} page
 * @param {import('@playwright/test').BrowserContext} context
 */
async function pickSourceTaskPage(page, context) {
  const hit = [page, ...context.pages()].find(
    (p) => p && !p.isClosed() && String(p.url() || '').includes(`/task-detail/${SOURCE_TASK_ID}`),
  );
  if (hit) {
    await hit.bringToFront().catch(() => {});
    e2eLog(`reuse tab ${hit.url()}`);
    return hit;
  }
  e2eLog(`goto source from ${page.url()}`);
  await page.goto(SOURCE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  return page;
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {import('@playwright/test').BrowserContext} context
 */
async function ensureLoggedIn(page, context) {
  e2eLog(`loginViaGatewayApi from ${page.url()}`);
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
    if (!activate.ok()) {
      throw new Error(`activate-session HTTP ${activate.status()}`);
    }
  }
  const cookies = await page.context().cookies();
  const mirrored = cookies
    .filter((c) => ['sessionid', 'userId', 'csrftoken', 'token'].includes(c.name))
    .map((c) => ({
      name: c.name,
      value: c.value,
      url: `${API_ORIGIN}/`,
      httpOnly: c.httpOnly,
      secure: true,
    }));
  if (mirrored.length) await page.context().addCookies(mirrored);

  const work = await pickSourceTaskPage(page, context);
  if (work.url().includes('/auth/login')) {
    await playwrightLoginWithLegalAccept(work, {
      email: EMAIL,
      password: PASSWORD,
      baseURL: SITE_BASE,
      skipGoto: true,
    });
    await work.goto(SOURCE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  }
  const privacyContinue = work.getByRole('button', { name: '同意并继续' });
  if (await privacyContinue.isVisible().catch(() => false)) {
    await privacyContinue.click();
    await privacyContinue.waitFor({ state: 'hidden', timeout: 30000 }).catch(() => {});
  }
  await expect(work).not.toHaveURL(/\/auth\/login/, { timeout: 30000 });
  await expect(work).toHaveURL(new RegExp(`/task-detail/${SOURCE_TASK_ID}`), { timeout: 30000 });
  return work;
}

/**
 * @param {import('@playwright/test').Page} page
 */
async function confirmForkAutoRun(page) {
  const forkBtn = page.locator('#task-fork-btn');
  await expect(forkBtn).toBeVisible({ timeout: 60000 });
  await forkBtn.click();
  const modal = page.getByTestId('fork-auto-run-confirm-modal');
  await expect(modal).toBeVisible({ timeout: 15000 });
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
    throw new Error('Fork 自动运行仍显示 OAuth 绑定入口');
  }
  const submit = page.getByTestId('fork-confirm-submit');
  await expect(submit).toBeEnabled({ timeout: 60000 });
  return submit;
}

test.describe('TaskDetail Fork 自动运行 push+PR+回帖', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑公网 CDP');
  test.skip(() => process.env.PLAYWRIGHT_INTEGRATION !== '1', '需 PLAYWRIGHT_INTEGRATION=1');
  test.setTimeout(1920000);

  test('Fork 自动运行后应推送成功、出现 PR、并有评论回帖', async ({ page, context }) => {
    const acceptDialog = (/** @type {import('@playwright/test').Page} */ p) => {
      p.on('dialog', (d) => void d.accept());
    };
    acceptDialog(page);
    context.on('page', acceptDialog);
    e2eLog(`fixture page=${page.url()} pages=${context.pages().map((p) => p.url()).join(' | ')}`);

    const work = await ensureLoggedIn(page, context);
    e2eLog(`on ${work.url()}`);

    const forkRespPromise = context.waitForResponse(
      (r) =>
        r.request().method() === 'POST' &&
        /\/api\/tasks\/todos\/tenant_id\//.test(r.url()) &&
        !r.url().includes(`/${SOURCE_TASK_ID}/`),
      { timeout: 180000 },
    );
    const submit = await confirmForkAutoRun(work);
    const popupPromise = context.waitForEvent('page', { timeout: 180000 }).catch(() => null);
    await submit.click();
    const forkResp = await forkRespPromise;
    const forkBody = await forkResp.json().catch(() => ({}));
    expect(forkResp.ok(), `fork POST ${forkResp.status()} ${JSON.stringify(forkBody)}`).toBeTruthy();
    const bs = forkBody.branch_strategy || {};
    const workBranch = String(bs.work_branch_name || bs.target_branch_name || '');
    const newId = String(forkBody.id || '').replace(/^task_/, '');
    e2eLog(`forked ${forkBody.id} branch=${workBranch}`);
    expect(workBranch, 'fork 工作分支应带新任务 ID').toContain(newId);
    expect(workBranch, 'fork 不得沿用祖先任务工作分支').not.toContain('882908895993950208');

    const popup = await popupPromise;
    const resolveForkedPage = () =>
      [work, popup, ...context.pages()].find((p) => p && isForkedTaskUrl(p.url(), SOURCE_TASK_ID)) || null;
    await expect.poll(() => Boolean(resolveForkedPage()), { timeout: 180000, intervals: [500, 1000, 2000] }).toBe(true);
    const forkedPage = resolveForkedPage();
    expect(forkedPage).toBeTruthy();
    await forkedPage.bringToFront().catch(() => {});
    e2eLog(`forked tab ${forkedPage.url()}`);

    await expect(forkedPage.locator('#comments-container')).toBeVisible({ timeout: 180000 });
    await expect
      .poll(
        async () => {
          if (await forkedPage.getByTestId('container-http-unreachable-banner').isVisible().catch(() => false)) {
            throw new Error('容器无法连接，层图不会出现');
          }
          return forkedPage.getByTestId('comment-layer-ztree-panel').isVisible().catch(() => false);
        },
        { timeout: 1500000, intervals: [5000, 10000, 15000] },
      )
      .toBe(true);

    await expect
      .poll(
        async () => {
          const labels = forkedPage.getByTestId('layer-ztree-push-error-label');
          const n = await labels.count();
          for (let i = 0; i < n; i++) {
            const title = String((await labels.nth(i).getAttribute('title')) || '');
            const text = String((await labels.nth(i).innerText()) || '');
            if (REJECT_RE.test(title) || REJECT_RE.test(text)) {
              throw new Error(`层图仍是 fetch-first 推送失败：${text} ${title.slice(0, 400)}`);
            }
          }
          const prVisible = await forkedPage.getByTestId('layer-ztree-pr-btn').first().isVisible().catch(() => false);
          const replyCount = await forkedPage.locator('#comments-container .conversation-feed > *').count();
          return prVisible && replyCount >= 2;
        },
        { timeout: 1500000, intervals: [5000, 10000, 15000] },
      )
      .toBe(true);
  });
});
