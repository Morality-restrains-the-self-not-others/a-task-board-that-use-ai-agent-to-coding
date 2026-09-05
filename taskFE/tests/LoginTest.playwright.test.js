import { test, expect } from '@playwright/test';

test('Login test', async ({ page }) => {
  test.skip(process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前后端与账号就绪的登录 E2E，请本地或 CI 单独执行');
  // 1. 登录页面
  await page.goto('http://localhost:4000/login');
  
  // 等待登录页面加载
  await page.waitForLoadState('networkidle');
  
  // 检查页面标题
  console.log('Login page title:', await page.title());
  
  // 输入账号密码
  await page.fill('input#email', 'author@example.com');
  await page.fill('input#password', process.env.PLAYWRIGHT_TEST_PASSWORD);
  
  // 点击登录按钮
  await page.click('button:has-text("登录")');
  
  // 等待登录完成
  await page.waitForNavigation({ waitUntil: 'networkidle', timeout: 10000 });
  
  // 查看登录后的当前页面
  console.log('Current URL after login:', page.url());
  console.log('Page title after login:', await page.title());
  
  // 检查是否登录成功
  expect(page.url()).not.toContain('login');
  
  console.log('Login test completed!');
});
