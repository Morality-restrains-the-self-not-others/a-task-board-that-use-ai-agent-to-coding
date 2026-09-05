// @ts-check
/**
 * E2E: 项目列表页 — 从 GitLab 同步项目
 *
 * 账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 * 租户：默认取账号所属公司；可用 TEST_TENANT_ID 覆盖
 * （历史默认 850256677331562496 已与当前 E2E 账号公司不一致）
 *
 * 运行：
 *   ./Projects.gitlab-sync.playwright.test.sh
 *   SITE_BASE=http://127.0.0.1:4000 ./Projects.gitlab-sync.playwright.test.sh --headed
 *   E2E_LOGIN_VIA_UI=1 ./Projects.gitlab-sync.playwright.test.sh --headed
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { installApisixCorsWorkaround } from './helpers/remoteLoginE2e.js';
import {
  loginViaGatewayApi,
  gotoAuthenticatedPath,
} from './helpers/gatewayLoginE2e.js';

const portConfig = loadPortConfig();

const BASE_URL =
  (process.env.SITE_BASE || process.env.BASE_URL || '').replace(/\/$/, '') ||
  portConfig.vue?.publicBaseUrl ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`;

/** 与当前 E2E 账号公司一致；可用 TEST_TENANT_ID 覆盖 */
let TENANT_ID = process.env.TEST_TENANT_ID || '882297276515512320';
const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

function projectsUrl() {
  return `${BASE_URL}/tenant/${TENANT_ID}/projects/`;
}

const MOCK_REPOS = [
  {
    gitlab_project_id: 101,
    name: 'valuestream',
    path_with_namespace: 'example-user/valuestream',
    description: 'Playwright mock repo',
    http_url_to_repo: 'http://127.0.0.1:8012/example-user/valuestream.git',
    web_url: 'http://127.0.0.1:8012/example-user/valuestream',
    default_branch: 'main',
    imported_in_single_repo_project: false,
    imported_in_any_project: false,
  },
  {
    gitlab_project_id: 103,
    name: 'other-repo',
    path_with_namespace: 'example-user/other-repo',
    description: 'Second mock repo',
    http_url_to_repo: 'http://127.0.0.1:8012/example-user/other-repo.git',
    web_url: 'http://127.0.0.1:8012/example-user/other-repo',
    default_branch: 'main',
    imported_in_single_repo_project: false,
    imported_in_any_project: false,
  },
  {
    gitlab_project_id: 102,
    name: 'already-synced',
    path_with_namespace: 'example-user/already-synced',
    description: 'Already in single-repo project',
    http_url_to_repo: 'http://127.0.0.1:8012/example-user/already-synced.git',
    web_url: 'http://127.0.0.1:8012/example-user/already-synced',
    default_branch: 'main',
    imported_in_single_repo_project: true,
    imported_in_any_project: true,
  },
];

/** @param {import('@playwright/test').Page} page
 *  @param {{ installCors?: boolean }} [opts]
 */
async function loginToProjects(page, opts = {}) {
  const installCors = opts.installCors !== false;
  // CORS 剥离必须先于业务 mock 注册：Playwright 后注册的 route 优先；
  // 若 **/* continue 盖在 mock 之上，gitlab-resources / workspaces 会打到真 API。
  if (installCors) {
    await installApisixCorsWorkaround(page);
  }
  const useUiLogin = process.env.E2E_LOGIN_VIA_UI === '1';
  if (useUiLogin) {
    // UI 路径：Login.vue 提交前会对明文 password 做 generateDjangoPasswordHash
    await playwrightLoginWithLegalAccept(page, {
      email: CREDENTIALS.email,
      password: CREDENTIALS.password,
      baseURL: BASE_URL,
    });
    await page.goto(projectsUrl(), { waitUntil: 'domcontentloaded', timeout: 60000 });
  } else {
    // API 路径：明文 password + activate-session（见 gatewayLoginE2e.js）
    const loginJson = await loginViaGatewayApi(page, {
      email: CREDENTIALS.email,
      password: CREDENTIALS.password,
      siteOrigin: BASE_URL,
    });
    if (!process.env.TEST_TENANT_ID) {
      const companyId = String(loginJson?.user?.companies?.[0]?.id || '').trim();
      if (companyId) TENANT_ID = companyId;
    }
    await gotoAuthenticatedPath(page, projectsUrl());
  }
  await page.waitForLoadState('networkidle').catch(() => {});
}

