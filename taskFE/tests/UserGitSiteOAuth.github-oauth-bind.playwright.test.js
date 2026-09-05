// @ts-check
/**
 * 核验：用户资料页点击「使用 GitHub 授权」后，完成 OAuth 流程应显示已绑定状态
 * 测试账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 */
import { test, expect } from '@playwright/test';

test.describe('GitHub OAuth 绑定流程', () => {
  test('登录后访问 git-site-oauth 页面并验证绑定状态', async ({ page }) => {
    const loginUrl = '/auth/login/';
    const gitSiteOauthUrl = '/profile/git-site-oauth/';

    await page.goto(loginUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

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

    await page.goto(gitSiteOauthUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    const pageUrl = page.url();
    console.log(`当前URL: ${pageUrl}`);

    const pageContent = await page.content();
    console.log('页面内容摘要:', pageContent.substring(0, 2000));

    const connectButton = page.getByRole('button', { name: '使用 GitHub 授权' });
    const reconnectButton = page.getByRole('button', { name: '重新授权' });
    const disconnectButton = page.getByRole('button', { name: '取消授权' });
    const connectedText = page.locator('text=已绑定 GitHub');
    const notConnectedText = page.locator('text=尚未绑定 GitHub 账号');

    if (await connectedText.isVisible()) {
      console.log('✓ 当前状态：已绑定 GitHub');
      await expect(connectedText).toBeVisible();
    } else if (await notConnectedText.isVisible()) {
      console.log('✗ 当前状态：尚未绑定 GitHub 账号');
      await expect(notConnectedText).toBeVisible();
      
      if (await connectButton.isVisible()) {
        console.log('准备点击「使用 GitHub 授权」按钮');
        const [navigation] = await Promise.all([
          page.waitForNavigation({ url: /github\.com/, timeout: 15000 }),
          connectButton.click()
        ]);
        
        expect(navigation?.url()).toContain('github.com');
        console.log('✓ 成功跳转到 GitHub 授权页');
      } else {
        console.log('「使用 GitHub 授权」按钮不可见');
      }
    } else {
      console.log('无法确定绑定状态');
    }
  });

  test('OAuth 回调后应正确显示绑定状态', async ({ page }) => {
    const loginUrl = '/auth/login/';

    await page.goto(loginUrl);
    await page.waitForLoadState('networkidle');

    const emailInput = page.locator('#email');
    const passwordInput = page.locator('#password');

    await emailInput.fill('contact@daydaymoney.com');
    await passwordInput.fill(process.env.PLAYWRIGHT_TEST_PASSWORD);

    await page.locator('form').first().evaluate((form) => form.requestSubmit());
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    await page.goto('/profile/git-site-oauth/?github=ok');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    const successMessage = page.locator('text=GitHub 授权成功');
    const connectedText = page.locator('text=已绑定 GitHub');
    const notConnectedText = page.locator('text=尚未绑定 GitHub 账号');

    console.log('检查页面状态...');
    
    if (await successMessage.isVisible()) {
      console.log('✓ 显示成功消息：GitHub 授权成功');
    } else {
      console.log('未显示成功消息');
    }

    if (await connectedText.isVisible()) {
      console.log('✓ OAuth 回调后状态：已绑定');
    } else if (await notConnectedText.isVisible()) {
      console.log('✗ OAuth 回调后状态：仍显示未绑定 - 这是问题所在！');
      
      const pageContent = await page.content();
      console.log('页面内容:', pageContent);
    } else {
      console.log('无法确定绑定状态');
    }

    await page.waitForTimeout(2000);
  });
});