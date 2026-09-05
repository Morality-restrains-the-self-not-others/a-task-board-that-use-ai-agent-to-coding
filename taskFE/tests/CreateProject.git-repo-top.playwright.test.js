// @ts-check
/**
 * E2E: CreateProject — Git 仓库 URL 输入位于表单顶端
 *
 * 用户可能需要先做仓库授权，故 #gitRepo0 应排在 #projectName 之前。
 *
 * 账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 * 租户：850256677331562496
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';

const portConfig = loadPortConfig();

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
    waitUntil: 'domcontentloaded',
    timeout: 15000,
  });
  await page.waitForSelector('#gitRepo0', { timeout: 10000 });
  await page.waitForSelector('#projectName', { timeout: 10000 });
}

test.describe('CreateProject Git 仓库置顶', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await navigateToCreateProject(page);
  });

  test('#gitRepo0 在 DOM 中位于 #projectName 之前', async ({ page }) => {
    const gitRepo = page.locator('#gitRepo0');
    const projectName = page.locator('#projectName');
    await expect(gitRepo).toBeVisible();
    await expect(projectName).toBeVisible();

    const gitComesFirst = await page.evaluate(() => {
      const git = document.querySelector('#gitRepo0');
      const name = document.querySelector('#projectName');
      if (!git || !name) return false;
      // DOCUMENT_POSITION_FOLLOWING = 4 → name 在 git 之后
      return (git.compareDocumentPosition(name) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0;
    });
    expect(gitComesFirst).toBe(true);

    const gitBox = await gitRepo.boundingBox();
    const nameBox = await projectName.boundingBox();
    expect(gitBox).toBeTruthy();
    expect(nameBox).toBeTruthy();
    expect(gitBox.y).toBeLessThan(nameBox.y);
  });
});
