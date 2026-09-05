// @ts-check
/**
 * E2E: 项目编辑页 — 标签与运行模版保存（Mock API）
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import {
  loginViaGatewayApi,
  gotoAuthenticatedPath,
} from './helpers/gatewayLoginE2e.js';
import { installApisixCorsWorkaround } from './helpers/remoteLoginE2e.js';

const portConfig = loadPortConfig();

const BASE_URL =
  (process.env.SITE_BASE || process.env.BASE_URL || '').replace(/\/$/, '') ||
  portConfig.vue?.publicBaseUrl ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`;

const TENANT_ID = process.env.TEST_TENANT_ID || '850256677331562496';
const PROJECT_ID = '861581450509701120';
const EDIT_URL = `${BASE_URL}/tenant/${TENANT_ID}/projects/${PROJECT_ID}/edit/`;
const DETAIL_URL = `${BASE_URL}/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;

const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

const MOCK_PROJECT = {
  id: PROJECT_ID,
  name: 'E2E 编辑标签项目',
  description: 'desc',
  tags: [],
  git_repos: [],
  container_image_id: '',
  container_image: '',
  workspaces: ['857903329669984256'],
  server_run_template: {},
  company: TENANT_ID,
  created_at: '2026-07-05T12:00:00Z',
  updated_at: '2026-07-05T12:00:00Z',
};

/** @param {import('@playwright/test').Page} page */
async function loginAndOpenEdit(page) {
  await installApisixCorsWorkaround(page);
  await loginViaGatewayApi(page, {
    email: CREDENTIALS.email,
    password: CREDENTIALS.password,
    siteOrigin: BASE_URL,
  });
  await gotoAuthenticatedPath(page, EDIT_URL);
  await page.waitForLoadState('domcontentloaded');
}

test.describe('ProjectEdit 标签保存（Mock API）', () => {
  test('编辑页添加标签并 PUT 提交', async ({ page }) => {
    let putPayload = null;

    await page.route(`**/api/projects/tenant_id/${TENANT_ID}${PROJECT_ID}/`, async (route) => {
      const method = route.request().method();
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_PROJECT),
        });
        return;
      }
      if (method === 'PUT') {
        putPayload = route.request().postDataJSON();
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ...MOCK_PROJECT,
            ...putPayload,
            tags: putPayload?.tags || [],
          }),
        });
        return;
      }
      await route.continue();
    });

    await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { id: '857903329669984256', name: '默认工作空间' },
        ]),
      });
    });

    await page.route(`**/api/cloud/installed-images/tenant_id/${TENANT_ID}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.route(`**/api/cloud/server-config-default/tenant_id/${TENANT_ID}/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [] }),
      });
    });

    await loginAndOpenEdit(page);

    await expect(page.getByTestId('project-tags-input')).toBeVisible({ timeout: 30000 });
    await expect(page.getByTestId('project-run-template-panel')).toBeVisible();

    const tagInput = page.locator('#project-tags');
    await tagInput.fill('e2e-tag');
    await tagInput.press('Enter');

    await expect(page.getByText('e2e-tag', { exact: true })).toBeVisible();

    await page.getByRole('button', { name: '保存更改' }).click();

    await expect.poll(() => putPayload).not.toBeNull();
    expect(putPayload.tags).toEqual(['e2e-tag']);
    expect(putPayload).toHaveProperty('server_run_template');
  });

  test('已安装镜像下拉应展示 CPU 架构', async ({ page }) => {
    const MOCK_IMAGE_ID = '859671040643174400';
    const projectWithImage = {
      ...MOCK_PROJECT,
      container_image_id: MOCK_IMAGE_ID,
    };

    await page.route(`**/api/projects/tenant_id/${TENANT_ID}${PROJECT_ID}/`, async (route) => {
      const method = route.request().method();
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(projectWithImage),
        });
        return;
      }
      await route.continue();
    });

    await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: '857903329669984256', name: '默认工作空间' }]),
      });
    });

    await page.route(`**/api/cloud/installed-images/tenant_id/${TENANT_ID}**`, async (route) => {
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
        ]),
      });
    });

    await page.route(`**/api/cloud/server-config-default/tenant_id/${TENANT_ID}/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [] }),
      });
    });

    await loginAndOpenEdit(page);

    await expect(page.locator('#containerImage')).toHaveValue(MOCK_IMAGE_ID);
    await expect(page.getByText('支持 CPU 架构：x86_64')).toBeVisible();
    await expect(page.getByTestId('image-architecture-display')).toHaveText('x86_64');
  });

  test('保存后详情页展示标签', async ({ page }) => {
    const projectWithTag = { ...MOCK_PROJECT, tags: ['shown-tag'] };

    await page.route(`**/api/projects/tenant_id/${TENANT_ID}${PROJECT_ID}/`, async (route) => {
      const url = route.request().url();
      if (url.includes('/branches/')) {
        await route.continue();
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(projectWithTag),
      });
    });

    await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}**`, async (route) => {
      const url = route.request().url();
      if (url.includes('ids=')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: '857903329669984256', name: '默认工作空间' }]),
        });
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.route(`**/api/cloud/server-config-default/tenant_id/${TENANT_ID}/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [] }),
      });
    });

    await installApisixCorsWorkaround(page);
    await loginViaGatewayApi(page, {
      email: CREDENTIALS.email,
      password: CREDENTIALS.password,
      siteOrigin: BASE_URL,
    });
    await gotoAuthenticatedPath(page, DETAIL_URL);
    await page.waitForLoadState('domcontentloaded');

    await expect(page.getByText('shown-tag', { exact: true })).toBeVisible({ timeout: 30000 });
    await expect(page.getByText('运行模版摘要')).toBeVisible();
  });
});
