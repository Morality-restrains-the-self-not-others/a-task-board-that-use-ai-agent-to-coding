// @ts-check
/**
 * 集成：relayToTrae 直接启动 → 点击「启动」时 repo-credentials-precheck 应通过。
 * 回归任务 846269443533955072：GitLab :8012 未启动时曾报
 * 「仓库克隆 token 换发失败（http 502）」；GitLab + gitOauth 就绪后预检应 200。
 *
 * 环境：localhost:4000、Django :8001、gitOauth :8002、GitLab :8012、go-relay :8797。
 * 凭据：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD。
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const SITE_ORIGIN = String(process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const TASK_ID = process.env.PLAYWRIGHT_RELAY_TASK_ID || '846269443533955072';
const TASK_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?relayToTrae=true`;

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

async function assertGitLabReachable() {
  const r = await fetch('http://127.0.0.1:8012/users/sign_in', { signal: AbortSignal.timeout(8000) }).catch(() => null);
  if (!r?.ok) {
    throw new Error('GitLab :8012 不可达，请先启动 gitService（./run.sh managed）');
  }
}

async function assertGitOauthReachable() {
  const r = await fetch('http://127.0.0.1:8002/api/health/', { signal: AbortSignal.timeout(8000) }).catch(() => null);
  if (!r?.ok) {
    throw new Error('gitOauth :8002 不可达');
  }
}

test.describe('TaskDetail relayToTrae start precheck (GitLab token refresh)', () => {
  test('直接启动点击启动时预检通过，不出现 token 换发 502 提示', async ({ page }) => {
    test.skip(process.env.PRE_COMMIT === '1', 'pre-commit 不跑全栈集成');
    test.setTimeout(180_000);

    await assertGitLabReachable();
    await assertGitOauthReachable();

    /** @type {null | { status: number, body: Record<string, unknown> }} */
    let precheckCapture = null;

    page.on('response', async (resp) => {
      const url = resp.url();
      if (!url.includes('/relay-to-trae/repo-credentials-precheck/') || resp.request().method() !== 'POST') {
        return;
      }
      let body = {};
      try {
        body = await resp.json();
      } catch {
        /* ignore */
      }
      precheckCapture = { status: resp.status(), body };
    });

    await playwrightLoginWithLegalAccept(page, { email: EMAIL, password: PASSWORD, baseURL: SITE_ORIGIN });
    await page.goto(`${SITE_ORIGIN}${TASK_PATH}`);
    await page.waitForLoadState('domcontentloaded');

    const relayTab = page.getByTestId('server-config-relay-direct-tab');
    await expect(relayTab).toBeVisible({ timeout: 30_000 });
    await relayTab.click();

    const statusRow = page.getByTestId('relay-to-trae-status-row');
    await expect(statusRow.getByText(/relayToTrae 服务：在线/)).toBeVisible({ timeout: 30_000 });

    const stopBtn = page.getByTestId('relay-to-trae-stop-btn');
    if (await stopBtn.isVisible().catch(() => false)) {
      await stopBtn.click();
      await expect(statusRow.getByText(/onlineServiceJS：未启动/)).toBeVisible({ timeout: 60_000 });
    }

    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    await expect(startBtn).toBeEnabled({ timeout: 15_000 });
    await startBtn.click();

    await expect
      .poll(() => precheckCapture?.status, {
        timeout: 60_000,
        message: '应完成 repo-credentials-precheck',
      })
      .toBeDefined();

    expect(precheckCapture?.status, '预检不应因 token 换发失败返回 502').not.toBe(502);
    expect(String(precheckCapture?.body?.error_code || '')).not.toBe('REPO_CLONE_TOKEN_REFRESH_FAILED');

    await expect(page.getByText(/仓库克隆 token 换发失败/)).toHaveCount(0);
    await expect(page.getByText(/gitOauth 换发 token 失败/)).toHaveCount(0);

    if (precheckCapture?.status !== 200) {
      throw new Error(
        `repo-credentials-precheck 未通过：HTTP ${precheckCapture?.status} ${JSON.stringify(precheckCapture?.body).slice(0, 400)}`,
      );
    }
  });
});
