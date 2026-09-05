// @ts-check
/**
 * E2E: CreateProject 自动克隆子仓库勾选（OPT-20260812-006）
 *
 * 防止模板条件 `hasAnyGitRepoUrl` 回归导致开关消失：
 * 1. 填仓库 URL 后断言 `create-project-auto-clone-nested-repos` 可见且默认勾选
 * 2. 取消勾选后提交，mock POST 校验 body.auto_clone_nested_repos === false
 * 3. 保持勾选提交，mock POST 校验 body.auto_clone_nested_repos === true
 *
 * 采用与 TaskDetail.runtime-hydrate-lifecycle 一致的 mock 方案（假 Cookie +
 * page.route 拦截全部 /api/），不依赖真实登录。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE_URL = process.env.BASE_URL || `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`;
const TENANT_ID = process.env.TEST_TENANT_ID || '850256677331562496';
const REPO_URL = process.env.PLAYWRIGHT_PUBLIC_GIT_URL || 'https://github.com/octocat/Hello-World.git';
const CREATE_PROJECT_PATH = `/tenant/${TENANT_ID}/create-project/`;

const VALIDATE_REQ_BODY = { results: [{ url: REPO_URL, is_accessible: true, token_status: 'token_available' }] };

/**
 * 安装 mock 路由：覆盖创建项目页依赖的全部 /api/ 请求。
 * @param {import('@playwright/test').Page} page
 * @param {{ capturedBodies: Array<Record<string, unknown>> }} state
 */
async function installApiMocks(page, state) {
  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    // 租户守卫依赖 /me/：返回该 tenant 在公司列表内 → 放行
    if (url.includes('/api/accounts/users/me/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'mock-user',
          username: 'mock-user',
          companies: [{ id: TENANT_ID, name: 'Mock 租户' }],
        }),
      });
      return;
    }

    // 工作空间下拉
    if (url.includes('/api/projects/workspaces/tenant_id/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'ws-mock-1', name: 'Mock 工作空间' }]),
      });
      return;
    }

    // 已安装镜像
    if (url.includes('/api/cloud/installed-images/tenant_id/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    // Git 仓库 URL 批量校验：返回可访问，token 已就绪 → 无需 OAuth 授权按钮
    if (url.includes('/api/projects/validate-git-repos/') && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(VALIDATE_REQ_BODY),
      });
      return;
    }

    // 创建项目 POST：捕获 body 供断言，返回 mock id
    if (url.includes('/api/projects/tenant_id/') && method === 'POST') {
      const bodyText = req.postData() || '';
      let parsed = {};
      try {
        parsed = JSON.parse(bodyText);
      } catch (_) {
        /* 非 JSON 忽略 */
      }
      state.capturedBodies.push(parsed);
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: 'mock-project-auto-clone' }),
      });
      return;
    }

    // 其余 API 一律返回空对象，避免未 mock 请求挂起
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

async function gotoCreateProject(page) {
  await page.goto(`${BASE_URL}${CREATE_PROJECT_PATH}`, { waitUntil: 'domcontentloaded', timeout: 20000 });
  await page.waitForSelector('#projectName', { timeout: 15000 });
  await page.waitForFunction(() => {
    const select = document.querySelector('#workspace');
    return select && select.options.length >= 1;
  }, { timeout: 15000 });
}

async function fillRequiredFields(page) {
  await page.fill('#projectName', `E2E-AutoClone-${Date.now()}`);
  await page.fill('#projectDescription', `自动克隆子仓库 E2E ${new Date().toISOString()}`);
  await page.selectOption('#workspace', { label: 'Mock 工作空间' });
}

async function fillRepoUrl(page) {
  const urlInput = page.locator('#gitRepo0');
  await urlInput.fill(REPO_URL);
  await urlInput.blur();
  // 等待批量校验完成，校验通过后创建按钮才可用
  await expect(page.getByTestId('create-project-auto-clone-nested-repos')).toBeVisible({ timeout: 15000 });
  await page.waitForTimeout(800);
}

test.describe('CreateProject 自动克隆子仓库', () => {
  test('填仓库 URL 后勾选框可见且默认勾选', async ({ page, context }) => {
    await context.addCookies([
      { name: 'userId', value: 'e2e-user', url: `${BASE_URL}/` },
      { name: 'csrftoken', value: 'e2e-csrf', url: `${BASE_URL}/` },
    ]);
    const state = { capturedBodies: [] };
    await installApiMocks(page, state);

    await gotoCreateProject(page);
    await fillRequiredFields(page);

    // 未填 URL 时不显示勾选框
    await expect(page.getByTestId('create-project-auto-clone-nested-repos')).toHaveCount(0);

    await fillRepoUrl(page);
    const checkbox = page.getByTestId('create-project-auto-clone-nested-repos');
    await expect(checkbox).toBeVisible();
    await expect(checkbox).toBeChecked();
    await expect(page.getByTestId('create-project-auto-clone-nested-repos-label')).toContainText('自动克隆子仓库');
  });

  test('取消勾选后提交 POST body 为 auto_clone_nested_repos=false', async ({ page, context }) => {
    await context.addCookies([
      { name: 'userId', value: 'e2e-user', url: `${BASE_URL}/` },
      { name: 'csrftoken', value: 'e2e-csrf', url: `${BASE_URL}/` },
    ]);
    const state = { capturedBodies: [] };
    await installApiMocks(page, state);

    await gotoCreateProject(page);
    await fillRequiredFields(page);
    await fillRepoUrl(page);

    const checkbox = page.getByTestId('create-project-auto-clone-nested-repos');
    await expect(checkbox).toBeChecked();
    await checkbox.uncheck();
    await expect(checkbox).not.toBeChecked();

    const createBtn = page.locator('button:has-text("创建项目")');
    await expect(createBtn).toBeEnabled({ timeout: 15000 });
    await createBtn.click();

    // 等待 POST 被拦截并捕获 body
    await expect.poll(() => state.capturedBodies.length, { timeout: 10000 }).toBeGreaterThan(0);
    expect(state.capturedBodies.at(-1).auto_clone_nested_repos).toBe(false);
    expect(state.capturedBodies.at(-1).workspaces_ids).toEqual(['ws-mock-1']);
  });

  test('保持勾选提交 POST body 为 auto_clone_nested_repos=true', async ({ page, context }) => {
    await context.addCookies([
      { name: 'userId', value: 'e2e-user', url: `${BASE_URL}/` },
      { name: 'csrftoken', value: 'e2e-csrf', url: `${BASE_URL}/` },
    ]);
    const state = { capturedBodies: [] };
    await installApiMocks(page, state);

    await gotoCreateProject(page);
    await fillRequiredFields(page);
    await fillRepoUrl(page);

    const checkbox = page.getByTestId('create-project-auto-clone-nested-repos');
    await expect(checkbox).toBeChecked();

    const createBtn = page.locator('button:has-text("创建项目")');
    await expect(createBtn).toBeEnabled({ timeout: 15000 });
    await createBtn.click();

    await expect.poll(() => state.capturedBodies.length, { timeout: 10000 }).toBeGreaterThan(0);
    expect(state.capturedBodies.at(-1).auto_clone_nested_repos).toBe(true);
  });
});
