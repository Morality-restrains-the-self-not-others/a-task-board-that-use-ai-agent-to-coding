// @ts-check
/**
 * 核验：项目详情页点击「OAuth 授权」后，完成 OAuth 流程应跳回当前项目详情页，而非首页
 * 测试账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 * 说明：完整 OAuth 需真实 GitHub 授权，本测试验证 (1) 点击后发起 OAuth 流程 (2) 回调重定向逻辑由后端单测覆盖
 */
import { test, expect } from '@playwright/test';

test.describe('项目详情页 OAuth 授权', () => {
  test('点击 OAuth 授权后应跳转到 GitHub 授权页（发起 OAuth 流程）', async ({ page }) => {
    const loginUrl = '/auth/login/';
    const projectDetailUrl =
      '/tenant/822445079575764992/projects/822447075168653312/';

    await page.goto(loginUrl);
    await page.waitForLoadState('networkidle');

    const emailPasswordTab = page
      .locator('text=邮箱')
      .or(page.locator('button:has-text("邮箱")'))
      .first();
    if (await emailPasswordTab.isVisible()) {
      await emailPasswordTab.click();
      await page.waitForTimeout(300);
    }

    const emailInput = page.locator('#email');
    const passwordInput = page.locator('#password');
    await expect(emailInput).toBeVisible();
    await expect(passwordInput).toBeVisible();

    await emailInput.fill('contact@daydaymoney.com');
    await passwordInput.fill(process.env.PLAYWRIGHT_TEST_PASSWORD);

    await page.locator('form').first().evaluate((form) => form.requestSubmit());
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    await page.goto(projectDetailUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1500);

    const oauthButton = page.getByRole('button', { name: 'OAuth 授权' });
    if (!(await oauthButton.isVisible())) {
      test.skip(true, '项目未设置 git_repos，OAuth 按钮不显示，跳过 OAuth 流程验证');
    }

    const [navigation] = await Promise.all([
      page.waitForNavigation({ url: /github\.com/, timeout: 15000 }),
      oauthButton.click()
    ]);

    expect(navigation?.url()).toContain('github.com');
  });
});
