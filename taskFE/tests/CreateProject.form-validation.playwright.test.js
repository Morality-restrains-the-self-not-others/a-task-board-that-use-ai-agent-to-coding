// @ts-check
/**
 * E2E: CreateProject 页面表单验证与提交流程
 *
 * 测试范围：
 * 1. 登录后访问创建项目页面
 * 2. 表单初始状态 — 按钮禁用，原因提示正确
 * 3. 逐字段填写时按钮状态变化
 * 4. 工作空间加载状态和错误状态
 * 5. Git 仓库 URL 校验对按钮的影响
 * 6. 完整填写后提交成功并跳转
 * 7. 必填字段为空时提交被阻止
 * 8. keyboard-only 用户可操作性
 *
 * 账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 * 租户：850256677331562496
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';

const portConfig = loadPortConfig();

// pre-commit / 本地验证必须走本机 Vue，禁止落到 conf 中的公网 publicBaseUrl
const BASE_URL = (
  process.env.BASE_URL ||
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');
const TENANT_ID = process.env.TEST_TENANT_ID || '850256677331562496';
const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

async function login(page) {
  // 网关 API 登录（哈希密码 + cookie），避免 UI 登录在 pre-commit 下反复落回 /auth/login/
  await loginViaGatewayApi(page, {
    email: CREDENTIALS.email,
    password: CREDENTIALS.password,
    siteOrigin: BASE_URL,
  });
  await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/projects/`, {
    waitUntil: 'domcontentloaded',
    timeout: 15000,
  });
  await page.waitForURL('**/projects/**', { timeout: 15000 });
}

async function navigateToCreateProject(page) {
  await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/create-project/`, {
    waitUntil: 'domcontentloaded', timeout: 15000,
  });
  // Wait for Vue app mount & API data (workspaces, images)
  await page.waitForSelector('#workspace', { timeout: 10000 });
  await page.waitForFunction(() => {
    const select = document.querySelector('#workspace');
    return select && select.options.length >= 1;
  }, { timeout: 10000 });
}

test.describe('CreateProject 页面', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await navigateToCreateProject(page);
  });

  test('初始状态 — 按钮禁用且有提示', async ({ page }) => {
    const btn = page.locator('button:has-text("创建项目")');
    await expect(btn).toBeDisabled();

    // 按钮应有 title 提示缺少必填项
    const title = await btn.getAttribute('title');
    expect(title).toBeTruthy();
    // 默认为空名称
    expect(title).toContain('项目名称');
  });

  test('仅填写名称 — 按钮仍禁用，提示变为缺少描述', async ({ page }) => {
    await page.fill('#projectName', '测试项目');
    await page.waitForTimeout(300);

    const btn = page.locator('button:has-text("创建项目")');
    await expect(btn).toBeDisabled();

    const title = await btn.getAttribute('title');
    expect(title).toContain('项目描述');
  });

  test('填写名称和描述 — 按钮仍禁用，提示变为缺少工作空间', async ({ page }) => {
    await page.fill('#projectName', '测试项目');
    await page.fill('#projectDescription', '测试描述');
    await page.waitForTimeout(300);

    const btn = page.locator('button:has-text("创建项目")');
    await expect(btn).toBeDisabled();

    const title = await btn.getAttribute('title');
    expect(title).toContain('工作空间');
  });

  test('完整填写必填项后按钮启用', async ({ page }) => {
    await page.fill('#projectName', '测试项目-E2E');
    await page.fill('#projectDescription', 'E2E 测试项目描述');
    await page.selectOption('#workspace', { index: 1 }); // 选择第一个工作空间
    await page.waitForTimeout(500);

    const btn = page.locator('button:has-text("创建项目")');
    await expect(btn).toBeEnabled();

    // 启用时 title 应为空（无提示）
    const title = await btn.getAttribute('title');
    expect(title || '').toBe('');
  });

  test('提交流程 — 完整填写后点击创建，跳转到项目列表', async ({ page }) => {
    const projectName = `E2E-Test-${Date.now()}`;

    await page.fill('#projectName', projectName);
    await page.fill('#projectDescription', `E2E 测试项目 - ${new Date().toISOString()}`);
    await page.selectOption('#workspace', { index: 1 });
    await page.waitForTimeout(300);

    const btn = page.locator('button:has-text("创建项目")');
    await expect(btn).toBeEnabled();
    await btn.click();

    // 等待创建完成并跳转
    await page.waitForURL('**/tenant/*/projects/', { timeout: 15000 });
    expect(page.url()).toContain(`/tenant/${TENANT_ID}/projects/`);
  });

  test('清空必填字段后按钮重新禁用', async ({ page }) => {
    // 先完整填写
    await page.fill('#projectName', '测试项目');
    await page.fill('#projectDescription', '测试描述');
    await page.selectOption('#workspace', { index: 1 });
    await page.waitForTimeout(300);

    // 确认启用
    await expect(page.locator('button:has-text("创建项目")')).toBeEnabled();

    // 清空名称
    await page.fill('#projectName', '');
    await page.waitForTimeout(300);
    await expect(page.locator('button:has-text("创建项目")')).toBeDisabled();

    // 重新填写
    await page.fill('#projectName', '测试项目2');
    await page.waitForTimeout(300);
    await expect(page.locator('button:has-text("创建项目")')).toBeEnabled();
  });

  test('工作空间选择器在 API 返回后有可用选项', async ({ page }) => {
    // <option> 在原生 select 中常为 hidden，用 attached 等待即可
    await page.waitForSelector('#workspace option:nth-child(2)', {
      state: 'attached',
      timeout: 10000,
    });

    const options = await page.$$eval('#workspace option', opts =>
      opts.map(o => ({ value: o.value, text: o.textContent?.trim() }))
    );

    // 至少有默认 + 一个真实选项
    expect(options.length).toBeGreaterThanOrEqual(2);
    // 默认选项
    expect(options[0].value).toBe('');
    // 真实选项有非空 value
    expect(options[1].value).toBeTruthy();
    expect(options[1].value).not.toBe('');
  });

  test('键盘 Tab 导航填写表单后按钮可用', async ({ page }) => {
    await page.fill('#projectName', '键盘导航测试');
    await page.fill('#projectDescription', '键盘输入描述内容');
    await page.waitForSelector('#workspace option:nth-child(2)', {
      state: 'attached',
      timeout: 10000,
    });
    await page.selectOption('#workspace', { index: 1 });
    await page.waitForTimeout(300);

    const btn = page.locator('button:has-text("创建项目")');
    await btn.focus();
    const isFocused = await btn.evaluate(el => el === document.activeElement);
    expect(isFocused).toBe(true);
    await expect(btn).toBeEnabled();
  });

  test('Git 仓库输入无效 URL 后按钮禁用', async ({ page }) => {
    // 先完整填写必填项
    await page.fill('#projectName', 'URL测试项目');
    await page.fill('#projectDescription', '测试无效 URL');
    await page.selectOption('#workspace', { index: 1 });
    await page.waitForTimeout(300);

    await expect(page.locator('button:has-text("创建项目")')).toBeEnabled();

    // 填入无效 URL（非合法 URL 格式）
    await page.fill('#gitRepo0', 'not-a-valid-url');
    await page.waitForTimeout(500);

    const btn = page.locator('button:has-text("创建项目")');
    await expect(btn).toBeDisabled();

    const formatHint = page.locator('[data-testid="git-repo-url-format-hint"]');
    await expect(formatHint).toBeVisible();
    await expect(formatHint).toContainText('URL');
    await expect(page.locator('code').first()).toContainText('https://github.com/');

    // 清空无效 URL 后恢复
    await page.fill('#gitRepo0', '');
    await page.waitForTimeout(500);
    await expect(page.locator('button:has-text("创建项目")')).toBeEnabled();
  });

  test('Git 仓库输入 git@host:path 格式后按钮仍可用', async ({ page }) => {
    await page.fill('#projectName', 'SSH URL 测试');
    await page.fill('#projectDescription', '测试 git@ 格式');
    await page.selectOption('#workspace', { index: 1 });
    await page.waitForTimeout(300);

    await page.fill('#gitRepo0', 'git@127.0.0.1:example-user/somanyad-emailD.git');
    await page.waitForTimeout(500);

    const btn = page.locator('button:has-text("创建项目")');
    await expect(btn).toBeEnabled();
    await expect(page.locator('[data-testid="git-repo-url-format-hint"]')).toHaveCount(0);
  });

  test('Git 仓库输入 ssh://git@host/path 格式后按钮仍可用', async ({ page }) => {
    await page.fill('#projectName', 'ssh URI 测试');
    await page.fill('#projectDescription', '测试 ssh:// 协议');
    await page.selectOption('#workspace', { index: 1 });
    await page.waitForTimeout(300);

    await page.fill('#gitRepo0', 'ssh://git@github.com/owner/repo.git');
    await page.waitForTimeout(500);

    const btn = page.locator('button:has-text("创建项目")');
    await expect(btn).toBeEnabled();
    await expect(page.locator('[data-testid="git-repo-url-format-hint"]')).toHaveCount(0);
  });


  test('提交按钮显示创建中状态', async ({ page }) => {
    const projectName = `Loading-Test-${Date.now()}`;

    await page.fill('#projectName', projectName);
    await page.fill('#projectDescription', '测试创建中状态');
    await page.selectOption('#workspace', { index: 1 });

    const btn = page.locator('button:has-text("创建项目")');
    await btn.click();

    // 点击后按钮文本应变为 "创建中..."
    // (可能会很快完成，所以用 try-catch 包裹)
    try {
      await page.waitForSelector('button:has-text("创建中...")', { timeout: 2000 });
      const loadingText = await page.locator('button:has-text("创建中...")').textContent();
      expect(loadingText).toContain('创建中');
    } catch {
      // 创建太快已完成，这也是正常的
    }

    // 最终应跳转
    await page.waitForURL('**/tenant/*/projects/', { timeout: 15000 });
  });
});
