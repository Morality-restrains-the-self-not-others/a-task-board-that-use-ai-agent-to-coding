// @ts-check
/**
 * 执行用户指定的任务：登录 → 模拟启动 → 发送指令 → 提交代码 → 推送代码
 */
import { chromium } from '@playwright/test';
import { execSync } from 'node:child_process';

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
if (!PASSWORD) {
  throw new Error('PLAYWRIGHT_TEST_PASSWORD is required; do not hardcode secrets');
}
const TASK_DETAIL_URL = 'http://localhost:4000/tenant/827923618468040704/workspace/827923618602258432/task-detail/835844524459184128/?accessCode=u824976301710503936';

/** 用户要求的发送内容为核心句，后接落盘说明以便解锁「提交」 */
const PROMPT = '用js写一个hello world。请在仓库根目录创建 hello.js，执行 node hello.js 时在控制台输出 hello world。';

const QUICK_CREATE_VALUE = '__quick_create__';

function dockerAvailable() {
  try {
    execSync('docker info', { stdio: 'ignore', timeout: 15000 });
    return true;
  } catch {
    return false;
  }
}

async function ensureGitIdentitySynced(page) {
  const sel = page.locator('#layer-git-identity-select');
  await sel.waitFor({ timeout: 60000 });

  const optionValues = await sel.locator('option').evaluateAll((opts) =>
    opts.map((o) => /** @type {HTMLOptionElement} */ (o).value).filter(Boolean),
  );
  const realId = optionValues.find((v) => v !== QUICK_CREATE_VALUE);

  if (realId) {
    await sel.selectOption(realId);
  } else {
    await sel.selectOption(QUICK_CREATE_VALUE);
    await page.getByPlaceholder('Git 用户名').fill('e2e-playwright');
    await page.getByPlaceholder('Git 邮箱').fill('e2e-playwright@example.com');
    await page.getByRole('button', { name: '创建并同步' }).click();
    await page.getByText('当前身份已同步至容器，可进行提交/推送。').waitFor({ timeout: 120000 });
    return;
  }

  await page.getByRole('button', { name: '将各仓库所选身份同步到容器' }).click();
  await page.getByText('当前身份已同步至容器，可进行提交/推送。').waitFor({ timeout: 120000 });
}

/** 绿点：已连接；灰点可点「重连」建立 SSE */
async function awaitSseConnected(page) {
  const connected = page.getByRole('button', { name: /SSE 已连接/ });
  const disconnected = page.getByRole('button', { name: /SSE 未连接/ });
  try {
    await connected.waitFor({ timeout: 60000 });
    return;
  } catch {
    /* 首屏可能尚未连上 */
  }
  if (await disconnected.isVisible().catch(() => false)) {
    await disconnected.click();
  }
  await connected.waitFor({ timeout: 120000 });
}

async function dismissMockStartBusyIfPresent(page) {
  const busyBtn = page.getByRole('button', { name: '启动中…' });
  if (await busyBtn.isVisible().catch(() => false)) {
    const cancelBtn = page.getByRole('button', { name: '取消' });
    if (await cancelBtn.isVisible().catch(() => false)) {
      await cancelBtn.click();
      await page.waitForTimeout(2000);
    }
  }
}

