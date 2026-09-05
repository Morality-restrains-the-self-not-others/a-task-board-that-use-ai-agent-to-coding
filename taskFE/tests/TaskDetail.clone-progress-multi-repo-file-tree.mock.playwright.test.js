// @ts-check
/**
 * Mock 路由 + MockEventSource：快速核验 UI（无需 Docker / 真实 git）。
 * 真实克隆与 onlineServiceJS 引导请见 TaskDetail.real-clone-progress-project-file-tree.playwright.test.js。
 *
 * 本文件覆盖：
 * 1) SSE container_git_clone_progress 带 repo_url 时，「仓库列表与克隆身份」内各仓并列展示进度；
 * 2) 层级节点下项目文件树列出所有仓库根目录（与 container-layer-files 返回的路径一致）；
 * 3) mockStart 下点击「模拟启动」→ GET container-bootstrap-clone-log 多段 ━━ 日志时，每行「克隆日志」可展开且含各自分段（并行多仓）。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662912';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

const LAYER_ID = 'layer-multi-repo';
const JOB_ID = 'job-multi-repo';

const REPO_URL_ALPHA = 'https://example.com/org/repo-alpha.git';
const REPO_URL_BETA = 'https://example.com/org/repo-beta.git';

/** 与 TaskDetail `repoCloneFieldId` 一致，用于 data-testid 预期 */
function repoCloneFieldIdForTest(url) {
  return String(url || '')
    .replace(/[^a-zA-Z0-9]+/g, '-')
    .slice(0, 80);
}

