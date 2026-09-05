/**
 * CDP 9222 验收：推送并创建PR 成功后出现 PR 按钮，点击打开审查页。
 *
 * 运行：
 *   node taskFE/tests/TaskDetail.ztree-pr-btn-after-push-cdp.mjs
 */
import { chromium } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';

const CDP_URL = (process.env.CDP_URL || 'http://127.0.0.1:9222').replace(/\/$/, '');
const SITE_BASE = (process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:4000').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '861623708318031872';
const TASK_ID = 'task_ztree_pr_btn_mock';
const LAYER_ID = '20260713_pr_btn_layer';
const PR_URL = 'https://gitlab.daydaymoney.com/example-user/somanyad/-/merge_requests/42';
const TASK_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

function fail(msg) {
  console.error('FAIL:', msg);
  process.exitCode = 1;
}

async function main() {
  const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
  const context = browser.contexts()[0] || (await browser.newContext());
  const page = await context.newPage();
  page.on('dialog', async (d) => {
    try {
      await d.accept();
    } catch {
      /* dialog already closed */
    }
  });

  let graphGeneration = 0;

  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: TASK_ID,
          title: 'Mock push create PR button',
          description: '',
          created_at: '2026-07-13T00:00:00Z',
          created_by: { username: 'mock' },
          comments: [],
          ai_comments: [],
          assignees: [],
          workspace_id: WORKSPACE_ID,
          projects: [
            {
              stored_repo_address: 'https://gitlab.daydaymoney.com/example-user/somanyad.git',
              target_branch: 'feature/mock-pr-btn',
            },
          ],
        }),
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
      const ahead = graphGeneration === 0 ? 1 : 0;
      const gitRemote =
        ahead > 0
          ? { is_git: true, ahead: 1, no_upstream: false, upstream: 'origin/feature/mock-pr-btn' }
          : {
              is_git: true,
              ahead: 0,
              no_upstream: false,
              upstream: 'origin/feature/mock-pr-btn',
              last_pushed_count: 1,
              pr_html_url: PR_URL,
            };
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          layers_root: '/workspace/layers',
          bootstrap_layer_id: LAYER_ID,
          layers: [
            {
              layer_id: LAYER_ID,
              parent_layer_id: null,
              created_at: '2026-07-13T06:14:44Z',
              command: 'mock push pr btn',
              job_status: 'completed',
              mind_state: 'idle_done',
              git_worktree_dirty: false,
              git_remote: gitRemote,
              queue_depth: 0,
            },
          ],
          jobs: [
            {
              id: 'job-mock-pr-btn',
              layer_id: LAYER_ID,
              status: 'completed',
              command_kind: 'trae',
              command: 'mock push pr btn',
              created_at: '2026-07-13T06:15:00Z',
            },
          ],
        }),
      });
      return;
    }

    if (url.includes('container-layer-git-push') && method === 'POST') {
      graphGeneration = 1;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ok: true,
          git_remote: { is_git: true, ahead: 0, no_upstream: false },
          github_pull_request: {
            html_url: PR_URL,
            number: 42,
            provider: 'gitlab',
          },
          github_oauth_multirepo: {
            repos: [
              {
                rel_prefix: 'somanyad',
                push_ok: true,
                provider: 'gitlab',
                github_slug: 'example-user/somanyad',
                pr: { html_url: PR_URL, number: 42 },
              },
            ],
          },
        }),
      });
      return;
    }

    if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    if (url.includes('/progress-system/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ columns: [] }),
      });
      return;
    }

    if (url.includes('/cloud/compute/container-clone-log/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ layer_id: LAYER_ID, text: '' }),
      });
      return;
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  try {
    await page.goto(`${SITE_BASE}${TASK_PATH}`, { waitUntil: 'domcontentloaded', timeout: 60000 });

    const ztree = page.getByTestId('comment-layer-ztree-panel');
    await ztree.waitFor({ state: 'visible', timeout: 60000 });

    const beforePr = await ztree.getByTestId('layer-ztree-pr-btn').count();
    if (beforePr !== 0) {
      fail(`推送前不应有 PR 按钮，实际 ${beforePr}`);
      return;
    }

    const pushBtn = ztree.getByTestId('layer-ztree-push-btn');
    await pushBtn.waitFor({ state: 'visible', timeout: 15000 });
    const pushText = ((await pushBtn.innerText()) || '').trim();
    if (pushText !== '推送并创建PR') {
      fail(`推送按钮文案应为「推送并创建PR」，实际「${pushText}」`);
      return;
    }

    await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('container-layer-git-push') && r.request().method() === 'POST',
        { timeout: 30000 },
      ),
      pushBtn.click(),
    ]);

    const prBtn = ztree.getByTestId('layer-ztree-pr-btn');
    await prBtn.waitFor({ state: 'visible', timeout: 15000 });

    const pushAfter = await ztree.getByTestId('layer-ztree-push-btn').count();
    if (pushAfter !== 0) {
      fail(`推送后应隐藏推送按钮，仍有 ${pushAfter} 个`);
      return;
    }

    const href = await prBtn.getAttribute('href');
    if (href !== PR_URL) {
      fail(`PR 锚点 href 应为 ${PR_URL}，实际 ${href}`);
      return;
    }
    const target = await prBtn.getAttribute('target');
    if (target !== '_blank') {
      fail(`PR 锚点 target 应为 _blank，实际 ${target}`);
      return;
    }

    console.log('OK: 推送后出现可点击 PR 锚点', PR_URL);
    process.exitCode = 0;
  } catch (e) {
    fail(String(e?.message || e));
  } finally {
    await page.unroute('**/api/**').catch(() => {});
    await page.close().catch(() => {});
    process.exit(process.exitCode || 0);
  }
}

main().catch((e) => {
  fail(String(e?.message || e));
  process.exit(1);
});
