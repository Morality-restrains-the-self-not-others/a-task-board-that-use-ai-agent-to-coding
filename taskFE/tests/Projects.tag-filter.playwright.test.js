// @ts-check
/**
 * E2E: 项目列表 — 按标签筛选（Mock API）
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
const PROJECTS_URL = `${BASE_URL}/tenant/${TENANT_ID}/projects/`;

const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

const MOCK_PROJECTS = [
  {
    id: '920001',
    name: 'Tag Filter A',
    description: 'alpha',
    tags: ['frontend', 'vue'],
    status: '进行中',
    created_at: '2026-07-06T00:00:00Z',
  },
  {
    id: '920002',
    name: 'Tag Filter B',
    description: 'beta',
    tags: ['backend'],
    status: '进行中',
    created_at: '2026-07-05T00:00:00Z',
  },
];

/** @param {import('@playwright/test').Page} page */
async function loginToProjects(page) {
  await installApisixCorsWorkaround(page);
  await loginViaGatewayApi(page, {
    email: CREDENTIALS.email,
    password: CREDENTIALS.password,
    siteOrigin: BASE_URL,
  });
  await page.route(`**/api/projects/tenant_id/${TENANT_ID}**`, async (route) => {
    const url = route.request().url();
    if (route.request().method() === 'GET' && url.endsWith(`/api/projects/tenant_id/${TENANT_ID}`)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_PROJECTS),
      });
      return;
    }
    await route.continue();
  });
  await gotoAuthenticatedPath(page, PROJECTS_URL);
  await page.waitForLoadState('domcontentloaded');
}

test.describe('Projects 标签筛选（Mock API）', () => {
  test('点击标签筛选项目并同步 URL query', async ({ page }) => {
    await loginToProjects(page);

    await expect(page.getByTestId('projects-tag-filter')).toBeVisible({ timeout: 30000 });
    await expect(page.getByText('Tag Filter A')).toBeVisible();
    await expect(page.getByText('Tag Filter B')).toBeVisible();

    await page.getByTestId('projects-tag-filter-backend').click();

    await expect(page.getByText('Tag Filter A')).not.toBeVisible();
    await expect(page.getByText('Tag Filter B')).toBeVisible();
    await expect(page).toHaveURL(/tags=backend/);

    await page.getByTestId('projects-clear-filters-btn').click();
    await expect(page.getByText('Tag Filter A')).toBeVisible();
    await expect(page.url()).not.toMatch(/tags=/);
  });
});
