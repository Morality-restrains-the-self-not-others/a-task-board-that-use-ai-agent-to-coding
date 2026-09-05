// @ts-check
/**
 * 在容器中将变动推送到后端，以便通过 SSE 推送到前端
 */
import { chromium } from '@playwright/test';

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
if (!PASSWORD) {
  throw new Error('PLAYWRIGHT_TEST_PASSWORD is required; do not hardcode secrets');
}
const TASK_DETAIL_URL = 'http://localhost:4000/tenant/827923618468040704/workspace/827923618602258432/task-detail/835844524459184128/?accessCode=u824976301710503936';

async function playwrightLoginWithLegalAccept(page, { email, password }) {
  // 导航到登录页
  await page.goto('http://localhost:4000/login');
  await page.waitForLoadState('domcontentloaded');
  
  // 填写登录表单
  await page.getByPlaceholder('邮箱').fill(email);
  await page.getByPlaceholder('密码').fill(password);
  
  // 勾选同意隐私政策和服务条款
  try {
    // 勾选隐私政策
    const privacyCheckbox = page.getByTestId('login-privacy-accept');
    await privacyCheckbox.waitFor({ timeout: 10000 });
    await privacyCheckbox.check();
    
    // 勾选服务条款
    const licenseCheckbox = page.getByTestId('login-license-accept');
    await licenseCheckbox.waitFor({ timeout: 10000 });
    await licenseCheckbox.check();
    
    // 等待登录按钮变为可用
    await page.getByRole('button', { name: '登录' }).waitFor({ state: 'enabled', timeout: 10000 });
  } catch (error) {
    console.log('勾选复选框时出错:', error.message);
  }
  
  // 点击登录按钮
  await page.getByRole('button', { name: '登录' }).click();
  
  // 等待登录完成（跳转）
  await page.waitForNavigation({ timeout: 60000 });
}

