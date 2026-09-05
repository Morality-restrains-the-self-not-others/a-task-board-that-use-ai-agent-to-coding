// @ts-check
/**
 * 复现：已登录（profile 有效）但 userId cookie 缺失时，git-site-oauth 误报「非本人」。
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const LOGIN = {
  email: 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

const OWN_PROFILE_WARNING =
  '仅登录用户本人可在账号中心管理 GitHub 授权';

test.describe('GitSiteOAuth userId cookie 缺失场景', () => {
  test('清除 userId cookie 后访问应仍视为本人（当前会误报）', async ({ page }) => {
    await playwrightLoginWithLegalAccept(page, LOGIN);

    await page.context().clearCookies({ name: 'userId' });
    await page.evaluate(() => {
      document.cookie = 'userId=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/';
    });

    const profileViaPage = await page.evaluate(async () => {
      const r = await fetch('/api/accounts/users/profile/', {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      });
      const body = await r.json().catch(() => ({}));
      return { status: r.status, user_id: body?.user_id };
    });
    console.log('[diag] profile after clearing userId cookie:', profileViaPage);

    await page.goto('/profile/git-site-oauth/', { waitUntil: 'networkidle' });

    const docCookieUserId = await page.evaluate(() => {
      const row = document.cookie.split('; ').find((r) => r.startsWith('userId='));
      return row ? decodeURIComponent(row.split('=')[1]) : '';
    });
    console.log('[diag] document.cookie userId after clear:', docCookieUserId || '(empty)');

    const warning = page.getByText(OWN_PROFILE_WARNING, { exact: false });
    const hasWarning = await warning.isVisible().catch(() => false);
    console.log('[diag] own-profile warning visible:', hasWarning);

    // 记录当前行为：若 profile 仍 200，警告不应出现（修复目标）
    if (profileViaPage.status === 200 && profileViaPage.user_id) {
      await expect(warning).not.toBeVisible();
    } else {
      console.log('[diag] profile 不可用，跳过断言');
    }
  });
});
