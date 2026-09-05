// @ts-check
/**
 * 核验：多仓任务 bootstrap 克隆不得因同用户重复 refresh 吊销 access_token 而失败。
 * 目标任务含 somanyad + somanyad-emailD；修复前 emailD 常报 HTTP Basic Access denied。
 *
 * 运行（CDP 9222）：
 * PLAYWRIGHT_RELAY_TASK_ID=task_12590983282794675865 \
 * bash tests/TaskDetail.relay-bootstrap-clone-auth.playwright.test.js.sh
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '861623708318031872';
const TASK_ID = process.env.PLAYWRIGHT_RELAY_TASK_ID || 'task_12590983282794675865';
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const TASK_PATH =
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?relayToTrae=true`;

test.describe('TaskDetail relay bootstrap clone auth', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑 relay 全栈 E2E');

  test('启动后两仓克隆成功且无 HTTP Basic Access denied', async ({ page }) => {
    test.setTimeout(420_000);

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
      .poll(
        async () => {
          const bodyText = await page.locator('body').innerText();
          if (/HTTP Basic:\s*Access denied/i.test(bodyText)) return 'auth-denied';
          if (/BOOTSTRAP_FAILED/i.test(bodyText)) return 'bootstrap-failed';
          if (/Authentication failed for/i.test(bodyText)) return 'auth-failed';
          if (bodyText.includes('任务引导完成') || /BOOTSTRAP_COMPLETE/i.test(bodyText)) {
            return 'bootstrap-ok';
          }
          return 'pending';
        },
        {
          timeout: 300_000,
          message: '应完成引导克隆且无 Git HTTP Basic 认证失败',
        },
      )
      .toBe('bootstrap-ok');

    const finalText = await page.locator('body').innerText();
    expect(finalText).not.toMatch(/HTTP Basic:\s*Access denied/i);
    expect(finalText).not.toMatch(/BOOTSTRAP_FAILED/i);
    expect(finalText).not.toMatch(/Authentication failed for/i);
  });
});
