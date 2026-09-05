// @ts-check
/**
 * OPT-20260717-029: 核验任务详情页嵌套子仓库「重新克隆」按钮发起的 API 请求。
 *
 * 测试策略：
 * - 使用 mock 模式，拦截全部 API 请求
 * - 模拟任务详情返回含关联项目与嵌套子仓库数据
 * - 通过 MockEventSource 模拟 SSE 推送 clone progress → 使 reClone 按钮可见
 * - 点击重新克隆后断言 POST .../cloud/repo-reclone/ 请求体含 parent_repo_url 和 clone_alias
 *
 * 运行：
 *   npx playwright test tests/TaskDetail.nested-repos-reclone.playwright.test.js --config=playwright.verify.config.js
 */
import { test, expect } from '@playwright/test'
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js'
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js'

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID
const WORKSPACE_ID =
  process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID
const TASK_ID = '830423831930662912'
const PARENT_REPO_URL = 'https://gitlab.daydaymoney.com/g/parent.git'
const CLONE_ALIAS = 'nested-submodule'
const NESTED_REPO_URL = 'https://gitlab.daydaymoney.com/g/nested-sub.git'

const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
)

test.describe('TaskDetail 嵌套子仓库重新克隆', () => {
  test('点击重新克隆按钮应发起 POST 到 repo-reclone 并携带 parent_repo_url 与 clone_alias', async ({
    page,
  }) => {
    /** 捕获所有 repo-reclone POST 请求 */
    const recloneRequests = []

    // ── 设置 userId cookie ──
    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ])

    // ── 注入 MockEventSource ──
    // 发送 container_git_clone_progress 事件使克隆进度数据映射至失败态
    await page.addInitScript(
      ({ tenantId, workspaceId, taskId, nestedRepoUrl, cloneAlias }) => {
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
              const expected = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`
              if (!this.url.includes(expected)) return
              // 推送克隆失败进度，使组件状态变为 error → 显示重新克隆按钮
              if (typeof this.onmessage === 'function') {
                this.onmessage({
                  data: JSON.stringify({
                    status: 'container_git_clone_progress',
                    repo_url: nestedRepoUrl,
                    progress: 0,
                    message: `【项目克隆】(1/1) 失败 ${cloneAlias}: fatal: repository not found`,
                  }),
                })
              }
            }, 120)
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
        nestedRepoUrl: NESTED_REPO_URL,
        cloneAlias: CLONE_ALIAS,
      },
    )

    // ── 拦截所有 API 路由 ──
    await page.route('**/api/**', async (route) => {
      const req = route.request()
      const url = req.url()
      const method = req.method()

      // 任务详情 GET
      if (
        url.includes(`/todos/${TASK_ID}/`) &&
        method === 'GET' &&
        !url.includes('repo-clone-git-identities')
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock nested repos reclone task',
            description: 'Test nested repo reclone flow',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            parameters: {},
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
              id: 'proj-1',
              name: 'Mock Project',
              git_repos: [PARENT_REPO_URL],
            },
          ]),
        })
        return
      }

      // 嵌套子仓库 API（nested-git-repos）─ 模拟一个失败状态的子仓库
      if (url.includes('/cloud/nested-git-repos/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            repos: [
              {
                path: CLONE_ALIAS,
                url: NESTED_REPO_URL,
                source: 'gitmodules',
              },
            ],
          }),
        })
        return
      }

      // container-task-ui-context（容器就绪，使重新克隆按钮可用）
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

      // repo-reclone POST ── 捕获请求
      if (url.includes('/cloud/repo-reclone/') && method === 'POST') {
        recloneRequests.push(req)
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ ok: true, result: { status: 'ok' } }),
        })
        return
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: '{}',
      })
    })

    // ── 导航到任务详情 ──
    await page.goto(TASK_DETAIL_PATH, { waitUntil: 'domcontentloaded' })

    // 等待嵌套子仓库状态区出现
    const statusSection = page.getByTestId('task-nested-repos-clone-status')
    await expect(statusSection).toBeVisible({ timeout: 30000 })

    // 验证子仓库汇总展示
    const summary = page.getByTestId('task-nested-repos-clone-summary')
    await expect(summary).toBeVisible({ timeout: 15000 })

    // 等待重新克隆按钮出现（需要克隆进度数据 → error 态 → 按钮可见）
    const recloneBtn = page.getByTestId('task-nested-repos-clone-reclone')
    await expect(recloneBtn).toBeVisible({ timeout: 20000 })

    // 点击重新克隆按钮
    await recloneBtn.click()

    // 断言已捕获 repo-reclone POST 请求
    await expect
      .poll(() => recloneRequests.length, { timeout: 10000, message: '等待 repo-reclone POST' })
      .toBeGreaterThanOrEqual(1)

    // 断言请求体包含 parent_repo_url 和 clone_alias
    const postedBody = recloneRequests[recloneRequests.length - 1].postDataJSON()
    expect(postedBody).toBeDefined()
    expect(postedBody.parent_repo_url).toBe(PARENT_REPO_URL)
    expect(postedBody.clone_alias).toBe(CLONE_ALIAS)
  })
})
