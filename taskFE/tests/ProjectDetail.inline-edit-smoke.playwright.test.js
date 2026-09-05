// @ts-check
/**
 * E2E 冒烟：项目详情页 — 点击「项目标签 / 是否允许自动运行 / 已安装镜像」进入编辑控件
 *
 * 鉴权：注入 userId cookie + Mock me/profile（不依赖网关 /api/auth/，避免 502 阻塞冒烟）
 * 业务：Mock 项目详情与已安装镜像列表
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import { readE2eOrigins } from './helpers/gatewayLoginE2e.js';

const portConfig = loadPortConfig();

const origins = readE2eOrigins();
const BASE_URL =
  (process.env.SITE_BASE || process.env.BASE_URL || process.env.PLAYWRIGHT_SITE_ORIGIN || '').replace(/\/$/, '') ||
  origins.siteOrigin ||
  portConfig.vue?.publicBaseUrl ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`;

const TENANT_ID = process.env.TEST_TENANT_ID || process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.TEST_WORKSPACE_ID || '857903329669984256';
const PROJECT_ID = process.env.TEST_PROJECT_ID || '861581450509701120';
const MOCK_IMAGE_ID = process.env.PLAYWRIGHT_CONTAINER_IMAGE_ID || '859671040643174400';
const MOCK_USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488';
const DETAIL_URL = `${BASE_URL}/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;

const MOCK_PROJECT = {
  id: PROJECT_ID,
  name: 'E2E 点击即编辑冒烟',
  description: 'desc',
  tags: ['smoke-tag'],
  git_repos: [],
  container_image_id: MOCK_IMAGE_ID,
  container_image: 'task2app-trae',
  workspaces: [WORKSPACE_ID],
  server_run_template: {
    platform: 'aliyun',
    region: 'cn-hongkong',
    default_auto_run: false,
  },
  company: TENANT_ID,
  created_at: '2026-07-12T00:00:00Z',
  updated_at: '2026-07-12T00:00:00Z',
};

/** @param {import('@playwright/test').Page} page */
async function seedMockSession(page) {
  await page.context().addCookies([
    {
      name: 'userId',
      value: MOCK_USER_ID,
      url: `${BASE_URL}/`,
    },
  ]);
}

/** @param {import('@playwright/test').Page} page */
async function registerDetailMocks(page) {
  await page.route('**/*', async (route) => {
    const headers = { ...route.request().headers() };
    delete headers['x-requested-with'];
    const url = route.request().url();
    const method = route.request().method();

    if (url.includes('/accounts/users/me/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: MOCK_USER_ID, user_id: MOCK_USER_ID, email: 'smoke@example.com' }),
      });
      return;
    }

    if (url.includes('/api/accounts/users/profile/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user_id: MOCK_USER_ID,
          email: 'smoke@example.com',
          username: 'smoke',
        }),
      });
      return;
    }

    if (url.includes('/api/') && url.includes(`/projects/${PROJECT_ID}`) && !url.includes('/branches') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_PROJECT),
      });
      return;
    }

    if (url.includes('/installed-images/') && method === 'GET' && !url.includes('/regions/') && !url.includes('/catalog')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: MOCK_IMAGE_ID,
            name: 'task2app-trae',
            version: 'x86_64_2026-05-08',
            target_architectures: ['x86_64'],
          },
          {
            id: '9',
            name: 'other-image',
            version: '1.0',
            target_architectures: ['x86_64'],
          },
        ]),
      });
      return;
    }

    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}?ids=`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: WORKSPACE_ID, name: 'E2E Workspace' }]),
      });
      return;
    }

    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: WORKSPACE_ID, name: 'E2E Workspace' }]),
      });
      return;
    }

    if (url.includes('/api/accounts/git-oauth/providers') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ providers: [] }),
      });
      return;
    }

    await route.continue({ headers });
  });
}

/** @param {import('@playwright/test').Page} page */
async function openDetailWithMocks(page) {
  await registerDetailMocks(page);
  await seedMockSession(page);
  await page.goto(DETAIL_URL, { waitUntil: 'domcontentloaded', timeout: 60_000 });
  if (page.url().includes('/auth/login')) {
    throw new Error(`Mock 会话未生效，仍被重定向到登录页: ${page.url()}`);
  }
  await page.waitForLoadState('networkidle').catch(() => null);
}

test.describe('ProjectDetail inline edit smoke', () => {
  test('点击项目标签进入编辑控件', async ({ page }) => {
    await openDetailWithMocks(page);

    await expect(page.getByTestId('project-tags-display')).toBeVisible({ timeout: 30000 });
    await page.getByTestId('project-tags-display').click();
    await expect(page.getByTestId('project-tags-edit')).toBeVisible();
    await expect(page.getByTestId('project-tags-input')).toBeVisible();
    await expect(page.getByTestId('project-tags-save')).toBeVisible();
  });

  test('点击是否允许自动运行进入编辑控件', async ({ page }) => {
    await openDetailWithMocks(page);

    await expect(page.getByTestId('project-default-auto-run-display')).toBeVisible({ timeout: 30000 });
    await page.getByTestId('project-default-auto-run-display').click();
    await expect(page.getByTestId('project-default-auto-run-edit')).toBeVisible();
    await expect(page.getByTestId('project-default-auto-run')).toBeVisible();
    await expect(page.getByTestId('project-default-auto-run-save')).toBeVisible();
  });

  test('点击已安装镜像进入编辑控件', async ({ page }) => {
    await openDetailWithMocks(page);

    await expect(page.getByTestId('project-container-image-display')).toBeVisible({ timeout: 30000 });
    await page.getByTestId('project-container-image-display').click();
    await expect(page.getByTestId('project-container-image-edit')).toBeVisible();
    await expect(page.getByTestId('project-container-image-select')).toBeVisible();
    await expect(page.getByTestId('project-container-image-save')).toBeVisible();
  });
});
