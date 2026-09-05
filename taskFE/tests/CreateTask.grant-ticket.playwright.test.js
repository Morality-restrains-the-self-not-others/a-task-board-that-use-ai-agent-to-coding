// @ts-check
/**
 * OPT-20260829-014：创建任务 auto_run 时无 grant_ticket 禁用提交；回流后可提交。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { addE2eSession, matchTenantShellApi, mockMeBody } from './playwrightE2eSession.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const PROJECT_ID = 'proj_create_grant_ticket';
const GITHUB_URL = 'https://github.com/ruandao/helloworld.git';
const IMAGE_ID = 'img-create-grant-1';
const WP = `/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`;

const RUN_TEMPLATE = {
  label: 'mock-run',
  default_auto_run: true,
  platform: 'aliyun',
  region: 'cn-hangzhou',
  selected_instance: 'ecs.n1',
};

/**
 * @param {import('@playwright/test').Page} page
 */
async function installWorkPanelRoutes(page) {
  await page.route('**/api/**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();
    if (url.includes('/api/accounts/users/me/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...mockMeBody(TENANT_ID),
          current_workspace: { id: WORKSPACE_ID, name: 'WS' },
        }),
      });
      return;
    }
    const shell = matchTenantShellApi(url, method, TENANT_ID);
    if (shell) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(shell) });
      return;
    }
    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}`) && method === 'GET') {
      if (url.includes('create-task-field-settings')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ fields: { feature_params: false } }),
        });
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [{ id: WORKSPACE_ID, name: 'WS' }],
          results: [{ id: WORKSPACE_ID, name: 'WS' }],
        }),
      });
      return;
    }
    if (url.includes(`/api/projects/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: PROJECT_ID,
            name: 'helloworld',
            git_repos: [GITHUB_URL],
            server_run_template: RUN_TEMPLATE,
          },
        ]),
      });
      return;
    }
    if (url.includes('/api/cloud/installed-images/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: IMAGE_ID, name: 'trae', version: '1' }]),
      });
      return;
    }
    if (url.includes('/api/git-identities/user/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          identities: [
            { id: 'gid-create-1', name: 'alice', email: 'alice@example.com', is_default: true },
          ],
        }),
      });
      return;
    }
    if (url.includes('/api/git-oauth/user-app-connection/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ connected: true }),
      });
      return;
    }
    if (url.includes('/progress-system/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ columns: [{ id: 'col1', name: '待办' }] }),
      });
      return;
    }
    if (url.includes('/api/tasks/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ results: [], columns: [] }),
      });
      return;
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

async function openCreateAndEnableAutoRun(page) {
  await expect(page.locator('#create-task-btn')).toBeVisible({ timeout: 20000 });
  await page.locator('#create-task-btn').click();
  await expect(page.getByTestId('create-task-submit-btn')).toBeVisible({ timeout: 15000 });
  await expect(page.getByTestId('create-task-repo-url')).toBeVisible({ timeout: 20000 });
  await page.locator('#task-title').fill('grant ticket create');
  const desc = page.locator('#task-description');
  await desc.click();
  await desc.fill('');
  await desc.pressSequentially('$');
  const picker = page.getByTestId('task-description-image-mention-picker');
  await expect(picker).toBeVisible({ timeout: 15000 });
  await picker.locator('li').first().click();
  const auto = page.getByTestId('task-auto-run-checkbox');
  await expect(auto).toBeEnabled({ timeout: 20000 });
  await auto.check();
  const ident = page.getByTestId('create-task-repo-git-identity-select');
  await expect(ident).toBeVisible({ timeout: 10000 });
  await ident.selectOption('gid-create-1');
}

test.describe('CreateTask grant_ticket（OPT-20260829-014）', () => {
  test('auto_run 且无 session ticket 时提交禁用', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-create-grant-block' });
    await installWorkPanelRoutes(page);
    const waitProjects = page.waitForResponse(
      (r) => r.url().includes(`/api/projects/tenant_id/${TENANT_ID}`) && !r.url().includes('/workspaces/') && r.ok(),
      { timeout: 20000 },
    );
    await page.goto(WP);
    await waitProjects;
    await openCreateAndEnableAutoRun(page);
    await expect(page.getByTestId('create-task-submit-btn')).toBeDisabled({ timeout: 15000 });
    await expect(page.getByTestId('create-task-submit-blocked-reason')).toContainText('OAuth');
  });

  test('回流 grant_ticket 后可提交', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-create-grant-ok' });
    await installWorkPanelRoutes(page);
    const ticket = 'grant-ticket-014';
    const waitProjects = page.waitForResponse(
      (r) => r.url().includes(`/api/projects/tenant_id/${TENANT_ID}`) && !r.url().includes('/workspaces/') && r.ok(),
      { timeout: 20000 },
    );
    await page.goto(
      `${WP}&grant_ticket=${ticket}&repo_url=${encodeURIComponent(GITHUB_URL)}`,
    );
    await waitProjects;
    await page.evaluate((t) => {
      sessionStorage.setItem('gitOauthGrantTickets', JSON.stringify({ '*': t, 'github.com': t }));
    }, ticket);
    await openCreateAndEnableAutoRun(page);
    await expect(page.getByTestId('create-task-submit-btn')).toBeEnabled({ timeout: 15000 });
  });
});
