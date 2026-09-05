// @ts-check
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { openLayerFilesChangesTab } from './openLayerFilesChangesTab.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '837978129569890304';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`
);

const LAYER_ID = 'layer-1';
const JOB_ID = 'job-1';

test.describe('TaskDetail 提交文件变动后应刷新项目文件树提交日志', () => {
  test('文件变动列表提交后，项目文件树应重新拉取并显示最新提交日志', async ({ page }) => {
    let gitLogApiCallCount = 0;
    let filesApiCallCount = 0;
    let commitCount = 1;

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
              changes: [{ path: 'somanyad/hello.txt', kind: 'added', git_staged: true }],
              truncated: false,
            },
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-files/') && method === 'GET') {
        filesApiCallCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layer_id: LAYER_ID,
            truncated: false,
            files: [
              { path: 'somanyad/hello.txt', size: 12, mtime: '2026-01-01T00:00:00' },
            ],
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-git-log/') && method === 'GET') {
        gitLogApiCallCount += 1;
        const commits = [];
        for (let i = 1; i <= commitCount; i++) {
          commits.push({
            hash: `commit-${i}`,
            short: `c${i}`,
            subject: `commit-${i}: test commit ${i}`,
            author: 'e2e-user',
            email: 'e2e@example.com',
            date: '2026-01-01',
          });
        }
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layer_id: LAYER_ID,
            path: 'somanyad',
            repo: 'somanyad',
            status: 'ok',
            commits: commits,
            text: commits.map(c => `commit ${c.hash}\nAuthor: ${c.author}\n\n    ${c.subject}`).join('\n\n'),
            truncated: false,
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-git-commit/') && method === 'POST') {
        commitCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ok: true,
            summary: '提交成功',
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-create/') && method === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            ok: true,
            layer_id: 'layer-2',
            parent_layer_id: LAYER_ID,
            created_at: '2026-01-01T00:00:02Z',
          }),
        });
        return;
      }

      if (url.includes('/profile/git-identities') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            identities: [
              { id: 'identity-1', label: 'Default Identity', git_user_name: 'test-user', git_user_email: 'test@example.com', is_default: true },
            ],
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
    await expect(fileTreeRoot).toBeAttached({ timeout: 15000 });

    const layerChangesPanel = await openLayerFilesChangesTab(page);

    await layerChangesPanel.locator('details').first().click();

    const commitButton = layerChangesPanel.getByTestId('task-detail-layer-changes-commit-staged');
    await expect(commitButton).toBeVisible({ timeout: 10000 });

    const commitMessageInput = layerChangesPanel.getByTestId('task-detail-layer-changes-commit-message');
    await expect(commitMessageInput).toBeVisible({ timeout: 10000 });
    await commitMessageInput.fill('test commit message');

    await expect(commitButton).toBeEnabled({ timeout: 10000 });

    const initialGitLogCallCount = gitLogApiCallCount;

    const alertPromise = page.waitForEvent('dialog');

    await commitButton.click();

    const alert = await alertPromise;
    expect(alert.message()).toBe('提交成功');
    await alert.accept();

    await expect
      .poll(() => filesApiCallCount, { timeout: 10000 })
      .toBeGreaterThan(0);

    const initialFilesCallCount = filesApiCallCount;

    await expect
      .poll(() => filesApiCallCount, { timeout: 10000 })
      .toBeGreaterThan(initialFilesCallCount);

    await page.getByTestId('layer-files-tab-tree').click();
    await expect(fileTreeRoot).toBeVisible({ timeout: 10000 });

    const somanyadDirBtn = fileTreeRoot.getByRole('button', { name: 'somanyad' }).first();
    await expect(somanyadDirBtn).toBeVisible({ timeout: 10000 });
    await somanyadDirBtn.click();

    await expect
      .poll(() => gitLogApiCallCount, { timeout: 10000 })
      .toBeGreaterThan(0);

    await expect(fileTreeRoot.getByText('commit-2')).toBeVisible({ timeout: 10000 });
  });
});