// @ts-check
/**
 * OPT-20260723-028: 改后执行 completed → Feed 出现含 PR 的 Agent 回复。
 *
 * 测试策略：
 * - Mock 任务详情返回含人工评论 + 容器 Agent 评论的数据
 * - Agent 评论的 assistant_response 含 PR 链接格式内容
 * - 断言 ConversationFeed 中展示「容器 Agent」badge 及 Agent 回复板块
 * - 回复内容中包含 PR 链接（markdown 格式 [PR](url)）
 *
 * 运行：
 *   npx playwright test tests/TaskDetail.edit-run-pr-backfill.playwright.test.js --config=playwright.verify.config.js
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

const HUMAN_COMMENT_ID = 'comment-human-1'
const AGENT_COMMENT_ID = 'agent-comment-1'
const PR_URL = 'https://github.com/mock-org/mock-repo/pull/42'

test.describe('TaskDetail 改后执行 PR 回填 Feed', () => {
  test('容器 Agent 回复含 PR 链接时 Feed 中可见', async ({ page }) => {
    // 模拟 SSE：无操作（不发送实质事件，只保持连接）
    await page.addInitScript(() => {
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
            const ev = { type: 'open' }
            for (const cb of this._listeners.open || []) {
              try { cb(ev) } catch (_) {}
            }
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
      }
      window.EventSource = MockEventSource
    })

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ])

    await page.route('**/api/**', async (route) => {
      const req = route.request()
      const url = req.url()
      const method = req.method()

      // 任务详情 ── 返回含容器 Agent 评论的数据
      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Edit-run PR backfill test',
            description: 'Test for PR link in agent reply after edit-run',
            priority: 1,
            created_at: '2026-07-01T10:00:00Z',
            created_by: { id: 'u1', username: 'test-user' },
            // 人工评论
            comments: [
              {
                id: HUMAN_COMMENT_ID,
                content: '请修改后运行',
                created_at: '2026-07-01T10:01:00Z',
                created_by: { id: 'u1', username: 'test-user' },
              },
            ],
            ai_comments: [],
            // 容器 Agent 评论（对应 EDIT_RUN_AGENT_COMMENT 完成后的响应）
            // container_agent_comments 通过 parent_comment_id 挂到人工评论下
            container_agent_comments: [
              {
                id: AGENT_COMMENT_ID,
                parent_comment_id: HUMAN_COMMENT_ID,
                content: '',
                created_at: '2026-07-01T10:05:00Z',
                // assistant_response 含 PR 链接（markdown 格式）
                assistant_response: `Agent 已完成代码修改并提交 PR。

[PR #42: fix: update config for production](${PR_URL})

请审核变更内容。`,
              },
            ],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            parameters: {},
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

      // container-task-ui-context
      if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', container_endpoint_registered: true }),
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

    // 等待评论面板加载
    const feed = page.getByTestId('task-detail-conversation-feed')
    await expect(feed).toBeVisible({ timeout: 30000 })

    // 断言容器 Agent badge 可见
    await expect(feed.getByText('容器 Agent')).toBeVisible({ timeout: 10000 })

    // 断言「Agent 回复」标题可见
    await expect(feed.getByText('Agent 回复')).toBeVisible({ timeout: 5000 })

    // 断言 agent 回复中包含 PR 链接
    // assistantLooksLikeMarkdown 检测到 markdown 链接 → 通过 SafeMarkdownBlock 渲染
    const agentReply = feed.locator('text=PR #42')
    await expect(agentReply).toBeVisible({ timeout: 5000 })

    // 断言链接指向正确 URL
    const prLink = feed.locator(`a[href="${PR_URL}"]`)
    await expect(prLink).toBeVisible({ timeout: 5000 })
  })
})
