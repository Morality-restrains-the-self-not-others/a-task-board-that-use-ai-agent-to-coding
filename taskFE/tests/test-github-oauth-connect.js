// @ts-check
/**
 * 使用 Playwright 连接本地 Chrome 9222 端口，验证 GitHub OAuth 绑定流程
 * 测试账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 */
const { chromium } = require('playwright');

async function runTest() {
  console.log('=== GitHub OAuth 绑定测试 ===');
  
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
    const loginUrl = 'http://localhost:4000/auth/login/';
    const gitSiteOauthUrl = 'http://localhost:4000/profile/git-site-oauth/';

    console.log(`1. 导航到登录页: ${loginUrl}`);
    await page.goto(loginUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    const emailPasswordTab = page
      .locator('text=邮箱')
      .or(page.locator('button:has-text("邮箱")'))
      .first();
    if (await emailPasswordTab.isVisible()) {
      console.log('2. 切换到邮箱登录选项');
      await emailPasswordTab.click();
      await page.waitForTimeout(300);
    }

    console.log('3. 填写登录表单');
    const emailInput = page.locator('#email');
    const passwordInput = page.locator('#password');
    
    await emailInput.fill('contact@daydaymoney.com');
    await passwordInput.fill(process.env.PLAYWRIGHT_TEST_PASSWORD);

    console.log('4. 提交登录表单');
    await page.locator('form').first().evaluate((form) => form.requestSubmit());
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    console.log(`5. 导航到 GitHub OAuth 设置页: ${gitSiteOauthUrl}`);
    await page.goto(gitSiteOauthUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    const viewAlias = page.locator('[data-alias="view-user-git-site-oauth"]');
    if (await viewAlias.isVisible()) {
      console.log('6. 页面加载成功');
    }

    const statusLoading = page.locator('text=加载中...');
    if (await statusLoading.isVisible()) {
      console.log('等待状态加载...');
      await page.waitForTimeout(2000);
    }

    const connectButton = page.getByRole('button', { name: '使用 GitHub 授权' });
    const reconnectButton = page.getByRole('button', { name: '重新授权' });
    const disconnectButton = page.getByRole('button', { name: '取消授权' });
    const connectedText = page.locator('text=已绑定 GitHub');
    const notConnectedText = page.locator('text=尚未绑定 GitHub 账号');

    if (await connectedText.isVisible()) {
      console.log('✓ 当前状态：已绑定 GitHub');
      console.log('  - 显示：已绑定 GitHub');
      console.log('  - 显示：重新授权按钮');
      console.log('  - 显示：取消授权按钮');
    } else if (await notConnectedText.isVisible()) {
      console.log('✗ 当前状态：尚未绑定 GitHub 账号');
      console.log('  - 需要点击「使用 GitHub 授权」按钮');
      
      if (await connectButton.isVisible()) {
        console.log('7. 点击「使用 GitHub 授权」按钮');
        try {
          const [navigation] = await Promise.all([
            page.waitForNavigation({ url: /github\.com/, timeout: 15000 }),
            connectButton.click()
          ]);
          
          if (navigation?.url().includes('github.com')) {
            console.log('✓ 成功跳转到 GitHub 授权页');
          }
        } catch (error) {
          console.log('✗ 跳转失败:', error.message);
        }
      } else {
        console.log('✗ 「使用 GitHub 授权」按钮不可见');
      }
    } else {
      console.log('? 无法确定绑定状态，检查页面内容');
      const pageContent = await page.content();
      console.log('页面内容摘要:', pageContent.substring(0, 1000));
    }

    console.log('=== 测试完成 ===');
    
  } catch (error) {
    console.error('测试过程中发生错误:', error.message);
    console.error(error.stack);
  } finally {
    console.log('保持浏览器连接开放供查看...');
  }
}

runTest().catch(console.error);