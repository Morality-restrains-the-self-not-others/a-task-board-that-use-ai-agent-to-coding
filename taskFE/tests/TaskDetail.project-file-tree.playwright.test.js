// @ts-check
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662912';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`
);

const LAYER_ID = 'layer-1';
const JOB_ID = 'job-1';

test.describe('TaskDetail 项目文件树（mock）', () => {
  test('展开文件树应请求 layer files，点击文件可预览内容', async ({ page }) => {
    let filesApiHits = 0;
    let fileContentHits = 0;
    let gitLogHits = 0;

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
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock Task',
            description: 'Mock task detail',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
          }),
        });
        return;
      }

      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: [] }) });
        return;
      }

      if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            container_endpoint_registered: true,
            container_page_url: 'http://127.0.0.1:18080/ui/mock-token',
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
            bootstrap_layer_id: LAYER_ID,
            layers: [
              {
                layer_id: LAYER_ID,
                parent_layer_id: null,
                created_at: '2026-01-01T00:00:00Z',
                command: 'mock command',
                job_status: 'completed',
              },
            ],
            jobs: [
              {
                id: JOB_ID,
                layer_id: LAYER_ID,
                status: 'completed',
                command_kind: 'trae',
                command: 'echo mock',
                created_at: '2026-01-01T00:00:01Z',
                output: 'job-output',
              },
            ],
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-job-execution-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            job: { id: JOB_ID, status: 'completed', layer_id: LAYER_ID, command: 'echo mock', output: 'job-output' },
            steps: { note: '', steps: [] },
            layer_changes: {
              layer_id: LAYER_ID,
              change_count: 1,
              changes: [{ path: 'repo-a/src/a.py', kind: 'modified' }],
              truncated: false,
            },
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-files/') && method === 'GET') {
        filesApiHits += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layer_id: LAYER_ID,
            truncated: false,
            files: [
              { path: 'repo-a/src/a.py', size: 12, mtime: '2026-01-01T00:00:00' },
              { path: 'repo-a/README.md', size: 34, mtime: '2026-01-01T00:00:00' },
              { path: 'repo-b/main.py', size: 56, mtime: '2026-01-01T00:00:00' },
            ],
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-file-content/') && method === 'GET') {
        fileContentHits += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            path: 'repo-a/src/a.py',
            content: 'print(\"tree preview\")',
            truncated: false,
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-git-log/') && method === 'GET') {
        gitLogHits += 1;
        const u = new URL(url);
        const reqPath = u.searchParams.get('path') || '';
        const isRepoRoot = reqPath === 'repo-a' || reqPath === 'repo-b';
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layer_id: LAYER_ID,
            path: reqPath,
            repo: 'repo-a',
            status: 'ok',
            commits: [{ hash: 'abc', short: 'abc', subject: 'init', author: 't', email: 't@e', date: '2026-01-01' }],
            text: 'commit abc\nAuthor: t\n\n    init',
            truncated: false,
            is_repo_root: isRepoRoot,
            ...(isRepoRoot ? { current_branch: 'feature/mock-work' } : {}),
          }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    await expect(ztreePanel).toBeVisible({ timeout: 30000 });
    await ztreePanel.locator('button.break-words.flex-1.min-w-0').nth(0).click();

    const execPanel = page.getByTestId('comment-layer-ztree-exec-log-panel');
    await expect(execPanel).toBeVisible({ timeout: 15000 });

    const fileTreeRoot = page.getByTestId('comment-layer-ztree-project-file-tree');
    await expect(fileTreeRoot).toBeVisible({ timeout: 15000 });

    await expect
      .poll(() => filesApiHits, { timeout: 10000 })
      .toBeGreaterThan(0);

    const repoDirBtn = fileTreeRoot.getByRole('button', { name: 'repo-a' }).first();
    await expect(repoDirBtn).toBeVisible({ timeout: 10000 });
    await repoDirBtn.scrollIntoViewIfNeeded();
    await repoDirBtn.click();

    await expect
      .poll(() => gitLogHits, { timeout: 10000 })
      .toBeGreaterThan(0);
    await expect(fileTreeRoot.getByText('提交日志')).toBeVisible({ timeout: 10000 });
    await expect(fileTreeRoot.getByTestId('git-log-current-branch')).toHaveText('· feature/mock-work', {
      timeout: 10000,
    });
    await expect(fileTreeRoot.getByText('commit abc')).toBeVisible({ timeout: 10000 });

    const expandRepo = fileTreeRoot.getByRole('button', { name: '展开或折叠子目录' }).first();
    await expandRepo.click();

    const srcDirBtn = fileTreeRoot.getByRole('button', { name: 'src' }).first();
    await expect(srcDirBtn).toBeVisible({ timeout: 10000 });
    await srcDirBtn.scrollIntoViewIfNeeded();
    await srcDirBtn.click();

    await expect
      .poll(() => gitLogHits, { timeout: 10000 })
      .toBeGreaterThanOrEqual(2);
    await expect(fileTreeRoot.getByTestId('git-log-current-branch')).toHaveCount(0);

    const expandSrc = fileTreeRoot.getByRole('button', { name: '展开或折叠子目录' }).nth(1);
    await expandSrc.click();

    const fileBtn = fileTreeRoot.getByRole('button', { name: 'a.py' }).first();
    await expect(fileBtn).toBeVisible({ timeout: 10000 });
    await fileBtn.click();

    await expect
      .poll(() => fileContentHits, { timeout: 10000 })
      .toBeGreaterThan(0);
    await expect(fileTreeRoot.getByText('print(\"tree preview\")')).toBeVisible({ timeout: 10000 });
  });

  /**
   * Django 仅转发「字符串路径」列表；单仓在子目录时与容器层一致，形如 goPractice/README.md（含仓目录名）。
   * 预览请求必须使用与列表相同的全路径，供容器 /api/layers/.../files/... 解析（见 trae resolveAbsolutePathForLayerListedFile）。
   */
  test('string 文件列表时点击文件：layer file content 的 path 含仓目录前缀', async ({ page }) => {
    const contentPaths = [];

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
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock Task',
            description: 'Mock',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
          }),
        });
        return;
      }

      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: [] }) });
        return;
      }

      if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            container_endpoint_registered: true,
            container_page_url: 'http://127.0.0.1:18080/ui/mock-token',
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
            bootstrap_layer_id: LAYER_ID,
            layers: [
              {
                layer_id: LAYER_ID,
                parent_layer_id: null,
                created_at: '2026-01-01T00:00:00Z',
                command: 'mock',
                job_status: 'completed',
              },
            ],
            jobs: [
              {
                id: JOB_ID,
                layer_id: LAYER_ID,
                status: 'completed',
                command_kind: 'trae',
                command: 'echo',
                created_at: '2026-01-01T00:00:01Z',
                output: '',
              },
            ],
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-job-execution-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            job: { id: JOB_ID, status: 'completed', layer_id: LAYER_ID, command: 'echo', output: '' },
            steps: { note: '', steps: [] },
            layer_changes: { layer_id: LAYER_ID, change_count: 0, changes: [], truncated: false },
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-files/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layer_id: LAYER_ID,
            truncated: false,
            files: ['goPractice/README.md', 'otherRepo/main.py'],
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-file-content/') && method === 'GET') {
        const u = new URL(url);
        const p = u.searchParams.get('path') || '';
        contentPaths.push(p);
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            path: p,
            content: '# mock readme',
            truncated: false,
          }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    await expect(ztreePanel).toBeVisible({ timeout: 30000 });
    await ztreePanel.locator('button.break-words.flex-1.min-w-0').nth(0).click();

    const fileTreeRoot = page.getByTestId('comment-layer-ztree-project-file-tree');
    await expect(fileTreeRoot).toBeVisible({ timeout: 15000 });

    const expandGo = fileTreeRoot.getByRole('button', { name: '展开或折叠子目录' }).first();
    await expandGo.click();
    const readmeBtn = fileTreeRoot.getByRole('button', { name: 'README.md' }).first();
    await expect(readmeBtn).toBeVisible({ timeout: 10000 });
    await readmeBtn.click();

    await expect
      .poll(() => contentPaths, { timeout: 10000 })
      .toEqual(expect.arrayContaining(['goPractice/README.md']));
    await expect(fileTreeRoot.getByText('# mock readme')).toBeVisible({ timeout: 10000 });
  });
});
