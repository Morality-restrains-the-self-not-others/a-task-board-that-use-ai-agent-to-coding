// @ts-check
/**
 * 测试 GitHub OAuth 完整回调流程
 * 模拟从授权到回调返回的整个流程
 */
const { chromium } = require('playwright');

async function runTest() {
  console.log('=== GitHub OAuth 完整回调流程测试 ===');
  
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

    // 3. 导航到 OAuth 设置页
    console.log('\n2. 导航到 GitHub OAuth 设置页');
    const gitSiteOauthUrl = 'http://localhost:4000/profile/git-site-oauth/';
    await page.goto(gitSiteOauthUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // 4. 检查初始状态
    console.log('\n3. 检查初始绑定状态');
    const initialContent = await page.content();
    
    if (initialContent.includes('尚未绑定 GitHub 账号')) {
      console.log('   当前状态：尚未绑定');
    } else if (initialContent.includes('已绑定 GitHub')) {
      console.log('   当前状态：已绑定');
    }

    // 5. 模拟 OAuth 回调流程
    console.log('\n4. 模拟 OAuth 回调流程');
    
    // 首先获取当前页面的完整 URL（包含查询参数）
    const currentUrl = page.url();
    console.log(`   当前页面 URL: ${currentUrl}`);
    
    // 模拟后端回调后重定向回来（带 github=ok 参数）
    const callbackUrl = currentUrl.includes('?') 
      ? `${currentUrl}&github=ok` 
      : `${currentUrl}?github=ok`;
    
    console.log(`   模拟回调重定向到: ${callbackUrl}`);
    await page.goto(callbackUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    // 6. 检查回调后的状态
    console.log('\n5. 检查回调后的绑定状态');
    const afterCallbackContent = await page.content();
    
    if (afterCallbackContent.includes('GitHub 授权成功')) {
      console.log('   ✓ 显示成功消息：GitHub 授权成功');
    } else {
      console.log('   ✗ 未显示成功消息');
    }

    if (afterCallbackContent.includes('已绑定 GitHub')) {
      console.log('   ✓ 绑定状态：已绑定');
    } else if (afterCallbackContent.includes('尚未绑定 GitHub 账号')) {
      console.log('   ✗ 绑定状态：仍显示未绑定 - 这是问题所在！');
      
      // 检查是否是因为没有重新获取状态
      console.log('\n6. 检查 API 请求是否被发送');
      const status = await page.evaluate(async () => {
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
      
      console.log('   API 获取的绑定状态:', JSON.stringify(status, null, 2));
      
      if (status.connected === false) {
        console.log('   问题分析：API 返回 connected: false，说明后端确实没有绑定记录');
        console.log('   可能原因：');
        console.log('   1. OAuth 回调时用户未登录（session 丢失）');
        console.log('   2. OAuth 回调处理失败（日志中可能有错误）');
        console.log('   3. 用户账号在回调时发生了变化');
      }
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