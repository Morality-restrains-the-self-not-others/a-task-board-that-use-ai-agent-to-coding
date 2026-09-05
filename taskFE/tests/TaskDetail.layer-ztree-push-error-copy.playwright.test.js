// @ts-check
/**
 * OPT-20260829-016：层图 zTree push 失败「复制」把失败原文 + traceId 写入剪贴板。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { addE2eSession } from './playwrightE2eSession.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const TASK_ID = 'task_layer_ztree_push_copy';
const COMMENT_ID = 'comment-ztree-push-copy-001';
const PROJECT_ID = 'proj_ztree_push_copy';
const GITHUB_URL = 'https://github.com/ruandao/helloworld.git';
const PUSH_ERROR = 'remote: Permission to ruandao/helloworld.git denied to alice.';
const TRACE_ID = 'tid-ztree-copy-016';
const PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`;

test.describe('TaskDetail 层图 zTree push 失败复制（OPT-20260829-016）', () => {
  test('点击 layer-ztree-push-error-copy 写入失败原文与 traceId', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-ztree-copy' });
    await page.addInitScript(() => {
      window.__pwClipboard = '';
      const proto = navigator.clipboard;
      if (proto && typeof proto.writeText === 'function') {
        proto.writeText = async (text) => {
          window.__pwClipboard = String(text || '');
        };
      }
    });

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'zTree copy',
            description: 'push error copy',
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
          body: JSON.stringify({
            results: [{
              id: COMMENT_ID,
              content: '帮我改一下这个仓库',
              created_at: '2026-07-22T10:00:00Z',
              created_by: { username: '测试用户' },
              repo_identities: [{ repo_url: GITHUB_URL, git_identity_id: 'gid-copy-1' }],
            }],
            next_cursor: null,
            has_more: false,
          }),
        });
        return;
      }
      if (url.includes('/api/ai-comment/task-detail/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ results: [] }) });
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
          body: JSON.stringify({ connected: true, bind_status: 'active' }),
        });
        return;
      }
      if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            container_endpoint_registered: true,
            container_page_url: 'http://127.0.0.1:18080/ui/dev-local-token',
          }),
        });
        return;
      }
      if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layers_root: '/workspace/layers',
            bootstrap_layer_id: 'layer-copy-1',
            layers: [{
              layer_id: 'layer-copy-1',
              created_at: '2026-07-22T10:01:00Z',
              git_remote: { last_push_error: PUSH_ERROR, last_push_error_trace_id: TRACE_ID },
            }],
            jobs: [],
          }),
        });
        return;
      }
      if (url.includes('comment-container-bindings') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            bindings: [{
              comment_id: COMMENT_ID,
              status: 'running',
              container_name: 'mock-container-copy',
              csc_id: 'csc-copy-1',
              start_trace_id: 'trace-copy-1',
              logs: [],
            }],
          }),
        });
        return;
      }
      if (url.includes('comment-container-bindings') && method === 'POST') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true }) });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(PATH);
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByText('帮我改一下这个仓库')).toBeVisible({ timeout: 15000 });

    const execDetails = page.getByTestId('comment-execution-details');
    await expect(execDetails).toBeVisible({ timeout: 45000 });
    if ((await execDetails.getAttribute('open')) === null) {
      await page.getByTestId('comment-execution-details-summary').click();
    }

    const ztreeTab = page.getByTestId('comment-execution-tab-ztree');
    await expect(ztreeTab).toBeVisible({ timeout: 15000 });
    await ztreeTab.click();

    const copyBtn = page.getByTestId('layer-ztree-push-error-copy');
    await expect(copyBtn).toBeVisible({ timeout: 20000 });
    await copyBtn.click();

    const clip = await page.evaluate(() => String(window.__pwClipboard || ''));
    expect(clip).toContain(PUSH_ERROR);
    expect(clip).toMatch(/traceId:\s*tid-ztree-copy-016/);
  });
});
