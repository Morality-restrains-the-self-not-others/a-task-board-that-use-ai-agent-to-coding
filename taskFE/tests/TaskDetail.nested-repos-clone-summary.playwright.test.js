// @ts-check
/**
 * OPT-20260719-009: 子仓库克隆状态在进度 Map 清空后凭引导日志显示已完成。
 *
 * 测试策略：
 * - Mock 容器 bootstrap-clone-log API 返回含「已移入」「克隆完成」的分段日志
 * - Mock progress Map 为空（不推送含 repo_url 的 SSE progress 条目）
 * - Mock nested-git-repos API 返回两个子仓库
 * - SSR clone progress SSE 推送 container_git_clone_progress（bootstrap_log_text 含分段）
 * - 断言 summary 显示非 0/N、首仓 badge 显示「已完成」
 *
 * 运行：
 *   npx playwright test tests/TaskDetail.nested-repos-clone-summary.playwright.test.js --config=playwright.verify.config.js
 */
import { test, expect } from '@playwright/test'
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js'
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js'

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID

const TASK_ID = '830423831930662912'
const PARENT_REPO_URL = 'https://gitlab.daydaymoney.com/g/parent.git'
const CLONE_ALPHA = 'sub-alpha'
const NESTED_URL_ALPHA = 'https://gitlab.daydaymoney.com/g/sub-alpha.git'
const CLONE_BETA = 'sub-beta'
const NESTED_URL_BETA = 'https://gitlab.daydaymoney.com/g/sub-beta.git'

const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
)

// 引导日志格式：━━ 分段 + [bootstrap-clone] 已移入（触发 done 状态的关键标志）
const BOOTSTRAP_LOG_TEXT = [
  '【项目克隆】e2e 子仓库引导日志…',
  '',
  '━━ (1/2) https://gitlab.daydaymoney.com/g/sub-alpha.git',
  '→ sub-alpha',
  'Receiving objects: 100% (1/1)',
  '[bootstrap-clone] 已移入 sub-alpha',
  '',
  '━━ (2/2) https://gitlab.daydaymoney.com/g/sub-beta.git',
  '→ sub-beta',
  'Receiving objects: 100% (1/1)',
  '[bootstrap-clone] 已移入 sub-beta',
  '',
  '【项目克隆】克隆完成。',
].join('\n')

const BOOTSTRAP_LOG_SEGMENTS = [
  {
    repo_url: NESTED_URL_ALPHA,
    text: [
      '━━ (1/2) https://gitlab.daydaymoney.com/g/sub-alpha.git',
      '→ sub-alpha',
      'Receiving objects: 100% (1/1)',
      '[bootstrap-clone] 已移入 sub-alpha',
    ].join('\n'),
  },
  {
    repo_url: NESTED_URL_BETA,
    text: [
      '━━ (2/2) https://gitlab.daydaymoney.com/g/sub-beta.git',
      '→ sub-beta',
      'Receiving objects: 100% (1/1)',
      '[bootstrap-clone] 已移入 sub-beta',
    ].join('\n'),
  },
]

