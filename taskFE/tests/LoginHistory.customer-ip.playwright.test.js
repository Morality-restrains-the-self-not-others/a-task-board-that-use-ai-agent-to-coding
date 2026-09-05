// @ts-check
/**
 * 公网客户登录历史：侧栏入口 + 表格「用户入口」+ 最新 IP 非 RFC1918
 * （OPT-20260825-036 / OPT-20260826-002）。
 *
 * 运行：bash tests/LoginHistory.profile-and-admin.playwright.test.sh
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { isRfc1918Ipv4 } from './helpers/loginHistoryIp.js';

const SITE = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  'https://www.daydaymoney.com'
).replace(/\/$/, '');
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PLAYWRIGHT_TEST_PASSWORD required (no hardcoded fallback)');

/**
 * @param {import('@playwright/test').Page} page
 */
async function dismissPrivacyGate(page) {
  const btn = page.getByRole('button', { name: '同意并继续' });
  if (await btn.isVisible({ timeout: 4000 }).catch(() => false)) {
    await btn.click();
    await expect(btn).toBeHidden({ timeout: 20000 });
  }
}

test.describe('客户登录历史（公网）', () => {
  test('侧栏「登录历史」可见，表格含入口与非 RFC1918 IP', async ({ page }) => {
    test.setTimeout(180000);

    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      baseURL: SITE,
    });
    await dismissPrivacyGate(page);

    await page.goto(`${SITE}/profile/login-history/`, { waitUntil: 'domcontentloaded' });
    await page.reload({ waitUntil: 'networkidle' }).catch(() => {});
    await dismissPrivacyGate(page);

    await expect(page.getByTestId('user-center-nav-login-history')).toBeVisible({ timeout: 30000 });
    await expect(page.getByRole('heading', { name: '登录历史' })).toBeVisible();

    await expect(page.locator('table thead')).toContainText('入口');
    await expect(page.locator('table thead')).toContainText('IP');

    const rows = page.getByTestId('login-history-row');
    await expect(rows.first()).toBeVisible({ timeout: 45000 });
    // 邮箱密码登录可能记「用户入口」；同会话再进站常记「切换会话」
    const customerEntry = rows.filter({ hasText: /用户入口|切换会话/ });
    await expect(customerEntry.first()).toBeVisible();

    const ipCell = page.locator('[data-testid="login-history-row"] td.px-4.py-3.text-sm.font-mono').first();
    await expect(ipCell).toBeVisible();
    const ip = (await ipCell.innerText()).trim();
    expect(ip, '最新登录 IP 不得为空').not.toMatch(/^—$|^$/);
    expect(isRfc1918Ipv4(ip), `最新 IP ${ip} 仍是 RFC1918（曾出现 172.26.0.1 网桥）`).toBe(false);
  });
});
