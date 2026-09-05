// @ts-check
/**
 * OPT-20260830-001：任务详情与项目详情「历史版本」打开/收起。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { addE2eSession, matchTenantShellApi } from './playwrightE2eSession.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const TASK_ID = 'task_entity_revision_pw';
const PROJECT_ID = 'proj_entity_revision_pw';

test.describe('历史版本面板（OPT-20260830-001）', () => {
  test('任务详情：打开 entity-revision 空态后收起', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-rev-task' });
    const path = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`;

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();
      if (url.includes('/revisions/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ results: [], items: [] }),
        });
        return;
      }
      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Revision task',
            description: 'hist',
            created_at: '2026-07-22T00:00:00Z',
            created_by: { username: 'mock' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            projects: [],
          }),
        });
        return;
      }
      if (url.includes(`/api/tasks/${TASK_ID}/comments/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ results: [], next_cursor: null, has_more: false }),
        });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(path);
    await page.waitForLoadState('domcontentloaded');
    const openBtn = page.getByTestId('entity-revision-open');
    await expect(openBtn).toBeVisible({ timeout: 20000 });
    await openBtn.click();
    await expect(page.getByTestId('entity-revision-panel')).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('entity-revision-empty')).toBeVisible();
    await page.getByTestId('entity-revision-close').click();
    await expect(page.getByTestId('entity-revision-panel')).toHaveCount(0);
  });

  test('项目详情：打开 entity-revision 后收起', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-rev-proj' });
    const path = `/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();
      const shell = matchTenantShellApi(url, method, TENANT_ID);
      if (shell) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(shell) });
        return;
      }
      if (url.includes('/revisions/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ results: [], items: [] }),
        });
        return;
      }
      if (url.includes(`/api/projects/${PROJECT_ID}/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: PROJECT_ID,
            name: 'rev-proj',
            description: 'hist',
            git_repos: [],
            git_repos_status: [],
            workspaces: [],
            company: TENANT_ID,
            tags: [],
            server_run_template: {},
          }),
        });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(path);
    await page.waitForLoadState('domcontentloaded');
    const openBtn = page.getByTestId('entity-revision-open');
    await expect(openBtn).toBeVisible({ timeout: 20000 });
    await openBtn.click();
    await expect(page.getByTestId('entity-revision-panel')).toBeVisible({ timeout: 10000 });
    await page.getByTestId('entity-revision-close').click();
    await expect(page.getByTestId('entity-revision-panel')).toHaveCount(0);
  });
});
