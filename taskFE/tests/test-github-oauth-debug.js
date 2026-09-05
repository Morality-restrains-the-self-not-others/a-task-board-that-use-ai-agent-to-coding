// @ts-check
/**
 * 调试 GitHub OAuth 绑定流程的详细测试
 */
const { chromium } = require('playwright');

async function runTest() {
  console.log('=== GitHub OAuth 绑定调试测试 ===');
  
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
    // 1. 清除可能存在的状态
    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });

    // 2. 导航到登录页
    const loginUrl = 'http://localhost:4000/auth/login/';
    console.log(`\n1. 导航到登录页: ${loginUrl}`);
    await page.goto(loginUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // 3. 填写登录表单
    console.log('\n2. 填写登录表单');
    const emailInput = page.locator('#email');
    const passwordInput = page.locator('#password');
    
    await emailInput.fill('contact@daydaymoney.com');
    await passwordInput.fill(process.env.PLAYWRIGHT_TEST_PASSWORD);

    // 4. 同意条款并提交
    console.log('\n3. 同意条款并提交');
    const acceptAllCheckbox = page.locator('[data-testid="login-accept-all"]');
    await acceptAllCheckbox.check();

    await page.locator('form').first().evaluate((form) => form.requestSubmit());
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    // 5. 检查登录状态
    const currentUrl = page.url();
    console.log(`\n4. 登录后URL: ${currentUrl}`);
    
    const userIdCookie = await page.evaluate(() => document.cookie.split('; ').find(row => row.startsWith('userId='))?.split('=')[1]);
    console.log(`   userId Cookie: ${userIdCookie}`);

    // 6. 导航到 GitHub OAuth 设置页
    const gitSiteOauthUrl = 'http://localhost:4000/profile/git-site-oauth/';
    console.log(`\n5. 导航到 GitHub OAuth 设置页: ${gitSiteOauthUrl}`);
    await page.goto(gitSiteOauthUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    // 7. 检查页面内容
    const pageContent = await page.content();
    console.log(`\n6. 页面内容检查:`);
    
    if (pageContent.includes('尚未绑定 GitHub 账号')) {
      console.log('   ✗ 当前状态：尚未绑定 GitHub 账号');
    }
    if (pageContent.includes('已绑定 GitHub')) {
      console.log('   ✓ 当前状态：已绑定 GitHub');
    }
    if (pageContent.includes('使用 GitHub 授权')) {
      console.log('   ✓ 「使用 GitHub 授权」按钮存在');
    }

    // 8. 检查 localStorage 中的 OAuth 返回状态
    const githubOauthStorage = await page.evaluate(() => {
      const keys = Object.keys(localStorage).filter(k => k.startsWith('github_app_return:'));
      return keys.map(k => ({ key: k, value: localStorage.getItem(k) }));
    });
    console.log(`\n7. localStorage 中的 GitHub OAuth 状态:`);
    console.log(JSON.stringify(githubOauthStorage, null, 2));

    // 9. 检查网络请求（获取绑定状态的 API）
    console.log(`\n8. 检查 API 请求状态:`);
    const response = await page.evaluate(async () => {
      try {
        const res = await fetch('/api/accounts/github/app/connection/', {
          headers: { Accept: 'application/json' },
          credentials: 'include'
        });
        const data = await res.json();
        return { status: res.status, data };
      } catch (e) {
        return { error: e.message };
      }
    });
    console.log(`   API 响应状态: ${response.status}`);
    console.log(`   API 响应数据:`, JSON.stringify(response.data, null, 2));

    // 10. 如果未绑定，尝试启动 OAuth 流程并观察
    if (pageContent.includes('尚未绑定 GitHub 账号') && pageContent.includes('使用 GitHub 授权')) {
      console.log(`\n9. 尝试启动 OAuth 流程:`);
      
      const connectButton = page.getByRole('button', { name: '使用 GitHub 授权' });
      if (await connectButton.isVisible()) {
        console.log('   点击「使用 GitHub 授权」按钮');
        
        // 监听网络请求
        page.on('request', request => {
          if (request.url().includes('/api/git-oauth/github-app-start/')) {
            console.log(`   → 请求 OAuth start: ${request.url()}`);
          }
        });

        page.on('response', response => {
          if (response.url().includes('/api/git-oauth/github-app-start/')) {
            response.json().then(data => {
              console.log(`   ← OAuth start 响应:`, data);
            }).catch(() => {});
          }
        });

        try {
          const [navigation] = await Promise.all([
            page.waitForNavigation({ url: /github\.com/, timeout: 15000 }),
            connectButton.click()
          ]);
          
          if (navigation?.url().includes('github.com')) {
            console.log('   ✓ 成功跳转到 GitHub 授权页');
            console.log(`   GitHub URL: ${navigation.url()}`);
          }
        } catch (error) {
          console.log('   ✗ 跳转失败:', error.message);
        }
      }
    }

    console.log('\n=== 调试测试完成 ===');
    
  } catch (error) {
    console.error('测试过程中发生错误:', error.message);
    console.error(error.stack);
  } finally {
    console.log('\n保持浏览器连接开放供查看...');
  }
}

runTest().catch(console.error);