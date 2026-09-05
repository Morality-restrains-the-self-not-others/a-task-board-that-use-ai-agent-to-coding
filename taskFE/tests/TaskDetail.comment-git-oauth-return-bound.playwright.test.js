// @ts-check
/**
 * 任务详情执行细节 Git OAuth：回流后须查询 user-app-connection 并显示已绑定。
 * 通过 CDP 9222 连接本机 Chrome。
 *
 * 运行：PLAYWRIGHT_INTEGRATION=1 bash taskFE/tests/TaskDetail.comment-git-oauth-return-bound.playwright.test.sh
 */
import { test, expect, chromium } from '@playwright/test';

const CDP_URL = (process.env.CDP_URL || 'http://127.0.0.1:9222').replace(/\/$/, '');
const PAGE_URL =
  process.env.PAGE_URL ||
  'https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878932440129761280/?accessCode=DR2AKvP9J9';
const EMAIL = process.env.PW_E2E_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PW_E2E_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const CONNECTION_ROUTE = /\/api\/git-oauth\/user-app-connection\/?/;

async function connectPage() {
  const browser = await chromium.connectOverCDP(CDP_URL);
  const context = browser.contexts()[0] || (await browser.newContext());
  const page = await context.newPage();
  return { browser, page };
}

async function loginOnWww(page) {
  await page.goto('https://www.daydaymoney.com/auth/login/', { waitUntil: 'domcontentloaded', timeout: 60_000 });
  const emailTab = page.locator('button:has-text("邮箱/密码")').first();
  if (await emailTab.isVisible().catch(() => false)) {
    await emailTab.click();
  }
  await page.locator('#email').fill(EMAIL);
  await page.locator('#password').fill(PASSWORD);
  const agreeBoxes = await page.locator('input[type=checkbox]').all();
  for (const box of agreeBoxes) {
    const checked = await box.isChecked().catch(() => false);
    if (!checked) await box.check({ force: true }).catch(() => {});
  }
  await page.getByRole('button', { name: '登录', exact: true }).first().click();
  await page.waitForURL((url) => !String(url).includes('/auth/login'), { timeout: 45_000 }).catch(() => {});
}

test.describe('TaskDetail 评论 Git OAuth 回流后绑定芯片', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑公网 CDP');
  test.skip(() => process.env.PLAYWRIGHT_INTEGRATION !== '1', '需 PLAYWRIGHT_INTEGRATION=1');
  test.setTimeout(180_000);

  test('拦截 user-app-connection 为已绑定后芯片应为已绑定', async () => {
    const { page } = await connectPage();
    const connectionUrls = [];

    await loginOnWww(page);

    await page.route(CONNECTION_ROUTE, async (route) => {
      connectionUrls.push(route.request().url());
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          connected: true,
          bind_status: 'active',
          connections: [{ connected: true, provider: 'gitlab', service_provider: 'tencent-sh-1' }],
        }),
      });
    });

    const withOk = new URL(PAGE_URL);
    withOk.searchParams.set('gitlab', 'ok');
    await page.goto(withOk.toString(), { waitUntil: 'domcontentloaded', timeout: 90_000 });
    await expect(page).not.toHaveURL(/\/auth\/login/, { timeout: 15_000 });
    await expect
      .poll(async () => connectionUrls.length, { timeout: 45_000 })
      .toBeGreaterThan(0);
    const chip = page.getByTestId('comment-execution-git-oauth').first();
    await expect(chip).toBeVisible({ timeout: 45_000 });
    await expect(chip).toHaveAttribute('data-kind', 'bound');
    await expect(chip).toContainText('已绑定');
    await expect(page.getByTestId('comment-execution-git-oauth-bind')).toHaveCount(0);
    await page.close();
  });

  test('公网真实 API：gitlab=ok 回流后芯片已绑定且 probe 不打回未绑定', async () => {
    const { page } = await connectPage();
    const connectionHits = [];
    page.on('response', async (res) => {
      if (!CONNECTION_ROUTE.test(res.url())) return;
      let body = null;
      try {
        body = await res.json();
      } catch {
        body = null;
      }
      connectionHits.push({
        url: res.url(),
        status: res.status(),
        connected: body?.connected,
        access_token_valid: Object.prototype.hasOwnProperty.call(body || {}, 'access_token_valid')
          ? body.access_token_valid
          : '(omitted)',
      });
    });

    await loginOnWww(page);

    const withOk = new URL(PAGE_URL);
    withOk.searchParams.set('gitlab', 'ok');
    withOk.searchParams.set('_spa', String(Date.now()));
    await page.goto(withOk.toString(), { waitUntil: 'domcontentloaded', timeout: 90_000 });
    await expect(page).not.toHaveURL(/\/auth\/login/, { timeout: 15_000 });

    const chip = page.getByTestId('comment-execution-git-oauth').first();
    await expect(chip).toBeVisible({ timeout: 45_000 });
    await expect
      .poll(async () => (await chip.getAttribute('data-kind')) || '', { timeout: 45_000 })
      .not.toBe('loading');
    await expect(chip).toHaveAttribute('data-kind', 'bound');
    await expect(chip).toContainText('已绑定');
    // once-probe 约 6s 超时内若把缺字段当无效，会把芯片打回未绑定
    await page.waitForTimeout(4000);
    await expect(chip).toHaveAttribute('data-kind', 'bound');
    await expect(page.getByTestId('comment-execution-git-oauth-bind')).toHaveCount(0);
    expect(connectionHits.some((row) => row.status === 200 && row.connected === true)).toBe(true);
    await page.close();
  });

  test('accessCode 页 connection 401 不得踢到登录', async () => {
    const { page } = await connectPage();

    await page.route(CONNECTION_ROUTE, async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({
          detail: '无法解析登录凭据，请重新登录',
          redirect_url: '/auth/login/?next=%2Ftenant%2F',
        }),
      });
    });

    await page.goto(PAGE_URL, { waitUntil: 'domcontentloaded', timeout: 90_000 });
    await page.waitForTimeout(3000);
    await expect(page).not.toHaveURL(/\/auth\/login/);
    await expect(page).toHaveURL(/task-detail/);
    await page.close();
  });
});
