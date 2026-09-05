// @ts-check
/**
 * 回归：父仓内嵌套子仓有变更时，zTree「提交」按钮可点，
 * 变更面板显示「N 个文件变化」（暂存/未暂存）而非「N 个相对父层差异」。
 *
 * 对应：layerZtreeNodes.js normalizeLayerGitDirty + actionUiForJob
 *       layerChangesDirty.js formatLayerSubmitFileChangesCaption
 *
 * 纯 mock，无真实账号依赖。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { openLayerFilesChangesTab } from './openLayerFilesChangesTab.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = 'task_ztree_nested_dirty_mock';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

const LAYER_ID = '20260719_083000_nested';
const JOB_ID = 'job-nested-dirty-001';

test.describe('TaskDetail zTree 嵌套子仓 dirty 提交', () => {
  test('嵌套子仓有变更时提交按钮可用，变更面板显示文件变化而非父层差异', async ({ page }) => {
    test.setTimeout(120000);

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      // 任务详情
      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock nested git dirty',
            description: '',
            created_at: '2026-07-19T00:00:00Z',
            created_by: { username: 'mock' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            projects: [
              {
                stored_repo_address: 'https://gitlab.daydaymoney.com/example-user/parent-repo.git',
                target_branch: 'feature/nested-dirty',
              },
            ],
          }),
        });
        return;
      }

      // 容器 UI context
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

      // layer-graph：有 git、工作区 dirty=true（嵌套子仓有变更）
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
                created_at: '2026-07-19T08:30:00Z',
                command: '修改嵌套子仓库文件',
                job_status: 'completed',
                mind_state: 'idle_done',
                // git_worktree_dirty=true：表示有未提交变更（嵌套子仓变 dirty）
                git_worktree_dirty: true,
                git_remote: {
                  is_git: true,
                  ahead: 2,
                  no_upstream: false,
                  upstream: 'origin/feature/nested-dirty',
                  current_branch: 'feature/nested-dirty',
                },
                queue_depth: 0,
              },
            ],
            jobs: [
              {
                id: JOB_ID,
                layer_id: LAYER_ID,
                status: 'completed',
                command_kind: 'trae',
                command: '修改嵌套子仓库文件',
                created_at: '2026-07-19T08:30:30Z',
              },
            ],
          }),
        });
        return;
      }

      // job execution log：返回含未暂存 git 变更的文件列表（非 git_layer_diff_only）
      if (url.includes('container-job-execution-log') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            job_id: JOB_ID,
            layer_changes: {
              layer_id: LAYER_ID,
              same: false,
              truncated: false,
              change_count: 2,
              changes: [
                {
                  path: 'parent-repo/submodule/src/helper.py',
                  kind: 'modified',
                  // git_unstaged=true 表示工作区有未暂存变更 → 可提交
                  git_staged: false,
                  git_unstaged: true,
                  git_layer_diff_only: false,
                },
                {
                  path: 'parent-repo/submodule/src/new_feature.py',
                  kind: 'added',
                  git_staged: true,
                  git_unstaged: false,
                  git_layer_diff_only: false,
                },
              ],
            },
            steps: [],
            output: '',
          }),
        });
        return;
      }

      // 克隆日志
      if (url.includes('/cloud/compute/container-clone-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ layer_id: LAYER_ID, text: '' }),
        });
        return;
      }

      // 其余 API 返回空
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: '{}',
      });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 等待 zTree 面板可见
    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    await expect(ztreePanel).toBeVisible({ timeout: 30000 });

    // 选中层节点，展开变更面板
    const layerRow = ztreePanel.locator('button.break-words.flex-1.min-w-0').nth(0);
    await layerRow.click();

    const changesPanel = await openLayerFilesChangesTab(page);

    await expect(page.getByTestId('layer-files-tab-changes')).toContainText(/· 2/);
    await expect(changesPanel).toContainText(/个文件发生变动/);

    // 断言：提交按钮可用（nested dirty → submitDisabled 不为 true）
    const submitBtn = ztreePanel.getByTestId('layer-ztree-submit-btn');
    await expect(submitBtn).toBeVisible({ timeout: 15000 });
    await expect(submitBtn).toBeEnabled();

    // 断言提交按钮的 title 不包含「暂无未提交变更」
    await expect(submitBtn).not.toHaveAttribute('title', /暂无未提交变更/);
  });
});
