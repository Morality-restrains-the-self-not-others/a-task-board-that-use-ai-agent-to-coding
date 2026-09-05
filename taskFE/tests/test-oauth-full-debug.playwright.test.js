// @ts-check
/**
 * 详细调试 GitHub OAuth 完整流程
 */
import { test, expect } from '@playwright/test';

test.describe('GitHub OAuth 完整流程调试', () => {
  test('完整 OAuth 绑定流程测试', async ({ page }) => {
    // 1. 登录
    const loginUrl = '/auth/login/';
    await page.goto(loginUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    await page.locator('#email').fill('contact@daydaymoney.com');
    await page.locator('#password').fill(process.env.PLAYWRIGHT_TEST_PASSWORD);
    await page.locator('[data-testid="login-accept-all"]').check();
    await page.locator('form').first().evaluate((form) => form.requestSubmit());
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    console.log('登录后 URL:', page.url());

    // 2. 检查登录状态
    const userIdCookie = await page.evaluate(() => 
      document.cookie.split('; ').find(row => row.startsWith('userId='))?.split('=')[1]
    );
    console.log('userId Cookie:', userIdCookie);

    // 3. 导航到 OAuth 设置页
    await page.goto('/profile/git-site-oauth/');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    console.log('OAuth 页面 URL:', page.url());

    // 4. 检查当前绑定状态
    const connectionStatus = await page.evaluate(async () => {
      try {
        const res = await fetch('/api/accounts/github/app/connection/', {
          headers: { Accept: 'application/json' },
          credentials: 'include'
        });
        return await res.json();
      } catch (e) {
        return { error: e.message };
      }
    });

    console.log('当前绑定状态:', JSON.stringify(connectionStatus, null, 2));

    if (connectionStatus.connected) {
      console.log('用户已绑定 GitHub，跳过测试');
      return;
    }

    // 5. 检查页面状态显示
    const pageContent = await page.content();
    if (pageContent.includes('尚未绑定 GitHub 账号')) {
      console.log('页面显示：尚未绑定 GitHub 账号');
    }

    // 6. 尝试启动 OAuth 流程
    const connectButton = page.getByRole('button', { name: '使用 GitHub 授权' });
    if (await connectButton.isVisible()) {
      console.log('点击「使用 GitHub 授权」按钮');
      
      let startResponse = null;
      page.on('response', async (response) => {
        if (response.url().includes('/api/git-oauth/github-app-start/')) {
          try {
            startResponse = await response.json();
            console.log('OAuth Start 响应:', JSON.stringify(startResponse, null, 2));
          } catch (e) {
            console.log('OAuth Start 响应解析失败:', e.message);
          }
        }
      });

      try {
        const [navigation] = await Promise.all([
          page.waitForNavigation({ url: /github\.com/, timeout: 15000 }),
          connectButton.click()
        ]);

        if (navigation?.url().includes('github.com')) {
          console.log('成功跳转到 GitHub 授权页');
          console.log('GitHub URL:', navigation.url());
          
          // 检查 URL 参数
          const url = new URL(navigation.url());
          console.log('State 参数:', url.searchParams.get('state'));
        }
      } catch (error) {
        console.log('跳转失败:', error.message);
      }
    } else {
      console.log('「使用 GitHub 授权」按钮不可见');
    }
  });

  test('模拟 OAuth 回调后状态检查', async ({ page }) => {
    // 1. 登录
    await page.goto('/auth/login/');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    await page.locator('#email').fill('contact@daydaymoney.com');
    await page.locator('#password').fill(process.env.PLAYWRIGHT_TEST_PASSWORD);
    await page.locator('[data-testid="login-accept-all"]').check();
    await page.locator('form').first().evaluate((form) => form.requestSubmit());
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    // 2. 模拟 OAuth 回调返回（带 github=ok 参数）
    await page.goto('/profile/git-site-oauth/?github=ok');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    // 3. 检查页面内容
    const pageContent = await page.content();
    
    console.log('=== 回调后页面检查 ===');
    
    if (pageContent.includes('GitHub 授权成功')) {
      console.log('✓ 显示成功消息');
    } else {
      console.log('✗ 未显示成功消息');
    }

    if (pageContent.includes('已绑定 GitHub')) {
      console.log('✓ 显示已绑定状态');
    } else if (pageContent.includes('尚未绑定 GitHub 账号')) {
      console.log('✗ 仍显示未绑定状态 - 这是问题所在！');
      
      // 检查 API 返回的状态
      const status = await page.evaluate(async () => {
        const res = await fetch('/api/accounts/github/app/connection/', {
          headers: { Accept: 'application/json' },
          credentials: 'include'
        });
        return await res.json();
      });
      
      console.log('API 返回的绑定状态:', JSON.stringify(status, null, 2));
      
      if (status.connected === false) {
        console.log('问题分析：API 返回 connected: false');
        console.log('可能原因：');
        console.log('1. OAuth 回调时用户未登录（session 丢失）');
        console.log('2. OAuth 回调处理失败（数据库未更新）');
        console.log('3. 用户账号在回调时发生了变化');
      }
    }
  });
});