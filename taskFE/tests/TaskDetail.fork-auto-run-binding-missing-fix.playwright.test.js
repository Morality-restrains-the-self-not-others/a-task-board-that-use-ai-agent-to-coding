// @ts-check
/**
 * 公网验收：Fork 源任务并自动运行后，层图不得再因 comment L2 缺失
 * 报 BINDING_MISSING / HTTP 409 layer-github-oauth-access-tokens。
 *
 * 源任务：task_882943770763489280（gitlab-tencent-sh-1 …/ram-work）
 *
 * 运行：
 *   PLAYWRIGHT_INTEGRATION=1 bash tests/TaskDetail.fork-auto-run-binding-missing-fix.playwright.test.sh
 */
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const SITE_BASE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'https://www.daydaymoney.com').replace(/\/$/, '');
const API_ORIGIN = (process.env.PLAYWRIGHT_API_ORIGIN || 'https://api.daydaymoney.com').replace(/\/$/, '');
const GATEWAY_ORIGIN = (
  process.env.PLAYWRIGHT_GATEWAY_ORIGIN ||
  process.env.GATEWAY_URL ||
  API_ORIGIN
).replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '882297276515512320';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || 'ws_882297280953085952';
const SOURCE_TASK_ID = process.env.PLAYWRIGHT_TASK_ID || 'task_882943770763489280';
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || process.env.PW_E2E_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD || process.env.PW_E2E_PASSWORD || '';
const SOURCE_URL =
  process.env.PLAYWRIGHT_TASK_DETAIL_URL ||
  `${SITE_BASE}/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${SOURCE_TASK_ID}/`;
const BINDING_FAIL_RE =
  /BINDING_MISSING|缺少绑定|请先完成该仓库的 Git 授权|layer-github-oauth-access-tokens/;

test.skip(!PASSWORD, 'PLAYWRIGHT_TEST_PASSWORD required (no hardcoded fallback)');

/**
 * @param {import('@playwright/test').Page} page
 */
async function mirrorSessionCookiesToApiOrigin(page) {
  const cookies = await page.context().cookies();
  const names = new Set(['sessionid', 'userId', 'csrftoken', 'token']);
  const mirrored = cookies
    .filter((c) => names.has(c.name))
    .map((c) => ({
      name: c.name,
      value: c.value,
      url: `${API_ORIGIN}/`,
      httpOnly: c.httpOnly,
      secure: true,
    }));
  if (mirrored.length > 0) {
    await page.context().addCookies(mirrored);
  }
}

/**
 * @param {import('@playwright/test').Page} page
 */
async function ensureLoggedIn(page) {
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
    });
    if (!activate.ok()) {
      throw new Error(`activate-session HTTP ${activate.status()} ${(await activate.text().catch(() => '')).slice(0, 200)}`);
    }
  }
  await mirrorSessionCookiesToApiOrigin(page);
  await page.goto(SOURCE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
  if (page.url().includes('/auth/login')) {
    const emailTab = page.getByRole('button', { name: /邮箱\/密码|邮箱.*密码/ }).first();
    if (await emailTab.isVisible().catch(() => false)) await emailTab.click();
    for (const name of [/全部条款/, /隐私政策/, /许可及服务协议|软件许可/]) {
      const box = page.getByRole('checkbox', { name });
      if (await box.isVisible().catch(() => false)) {
        await box.check({ force: true }).catch(() => {});
      }
    }
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
  await expect(page).not.toHaveURL(/\/auth\/login/, { timeout: 30000 });
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
      const options = identitySelect.locator('option');
      const n = await options.count();
      for (let i = 0; i < n; i++) {
        const value = String((await options.nth(i).getAttribute('value')) || '').trim();
        if (value) {
          await identitySelect.selectOption(value);
          break;
        }
      }
    }
  }

  const bindLink = page.getByTestId('fork-auto-run-oauth-bind');
  if (await bindLink.isVisible().catch(() => false)) {
    throw new Error('Fork 自动运行仍显示 OAuth 绑定入口，无法在无交互下完成授权');
  }

  const submit = page.getByTestId('fork-confirm-submit');
  await expect(submit).toBeEnabled({ timeout: 60000 });
  return submit;
}

/**
 * @param {string} url
 * @param {string} sourceTaskId
 */
function isForkedTaskUrl(url, sourceTaskId) {
  const m = String(url || '').match(/\/task-detail\/(task_[A-Za-z0-9]+)/);
  return Boolean(m && m[1] && m[1] !== sourceTaskId);
}

