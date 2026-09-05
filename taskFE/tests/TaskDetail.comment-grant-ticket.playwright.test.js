// @ts-check
/**
 * OPT-20260829-017：@镜像 后无 grant_ticket 时「提交并运行」disabled；
 * 回流 grant_ticket= 后可点且 POST 含 grant_ticket。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { addE2eSession } from './playwrightE2eSession.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const TASK_ID = 'task_comment_grant_ticket';
const PROJECT_ID = 'proj_comment_grant_ticket';
const GITHUB_URL = 'https://github.com/ruandao/helloworld.git';
const IMAGE_ID = 'img-grant-1';
const BASE_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/`;

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ capturePosts?: object[] }} [opts]
 */
async function installRoutes(page, opts = {}) {
  const capturePosts = opts.capturePosts || null;
  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    if (url.includes(`/api/tasks/${TASK_ID}/comments/tenant_id/${TENANT_ID}`) && method === 'POST') {
      let body = {};
      try {
        body = JSON.parse(req.postData() || '{}');
      } catch {
        body = {};
      }
      if (capturePosts) capturePosts.push(body);
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ id: 'c-new', content: body.content || '' }),
      });
      return;
    }
    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: TASK_ID,
          title: 'Grant ticket comment',
          description: 'L2',
          created_at: '2026-07-22T00:00:00Z',
          created_by: { username: 'mock' },
          comments: [],
          ai_comments: [],
          assignees: [],
          workspace_id: WORKSPACE_ID,
          projects: [{ project_id: PROJECT_ID, repo_index: 0, base_branch: 'main' }],
        }),
      });
      return;
    }
    if (url.includes(`/api/tasks/${TASK_ID}/comments/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ results: [], next_cursor: null, has_more: false }),
      });
      return;
    }
    if (url.includes(`/api/projects/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: PROJECT_ID, name: 'helloworld', git_repos: [GITHUB_URL] }]),
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
    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}/${WORKSPACE_ID}/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: WORKSPACE_ID, container_image_at_mode_enabled: true }),
      });
      return;
    }
    if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: IMAGE_ID, name: 'trae', version: '1' }]),
      });
      return;
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

async function mentionImage(page) {
  const editor = page.getByTestId('comment-content-editor');
  await expect(editor).toBeVisible({ timeout: 20000 });
  await editor.click();
  await editor.pressSequentially('$');
  const picker = page.getByTestId('comment-image-mention-picker');
  await expect(picker).toBeVisible({ timeout: 10000 });
  await picker.locator('li').first().click();
}

test.describe('TaskDetail 评论 grant_ticket（OPT-20260829-017）', () => {
  test('无 grant_ticket 时提交并运行 disabled', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-grant-blocked' });
    await installRoutes(page);
    await page.goto(`${BASE_PATH}?accessCode=u824976301710503936`);
    await page.waitForLoadState('domcontentloaded');
    await mentionImage(page);
    const submit = page.getByTestId('task-detail-comment-submit');
    await expect(submit).toBeDisabled({ timeout: 15000 });
    await expect(page.getByTestId('comment-composer-oauth-blocked-reason')).toBeVisible();
  });

  test('回流 grant_ticket 后可提交且 POST 含 grant_ticket', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-grant-ok' });
    /** @type {object[]} */
    const posts = [];
    await installRoutes(page, { capturePosts: posts });
    const ticket = 'grant-ticket-017';
    await page.goto(
      `${BASE_PATH}?accessCode=u824976301710503936&grant_ticket=${ticket}&repo_url=${encodeURIComponent(GITHUB_URL)}`,
    );
    await page.waitForLoadState('domcontentloaded');
    await mentionImage(page);
    const submit = page.getByTestId('task-detail-comment-submit');
    await expect(submit).toBeEnabled({ timeout: 15000 });
    await submit.click();
    await expect.poll(() => posts.length, { timeout: 15000 }).toBeGreaterThan(0);
    expect(String(posts[0].grant_ticket || '')).toBe(ticket);
  });
});
