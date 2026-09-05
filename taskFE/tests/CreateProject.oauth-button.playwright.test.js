// @ts-check
/**
 * E2E: CreateProject 页面 — 仓库不可访问时 OAuth 授权按钮
 *
 * 测试范围：
 * 1. 输入 GitHub 私有仓库 URL → 校验返回 is_accessible=false
 * 2. 显示错误消息 "该仓库无法访问，后续需要授权后才能访问"
 * 3. 同时显示 "OAuth 授权" 按钮（仅当 provider 可识别时）
 * 4. 点击 OAuth 按钮 → 调用 /api/git-oauth/github-start-from-gateway/ 并跳转
 * 5. 不可识别的仓库（非 github/gitlab）不显示 OAuth 按钮
 * 6. 仓库可访问时不显示 OAuth 按钮
 *
 * 账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 * 租户：850256677331562496
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const vueHost = clientReachableHost(portConfig.vue?.host, '127.0.0.1');

const BASE_URL = process.env.BASE_URL || `http://${vueHost}:${portConfig.vue.port}`;
const TENANT_ID = process.env.TEST_TENANT_ID || '850256677331562496';
const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

/** 登录到租户 */
async function login(page) {
  await page.goto(`${BASE_URL}/auth/login/`, { waitUntil: 'domcontentloaded', timeout: 15000 });
  await page.waitForSelector('input[type="email"]', { timeout: 10000 });

  const checkboxes = await page.$$('input[type="checkbox"]');
  for (const cb of checkboxes) {
    if (!(await cb.isChecked())) await cb.check();
  }

  await page.fill('input[type="email"]', CREDENTIALS.email);
  await page.fill('input[type="password"]', CREDENTIALS.password);
  await page.locator('button[type="submit"]').click();

  await page.waitForURL('**/projects/**', { timeout: 15000 });
  await page.waitForTimeout(2000);
}

