// @ts-check
/**
 * OPT-20260720-025: 默认展开辅助信息；点击收起后正文不可见。
 * 排队调度入口已迁至评论「自动执行」卡（F-102），不再出现在辅助信息内。
 *
 * 测试策略：
 * - Mock 任务详情 API，任务为顶层（无 parent_task），使排队调度面板可见
 * - localStorage 中不设折叠标志 → useTaskAuxInfoExpanded 默认 true
 * - 断言 aux-info 面板可见、正文可见、排队调度入口可见
 * - 点击切换按钮收起 → 断言正文不可见
 *
 * 运行：
 *   npx playwright test tests/TaskDetail.aux-info-collapse.playwright.test.js --config=playwright.verify.config.js
 */
import { test, expect } from '@playwright/test'
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js'
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js'

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID

const TASK_ID = '830423831930662912'
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
)

test.describe('TaskDetail 辅助信息折叠', () => {
  test('默认展开、含排队调度入口；收起后正文隐藏', async ({ page }) => {
    // 清空 localStorage 确保 auxInfoExpanded 默认为 true
    await page.addInitScript(() => {
      localStorage.removeItem('task-detail-aux-info-expanded')
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
            setTimeout(() => this._emitOpen(), 50)
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

      // 任务详情 ── 顶层任务（无 parent_task）确保排队调度展示
      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Aux info collapse test task',
            description: 'Test description for aux info collapse',
            priority: 1,
            created_at: '2026-06-01T10:00:00Z',
            created_by: { id: 'u1', username: 'test-user' },
            comments: [],
            ai_comments: [],
            container_agent_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            // 人读序号：断言 task-aux-info-display-no 展示 #3
            workspace_seq: 3,
            // 无 parent_task => isTopLevel = true => 排队调度面板可见
            parameters: {},
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

      // mock-trae-online-log
      if (url.includes('/task-detail/') && url.includes('/mock-trae-online-log') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ log: '', in_progress: false }),
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
            container_endpoint_registered: false,
          }),
        })
        return
      }

      // container-layer-graph
      if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ layers_root: '', layers: [], jobs: [] }) })
        return
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    })

    await page.goto(TASK_DETAIL_PATH, { waitUntil: 'domcontentloaded' })

    // ── 断言辅助信息面板可见 ──
    const auxPanel = page.getByTestId('task-aux-info-panel')
    await expect(auxPanel).toBeVisible({ timeout: 30000 })

    // ── 断言任务编号 #N 可见且匹配 mock 的 workspace_seq ──
    const displayNo = page.getByTestId('task-aux-info-display-no')
    await expect(displayNo).toBeVisible({ timeout: 10000 })
    await expect(displayNo).toContainText('#3')

    // ── 断言正文默认可见（展开态） ──
    const auxBody = page.getByTestId('task-aux-info-body')
    await expect(auxBody).toBeVisible({ timeout: 10000 })

    // ── 断言排队调度已迁至评论区自动执行卡（F-102） ──
    const auxQueueToggle = auxBody.getByTestId('task-queued-auto-run-toggle')
    await expect(auxQueueToggle).toHaveCount(0)
    const queueToggle = page.getByTestId('task-queued-auto-run-toggle')
    await expect(queueToggle).toBeVisible({ timeout: 10000 })
    const queuePageLink = page.getByTestId('queued-schedule-page-link')
    await expect(queuePageLink).toBeVisible({ timeout: 5000 })
    await expect(queuePageLink).toContainText('自动调度安排')
    const depPicker = page.getByTestId('comment-execution-dependency-picker')
    await expect(depPicker.getByTestId('task-queued-auto-run-toggle')).toBeVisible({ timeout: 5000 })

    // ── 断言 toggle 按钮显示「收起」 ──
    const toggleBtn = page.getByTestId('task-aux-info-toggle')
    await expect(toggleBtn).toBeVisible({ timeout: 5000 })
    await expect(toggleBtn).toContainText('收起')

    // ── 点击收起 ──
    await toggleBtn.click()

    // ── 断言正文不可见 ──
    await expect(auxBody).not.toBeVisible({ timeout: 5000 })

    // ── 断言 toggle 按钮显示「展开」 ──
    await expect(toggleBtn).toContainText('展开')
  })
})
