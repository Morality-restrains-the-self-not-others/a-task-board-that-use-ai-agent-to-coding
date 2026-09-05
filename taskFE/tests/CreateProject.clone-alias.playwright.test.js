// @ts-check
/**
 * E2E: CreateProject Git 仓库别名（clone_alias）
 *
 * 1. 创建页可见别名输入
 * 2. 提交带别名后，详情页展示别名；编辑页带回别名值
 *
 * 账号/租户沿用 CreateProject 表单测例约定。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();

const BASE_URL = process.env.BASE_URL || `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`;
const TENANT_ID = process.env.TEST_TENANT_ID || '850256677331562496';
const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

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
  await page.waitForTimeout(1500);
}

async function navigateToCreateProject(page) {
  await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/create-project/`, {
    waitUntil: 'domcontentloaded',
    timeout: 15000,
  });
  await page.waitForSelector('#workspace', { timeout: 10000 });
  await page.waitForFunction(() => {
    const select = document.querySelector('#workspace');
    return select && select.options.length >= 1;
  }, { timeout: 10000 });
}

test.describe('CreateProject Git 仓库别名', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await navigateToCreateProject(page);
  });

  test('创建页可见别名输入框', async ({ page }) => {
    const aliasInput = page.getByTestId('git-repo-clone-alias-input').first();
    await expect(aliasInput).toBeVisible();
    await expect(aliasInput).toHaveAttribute('placeholder', /别名/);
  });

  test('提交带别名后详情与编辑页带回', async ({ page }) => {
    const projectName = `E2E-Alias-${Date.now()}`;
    const alias = `alias-${Date.now().toString(36)}`;
    // 使用可公开访问的示例 URL，避免阻塞在校验失败（无 OAuth 时也可能标记不可访问）
    const repoUrl = process.env.PLAYWRIGHT_PUBLIC_GIT_URL || 'https://github.com/octocat/Hello-World.git';

    await page.fill('#projectName', projectName);
    await page.fill('#projectDescription', `别名 E2E ${new Date().toISOString()}`);
    await page.selectOption('#workspace', { index: 1 });

    const urlInput = page.locator('#gitRepo0');
    await urlInput.fill(repoUrl);
    await urlInput.blur();
    await page.getByTestId('git-repo-clone-alias-input').first().fill(alias);

    // 等待 URL 失焦校验结束（成功或需授权均可提交；若格式错则失败）
    await page.waitForTimeout(800);

    const createBtn = page.locator('button:has-text("创建项目")');
    await expect(createBtn).toBeEnabled({ timeout: 15000 });

    const createRespPromise = page.waitForResponse(
      (r) =>
        r.request().method() === 'POST' &&
        /\/api\/tenant\/[^/]+\/projects\/?$/.test(new URL(r.url()).pathname) &&
        r.status() >= 200 &&
        r.status() < 300,
      { timeout: 20000 },
    );
    await createBtn.click();
    const createResp = await createRespPromise;
    const created = await createResp.json();
    const projectId = String(created.id || '').trim();
    expect(projectId).toBeTruthy();

    const entries = Array.isArray(created.git_repo_entries) ? created.git_repo_entries : [];
    expect(entries.some((e) => String(e?.clone_alias || '') === alias)).toBeTruthy();

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/projects/${projectId}/`, {
      waitUntil: 'domcontentloaded',
      timeout: 15000,
    });
    await page.waitForSelector('[data-testid="project-detail-git-repos-section"]', { timeout: 15000 });
    const aliasDisplay = page.getByTestId('git-repo-clone-alias-display').first();
    await expect(aliasDisplay).toBeVisible({ timeout: 10000 });
    await expect(aliasDisplay).toContainText(alias);

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/projects/${projectId}/edit/`, {
      waitUntil: 'domcontentloaded',
      timeout: 15000,
    });
    await page.waitForSelector('[data-testid="git-repo-clone-alias-input"]', { timeout: 15000 });
    await expect(page.getByTestId('git-repo-clone-alias-input').first()).toHaveValue(alias);
  });
});
