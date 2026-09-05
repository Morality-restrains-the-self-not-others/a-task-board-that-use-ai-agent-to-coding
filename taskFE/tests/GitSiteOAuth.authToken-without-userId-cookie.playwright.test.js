// @ts-check
/**
 * 复现根因：authToken 有效 + userId cookie 缺失 → 路由/Navbar 认为已登录，但 isOwnProfile 为 false。
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const LOGIN = {
  email: 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

const OWN_PROFILE_WARNING =
  '仅登录用户本人可在账号中心管理 GitHub 授权';

test.describe('GitSiteOAuth authToken 无 userId cookie', () => {
  test('仅清除 userId cookie 保留 authToken 时会误显非本人警告', async ({ page }) => {
    await playwrightLoginWithLegalAccept(page, LOGIN);

    await page.evaluate(() => {
      document.cookie = 'userId=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/';
    });

    const authState = await page.evaluate(async () => {
      const authToken = localStorage.getItem('authToken');
      const cookieUserId = document.cookie.split('; ').find((r) => r.startsWith('userId='))?.split('=')[1] || '';
      const headers = { Accept: 'application/json' };
      if (authToken) headers.Authorization = `Token ${authToken}`;
      const r = await fetch('/api/accounts/users/profile/', {
        credentials: 'include',
        headers,
      });
      const body = await r.json().catch(() => ({}));
      return {
        authToken: authToken ? '(set)' : '(missing)',
        cookieUserId: cookieUserId || '(empty)',
        profileStatus: r.status,
        profileUserId: body?.user_id,
      };
    });
    console.log('[diag] auth state after userId cookie clear:', authState);

    expect(authState.authToken).toBe('(set)');
    expect(authState.cookieUserId).toBe('(empty)');
    expect(authState.profileStatus).toBe(200);
    expect(authState.profileUserId).toBeTruthy();

    await page.goto('/profile/git-site-oauth/', { waitUntil: 'networkidle' });

    const warning = page.getByText(OWN_PROFILE_WARNING, { exact: false });
    const hasWarning = await warning.isVisible().catch(() => false);
    console.log('[diag] own-profile warning visible (bug if true):', hasWarning);

    // 当前实现会误报；修复后此断言应通过
    await expect(warning).not.toBeVisible();
  });
});