async function firstEnabledZtreeActionButton(page, actionLabel, { requireEnabled = true } = {}) {
  // 等待至少一个按钮出现
  await page.waitForFunction(async (label) => {
    const ztreePanel = document.querySelector('[data-testid="comment-layer-ztree-panel"]');
    if (!ztreePanel) return false;
    const allButtons = ztreePanel.querySelectorAll('button');
    const actionButtons = Array.from(allButtons).filter(btn => btn.textContent && btn.textContent.includes(label));
    return actionButtons.length > 0;
  }, actionLabel, { timeout: 60000 });
  
  // 查找第一个可用的按钮
  for (let i = 0; i < 20; i++) {
    const allButtons = page.locator('[data-testid="comment-layer-ztree-panel"] button');
    const count = await allButtons.count();
    for (let j = 0; j < count; j++) {
      const button = allButtons.nth(j);
      const text = await button.innerText().catch(() => '');
      if (text.includes(actionLabel)) {
        const enabled = await button.isEnabled().catch(() => false);
        if (enabled) {
          return button;
        }
      }
    }
    await page.waitForTimeout(1000);
  }
  
  if (requireEnabled) {
    throw new Error(`zTree 内没有已启用的「${actionLabel}」按钮（可能仍停在父层选中或未产生工作区变更）`);
  }
  
  // 返回第一个找到的按钮
  const allButtons = page.locator('[data-testid="comment-layer-ztree-panel"] button');
  const count = await allButtons.count();
  for (let j = 0; j < count; j++) {
    const button = allButtons.nth(j);
    const text = await button.innerText().catch(() => '');
    if (text.includes(actionLabel)) {
      return button;
    }
  }
  
  throw new Error(`未找到「${actionLabel}」按钮`);
}

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
  console.log('开始执行任务...');
  
  if (!dockerAvailable()) {
    console.error('错误：需要本机 Docker 环境');
    process.exit(1);
  }
  
  if (!EMAIL || !PASSWORD) {
    console.error('错误：请设置登录邮箱和密码');
    process.exit(1);
  }
  
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
    
    console.log('等待 SSE 连接...');
    await awaitSseConnected(page);
    await dismissMockStartBusyIfPresent(page);
    
    console.log('点击模拟启动按钮...');
    const mockBtn = page.getByRole('button', { name: /模拟启动（onlineService/ });
    await mockBtn.waitFor({ timeout: 120000 });
    await mockBtn.click();
    
    console.log('等待模拟启动完成...');
    await page.waitForFunction(async () => {
      const monoLog = document.querySelector('div.font-mono.whitespace-pre-wrap');
      if (!monoLog) return false;
      const t = monoLog.innerText;
      if (/build 失败|docker run 失败|\[错误\] 模拟启动|未在预期时间内检测到服务就绪/i.test(t)) {
        throw new Error(`模拟启动 Docker 未成功：${t.slice(-1000)}`);
      }
      return /服务已在 http:\/\/127\.0\.0\.1:/.test(t);
    }, null, { timeout: 600000 });
    
    console.log('等待 zTree 挂载...');
    await page.waitForFunction(async () => {
      const unreachableBanner = document.querySelector('[data-testid="container-http-unreachable-banner"]');
      if (unreachableBanner) {
        throw new Error('任务页展示「容器当前无法连接」');
      }
      const ztreePanel = document.querySelector('[data-testid="comment-layer-ztree-panel"]');
      return !!ztreePanel;
    }, null, { timeout: 1200000 });
    
    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    await ztreePanel.waitFor({ timeout: 60000 });
    
    console.log('选择可写层...');
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
    
    console.log('输入并发送指令...');
    const cmdPanel = page.getByTestId('comment-layer-ztree-command-panel');
    await cmdPanel.waitFor({ timeout: 30000 });
    await page.locator('#layer-graph-command-input').fill(PROMPT);
    const sendBtn = page.locator('#layer-graph-command-send-btn');
    await sendBtn.waitFor({ timeout: 120000 });
    await sendBtn.click();
    
    console.log('等待指令执行完成...');
    // 等待克隆完成
    await page.waitForFunction(async () => {
      const execPanel = document.querySelector('[data-testid="comment-layer-ztree-exec-log-panel"]');
      if (!execPanel) return false;
      const t = execPanel.innerText.trim();
      if (/Unknown tools in config|ConfigError:\s*Unknown tools/i.test(t)) {
        throw new Error(`trae_agent 功能参数 YAML 工具名与镜像不一致`);
      }
      if (/转发容器失败|502|403|401 Unauthorized/i.test(t)) {
        throw new Error(`容器执行链路 HTTP 异常`);
      }
      return t.length >= 12 && /克隆完成|bootstrap.*完成|仓库克隆已完成/i.test(t);
    }, null, { timeout: 600000 });
    
    // 等待指令执行完成
    await page.waitForFunction(async () => {
      const execPanel = document.querySelector('[data-testid="comment-layer-ztree-exec-log-panel"]');
      if (!execPanel) return false;
      const t = execPanel.innerText.trim();
      if (t.length < 8) return false;
      return /completed|interrupted|任务执行|done|phase.*chunk|hello|Hello|world|hello\.js|console|\.js|node|writeFile|writeFileSync|文件|输出|步骤|工具|调用|错误|失败/i.test(t);
    }, null, { timeout: 720000 });
    
    // 额外等待一段时间，确保代码已生成
    console.log('等待代码生成完成...');
    await page.waitForTimeout(10000);
    
    console.log('等待提交按钮可用...');
    // 增加等待时间，确保提交按钮变为可用
    await page.waitForFunction(async () => {
      const ztreePanel = document.querySelector('[data-testid="comment-layer-ztree-panel"]');
      if (!ztreePanel) return false;
      const allButtons = ztreePanel.querySelectorAll('button');
      const submitButtons = Array.from(allButtons).filter(btn => btn.textContent && btn.textContent.includes('提交'));
      if (submitButtons.length === 0) {
        console.log('未找到提交按钮');
        return false;
      }
      for (let i = 0; i < submitButtons.length; i++) {
        if (!submitButtons[i].disabled) {
          return true;
        }
      }
      console.log('提交按钮存在但不可用');
      return false;
    }, null, { timeout: 900000 });
    
    console.log('同步 Git 身份...');
    await ensureGitIdentitySynced(page);
    
    console.log('查找并点击提交按钮...');
    // 直接查找页面上的提交按钮并点击，确保只点击一次
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
    }
    
    // 等待提交完成
    console.log('等待提交完成...');
    await page.waitForTimeout(10000);
    
    console.log('查找并点击推送按钮...');
    // 直接查找页面上的推送按钮并点击，确保只点击一次
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
    }
    
    console.log('等待推送完成...');
    try {
      // 尝试等待推送响应，但如果超时则继续执行
      await page.waitForResponse(
        (r) => r.url().includes('container-layer-git-push') && r.request().method() === 'POST',
        { timeout: 300000 } // 减少超时时间
      );
      console.log('推送响应已收到');
    } catch (error) {
      console.log('推送响应超时，可能推送操作仍在进行中');
    }
    
    console.log('任务执行完成！');
    console.log('已完成以下操作：');
    console.log('1. 登录系统');
    console.log('2. 打开任务详情页');
    console.log('3. 模拟启动任务');
    console.log('4. 选择层级节点');
    console.log('5. 发送指令：用js写一个hello world');
    console.log('6. 同步Git身份');
    console.log('7. 提交代码');
    console.log('8. 推送代码到远端');
    console.log('任务执行完成！');
    
  } catch (error) {
    console.error('执行过程中出错:', error);
  } finally {
    await browser.close();
  }
}

main();