/** @param {import('@playwright/test').Page} page */
async function openGitlabSyncModal(page) {
  const syncBtn = page.getByTestId('projects-gitlab-sync-btn').or(
    page.getByTestId('projects-gitlab-sync-btn-empty'),
  ).or(page.getByRole('button', { name: '从 GitLab 同步' }).first());
  await expect(syncBtn).toBeVisible({ timeout: 30000 });
  await syncBtn.click();
  const modal = page.getByTestId('gitlab-sync-projects-modal');
  await expect(modal).toBeVisible({ timeout: 15000 });
  return modal;
}

const DEFAULT_PURCHASED = [
  {
    region: 'local-gitlab',
    region_name: '本地 GitLab',
    gitlab_web_url: 'http://127.0.0.1:8012',
    provisioning_status: 'active',
  },
];

/** @type {typeof DEFAULT_PURCHASED} */
let purchasedResourcesFixture = DEFAULT_PURCHASED;

/** @param {import('@playwright/test').Page} page */
async function mockPurchasedGitlabResources(page) {
  // 勿用 .../gitlab-resources/**：尾部 /** 可能匹配不到无子路径的 URL
  await page.route(`**/api/tenant/${TENANT_ID}/billing/gitlab-resources**`, async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        tenant_id: TENANT_ID,
        resources: purchasedResourcesFixture,
      }),
    });
  });
}

/** @param {import('@playwright/test').Page} page */
async function mockGitlabRemoteRepos(page, payload) {
  await page.route(`**/api/projects/gitlab-remote-repos/tenant_id/${TENANT_ID}**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(payload),
    });
  });
}

/** @param {import('@playwright/test').Page} page */
async function mockWorkspaces(page) {
  await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}*`, async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    // 列表接口无额外 path；子资源 path 放行
    const path = new URL(route.request().url()).pathname;
    if (!path.match(new RegExp(`/workspaces/tenant_id/${TENANT_ID}/?$`))) {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: '900001', name: 'E2E 工作空间 A' },
        { id: '900002', name: 'E2E 工作空间 B' },
      ]),
    });
  });
}

