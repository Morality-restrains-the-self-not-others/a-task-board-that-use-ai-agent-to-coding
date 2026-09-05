// @ts-check
/**
 * GitHub OAuth 完整流程测试
 * 验证从登录到授权绑定的完整流程
 */
import { chromium } from 'playwright';

async function runTest() {
  console.log('=== GitHub OAuth 完整流程测试 ===');
  
  const browser = await chromium.connectOverCDP('http://localhost:9222');
  
  const defaultContext = browser.contexts()[0];
  let page;
  
  if (defaultContext) {
    page = defaultContext.pages()[0] || await defaultContext.newPage();
  } else {
    const context = await browser.newContext();
    page = await context.newPage();
  }

  try {
    // 1. 清除状态
    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });

    // 2. 登录
    console.log('\n1. 登录到系统');
    const loginUrl = 'http://localhost:4000/auth/login/';
    await page.goto(loginUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    const emailInput = page.locator('#email');
    const passwordInput = page.locator('#password');
    await emailInput.fill('contact@daydaymoney.com');
    await passwordInput.fill(process.env.PLAYWRIGHT_TEST_PASSWORD);

    const acceptAllCheckbox = page.locator('[data-testid="login-accept-all"]');
    await acceptAllCheckbox.check();

    await page.locator('form').first().evaluate((form) => form.requestSubmit());
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    // 检查登录是否成功
    const userIdCookie = await page.evaluate(() => 
      document.cookie.split('; ').find(row => row.startsWith('userId='))?.split('=')[1]
    );
    console.log(`   用户ID Cookie: ${userIdCookie}`);

    if (!userIdCookie) {
      console.log('   ✗ 登录可能未成功，userId cookie 不存在');
      return;
    }

    // 3. 导航到 OAuth 设置页
    console.log('\n2. 导航到 GitHub OAuth 设置页');
    const gitSiteOauthUrl = 'http://localhost:4000/profile/git-site-oauth/';
    await page.goto(gitSiteOauthUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // 4. 检查当前绑定状态
    console.log('\n3. 检查当前绑定状态');
    const initialStatus = await page.evaluate(async () => {
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

    console.log('   当前绑定状态:', JSON.stringify(initialStatus, null, 2));

    if (initialStatus.connected) {
      console.log('   ✓ 用户已绑定 GitHub');
      console.log('   GitHub Login:', initialStatus.github_login);
      console.log('   GitHub User ID:', initialStatus.github_user_id);
      return;
    }

    console.log('   ✗ 用户尚未绑定 GitHub');

    // 5. 尝试启动 OAuth 流程
    console.log('\n4. 尝试启动 OAuth 流程');
    
    // 监听网络请求
    let oauthStartResponse = null;
    page.on('response', async (response) => {
      if (response.url().includes('/api/git-oauth/github-app-start/')) {
        try {
          oauthStartResponse = await response.json();
          console.log('   OAuth Start API 响应:', JSON.stringify(oauthStartResponse, null, 2));
        } catch (e) {
          console.log('   OAuth Start API 响应解析失败:', e.message);
        }
      }
    });

    const connectButton = page.getByRole('button', { name: '使用 GitHub 授权' });
    if (await connectButton.isVisible()) {
      console.log('   点击「使用 GitHub 授权」按钮');
      
      try {
        const [navigation] = await Promise.all([
          page.waitForNavigation({ url: /github\.com/, timeout: 15000 }),
          connectButton.click()
        ]);
        
        if (navigation?.url().includes('github.com')) {
          console.log('   ✓ 成功跳转到 GitHub 授权页');
          console.log(`   GitHub URL: ${navigation.url()}`);
          
          // 检查 URL 中的参数
          const url = new URL(navigation.url());
          console.log('   URL 参数:');
          console.log('     client_id:', url.searchParams.get('client_id'));
          console.log('     redirect_uri:', url.searchParams.get('redirect_uri'));
          console.log('     scope:', url.searchParams.get('scope'));
          console.log('     state:', url.searchParams.get('state'));
          
          // 检查 state 是否包含 return_key（32位十六进制）
          const state = url.searchParams.get('state');
          if (state && state.includes(':')) {
            const parts = state.split(':');
            if (parts.length === 2 && /^[0-9a-f]{32}$/.test(parts[1])) {
              console.log('   ✓ state 包含有效的 return_key');
            }
          }
        }
      } catch (error) {
        console.log('   ✗ 跳转失败:', error.message);
      }
    } else {
      console.log('   ✗ 「使用 GitHub 授权」按钮不可见');
    }

    console.log('\n=== 测试完成 ===');
    
  } catch (error) {
    console.error('测试过程中发生错误:', error.message);
    console.error(error.stack);
  } finally {
    console.log('保持浏览器连接开放供查看...');
  }
}

runTest().catch(console.error);