test.describe('TaskDetail Fork 自动运行不再 BINDING_MISSING', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑公网 CDP');
  test.skip(() => process.env.PLAYWRIGHT_INTEGRATION !== '1', '需 PLAYWRIGHT_INTEGRATION=1');
  // 云主机拉镜像 / register-reachability 实测约 20–25min，10min 不够。
  test.setTimeout(1920000);

  test('Fork 自动运行后层图不得出现 BINDING_MISSING / HTTP 409 拉票失败', async ({ page, context }) => {
    page.on('dialog', (d) => void d.accept());

    /** @type {{ status: number, body: string }[]} */
    const tokenHits = [];

    const attachTokenListener = (/** @type {import('@playwright/test').Page} */ p) => {
      p.on('response', async (r) => {
        if (!r.url().includes('layer-github-oauth-access-tokens')) return;
        const body = (await r.text().catch(() => '')).slice(0, 4000);
        tokenHits.push({ status: r.status(), body });
      });
    };
    attachTokenListener(page);

    await ensureLoggedIn(page);

    const forkRespPromise = page.waitForResponse(
      (r) =>
        r.request().method() === 'POST' &&
        /\/api\/tasks\/todos\/tenant_id\//.test(r.url()) &&
        !r.url().includes(`/${SOURCE_TASK_ID}/`),
      { timeout: 180000 },
    );

    const submit = await confirmForkAutoRun(page);
    const popupPromise = context.waitForEvent('page', { timeout: 180000 }).catch(() => null);
    await submit.click();

    const forkResp = await forkRespPromise;
    const forkBody = await forkResp.json().catch(() => ({}));
    expect(forkResp.ok(), `fork POST ${forkResp.status()} ${JSON.stringify(forkBody)}`).toBeTruthy();

    const popup = await popupPromise;
    if (popup) attachTokenListener(popup);

    /** Fork 会预开 about:blank；真正的任务详情可能在原页或其它 tab。 */
    const resolveForkedPage = () =>
      [page, popup, ...context.pages()].find((p) => p && isForkedTaskUrl(p.url(), SOURCE_TASK_ID)) || null;

    await expect
      .poll(() => Boolean(resolveForkedPage()), {
        timeout: 180000,
        intervals: [500, 1000, 2000],
      })
      .toBe(true);
    const forkedPage = resolveForkedPage();
    expect(forkedPage, 'forked task-detail tab').toBeTruthy();

    const comments = forkedPage.locator('#comments-container');
    await expect(comments).toBeVisible({ timeout: 180000 });
    await expect
      .poll(
        async () => comments.locator('.conversation-feed, [data-testid="comment-layer-ztree-panel"]').count(),
        { timeout: 300000, intervals: [2000, 5000, 10000] },
      )
      .toBeGreaterThan(0);

    await expect
      .poll(
        async () => {
          if (await forkedPage.getByTestId('container-http-unreachable-banner').isVisible().catch(() => false)) {
            throw new Error('容器无法连接，层图不会出现');
          }
          if (await forkedPage.getByTestId('comment-layer-ztree-panel').isVisible().catch(() => false)) {
            return true;
          }
          return false;
        },
        { timeout: 600000, intervals: [5000, 10000, 15000] },
      )
      .toBe(true);

    await expect
      .poll(
        async () => {
          const bindingHits = tokenHits.filter(
            (h) => h.status === 409 && BINDING_FAIL_RE.test(h.body),
          );
          if (bindingHits.length > 0) {
            throw new Error(`layer-oauth 仍 BINDING_MISSING: ${JSON.stringify(bindingHits[0])}`);
          }
          const labels = forkedPage.getByTestId('layer-ztree-push-error-label');
          const n = await labels.count();
          for (let i = 0; i < n; i++) {
            const title = String((await labels.nth(i).getAttribute('title')) || '');
            const text = String((await labels.nth(i).innerText()) || '');
            if (BINDING_FAIL_RE.test(title) || BINDING_FAIL_RE.test(text)) {
              throw new Error(`层图仍显示授权缺失：${text} ${title.slice(0, 400)}`);
            }
          }
          const ztreeReady = await forkedPage.getByTestId('comment-layer-ztree-panel').isVisible().catch(() => false);
          return ztreeReady && tokenHits.every((h) => h.status !== 409);
        },
        { timeout: 600000, intervals: [3000, 5000, 10000] },
      )
      .toBe(true);

    const leftover = tokenHits.filter((h) => h.status === 409 && BINDING_FAIL_RE.test(h.body));
    expect(leftover, JSON.stringify(leftover)).toHaveLength(0);
    const labels = forkedPage.getByTestId('layer-ztree-push-error-label');
    const n = await labels.count();
    for (let i = 0; i < n; i++) {
      const title = String((await labels.nth(i).getAttribute('title')) || '');
      const text = String((await labels.nth(i).innerText()) || '');
      expect(
        BINDING_FAIL_RE.test(title) || BINDING_FAIL_RE.test(text),
        `层图 push 失败仍是授权缺失：${text} ${title.slice(0, 400)}`,
      ).toBeFalsy();
    }
  });
});
