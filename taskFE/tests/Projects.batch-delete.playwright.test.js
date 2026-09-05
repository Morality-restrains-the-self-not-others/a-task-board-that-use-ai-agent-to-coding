// @ts-check
/**
 * E2E: 项目列表页 — 批量删除项目（Mock API）
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

/** @param {import('@playwright/test').Page} page */
async function loginToProjects(page) {
  await installApisixCorsWorkaround(page);
  await loginViaGatewayApi(page, {
    email: CREDENTIALS.email,
    password: CREDENTIALS.password,
    siteOrigin: BASE_URL,
  });
  await gotoAuthenticatedPath(page, PROJECTS_URL);
  await page.waitForLoadState('networkidle').catch(() => {});
}

test.describe('Projects 批量删除（Mock API）', () => {
  test('选择模式批量删除并提交 project_ids', async ({ page }) => {
    await loginToProjects(page);

    const mockProjects = [
      {
        id: '910001',
        name: 'E2E Delete A',
        description: 'mock',
        status: '进行中',
        created_at: '2026-07-05T00:00:00Z',
      },
      {
        id: '910002',
        name: 'E2E Delete B',
        description: 'mock',
        status: '进行中',
        created_at: '2026-07-05T00:00:00Z',
      },
    ];

    await page.route(`**/api/projects/tenant_id/${TENANT_ID}**`, async (route) => {
      const url = route.request().url();
      if (route.request().method() === 'GET' && url.endsWith(`/api/projects/tenant_id/${TENANT_ID}`)) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockProjects),
        });
        return;
      }
      await route.continue();
    });

    await page.reload({ waitUntil: 'domcontentloaded' });
    await expect(page.getByText('E2E Delete A')).toBeVisible({ timeout: 30000 });

    let batchPayload = null;
    await page.route(`**/api/projects/batch-delete/tenant_id/${TENANT_ID}/**`, async (route) => {
      batchPayload = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          deleted: [{ id: '910001', name: 'E2E Delete A' }],
          errors: [],
        }),
      });
    });

    await page.getByTestId('projects-select-mode-btn').click();
    await page.getByTestId('project-card-910001').click();
    await page.getByTestId('projects-batch-delete-btn').click();
    await expect(page.getByTestId('batch-delete-projects-modal')).toBeVisible();
    await page.getByTestId('batch-delete-confirm-btn').click();

    await expect(page.getByText('已删除 1 个项目')).toBeVisible({ timeout: 10000 });
    expect(batchPayload).toMatchObject({
      project_ids: ['910001'],
    });
  });
});