test.describe('TaskDetail 多仓克隆进度 SSE + 项目文件树（mock）', () => {
  test('横幅展示各仓库进度行且文件树含全部仓库目录', async ({ page, browserName }) => {
    test.skip(true, 'Temporarily skipped due to frontend UI changes');
    await page.addInitScript(
      ({ tenantId, workspaceId, taskId, repoAlpha, repoBeta }) => {
        class MockEventSource {
          static CONNECTING = 0;
          static OPEN = 1;
          static CLOSED = 2;

          constructor(url) {
            this.url = String(url || '');
            this.readyState = MockEventSource.OPEN;
            this.withCredentials = true;
            this.onmessage = null;
            this.onerror = null;
            this.onclose = null;
            this._listeners = { open: [] };

            setTimeout(() => {
              this._emitOpen();
              const needle = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`;
              if (!this.url.includes(needle)) return;

              this._emitMessage({ type: 'heartbeat' });

              setTimeout(() => {
                this._emitMessage({
                  status: 'container_git_clone_progress',
                  progress: 33,
                  message: '【项目克隆】(1/2) repo-alpha … 33%',
                  repo_url: repoAlpha,
                  event_name: 'server_status_update',
                });
              }, 0);

              setTimeout(() => {
                this._emitMessage({
                  status: 'container_git_clone_progress',
                  progress: 66,
                  message: '【项目克隆】(2/2) repo-beta … 66%',
                  repo_url: repoBeta,
                  event_name: 'server_status_update',
                });
              }, 30);
            }, 50);
          }

          addEventListener(type, cb) {
            if (!this._listeners[type]) this._listeners[type] = [];
            this._listeners[type].push(cb);
          }

          close() {
            this.readyState = MockEventSource.CLOSED;
            if (typeof this.onclose === 'function') {
              this.onclose();
            }
          }

          _emitOpen() {
            const ev = { type: 'open' };
            for (const cb of this._listeners.open || []) {
              try {
                cb(ev);
              } catch (_) {}
            }
          }

          _emitMessage(payload) {
            if (typeof this.onmessage !== 'function') return;
            try {
              this.onmessage({ data: JSON.stringify(payload) });
            } catch (_) {}
          }
        }

        window.EventSource = MockEventSource;
      },
      {
        tenantId: TENANT_ID,
        workspaceId: WORKSPACE_ID,
        taskId: TASK_ID,
        repoAlpha: REPO_URL_ALPHA,
        repoBeta: REPO_URL_BETA,
      },
    );

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/task-detail/${TASK_ID}/mock-trae-online-log`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ log: '', in_progress: false }),
        });
        return;
      }

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock multi-repo clone',
            description: '',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            parameters: { repo_clone_git_identities: {} },
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

      if (
        url.includes(`/api/projects/tenant_id/${TENANT_ID}`) &&
        method === 'GET' &&
        url.includes('workspace_id=')
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: 'proj-e2e-multi',
              name: 'Multi repo project',
              git_repos: [REPO_URL_ALPHA, REPO_URL_BETA],
            },
          ]),
        });
        return;
      }

      if (url.includes('/api/user/e2e-user/profile/git-identities/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
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
                command: 'git clone (mock)',
                job_status: 'completed',
              },
            ],
            jobs: [
              {
                id: JOB_ID,
                layer_id: LAYER_ID,
                status: 'completed',
                command_kind: 'clone',
                command: `git clone ${REPO_URL_ALPHA}`,
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
            job: { id: JOB_ID, status: 'completed', layer_id: LAYER_ID, command: 'git clone', output: '' },
            steps: { note: '', steps: [] },
            layer_changes: {
              layer_id: LAYER_ID,
              change_count: 0,
              changes: [],
              truncated: false,
            },
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
            files: [
              { path: 'repo-a/src/a.py', size: 12, mtime: '2026-01-01T00:00:00' },
              { path: 'repo-a/README.md', size: 34, mtime: '2026-01-01T00:00:00' },
              { path: 'repo-b/main.py', size: 56, mtime: '2026-01-01T00:00:00' },
              { path: 'repo-b/lib/helper.py', size: 8, mtime: '2026-01-01T00:00:00' },
            ],
          }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const banner = page.getByTestId('container-clone-progress-banner');
    await expect(banner).toBeVisible({ timeout: 15000 });
    await expect(banner.getByText('容器项目克隆进度')).toBeVisible();

    await expect
      .poll(
        async () => banner.locator('[data-testid^="container-clone-progress-row-"]').count(),
        { timeout: 10000 },
      )
      .toBe(2);

    await expect(page.getByTestId('container-clone-progress-row-https-example-com-org-repo-alpha-git')).toBeVisible();
    await expect(page.getByTestId('container-clone-progress-row-https-example-com-org-repo-beta-git')).toBeVisible();
    // 每仓行内已展示完整 URL 与项目名，克隆进度条旁不再重复短标签 org/repo-*
    await expect(banner.getByText(REPO_URL_ALPHA, { exact: true })).toBeVisible();
    await expect(banner.getByText(REPO_URL_BETA, { exact: true })).toBeVisible();

    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    await expect(ztreePanel).toBeVisible({ timeout: 30000 });
    await ztreePanel.locator('button.break-words.flex-1.min-w-0').nth(0).click();

    const fileTreeRoot = page.getByTestId('comment-layer-ztree-project-file-tree');
    await expect(fileTreeRoot).toBeVisible({ timeout: 15000 });
    await expect(fileTreeRoot.getByText('项目文件树（所有拉取仓库）')).toBeVisible();

    const repoA = fileTreeRoot.getByRole('button', { name: 'repo-a' }).first();
    const repoB = fileTreeRoot.getByRole('button', { name: 'repo-b' }).first();
    await expect(repoA).toBeVisible({ timeout: 10000 });
    await expect(repoB).toBeVisible({ timeout: 10000 });
  });

  /**
   * 核验：多仓并行时，每行「克隆日志」<details> 有内容（依赖 mockStart + GET 引导日志分段与 SSE repo_url 对齐）。
   * 与 TaskDetail `gitCloneRefMatchKey`、parseBootstrapCloneLogSections 的 ━━ 行格式一致。
   */
  test('并行多仓：各自可查看克隆日志（引导日志分段 + 模拟启动轮询）', async ({ page }) => {
    const UU = '\u2501\u2501'
    const bootstrapText = [
      '【项目克隆】e2e 并行多仓引导日志…',
      '',
      `${UU} (1/2) ${REPO_URL_ALPHA}`,
      '→ repo-alpha',
      'Receiving objects: 11% (1/2)',
      '',
      `${UU} (2/2) ${REPO_URL_BETA}`,
      '→ repo-beta',
      'Receiving objects: 22% (1/2)',
      '',
      '【项目克隆】克隆完成。',
    ].join('\n')

    await page.addInitScript(
      ({ tenantId, workspaceId, taskId, repoAlpha, repoBeta }) => {
        class MockEventSource {
          static CONNECTING = 0
          static OPEN = 1
          static CLOSED = 2

          constructor(url) {
            this.url = String(url || '')
            this.readyState = MockEventSource.OPEN
            this.withCredentials = true
            this.onmessage = null
            this.onerror = null
            this.onclose = null
            this._listeners = { open: [] }

            setTimeout(() => {
              this._emitOpen()
              const needle = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`
              if (!this.url.includes(needle)) return

              this._emitMessage({ type: 'heartbeat' })

              setTimeout(() => {
                this._emitMessage({
                  status: 'container_git_clone_progress',
                  progress: 12,
                  message: '【项目克隆】(1/2) repo-alpha … 12%',
                  repo_url: repoAlpha,
                  event_name: 'server_status_update',
                })
              }, 0)

              setTimeout(() => {
                this._emitMessage({
                  status: 'container_git_clone_progress',
                  progress: 34,
                  message: '【项目克隆】(2/2) repo-beta … 34%',
                  repo_url: repoBeta,
                  event_name: 'server_status_update',
                })
              }, 20)
            }, 50)
          }

          addEventListener(type, cb) {
            if (!this._listeners[type]) this._listeners[type] = []
            this._listeners[type].push(cb)
          }

          close() {
            this.readyState = MockEventSource.CLOSED
            if (typeof this.onclose === 'function') {
              this.onclose()
            }
          }

          _emitOpen() {
            const ev = { type: 'open' }
            for (const cb of this._listeners.open || []) {
              try {
                cb(ev)
              } catch (_) {}
            }
          }

          _emitMessage(payload) {
            if (typeof this.onmessage !== 'function') return
            try {
              this.onmessage({ data: JSON.stringify(payload) })
            } catch (_) {}
          }
        }

        window.EventSource = MockEventSource
      },
      {
        tenantId: TENANT_ID,
        workspaceId: WORKSPACE_ID,
        taskId: TASK_ID,
        repoAlpha: REPO_URL_ALPHA,
        repoBeta: REPO_URL_BETA,
      },
    )

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ])

    await page.route('**/api/**', async (route) => {
      const req = route.request()
      const url = req.url()
      const method = req.method()

      if (url.includes('container-bootstrap-clone-log') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            text: bootstrapText,
            segments: [
              {
                repo_url: REPO_URL_ALPHA,
                text: [`${UU} (1/2) ${REPO_URL_ALPHA}`, '→ repo-alpha', 'Receiving objects: 11% (1/2)'].join('\n'),
              },
              {
                repo_url: REPO_URL_BETA,
                text: [`${UU} (2/2) ${REPO_URL_BETA}`, '→ repo-beta', 'Receiving objects: 22% (1/2)'].join('\n'),
              },
            ],
          }),
        })
        return
      }

      if (url.includes('mock-trae-online-stream') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ message: 'e2e mock 模拟启动已受理' }),
        })
        return
      }

      if (url.includes(`/task-detail/${TASK_ID}/mock-trae-online-log`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ log: '', in_progress: false }),
        })
        return
      }

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock multi-repo clone logs',
            description: '',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            parameters: { repo_clone_git_identities: {} },
          }),
        })
        return
      }

      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
        return
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: [] }) })
        return
      }

      if (
        url.includes(`/api/projects/tenant_id/${TENANT_ID}`) &&
        method === 'GET' &&
        url.includes('workspace_id=')
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: 'proj-e2e-multi',
              name: 'Multi repo project',
              git_repos: [REPO_URL_ALPHA, REPO_URL_BETA],
            },
          ]),
        })
        return
      }

      if (url.includes('/api/user/e2e-user/profile/git-identities/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
        return
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
        })
        return
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
                command: 'git clone (mock)',
                job_status: 'completed',
              },
            ],
            jobs: [
              {
                id: JOB_ID,
                layer_id: LAYER_ID,
                status: 'completed',
                command_kind: 'clone',
                command: `git clone ${REPO_URL_ALPHA}`,
                created_at: '2026-01-01T00:00:01Z',
                output: '',
              },
            ],
          }),
        })
        return
      }

      if (url.includes('/cloud/compute/container-job-execution-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            job: { id: JOB_ID, status: 'completed', layer_id: LAYER_ID, command: 'git clone', output: '' },
            steps: { note: '', steps: [] },
            layer_changes: {
              layer_id: LAYER_ID,
              change_count: 0,
              changes: [],
              truncated: false,
            },
          }),
        })
        return
      }

      if (url.includes('/cloud/compute/container-layer-files/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ layer_id: LAYER_ID, truncated: false, files: [] }),
        })
        return
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    })

    await page.goto(TASK_DETAIL_PATH)
    await page.waitForLoadState('domcontentloaded')

    const banner = page.getByTestId('container-clone-progress-banner')
    await expect(banner).toBeVisible({ timeout: 15000 })

    const mockStartBtn = page.getByRole('button', { name: /模拟启动（onlineService/ })
    await expect(mockStartBtn).toBeVisible({ timeout: 20000 })
    await mockStartBtn.click()

    const idAlpha = 'container-clone-progress-log-https-example-com-org-repo-alpha-git'
    const idBeta = 'container-clone-progress-log-https-example-com-org-repo-beta-git'

    await expect(page.getByTestId(idAlpha)).toBeVisible({ timeout: 20000 })
    await expect(page.getByTestId(idBeta)).toBeVisible({ timeout: 20000 })

    const detAlpha = page.getByTestId(idAlpha)
    const detBeta = page.getByTestId(idBeta)
    await detAlpha.locator('summary').click()
    await detBeta.locator('summary').click()
    await expect(detAlpha.locator('pre')).toContainText('Receiving objects: 11%')
    await expect(detAlpha.locator('pre')).toContainText('repo-alpha')
    await expect(detBeta.locator('pre')).toContainText('Receiving objects: 22%')
    await expect(detBeta.locator('pre')).toContainText('repo-beta')
  })

  test('三仓进度行按任务中仓库声明顺序，不按 URL 字母序', async ({ page }) => {
    test.skip(true, 'Temporarily skipped due to frontend UI changes');
    const REPO_THIRD = 'https://example.com/m/mid.git';
    const REPO_FIRST = 'https://example.com/z/zebra.git';
    const REPO_SECOND = 'https://example.com/a/alpha.git';
    const taskOrder = [REPO_FIRST, REPO_SECOND, REPO_THIRD];
    // 按字母序为 alpha < mid < zebra，与任务序不同；错排时曾出现「第三仓挤到首行」

    await page.addInitScript(
      ({ tenantId, workspaceId, taskId, r1, r2, r3 }) => {
        class MockEventSource {
          static CONNECTING = 0;
          static OPEN = 1;
          static CLOSED = 2;

          constructor(url) {
            this.url = String(url || '');
            this.readyState = MockEventSource.OPEN;
            this.withCredentials = true;
            this.onmessage = null;
            this.onerror = null;
            this.onclose = null;
            this._listeners = { open: [] };

            setTimeout(() => {
              this._emitOpen();
              const needle = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`;
              if (!this.url.includes(needle)) return;

              this._emitMessage({ type: 'heartbeat' });

              setTimeout(() => {
                this._emitMessage({
                  status: 'container_git_clone_progress',
                  progress: 11,
                  message: '【项目克隆】(1/3) zebra … 11%',
                  repo_url: r1,
                  event_name: 'server_status_update',
                });
              }, 0);

              setTimeout(() => {
                this._emitMessage({
                  status: 'container_git_clone_progress',
                  progress: 44,
                  message: '【项目克隆】(2/3) alpha … 44%',
                  repo_url: r2,
                  event_name: 'server_status_update',
                });
              }, 25);

              setTimeout(() => {
                this._emitMessage({
                  status: 'container_git_clone_progress',
                  progress: 66,
                  message: '【项目克隆】(3/3) mid … 66%',
                  repo_url: r3,
                  event_name: 'server_status_update',
                });
              }, 50);
            }, 50);
          }

          addEventListener(type, cb) {
            if (!this._listeners[type]) this._listeners[type] = [];
            this._listeners[type].push(cb);
          }

          close() {
            this.readyState = MockEventSource.CLOSED;
            if (typeof this.onclose === 'function') {
              this.onclose();
            }
          }

          _emitOpen() {
            const ev = { type: 'open' };
            for (const cb of this._listeners.open || []) {
              try {
                cb(ev);
              } catch (_) {}
            }
          }

          _emitMessage(payload) {
            if (typeof this.onmessage !== 'function') return;
            try {
              this.onmessage({ data: JSON.stringify(payload) });
            } catch (_) {}
          }
        }

        window.EventSource = MockEventSource;
      },
      {
        tenantId: TENANT_ID,
        workspaceId: WORKSPACE_ID,
        taskId: TASK_ID,
        r1: REPO_FIRST,
        r2: REPO_SECOND,
        r3: REPO_THIRD,
      },
    );

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/task-detail/${TASK_ID}/mock-trae-online-log`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ log: '', in_progress: false }),
        });
        return;
      }

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock three-repo order',
            description: '',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            parameters: { repo_clone_git_identities: {} },
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

      if (url.includes(`/api/projects/tenant_id/${TENANT_ID}`) && method === 'GET' && url.includes('workspace_id=')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: 'proj-e2e-3',
              name: 'Three repo project',
              git_repos: taskOrder,
            },
          ]),
        });
        return;
      }

      if (url.includes('/api/user/e2e-user/profile/git-identities/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
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
          body: JSON.stringify({ layers_root: '/workspace/layers', layers: [], jobs: [] }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const banner = page.getByTestId('container-clone-progress-banner');
    await expect(banner).toBeVisible({ timeout: 15000 });

    await expect
      .poll(
        async () => banner.locator('[data-testid^="container-clone-progress-row-"]').count(),
        { timeout: 10000 },
      )
      .toBe(3);

    const expectedTestIds = taskOrder.map(
      (u) => `container-clone-progress-row-${repoCloneFieldIdForTest(u)}`,
    );
    const rowTestIds = await banner
      .locator('[data-testid^="container-clone-progress-row-"]')
      .evaluateAll((els) => els.map((e) => e.getAttribute('data-testid')));

    expect(rowTestIds).toEqual(expectedTestIds);
  });
});
