// @ts-check
/**
 * 诊断 /profile/git-site-oauth/ 在已登录时仍显示「仅登录用户本人…」的原因。
 * 根因假设：路由守卫用 profile API 判定已登录，但 isOwnProfile 仅读 userId cookie。
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const LOGIN = {
  email: 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

const OWN_PROFILE_WARNING =
  '仅登录用户本人可在账号中心管理 GitHub 授权';

test.describe('GitSiteOAuth isOwnProfile 诊断', () => {
  test('登录后 /profile/git-site-oauth/ 应可管理授权（非本人警告）', async ({ page }) => {
    await playwrightLoginWithLegalAccept(page, LOGIN);

    const cookiesAfterLogin = await page.context().cookies();
    const userIdCookie = cookiesAfterLogin.find((c) => c.name === 'userId');
    const sessionCookie = cookiesAfterLogin.find((c) => c.name === 'sessionid');

    console.log('[diag] login cookies:', {
      userId: userIdCookie?.value ?? '(missing)',
      sessionid: sessionCookie?.value ? '(set)' : '(missing)',
    });

    const profileResp = await page.request.get('/api/accounts/users/profile/', {
      headers: { Accept: 'application/json' },
    });
    const profileBody = await profileResp.json().catch(() => ({}));
    console.log('[diag] profile API:', {
      status: profileResp.status(),
      user_id: profileBody?.user_id,
    });

    await page.goto('/profile/git-site-oauth/', { waitUntil: 'networkidle' });

    const docCookieUserId = await page.evaluate(() => {
      const row = document.cookie.split('; ').find((r) => r.startsWith('userId='));
      return row ? decodeURIComponent(row.split('=')[1]) : '';
    });
    console.log('[diag] document.cookie userId:', docCookieUserId || '(empty)');

    const warning = page.getByText(OWN_PROFILE_WARNING, { exact: false });
    const hasWarning = await warning.isVisible().catch(() => false);
    console.log('[diag] own-profile warning visible:', hasWarning);

    const view = page.locator('[data-alias="view-user-git-site-oauth"]');
    await expect(view).toBeVisible();

    // 期望：已登录用户访问 /profile/git-site-oauth/ 不应看到「非本人」警告
    await expect(warning).not.toBeVisible();

    const oauthHeading = page.getByRole('heading', { name: 'Git 网站授权设置' });
    await expect(oauthHeading).toBeVisible();
  });
});