test.describe('Projects GitLab 同步（Mock API）', () => {
  test.beforeEach(async ({ page }) => {
    purchasedResourcesFixture = DEFAULT_PURCHASED;
    // CORS workaround 先注册，再注册 mock（后注册优先）
    await installApisixCorsWorkaround(page);
    await mockWorkspaces(page);
    await mockPurchasedGitlabResources(page);
    await loginToProjects(page, { installCors: false });
  });

  test('点击「从 GitLab 同步」打开对话框并展示仓库列表', async ({ page }) => {
    await mockGitlabRemoteRepos(page, {
      provider_key: 'gitlab:local-gitlab',
      gitlab_website: 'http://127.0.0.1:8012',
      oauth_bound: true,
      gitlab_login: 'example-user',
      repos: MOCK_REPOS,
      error: null,
    });

    const modal = await openGitlabSyncModal(page);
    await expect(modal.getByText('从 GitLab 同步项目')).toBeVisible();
    await expect(modal.getByTestId('gitlab-sync-source-url')).toContainText('http://127.0.0.1:8012');
    await expect(modal.getByText('example-user/valuestream')).toBeVisible();
    await expect(modal.getByText('单仓已占用')).toBeVisible();
    await expect(modal.getByTestId('gitlab-sync-create-btn')).toBeEnabled();
    await expect(modal.getByTestId('gitlab-sync-combined-create-btn')).toBeEnabled();
  });

  test('无已购 GitLab 时引导开通且不展示平台默认来源', async ({ page }) => {
    purchasedResourcesFixture = [];
    const modal = await openGitlabSyncModal(page);
    await expect(modal.getByTestId('gitlab-sync-no-purchased-region')).toBeVisible();
    await expect(modal.getByText('gitlab.daydaymoney.com')).toHaveCount(0);
    await expect(modal.getByTestId('gitlab-sync-go-pricing')).toBeVisible();
  });

  test('多区域时展示下拉并切换后带 gitlab_host 请求', async ({ page }) => {
    purchasedResourcesFixture = [
      {
        region: 'tencent-sh-5',
        region_name: '腾讯五区',
        gitlab_web_url: 'https://gl5.example',
        provisioning_status: 'active',
      },
      {
        region: 'tencent-sh-1',
        region_name: '腾讯一区',
        gitlab_web_url: 'https://gl1.example',
        provisioning_status: 'active',
      },
    ];

    /** @type {string[]} */
    const remoteUrls = [];
    await page.route(`**/api/projects/gitlab-remote-repos/tenant_id/${TENANT_ID}**`, async (route) => {
      remoteUrls.push(route.request().url());
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          provider_key: 'gitlab:local-gitlab',
          gitlab_website: route.request().url().includes('gl1')
            ? 'https://gl1.example'
            : 'https://gl5.example',
          oauth_bound: true,
          gitlab_login: 'u1',
          repos: MOCK_REPOS,
          error: null,
        }),
      });
    });

    const modal = await openGitlabSyncModal(page);
    const select = modal.getByTestId('gitlab-sync-region-select');
    await expect(select).toBeVisible();
    expect(remoteUrls.some((u) => u.includes(encodeURIComponent('https://gl5.example')))).toBe(true);

    await select.selectOption('tencent-sh-1');
    await expect.poll(() =>
      remoteUrls.some((u) => u.includes(encodeURIComponent('https://gl1.example'))),
    ).toBe(true);
  });

  test('未绑定 OAuth 时显示授权引导', async ({ page }) => {
    await mockGitlabRemoteRepos(page, {
      provider_key: 'gitlab:local-gitlab',
      gitlab_website: 'http://127.0.0.1:8012',
      oauth_bound: false,
      gitlab_login: null,
      repos: [],
      error: '尚未绑定 GitLab OAuth，请先完成授权',
    });

    const modal = await openGitlabSyncModal(page);
    await expect(modal.getByText('尚未绑定 GitLab OAuth').first()).toBeVisible();
    await expect(modal.getByTestId('gitlab-sync-oauth-btn')).toBeVisible();
    await expect(modal.getByTestId('gitlab-sync-create-btn')).not.toBeVisible();
  });

  test('勾选仓库并批量创建项目（每个仓库各建一个）', async ({ page }) => {
    await mockGitlabRemoteRepos(page, {
      provider_key: 'gitlab:local-gitlab',
      gitlab_website: 'http://127.0.0.1:8012',
      oauth_bound: true,
      gitlab_login: 'example-user',
      repos: MOCK_REPOS,
      error: null,
    });

    let batchPayload = null;
    await page.route(`**/api/projects/batch-from-gitlab-repos/tenant_id/${TENANT_ID}**`, async (route) => {
      batchPayload = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          created: [{ id: '999001', name: 'valuestream', http_url_to_repo: MOCK_REPOS[0].http_url_to_repo }],
          skipped: [],
          errors: [],
        }),
      });
    });

    const modal = await openGitlabSyncModal(page);
    await modal.locator('#gitlab-sync-workspace').selectOption('900001');

    // 行内 checkbox 为 pointer-events-none，须点整行切换选中
    await modal.locator('li', { hasText: 'other-repo' }).click();
    await modal.locator('li', { hasText: 'already-synced' }).click();

    await modal.getByTestId('gitlab-sync-create-btn').click();
    // 成功且无 errors 时 composable 会关闭弹窗（见 createSelectedProjects）
    await expect(modal).not.toBeVisible({ timeout: 15000 });

    expect(batchPayload).toMatchObject({
      workspace_id: '900001',
      repos: expect.arrayContaining([
        expect.objectContaining({ name: 'valuestream', http_url_to_repo: MOCK_REPOS[0].http_url_to_repo }),
      ]),
    });
    expect(batchPayload.repos).toHaveLength(1);
  });

  test('勾选多个仓库并合并为一个项目', async ({ page }) => {
    await mockGitlabRemoteRepos(page, {
      provider_key: 'gitlab:local-gitlab',
      gitlab_website: 'http://127.0.0.1:8012',
      oauth_bound: true,
      gitlab_login: 'example-user',
      repos: MOCK_REPOS,
      error: null,
    });

    let combinedPayload = null;
    await page.route(`**/api/projects/combined-from-gitlab-repos/tenant_id/${TENANT_ID}**`, async (route) => {
      combinedPayload = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          created: {
            id: '999002',
            name: 'example-user',
            repos: [
              { http_url_to_repo: MOCK_REPOS[0].http_url_to_repo },
              { http_url_to_repo: MOCK_REPOS[1].http_url_to_repo },
            ],
          },
          skipped: [],
          errors: [],
        }),
      });
    });

    const modal = await openGitlabSyncModal(page);
    await modal.locator('#gitlab-sync-workspace').selectOption('900001');
    await expect(modal.getByTestId('gitlab-sync-combined-name-input')).toBeVisible();
    await modal.getByTestId('gitlab-sync-combined-create-btn').click();
    await expect(modal).not.toBeVisible({ timeout: 15000 });

    expect(combinedPayload).toMatchObject({
      workspace_id: '900001',
      name: expect.any(String),
      repos: expect.arrayContaining([
        expect.objectContaining({ http_url_to_repo: MOCK_REPOS[0].http_url_to_repo }),
        expect.objectContaining({ http_url_to_repo: MOCK_REPOS[1].http_url_to_repo }),
      ]),
    });
    expect(combinedPayload.repos).toHaveLength(3);
  });

  test('批量创建排除单仓已占用仓库', async ({ page }) => {
    await mockGitlabRemoteRepos(page, {
      provider_key: 'gitlab:local-gitlab',
      gitlab_website: 'http://127.0.0.1:8012',
      oauth_bound: true,
      gitlab_login: 'example-user',
      repos: MOCK_REPOS,
      error: null,
    });

    let batchPayload = null;
    await page.route(`**/api/projects/batch-from-gitlab-repos/tenant_id/${TENANT_ID}**`, async (route) => {
      batchPayload = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          created: [
            { id: '999001', name: 'valuestream', http_url_to_repo: MOCK_REPOS[0].http_url_to_repo },
            { id: '999003', name: 'other-repo', http_url_to_repo: MOCK_REPOS[1].http_url_to_repo },
          ],
          skipped: [],
          errors: [],
        }),
      });
    });

    const modal = await openGitlabSyncModal(page);
    await modal.locator('#gitlab-sync-workspace').selectOption('900001');
    await modal.getByTestId('gitlab-sync-create-btn').click();
    await expect(modal).not.toBeVisible({ timeout: 15000 });

    expect(batchPayload.repos).toHaveLength(2);
    expect(batchPayload.repos.map((r) => r.name)).not.toContain('already-synced');
  });

  test('刷新列表重新请求 gitlab-remote-repos', async ({ page }) => {
    let callCount = 0;
    await page.route(`**/api/projects/gitlab-remote-repos/tenant_id/${TENANT_ID}**`, async (route) => {
      callCount += 1;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          oauth_bound: true,
          gitlab_website: 'http://127.0.0.1:8012',
          gitlab_login: 'example-user',
          repos: MOCK_REPOS,
          error: null,
        }),
      });
    });

    const modal = await openGitlabSyncModal(page);
    await expect(modal.getByText('valuestream').first()).toBeVisible();
    expect(callCount).toBeGreaterThanOrEqual(1);

    await modal.getByTestId('gitlab-sync-refresh-btn').click();
    await page.waitForTimeout(800);
    expect(callCount).toBeGreaterThanOrEqual(2);
  });
});

