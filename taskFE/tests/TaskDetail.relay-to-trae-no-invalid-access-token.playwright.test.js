// @ts-check
/**
 * 回归：relayToTrae 直接启动后，侧车状态推送不得出现「无效的 access_token」。
 * 根因：onlineServiceJS 换票后 DB token 已更新，relay 仍用登记时的预埋 token 推送。
 *
 * 运行前：本机 go-relay 已启动（runAll 或 go_relayToTrae/bin/go_relayToTrae）。
 * 可选：PLAYWRIGHT_BASE_URL=http://daydaymoney.com PLAYWRIGHT_TEST_EMAIL=... PLAYWRIGHT_TEST_PASSWORD=...
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

/** @param {import('@playwright/test').Page} page */
async function loginFordaydaymoney(page, siteRoot) {
  await playwrightLoginWithLegalAccept(page, {
    email: EMAIL,
    password: PASSWORD,
    baseURL: siteRoot || undefined,
  });
  if (page.url().includes('/auth/login')) {
    await page.goto(siteRoot ? `${siteRoot}/auth/login/` : '/auth/login/');
    await page.waitForLoadState('domcontentloaded');
    const emailPwdTab = page.getByRole('button', { name: /邮箱\/密码|邮箱.*密码/ }).first();
    if (await emailPwdTab.isVisible().catch(() => false)) {
      await emailPwdTab.click();
    }
    for (const checkbox of [
      page.getByRole('checkbox', { name: /全部条款/ }),
      page.getByRole('checkbox', { name: /隐私政策/ }),
      page.getByRole('checkbox', { name: /软件许可/ }),
    ]) {
      if (await checkbox.isVisible().catch(() => false)) {
        if (!(await checkbox.isChecked().catch(() => false))) {
          await checkbox.check();
        }
      }
    }
    await page.locator('#email').fill(EMAIL);
    await page.locator('#password').fill(PASSWORD);
    await page.locator('form').first().evaluate((form) => form.requestSubmit());
    await page.waitForURL((u) => !u.pathname.includes('/auth/login'), { timeout: 60000 });
  }
}

const TASK_ID = '843742455533076480';
const ACCESS_CODE = 'u827923618451263488';
const TASK_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=${ACCESS_CODE}&relayToTrae=true`;

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

test.describe('TaskDetail relayToTrae invalid access_token regression', () => {
  test('直接启动后日志不应含无效的 access_token', async ({ page, baseURL }) => {
    test.setTimeout(180000);
    test.skip(
      process.env.PRE_COMMIT === '1',
      'pre-commit 无本机 relayToTrae / 外网任务，请本地单独执行',
    );

    const siteRoot = (process.env.PLAYWRIGHT_BASE_URL || baseURL || '').replace(/\/$/, '');

    await loginFordaydaymoney(page, siteRoot);

    await page.goto(siteRoot ? `${siteRoot}${TASK_PATH}` : TASK_PATH);
    await page.waitForLoadState('domcontentloaded');

    test.skip(
      page.url().includes('/auth/login'),
      '登录未成功，请设置 PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD',
    );

    const statusRow = page.getByTestId('relay-to-trae-status-row');
    await expect(statusRow).toBeVisible({ timeout: 60000 });

    const healthOnline = statusRow.getByText('relayToTrae 服务：在线');
    const healthOffline = statusRow.getByText('relayToTrae 服务：未连接');
    await expect
      .poll(
        async () => {
          if (await healthOnline.isVisible().catch(() => false)) return 'online';
          if (await healthOffline.isVisible().catch(() => false)) return 'offline';
          return 'pending';
        },
        { timeout: 20000, message: '应显示 relayToTrae 健康状态' },
      )
      .not.toBe('pending');

    test.skip(
      await healthOffline.isVisible().catch(() => false),
      '本机 relayToTrae 未连接，请先启动 go-relay（go_relayToTrae/bin/go_relayToTrae）',
    );

    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    await expect(startBtn).toBeVisible({ timeout: 15000 });
    await startBtn.click();

    await expect(page.getByTestId('relay-to-trae-stop-btn')).toBeVisible({ timeout: 120000 });

    await expect
      .poll(
        async () => {
          const bodyText = await page.locator('body').innerText();
          if (bodyText.includes('无效的 access_token')) return 'invalid-token-error';
          if (/onlineServiceJS|token-exchange|listening|控制台|任务引导完成/.test(bodyText)) {
            return 'started';
          }
          return 'pending';
        },
        { timeout: 120000, message: '应完成启动且日志不含无效 token' },
      )
      .toBe('started');

    const pageText = await page.locator('body').innerText();
    expect(pageText.includes('无效的 access_token')).toBe(false);
  });
});
