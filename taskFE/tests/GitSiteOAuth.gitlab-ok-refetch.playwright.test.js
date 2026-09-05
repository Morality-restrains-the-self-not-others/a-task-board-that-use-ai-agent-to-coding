// @ts-check
/**
 * 核验：GitLab OAuth 回跳带 ?provider=gitlab&gitlab=ok 时，页面应请求 gitlab 连接接口并显示已绑定。
 */
import { test, expect } from '@playwright/test';

test.describe('Git 网站 OAuth 回跳后状态刷新（GitLab）', () => {
  test('gitlab=ok 时应显示已绑定 GitLab', async ({ page, context }) => {
    const uid = '827923618451263488';
    await context.addCookies([
      {
        name: 'userId',
        value: uid,
        domain: 'localhost',
        path: '/',
      },
    ]);

    let hit = 0;
    await page.route('**/api/accounts/gitlab/app/connection**', async (route) => {
      if (route.request().method() !== 'GET') {
        await route.continue();
        return;
      }
      hit += 1;
      const connected = hit >= 2;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          connected,
          github_login: connected ? 'gitlab-mock' : null,
          github_user_id: connected ? '101' : null,
          scope: connected ? 'read_repository api' : null,
          authorize_scope: 'read_repository api read_user',
        }),
      });
    });

    await page.goto(`/user/${uid}/profile/git-site-oauth/?provider=gitlab&gitlab=ok`, {
      waitUntil: 'domcontentloaded',
    });

    await expect(page.getByText('已绑定 GitLab')).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('gitlab-mock')).toBeVisible();
    expect(hit).toBeGreaterThanOrEqual(2);
  });
});