test.describe('Projects GitLab 同步（真实 API）', () => {
  test.beforeEach(async ({ page }) => {
    await loginToProjects(page);
  });

  test('登录后项目页可见「从 GitLab 同步」按钮', async ({ page }) => {
    const syncBtn = page.getByRole('button', { name: '从 GitLab 同步' }).first();
    await expect(syncBtn).toBeVisible({ timeout: 30000 });
  });

  test('打开对话框并调用 gitlab-remote-repos API', async ({ page }) => {
    const remoteReposPromise = page.waitForResponse(
      (r) =>
        r.url().includes(`/api/projects/gitlab-remote-repos/tenant_id/${TENANT_ID}`) &&
        r.request().method() === 'GET',
      { timeout: 60000 },
    );

    const modal = await openGitlabSyncModal(page);
    const remoteResp = await remoteReposPromise;
    expect(remoteResp.status()).toBe(200);

    const data = await remoteResp.json();
    expect(data).toHaveProperty('oauth_bound');

    if (data.oauth_bound) {
      await expect(modal.getByText('来源：')).toBeVisible({ timeout: 15000 });
      await expect(modal.locator('#gitlab-sync-workspace')).toBeVisible();
      if (Array.isArray(data.repos) && data.repos.length > 0) {
        const first = data.repos[0];
        const label = first.path_with_namespace || first.name;
        await expect(modal.getByText(label, { exact: false })).toBeVisible({ timeout: 15000 });
      } else {
        await expect(modal.getByText(/未找到可同步|尚未绑定/i)).toBeVisible({ timeout: 15000 });
      }
    } else {
      await expect(modal.getByTestId('gitlab-sync-oauth-btn')).toBeVisible({ timeout: 15000 });
    }
  });

  test('OAuth 已绑定时可勾选仓库（不提交创建）', async ({ page }) => {
    const remoteResp = await page.waitForResponse(
      (r) =>
        r.url().includes(`/api/projects/gitlab-remote-repos/tenant_id/${TENANT_ID}`) &&
        r.request().method() === 'GET',
      { timeout: 60000 },
    ).catch(() => null);

    const modal = await openGitlabSyncModal(page);
    const resp = remoteResp || (await page.waitForResponse(
      (r) =>
        r.url().includes(`/api/projects/gitlab-remote-repos/tenant_id/${TENANT_ID}`) &&
        r.request().method() === 'GET',
      { timeout: 60000 },
    ));

    const data = await resp.json();
    test.skip(!data.oauth_bound, '用户尚未绑定 GitLab OAuth，跳过勾选测试');

    const selectable = (data.repos || []).filter((r) => !r.imported_in_single_repo_project);
    test.skip(selectable.length === 0, '无可单独创建的仓库，跳过勾选测试');

    await modal.locator('#gitlab-sync-workspace').selectOption({ index: 1 });
    const firstRepo = selectable[0];
    const row = modal.locator('li', { hasText: firstRepo.path_with_namespace || firstRepo.name }).first();
    // checkbox 为 pointer-events-none，点行切换
    await row.click();
    await expect(modal.getByTestId('gitlab-sync-create-btn')).toBeEnabled();
    await modal.getByRole('button', { name: '关闭' }).click();
    await expect(modal).not.toBeVisible();
  });
});
