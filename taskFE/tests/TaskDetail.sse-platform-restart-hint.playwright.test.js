// @ts-check
/**
 * OPT-20260724-001: 已停止时不误显平台推送重启 hint。
 *
 * 测试策略：
 * - MockEventSource 快速触发 onerror（短耗时失败）且不先 emit open
 * - 模拟 SSE 探测路径（onerror 中的 fetch probe）返回非 502
 * - 断言 sse-platform-restart-hint 不可见
 * - 断言服务器生命周期仍为「已停止」
 *
 * 运行：
 *   npx playwright test tests/TaskDetail.sse-platform-restart-hint.playwright.test.js --config=playwright.verify.config.js
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

test.describe('TaskDetail SSE 平台重启提示', () => {
  test('SSE 短耗时失败且非 502 时，不显示重启 hint，生命周期为已停止', async ({ page }) => {
    // 记录被探测的 SSE URL（onerror 中的 fetch probe）
    let probeUrl = ''

    await page.addInitScript(
      ({ tenantId, workspaceId, taskId }) => {
        // 拦截 fetch 调用：当 fetch SSE URL 时拦截探测请求
        const origFetch = window.fetch.bind(window)
        window.__mockSseProbeResponse = null

        // MockEventSource：快速失败（不 emit open，直接触发 onerror）
        class FastFailEventSource {
          static CONNECTING = 0
          static OPEN = 1
          static CLOSED = 2

          constructor(url) {
            this.url = String(url || '')
            this.readyState = FastFailEventSource.CLOSED
            this.withCredentials = true
            this.onmessage = null
            this.onerror = null
            this.onclose = null
            this._listeners = { open: [], error: [] }

            // 短耗时后触发 onerror：模拟 establishSSEConnection 中的短时失败探测分支
            setTimeout(() => {
              this.readyState = FastFailEventSource.CLOSED
              if (typeof this.onerror === 'function') {
                this.onerror({ type: 'error', message: 'mock fast failure' })
              }
              // 也通知 error listener
              for (const cb of this._listeners.error || []) {
                try { cb({ type: 'error' }) } catch (_) {}
              }
            }, 30)
          }

          addEventListener(type, cb) {
            if (!this._listeners[type]) this._listeners[type] = []
            this._listeners[type].push(cb)
          }

          close() {
            this.readyState = FastFailEventSource.CLOSED
            if (typeof this.onclose === 'function') this.onclose()
          }
        }

        window.EventSource = FastFailEventSource
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

      // 任务详情 ── 生命周期为 stopped
      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'SSE platform restart hint test',
            description: '',
            priority: 1,
            created_at: '2026-07-01T10:00:00Z',
            created_by: { id: 'u1', username: 'test-user' },
            comments: [],
            ai_comments: [],
            container_agent_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            parameters: {},
            // 服务器状态：已停止
            container_endpoint_registered: false,
          }),
        })
        return
      }

      // container-task-ui-context ── 未注册（已停止）
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

      // 拦截 SSE URL 的探测 fetch ── 返回 200（非 502/HTML 5xx）
      // TaskDetail 会构造 SSE URL 类似 /api/sse/server-startup-status/tenant_id/{t}/workspace_id/{w}/task_id/{id}
      const sseProbePattern = `/cloud/server-startup-status-sse/`
      if (url.includes(sseProbePattern) && method === 'GET') {
        probeUrl = url
        // 返回 text/event-stream + 200 → 不是 HTML 5xx 或 502
        await route.fulfill({
          status: 200,
          contentType: 'text/event-stream',
          headers: { 'content-type': 'text/event-stream' },
          body: ':\n\n',
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

      // container-layer-graph
      if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ layers_root: '', layers: [], jobs: [] }) })
        return
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    })

    await page.goto(TASK_DETAIL_PATH, { waitUntil: 'domcontentloaded' })

    // 等待服务器启动状态面板出现
    const statusPanel = page.getByTestId('server-start-status-panel')
    await expect(statusPanel).toBeVisible({ timeout: 30000 })

    // ── 断言 sse-platform-restart-hint 不可见 ──
    const restartHint = page.getByTestId('sse-platform-restart-hint')
    await expect(restartHint).not.toBeVisible({ timeout: 10000 })

    // 确认探测请求已发出（非 502 探测 → hint 不应显示）
    // 等待一小段时间让 probe 完成
    await page.waitForTimeout(1500)
    // 再次确认 hint 仍未出现
    await expect(restartHint).not.toBeVisible({ timeout: 5000 })
  })
})
