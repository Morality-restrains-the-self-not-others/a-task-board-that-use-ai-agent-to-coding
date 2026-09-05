// @ts-check
/**
 * 回归：容器心跳已连通、层图仅有引导空层锚点、引导克隆因 Git 授权不齐失败时，
 * 任务关联区须在 `comment-layer-ztree-panel` 内展示可行动失败文案，
 * 不得把 empty/bootstrap_pending 当成已就绪而无限展示「正在准备可写层」。
 *
 * 运行：
 *   npx playwright test tests/TaskDetail.writable-layer-credentials-stuck.playwright.test.js --config=playwright.config.headless.js
 */
import { test, expect } from '@playwright/test'
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js'
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js'

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID

const TASK_ID = '877902757669924864'
const COMMENT_ID = 'comment-writable-layer-stuck'
const MOCK_CSC_ID = 'csc-writable-layer-stuck'
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
)

const BOOTSTRAP_FAIL_TEXT = [
  '【项目克隆】引导失败（phase=task_detail_or_credentials code=REPO_CLONE_CREDENTIALS_INCOMPLETE）。',
  'repo-clone-credentials 未返回完整 repo_clone_credentials；请在任务详情为全部仓库绑定 Git 授权后重试。 缺失仓库(1): https://github.com/ruandao/somanyad detail=repo clone credentials incomplete',
].join('\n')

function taskDetailMock() {
  return {
    id: TASK_ID,
    title: 'Writable layer credentials stuck',
    description: '',
    created_at: '2026-08-20T00:00:00Z',
    created_by: { username: 'mock-user' },
    comments: [
      {
        id: COMMENT_ID,
        content: '触发容器引导克隆',
        created_at: '2026-08-20T00:00:00Z',
        created_by: { username: 'mock-user' },
      },
    ],
    ai_comments: [],
    assignees: [],
    workspace_id: WORKSPACE_ID,
  }
}

test.describe('TaskDetail 可写层卡住 — 克隆凭证不齐', () => {
  test('心跳已连通且层图为空时，clone-log 凭证失败展示错误而非无限等待', async ({ page }) => {
    test.setTimeout(120000)

    await page.addInitScript(() => {
      window.__TASK2APP_API_BASE_URL__ = ''
    })

    await page.addInitScript(
      ({ tenantId, workspaceId, taskId }) => {
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
            window.__mockEventSource = this
            setTimeout(() => this._emitOpen(), 20)
          }

          addEventListener(type, cb) {
            if (!this._listeners[type]) this._listeners[type] = []
            this._listeners[type].push(cb)
          }

          close() {
            this.readyState = MockEventSource.CLOSED
            if (typeof this.onclose === 'function') this.onclose()
          }

          _emitOpen() {
            const ev = { type: 'open' }
            for (const cb of this._listeners.open || []) {
              try {
                cb(ev)
              } catch (_) {}
            }
          }

          emit(payload) {
            const expected = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`
            if (!this.url.includes(expected)) return
            if (typeof this.onmessage === 'function') {
              this.onmessage({ data: JSON.stringify(payload) })
            }
          }
        }

        window.EventSource = MockEventSource
        window.__emitStartupSse = (payload) => {
          window.__mockEventSource?.emit(payload)
        }
      },
      { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID },
    )

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ])

    await page.route('**/api/**', async (route) => {
      const req = route.request()
      const url = req.url()
      const method = req.method()

      if (
        url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) &&
        method === 'GET'
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(taskDetailMock()),
        })
        return
      }

      if (url.includes(`/api/tasks/${TASK_ID}/comments/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(taskDetailMock().comments),
        })
        return
      }

      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
        return
      }

      if (
        url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) &&
        method === 'GET'
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ columns: [] }),
        })
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
            bootstrap_layer_id: '',
            layers: [
              {
                layer_id: '20260827_074114_cdea43',
                created_at: '2026-08-27T07:41:14Z',
                command: null,
                job_status: null,
                mind_state: 'pending',
                git_worktree_dirty: false,
                meta_kind: 'empty',
                bootstrap_pending: true,
              },
            ],
            jobs: [],
          }),
        })
        return
      }

      if (url.includes('container-bootstrap-clone-log') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            error_code: 'REPO_CLONE_CREDENTIALS_INCOMPLETE',
            text: BOOTSTRAP_FAIL_TEXT,
          }),
        })
        return
      }

      if (url.includes('/cloud/compute/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            runtime_status: 'Running',
            instance_id: 'i-mock',
            csc_id: MOCK_CSC_ID,
            comment_id: COMMENT_ID,
            platform: 'aliyun',
            region: 'cn-hangzhou',
            message: '运行中',
          }),
        })
        return
      }

      if (url.includes('comment-container-bindings') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            bindings: [
              {
                comment_id: COMMENT_ID,
                status: 'running',
                container_name: 'mock-container-writable-stuck',
                csc_id: MOCK_CSC_ID,
                start_trace_id: 'trace-writable-stuck-1',
                logs: [],
              },
            ],
          }),
        })
        return
      }

      if (url.includes('comment-container-bindings') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ ok: true }),
        })
        return
      }

      if (url.includes('/ai-comment/task-detail/') && url.includes('/ai-comments/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
        return
      }

      if (url.includes('/container-agent-comments/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
        return
      }

      if (url.includes('/comments/') && method === 'PATCH') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
        return
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    })

    await page.goto(TASK_DETAIL_PATH)
    await page.waitForLoadState('domcontentloaded')

    const execDetails = page.getByTestId('comment-execution-details')
    await expect(execDetails).toBeVisible({ timeout: 45000 })
    const openAttr = await execDetails.getAttribute('open')
    if (openAttr === null) {
      await page.getByTestId('comment-execution-details-summary').click()
    }

    const ztreeTab = page.getByTestId('comment-execution-tab-ztree')
    await expect(ztreeTab).toBeVisible({ timeout: 15000 })
    await ztreeTab.click()

    await page.waitForFunction(() => typeof window.__emitStartupSse === 'function', null, {
      timeout: 30000,
    })
    await page.evaluate(
      ({ commentId }) => {
        window.__emitStartupSse?.({
          event_name: 'container_heartbeat',
          status: 'ok',
          bidirectional_ok: true,
          uplink_ok: true,
          downlink_ok: true,
          probe_ok: true,
          comment_id: commentId,
          message: 'hb-ok',
        })
      },
      { commentId: COMMENT_ID },
    )

    const loadingErr = page.getByTestId('comment-layer-ztree-loading-error')
    await expect(loadingErr).toBeVisible({ timeout: 20000 })
    await expect(loadingErr).toContainText('Git 授权未齐')
    await expect(loadingErr).not.toContainText('等待可写层就绪')
    const panel = page.getByTestId('comment-layer-ztree-panel')
    await expect(panel).toBeVisible()
    await expect(panel.getByTestId('comment-layer-ztree-loading-error')).toBeVisible()
    await expect(page.locator('[data-testid="comment-layer-ztree-loading"] .animate-spin')).toHaveCount(0)
    await expect(page.getByText('正在准备可写层')).toHaveCount(0)
  })
})
