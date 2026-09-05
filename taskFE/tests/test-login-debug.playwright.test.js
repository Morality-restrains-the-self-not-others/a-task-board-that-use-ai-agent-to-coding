// @ts-check
/**
 * 调试登录流程，检查登录后认证状态是否正确设置
 */
import { test, expect } from '@playwright/test';

test.describe('登录流程调试', () => {
  test('检查登录后认证状态', async ({ page }) => {
    const loginUrl = '/auth/login/';
    
    await page.goto(loginUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    const emailInput = page.locator('#email');
    const passwordInput = page.locator('#password');
    await emailInput.fill('contact@daydaymoney.com');
    await passwordInput.fill(process.env.PLAYWRIGHT_TEST_PASSWORD);

    const acceptAllCheckbox = page.locator('[data-testid="login-accept-all"]');
    await acceptAllCheckbox.check();

    let loginResponse = null;
    page.on('response', async (response) => {
      if (response.url().includes('/api/auth/')) {
        try {
          loginResponse = await response.json();
          console.log('登录 API 响应:', JSON.stringify(loginResponse, null, 2));
        } catch (e) {
          console.log('登录 API 响应解析失败:', e.message);
        }
      }
    });

    await page.locator('form').first().evaluate((form) => form.requestSubmit());
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    console.log('登录后 URL:', page.url());

    const userIdCookie = await page.evaluate(() => 
      document.cookie.split('; ').find(row => row.startsWith('userId='))?.split('=')[1]
    );
    console.log('userId Cookie:', userIdCookie);

    const authToken = await page.evaluate(() => localStorage.getItem('authToken'));
    console.log('authToken:', authToken ? '已设置' : '未设置');

    const sessionIdCookie = await page.evaluate(() => 
      document.cookie.split('; ').find(row => row.startsWith('sessionid='))?.split('=')[1]
    );
    console.log('sessionid Cookie:', sessionIdCookie ? '已设置' : '未设置');

    await page.goto('/profile/git-site-oauth/');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    console.log('访问 OAuth 页面后的 URL:', page.url());

    if (page.url().includes('/auth/login/')) {
      console.log('✗ 被重定向回登录页，认证状态可能未正确设置');
    } else {
      console.log('✓ 成功访问受保护页面');
    }
  });
});