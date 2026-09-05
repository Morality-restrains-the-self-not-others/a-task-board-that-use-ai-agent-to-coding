// @ts-check
/**
 * 回归：任务 TaskProject.repo_address 与项目当前 git_repos 不一致时，
 * 「直接启动」应阻断启动直至用户确认不更新或同步地址。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '846269443533955072';
const TASK_DETAIL_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?relayToTrae=true`;

const CURRENT_REPO = 'http://localhost:8012/ljy/somanyad';
const STALE_REPO = 'http://localhost:8012/example-user/somanyad';

const mockTaskPayload = (projectsOverride = null) => ({
  id: TASK_ID,
  title: 'Stale repo relay task',
  workspace_id: WORKSPACE_ID,
  container_image_id: 'img-relay-stale',
  comments: [],
  ai_comments: [],
  assignees: [],
  projects: projectsOverride ?? [
    {
      project_id: 'proj-somanyad',
      repo_index: 0,
      base_branch: 'main',
      target_branch: 'feature/task',
      stored_repo_address: STALE_REPO,
      project_repo_url: CURRENT_REPO,
      repo_address_mismatch: true,
    },
  ],
});

test.describe('TaskDetail relayToTrae stale repo address gate', () => {
  test('仓库地址不一致时启动按钮禁用，确认不更新后可启动', async ({ page }) => {
    let startRequestCount = 0;

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockTaskPayload()),
        });
        return;
      }

      if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'img-relay-stale', name: 'Relay Image', version: '1.0' }]),
        });
        return;
      }

      if (url.includes('/relay-to-trae/env-prepare/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            env: {
              TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
              BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
              ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
            },
          }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/health/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/register/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', task_id: TASK_ID }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/token-init/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', token_initialized: true, task_id: TASK_ID }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/repo-credentials-precheck/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', message: '仓库凭证预检通过', repo_count: 1 }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/start/') && method === 'POST') {
        startRequestCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'accepted', request_id: 'relay-stale-repo-ok' }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.getByTestId('server-config-relay-direct-tab').click();

    const banner = page.getByTestId('relay-to-trae-stale-repo-mismatch-banner');
    await expect(banner).toBeVisible({ timeout: 15000 });
    await expect(banner.getByText(STALE_REPO)).toBeVisible();
    await expect(banner.getByText(CURRENT_REPO)).toBeVisible();

    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    await expect(startBtn).toBeDisabled();
    await startBtn.click({ force: true });
    expect(startRequestCount, '阻断时不应调用 start').toBe(0);

    await page.getByTestId('relay-to-trae-ack-stale-repo-btn').click();
    await expect(startBtn).toBeEnabled({ timeout: 5000 });
    await startBtn.click();

    await expect
      .poll(() => startRequestCount, { timeout: 15000, message: '确认不更新后应调用 start' })
      .toBeGreaterThan(0);
  });

  test('点击更新任务仓库地址后 mismatch 消失且可直接启动', async ({ page }) => {
    let patchBody = null;
    let taskProjects = mockTaskPayload().projects;

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockTaskPayload(taskProjects)),
        });
        return;
      }

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'PATCH') {
        patchBody = req.postDataJSON();
        taskProjects = [
          {
            project_id: 'proj-somanyad',
            repo_index: 0,
            base_branch: 'main',
            target_branch: 'feature/task',
            stored_repo_address: CURRENT_REPO,
            project_repo_url: CURRENT_REPO,
            repo_address_mismatch: false,
          },
        ];
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ ...mockTaskPayload(taskProjects), id: TASK_ID }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/register/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok' }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/token-init/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', token_initialized: true }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/repo-credentials-precheck/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'ok', repo_count: 1 }),
        });
        return;
      }

      if (url.includes('/relay-to-trae/start/') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'accepted', request_id: 'after-sync' }),
        });
        return;
      }

      if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`)) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'img-relay-stale', name: 'Relay Image', version: '1.0' }]),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.getByTestId('server-config-relay-direct-tab').click();

    await page.getByTestId('relay-to-trae-sync-stale-repo-btn').click();

    await expect
      .poll(() => patchBody, { timeout: 10000, message: '应 PATCH 同步 projects' })
      .toBeTruthy();
    expect(patchBody?.projects?.[0]?.project_id).toBe('proj-somanyad');

    await expect(page.getByTestId('relay-to-trae-stale-repo-mismatch-banner')).toHaveCount(0, {
      timeout: 10000,
    });

    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    await expect(startBtn).toBeEnabled();
    await startBtn.click();
    await expect(page.getByText(/启动请求已受理/)).toBeVisible({ timeout: 10000 });
  });
});