test.describe('TaskDetail 子仓库克隆状态摘要（进度 Map 清空后凭引导日志）', () => {
  test('引导日志含已移入 + 克隆完成时，summary 显示完成计数，badge 为已完成', async ({ page }) => {
    /** 捕获 repo-reclone POST 请求（避免真正的 API 调用） */
    await page.addInitScript(
      ({ tenantId, workspaceId, taskId, bootstrapText, bootstrapSegments }) => {
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
              // 只处理预期的 SSE URL
              const expected = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`
              if (!this.url.includes(expected)) return

              // 先发心跳（保持连接活跃）
              if (typeof this.onmessage === 'function') {
                this.onmessage({ data: JSON.stringify({ type: 'heartbeat' }) })
              }

              // 延迟推送 container_git_clone_progress，携带 bootstrap_log_text/segments
              // 模拟 progress 到达 100% → 全局完成清空 progress Map；
              // 但 bootstrap_log_text 仍含有[已移入]供子仓库推导 done 状态
              setTimeout(() => {
                if (typeof this.onmessage === 'function') {
                  this.onmessage({
                    data: JSON.stringify({
                      status: 'container_git_clone_progress',
                      progress: 100,
                      message: '【项目克隆】仓库克隆已完成',
                      bootstrap_log_text: bootstrapText,
                      bootstrap_log_segments: bootstrapSegments,
                    }),
                  })
                }
              }, 100)
            }, 50)
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
              try { cb(ev) } catch (_) {}
            }
          }
        }

        window.EventSource = MockEventSource
      },
      {
        tenantId: TENANT_ID,
        workspaceId: WORKSPACE_ID,
        taskId: TASK_ID,
        bootstrapText: BOOTSTRAP_LOG_TEXT,
        bootstrapSegments: BOOTSTRAP_LOG_SEGMENTS,
      },
    )

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ])

    // ── 拦截所有 API ──
    await page.route('**/api/**', async (route) => {
      const req = route.request()
      const url = req.url()
      const method = req.method()

      // 任务详情 GET
      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock nested repos clone summary',
            description: '',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            container_agent_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            parameters: { repo_clone_git_identities: {} },
            projects: [
              {
                project_id: 'proj-1',
                repo_index: 0,
                base_branch: 'main',
                repo_url: PARENT_REPO_URL,
              },
            ],
          }),
        })
        return
      }

      // 项目列表
      if (url.includes(`/api/projects/tenant_id/${TENANT_ID}`) && method === 'GET' && url.includes('workspace_id=')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: 'proj-1',
              name: 'Mock Project',
              git_repos: [PARENT_REPO_URL],
            },
          ]),
        })
        return
      }

      // 嵌套子仓库 API ── 返回两个子仓库，progress Map 为空，依赖引导日志推导状态
      if (url.includes('/cloud/nested-git-repos/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            repos: [
              { path: CLONE_ALPHA, url: NESTED_URL_ALPHA, source: 'gitmodules' },
              { path: CLONE_BETA, url: NESTED_URL_BETA, source: 'gitmodules' },
            ],
          }),
        })
        return
      }

      // 容器 bootstrap-clone-log（mockStart 轮询用）
      if (url.includes('/cloud/compute/container-bootstrap-clone-log/') && method === 'GET') {
        // 非 critical，可返回空或 bootstrap 文本
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            text: BOOTSTRAP_LOG_TEXT,
            segments: BOOTSTRAP_LOG_SEGMENTS,
          }),
        })
        return
      }

      // container-task-ui-context
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

      // container-layer-graph
      if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layers_root: '/workspace/layers',
            bootstrap_layer_id: 'layer-1',
            layers: [],
            jobs: [],
          }),
        })
        return
      }

      // mock-trae-online-log
      if (url.includes('/task-detail/') && url.includes('/mock-trae-online-log') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ log: '', in_progress: false }),
        })
        return
      }

      // 协作成员
      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
        return
      }

      // 进度列
      if (url.includes('/progress-system/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ columns: [] }),
        })
        return
      }

      // git-identities
      if (url.includes('/api/user/e2e-user/profile/git-identities/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ identities: [] }) })
        return
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: '{}',
      })
    })

    // ── 导航到任务详情页 ──
    await page.goto(TASK_DETAIL_PATH, { waitUntil: 'domcontentloaded' })

    // 等待子仓库克隆状态区出现
    const statusSection = page.getByTestId('task-nested-repos-clone-status')
    await expect(statusSection).toBeVisible({ timeout: 30000 })

    // 等待 summary 出现
    const summary = page.getByTestId('task-nested-repos-clone-summary')
    await expect(summary).toBeVisible({ timeout: 15000 })

    // 断言 summary 显示非零完成计数（期望 2/2 已完成）
    await expect(summary).toContainText('已完成')
    await expect(summary).toContainText('2/2')

    // 断言首行 badge 显示「已完成」
    const firstRowBadge = page.locator('[data-testid="task-nested-repos-clone-status-0"]')
    await expect(firstRowBadge).toBeVisible({ timeout: 10000 })
    await expect(firstRowBadge).toHaveText('已完成')
  })
})