/** 导航到创建项目页面并等待加载完成 */
async function navigateToCreateProject(page) {
  await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/create-project/`, {
    waitUntil: 'domcontentloaded', timeout: 15000,
  });
  await page.waitForSelector('#workspace', { timeout: 10000 });
  await page.waitForFunction(() => {
    const select = document.querySelector('#workspace');
    return select && select.options.length >= 1;
  }, { timeout: 10000 });
}

/** Mock validate-git-repo / validate-git-repos API 返回 is_accessible=false */
async function mockRepoInaccessible(page) {
  await page.route('**/validate-git-repos/**', async (route) => {
    const postData = route.request().postDataJSON() || {};
    const urls = Array.isArray(postData.urls) ? postData.urls : [];
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        results: urls.map((url) => ({
          url,
          repo_url: url,
          is_accessible: false,
          token_status: 'not_bound',
          message: '',
        })),
      }),
    });
  });
  await page.route('**/validate-git-repo/**', async (route) => {
    if (route.request().url().includes('validate-git-repos')) {
      await route.fallback();
      return;
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ is_accessible: false, message: '', token_status: 'not_bound' }),
    });
  });
}

/** Mock validate-git-repo / validate-git-repos API 返回 is_accessible=true */
async function mockRepoAccessible(page) {
  await page.route('**/validate-git-repos/**', async (route) => {
    const postData = route.request().postDataJSON() || {};
    const urls = Array.isArray(postData.urls) ? postData.urls : [];
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        results: urls.map((url) => ({
          url,
          repo_url: url,
          is_accessible: true,
          token_status: 'token_available',
          message: '',
        })),
      }),
    });
  });
  await page.route('**/validate-git-repo/**', async (route) => {
    if (route.request().url().includes('validate-git-repos')) {
      await route.fallback();
      return;
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ is_accessible: true, message: '', token_status: 'token_available' }),
    });
  });
}

/** 填写必填字段使表单有效 */
async function fillRequiredFields(page) {
  await page.fill('#projectName', 'OAuth-Test-Project');
  await page.fill('#projectDescription', 'Testing OAuth button visibility');
  await page.selectOption('#workspace', { index: 1 });
  await page.waitForTimeout(300);
}

test.describe('CreateProject OAuth 授权按钮', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('GitHub 私有仓库不可访问时显示错误消息和 OAuth 授权按钮', async ({ page }) => {
    await mockRepoInaccessible(page);
    await navigateToCreateProject(page);
    await fillRequiredFields(page);

    // 输入 GitHub 私有仓库 URL
    await page.fill('#gitRepo0', 'https://github.com/private-org/private-repo');
    await page.locator('#gitRepo0').blur();
    // 等待 debounce (500ms) + API 调用完成
    await page.waitForTimeout(1200);

    // 验证错误消息可见
    const errorMsg = page.locator('text=该仓库无法访问，后续需要授权后才能访问');
    await expect(errorMsg).toBeVisible();

    // 验证引导文案 + OAuth 授权按钮可见
    await expect(page.getByTestId('repo-oauth-authorize-prompt')).toHaveText('是否现在去授权');
    const oauthBtn = page.getByTestId('repo-oauth-authorize-button');
    await expect(oauthBtn).toBeVisible();
    await expect(oauthBtn).toHaveText('OAuth 授权');
    await expect(oauthBtn).toBeEnabled();
  });

  test('GitLab 私有仓库不可访问时显示 OAuth 授权按钮', async ({ page }) => {
    await mockRepoInaccessible(page);
    await navigateToCreateProject(page);
    await fillRequiredFields(page);

    // 输入 GitLab 私有仓库 URL
    await page.fill('#gitRepo0', 'https://gitlab.com/private-group/private-repo');
    await page.locator('#gitRepo0').blur();
    await page.waitForTimeout(1200);

    // 验证错误消息可见
    const errorMsg = page.locator('text=该仓库无法访问，后续需要授权后才能访问');
    await expect(errorMsg).toBeVisible();

    // 验证引导文案 + OAuth 授权按钮可见（GitLab 通过 heuristic 识别）
    await expect(page.getByTestId('repo-oauth-authorize-prompt')).toHaveText('是否现在去授权');
    const oauthBtn = page.getByTestId('repo-oauth-authorize-button');
    await expect(oauthBtn).toBeVisible();
    await expect(oauthBtn).toHaveText('OAuth 授权');
    await expect(oauthBtn).toBeEnabled();
  });

  test('不可识别的 Git 站点不显示 OAuth 按钮', async ({ page }) => {
    await mockRepoInaccessible(page);
    await navigateToCreateProject(page);
    await fillRequiredFields(page);

    // 输入非 github/gitlab 的仓库 URL（如自建 Gitea）
    await page.fill('#gitRepo0', 'https://git.example.com/user/repo');
    await page.locator('#gitRepo0').blur();
    await page.waitForTimeout(1200);

    // 验证错误消息可见
    const errorMsg = page.locator('text=该仓库无法访问，后续需要授权后才能访问');
    await expect(errorMsg).toBeVisible();

    // 验证 OAuth 授权按钮与引导文案不可见（未知 Git 站点）
    await expect(page.getByTestId('repo-oauth-authorize-prompt')).toHaveCount(0);
    await expect(page.getByTestId('repo-oauth-authorize-button')).toHaveCount(0);
  });

  test('仓库可访问时不显示错误消息和 OAuth 按钮', async ({ page }) => {
    await mockRepoAccessible(page);
    await navigateToCreateProject(page);
    await fillRequiredFields(page);

    // 输入可访问的仓库 URL
    await page.fill('#gitRepo0', 'https://github.com/public-org/public-repo');
    await page.locator('#gitRepo0').blur();
    await page.waitForTimeout(1200);

    // 验证错误消息不可见
    const errorMsg = page.locator('text=该仓库无法访问，后续需要授权后才能访问');
    await expect(errorMsg).not.toBeVisible();

    // 验证 OAuth 授权按钮与引导文案不可见
    await expect(page.getByTestId('repo-oauth-authorize-prompt')).toHaveCount(0);
    await expect(page.getByTestId('repo-oauth-authorize-button')).toHaveCount(0);
  });

  test('点击 OAuth 授权按钮触发 API 调用并跳转', async ({ page }) => {
    await mockRepoInaccessible(page);

    // Mock OAuth start API
    let startApiCalled = false;
    await page.route('**/api/git-oauth/github-start-from-gateway/**', async (route) => {
      startApiCalled = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ authorize_url: 'https://github.com/login/oauth/authorize?client_id=test' }),
      });
    });

    await navigateToCreateProject(page);
    await fillRequiredFields(page);

    await page.fill('#gitRepo0', 'https://github.com/private-org/private-repo');
    await page.locator('#gitRepo0').blur();
    await page.waitForTimeout(1200);

    const oauthBtn = page.locator('button:has-text("OAuth 授权")');
    await expect(oauthBtn).toBeVisible();

    // 点击 OAuth 按钮，可能触发导航；使用 Promise.race 避免阻塞
    try {
      await Promise.race([
        oauthBtn.click(),
        page.waitForTimeout(3000),
      ]);
    } catch {
      // 导航可能因 mock URL 失败而抛出，忽略
    }

    // 验证 start API 被调用了
    expect(startApiCalled).toBe(true);
  });

  test('OAuth 按钮在加载中时显示"跳转中..."并禁用', async ({ page }) => {
    await mockRepoInaccessible(page);

    // Mock OAuth start API 延迟响应
    await page.route('**/api/git-oauth/github-start-from-gateway/**', async (route) => {
      await new Promise(resolve => setTimeout(resolve, 2000));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ authorize_url: 'https://github.com/login/oauth/authorize?client_id=test' }),
      });
    });

    await navigateToCreateProject(page);
    await fillRequiredFields(page);

    await page.fill('#gitRepo0', 'https://github.com/private-org/private-repo');
    await page.locator('#gitRepo0').blur();
    await page.waitForTimeout(1200);

    const oauthBtn = page.locator('button:has-text("OAuth 授权")');
    await expect(oauthBtn).toBeVisible();

    // 点击后应立即变为"跳转中..."并禁用
    await oauthBtn.click();
    await page.waitForTimeout(200);

    const loadingBtn = page.locator('button:has-text("跳转中...")');
    await expect(loadingBtn).toBeVisible();
    await expect(loadingBtn).toBeDisabled();
  });

  test('添加/移除仓库行时 OAuth 按钮正确更新', async ({ page }) => {
    await mockRepoInaccessible(page);
    await navigateToCreateProject(page);
    await fillRequiredFields(page);

    // 第一行输入 GitHub URL
    await page.fill('#gitRepo0', 'https://github.com/private-org/repo-one');
    await page.locator('#gitRepo0').blur();
    await page.waitForTimeout(1200);

    // 验证第一行有 OAuth 按钮
    await expect(page.locator('button:has-text("OAuth 授权")')).toHaveCount(1);

    // 添加第二行仓库
    await page.locator('button:has-text("添加仓库")').click();
    await page.waitForTimeout(300);

    // 第二行输入另一个 GitLab URL（type=text，非 url）
    const gitInputs = page.locator('input[placeholder*="Git 仓库 URL"]');
    await expect(gitInputs).toHaveCount(2);
    await gitInputs.nth(1).fill('https://gitlab.com/private-group/repo-two');
    await gitInputs.nth(1).blur();
    await page.waitForTimeout(1200);

    // 两行都应该有 OAuth 按钮
    await expect(page.locator('button:has-text("OAuth 授权")')).toHaveCount(2);

    // 移除第二行
    const removeButtons = await page.$$('button:has-text("移除")');
    await removeButtons[0].click();
    await page.waitForTimeout(300);

    // 只剩一个 OAuth 按钮
    await expect(page.locator('button:has-text("OAuth 授权")')).toHaveCount(1);
  });

  test('OAuth API 超时后显示超时错误并恢复按钮', async ({ page }) => {
    await mockRepoInaccessible(page);

    // Mock OAuth start API 延迟超过 10s（超时阈值）
    await page.route('**/api/git-oauth/github-start-from-gateway/**', async (route) => {
      await new Promise(resolve => setTimeout(resolve, 12000));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ authorize_url: 'https://github.com/login/oauth/authorize' }),
      });
    });

    await navigateToCreateProject(page);
    await fillRequiredFields(page);

    await page.fill('#gitRepo0', 'https://github.com/private-org/private-repo');
    await page.locator('#gitRepo0').blur();
    await page.waitForTimeout(1200);

    const oauthBtn = page.locator('button:has-text("OAuth 授权")');
    await expect(oauthBtn).toBeVisible();

    // 点击 OAuth 按钮
    await oauthBtn.click();

    // 等待超时 (10s) + 一点余量
    await page.waitForTimeout(11000);

    // 按钮应恢复为 "OAuth 授权" 且可用
    await expect(page.locator('button:has-text("OAuth 授权")')).toBeVisible();
    await expect(page.locator('button:has-text("OAuth 授权")')).toBeEnabled();

    // 应显示超时错误消息
    const timeoutError = page.locator('text=OAuth 授权服务响应超时，请检查网络后重试');
    await expect(timeoutError).toBeVisible();
  });

  test('OAuth API 超时后可重试', async ({ page }) => {
    await mockRepoInaccessible(page);

    let callCount = 0;
    // 第一次请求超时，第二次成功
    await page.route('**/api/git-oauth/github-start-from-gateway/**', async (route) => {
      callCount++;
      if (callCount === 1) {
        // 第一次：延迟超过 10s 导致超时
        await new Promise(resolve => setTimeout(resolve, 12000));
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ authorize_url: 'https://github.com/login/oauth/authorize' }),
        });
      } else {
        // 第二次：正常响应
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ authorize_url: 'https://github.com/login/oauth/authorize?client_id=retry' }),
        });
      }
    });

    await navigateToCreateProject(page);
    await fillRequiredFields(page);

    await page.fill('#gitRepo0', 'https://github.com/private-org/private-repo');
    await page.locator('#gitRepo0').blur();
    await page.waitForTimeout(1200);

    // 第一次点击 → 超时
    await page.locator('button:has-text("OAuth 授权")').click();
    await page.waitForTimeout(11000);

    // 验证超时错误显示
    const timeoutError = page.locator('text=OAuth 授权服务响应超时，请检查网络后重试');
    await expect(timeoutError).toBeVisible();

    // 按钮恢复可用后再次点击 → 第二次调用
    const retryBtn = page.locator('button:has-text("OAuth 授权")');
    await expect(retryBtn).toBeEnabled();
    await retryBtn.click();
    await page.waitForTimeout(500);

    // 验证 API 被调用了两次
    expect(callCount).toBe(2);
  });

  test('OAuth start API 返回错误时显示后端错误消息', async ({ page }) => {
    await mockRepoInaccessible(page);

    // Mock OAuth start API 返回 503
    await page.route('**/api/git-oauth/github-start-from-gateway/**', async (route) => {
      await route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: JSON.stringify({ detail: 'Git OAuth 服务暂时不可用' }),
      });
    });

    await navigateToCreateProject(page);
    await fillRequiredFields(page);

    await page.fill('#gitRepo0', 'https://github.com/private-org/private-repo');
    await page.locator('#gitRepo0').blur();
    await page.waitForTimeout(1200);

    await page.locator('button:has-text("OAuth 授权")').click();
    await page.waitForTimeout(500);

    // 应显示后端返回的错误消息
    const serverError = page.locator('text=Git OAuth 服务暂时不可用');
    await expect(serverError).toBeVisible();

    // 按钮应恢复为可点击状态（允许重试）
    await expect(page.locator('button:has-text("OAuth 授权")')).toBeEnabled();
  });

  test('OAuth API 立即失败（网络错误）时显示通用错误并恢复', async ({ page }) => {
    await mockRepoInaccessible(page);

    // Mock OAuth start API 立即失败
    await page.route('**/api/git-oauth/github-start-from-gateway/**', async (route) => {
      await route.abort('failed');
    });

    await navigateToCreateProject(page);
    await fillRequiredFields(page);

    await page.fill('#gitRepo0', 'https://github.com/private-org/private-repo');
    await page.locator('#gitRepo0').blur();
    await page.waitForTimeout(1200);

    await page.locator('button:has-text("OAuth 授权")').click();
    await page.waitForTimeout(500);

    // 应显示错误消息（网络错误 → "Failed to fetch"）
    const errorMsg = page.locator('text=Failed to fetch');
    await expect(errorMsg).toBeVisible();

    // 按钮应恢复为可点击状态
    await expect(page.locator('button:has-text("OAuth 授权")')).toBeEnabled();
  });
});
