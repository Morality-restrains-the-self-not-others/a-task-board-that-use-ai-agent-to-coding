// @ts-check
/**
 * 核验：OAuth 回跳带 ?github=ok 时，若「连接」接口前几拍仍返回未绑定，页面应重试并最终显示已绑定。
 * 不依赖真实 GitHub / gitOauth：拦截 GET …/github/app/connection/ 并分次返回。
 */
import { test, expect } from '@playwright/test';

test.describe('Git 网站 OAuth 回跳后状态刷新', () => {
  test('github=ok 且连接接口延迟就绪时应显示已绑定', async ({ page, context }) => {
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
    await page.route('**/api/accounts/github/app/connection**', async (route) => {
      if (route.request().method() !== 'GET') {
        await route.continue();
        return;
      }
      hit += 1;
      const connected = hit >= 3;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          connected,
          github_login: connected ? 'playwright-mock' : null,
          github_user_id: connected ? 999001 : null,
          scope: connected ? 'repo' : null,
          authorize_scope: 'repo read:user',
        }),
      });
    });

    await page.goto(`/user/${uid}/profile/git-site-oauth/?github=ok`, {
      waitUntil: 'domcontentloaded',
    });

    await expect(page.getByText('已绑定 GitHub')).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('playwright-mock')).toBeVisible();
    expect(hit).toBeGreaterThanOrEqual(3);
  });
});