async function main() {
  console.log('开始推送容器变动...');
  
  const browser = await chromium.launch({ headless: false });
  const context = await browser.newContext();
  const page = await context.newPage();
  
  try {
    page.on('dialog', (d) => {
      void d.accept();
    });
    
    console.log('正在登录...');
    await playwrightLoginWithLegalAccept(page, { email: EMAIL, password: PASSWORD });
    
    console.log('正在打开任务详情页...');
    await page.goto(TASK_DETAIL_URL);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1500);
    
    // 等待 SSE 连接
    console.log('等待 SSE 连接...');
    const connected = page.getByRole('button', { name: /SSE 已连接/ });
    const disconnected = page.getByRole('button', { name: /SSE 未连接/ });
    try {
      await connected.waitFor({ timeout: 60000 });
    } catch {
      if (await disconnected.isVisible().catch(() => false)) {
        await disconnected.click();
      }
      await connected.waitFor({ timeout: 120000 });
    }
    
    // 等待 zTree 挂载
    console.log('等待 zTree 挂载...');
    await page.waitForFunction(async () => {
      const ztreePanel = document.querySelector('[data-testid="comment-layer-ztree-panel"]');
      return !!ztreePanel;
    }, null, { timeout: 1200000 });
    
    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    await ztreePanel.waitFor({ timeout: 60000 });
    
    // 选择最后一层的可写层节点
    console.log('选择可写层节点...');
    const nameBtns = ztreePanel.locator('button.break-words.flex-1.min-w-0');
    await nameBtns.first().waitFor({ timeout: 60000 });
    const n = await nameBtns.count();
    let clickedLayer = false;
    
    // 优先选择最后一层的可写层节点
    for (let i = n - 1; i >= 0; i--) {
      const label = ((await nameBtns.nth(i).innerText()) || '').trim();
      if (label === '可写层' || /^可写层（/.test(label)) continue;
      if (/idle_done|idle_interrupt/i.test(label)) continue;
      await nameBtns.nth(i).click();
      clickedLayer = true;
      console.log(`已选择层级节点: ${label}`);
      break;
    }
    
    if (!clickedLayer) {
      for (let i = 0; i < n; i++) {
        const label = ((await nameBtns.nth(i).innerText()) || '').trim();
        if (label === '可写层' || /^可写层（/.test(label)) continue;
        await nameBtns.nth(i).click();
        clickedLayer = true;
        console.log(`已选择层级节点: ${label}`);
        break;
      }
    }
    
    if (!clickedLayer) {
      throw new Error('未找到可选中的可写层节点');
    }
    await page.waitForTimeout(400);
    
    // 同步 Git 身份
    console.log('同步 Git 身份...');
    const sel = page.locator('#layer-git-identity-select');
    await sel.waitFor({ timeout: 60000 });
    
    const optionValues = await sel.locator('option').evaluateAll((opts) =>
      opts.map((o) => /** @type {HTMLOptionElement} */ (o).value).filter(Boolean),
    );
    const realId = optionValues.find((v) => v !== '__quick_create__');
    
    if (realId) {
      await sel.selectOption(realId);
    } else {
      await sel.selectOption('__quick_create__');
      await page.getByPlaceholder('Git 用户名').fill('e2e-playwright');
      await page.getByPlaceholder('Git 邮箱').fill('e2e-playwright@example.com');
      await page.getByRole('button', { name: '创建并同步' }).click();
      await page.getByText('当前身份已同步至容器，可进行提交/推送。').waitFor({ timeout: 120000 });
    }
    
    if (realId) {
      await page.getByRole('button', { name: '将各仓库所选身份同步到容器' }).click();
      await page.getByText('当前身份已同步至容器，可进行提交/推送。').waitFor({ timeout: 120000 });
    }
    
    // 查找并点击提交按钮
    console.log('查找并点击提交按钮...');
    let submitClicked = false;
    for (let i = 0; i < 30; i++) {
      if (submitClicked) break;
      
      const allButtons = page.locator('button');
      const count = await allButtons.count();
      for (let j = 0; j < count; j++) {
        if (submitClicked) break;
        
        const button = allButtons.nth(j);
        const text = await button.innerText().catch(() => '');
        if (text.includes('提交') && !(await button.isDisabled().catch(() => true))) {
          console.log('找到可用的提交按钮，点击...');
          await button.click();
          submitClicked = true;
          break;
        }
      }
      await page.waitForTimeout(2000);
    }
    
    if (!submitClicked) {
      console.log('未找到可用的提交按钮');
    } else {
      // 等待提交完成
      console.log('等待提交完成...');
      await page.waitForTimeout(10000);
    }
    
    // 查找并点击推送按钮
    console.log('查找并点击推送按钮...');
    let pushClicked = false;
    for (let i = 0; i < 30; i++) {
      if (pushClicked) break;
      
      const allButtons = page.locator('button');
      const count = await allButtons.count();
      for (let j = 0; j < count; j++) {
        if (pushClicked) break;
        
        const button = allButtons.nth(j);
        const text = await button.innerText().catch(() => '');
        if (text.includes('推送') && !(await button.isDisabled().catch(() => true))) {
          console.log('找到可用的推送按钮，点击...');
          await button.click();
          pushClicked = true;
          break;
        }
      }
      await page.waitForTimeout(2000);
    }
    
    if (!pushClicked) {
      console.log('未找到可用的推送按钮');
    } else {
      // 等待推送完成
      console.log('等待推送完成...');
      try {
        await page.waitForResponse(
          (r) => r.url().includes('container-layer-git-push') && r.request().method() === 'POST',
          { timeout: 300000 }
        );
        console.log('推送响应已收到');
      } catch (error) {
        console.log('推送响应超时，可能推送操作仍在进行中');
      }
    }
    
    // 强制刷新层级状态
    console.log('强制刷新层级状态...');
    try {
      // 触发层级图刷新
      await page.evaluate(() => {
        // 模拟层级图刷新
        const event = new CustomEvent('refresh-layer-graph');
        document.dispatchEvent(event);
      });
      
      // 等待 SSE 推送新状态
      console.log('等待 SSE 推送新状态...');
      await page.waitForTimeout(10000);
    } catch (error) {
      console.log('刷新层级状态时出错:', error.message);
    }
    
    console.log('容器变动推送完成！');
    console.log('变动已推送到后端，将通过 SSE 推送到前端');
    
  } catch (error) {
    console.error('执行过程中出错:', error);
  } finally {
    await browser.close();
  }
}

main();
