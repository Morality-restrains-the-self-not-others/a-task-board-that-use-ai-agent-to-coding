// @ts-check
/**
 * OPT-20260824-005: 排队调度设置已自任务详情拆出至「工作空间的排队调度」页面。
 *
 * 测试策略：
 * - PUT API 模拟成功（模拟返回 200 快照）
 * - 打开工作空间排队调度页面（queue-schedule）
 * - 修改启用开关后点击保存
 * - 断言 schedule-saved-tip 可见
 *
 * 运行：
 *   npx playwright test tests/TaskDetail.schedule-rhythm-save-success.playwright.test.js --config=playwright.verify.config.js
 */
import { test, expect } from '@playwright/test'
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js'

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID

const QUEUE_SCHEDULE_PATH = `/tenant/${TENANT_ID}/queue-schedule/?workspace_id=${WORKSPACE_ID}`

const SNAPSHOT = {
  workspace_id: WORKSPACE_ID,
  tenant_id: TENANT_ID,
  schedule_rhythm: {
    enabled: true,
    timezone: 'Asia/Shanghai',
    windows: [
      { id: 'w0', daily_start: '22:00', daily_end: '06:00', max_queued_machines: 1, auto_close: false },
    ],
    max_queued_machines: 1,
    auto_close: false,
    in_window: false,
  },
  in_window: false,
  window_message: '等待时段 22:00–06:00 (Asia/Shanghai)',
  queued_slots_used: 0,
  max_queued_machines: 1,
  members: [],
}

test.describe('工作空间排队调度保存成功反馈', () => {
  test('保存节奏成功后提示「调度设置已保存」', async ({ page }) => {
    /** PUT 请求计数（用于验证保存已触发） */
    let putCount = 0

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ])

    await page.route('**/api/**', async (route) => {
      const req = route.request()
      const url = req.url()
      const method = req.method()

      // /me/ —— 页面初始化解析当前工作空间 + 租户路由守卫校验公司成员
      // （守卫读 me.companies，缺少该字段会把 URL tenant 判为陈旧并重定向首页）
      if (url.includes('/api/accounts/users/me/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'e2e-user',
            username: 'e2e-user',
            current_workspace: { id: WORKSPACE_ID, name: 'E2E 工作空间' },
            current_company: { id: TENANT_ID },
            companies: [{ id: TENANT_ID, name: 'E2E 公司' }],
          }),
        })
        return
      }

      // GET 快照
      if (url.includes(`/api/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/queue-schedule/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(SNAPSHOT),
        })
        return
      }

      // PUT 保存节奏 —— 模拟成功并回显新快照
      if (url.includes(`/api/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/queue-schedule/`) && method === 'PUT') {
        putCount += 1
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ...SNAPSHOT,
            schedule_rhythm: {
              ...SNAPSHOT.schedule_rhythm,
              enabled: true,
              windows: [
                { id: 'w0', daily_start: '22:00', daily_end: '06:00', max_queued_machines: 2, auto_close: false },
              ],
            },
          }),
        })
        return
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    })

    await page.goto(QUEUE_SCHEDULE_PATH, { waitUntil: 'domcontentloaded' })

    // 页面标题 + 状态条出现
    await expect(page.getByText('自动调度安排')).toBeVisible({ timeout: 30000 })
    const statusBar = page.getByTestId('schedule-status-bar')
    await expect(statusBar).toBeVisible({ timeout: 10000 })

    // 保存设置
    const saveBtn = page.getByTestId('save-schedule')
    await expect(saveBtn).toBeVisible({ timeout: 5000 })
    await saveBtn.click()

    // 断言 PUT 请求已发出
    await expect
      .poll(() => putCount, { timeout: 10000, message: '等待 PUT 请求' })
      .toBeGreaterThanOrEqual(1)

    // 断言保存成功提示可见
    const successHint = page.getByTestId('schedule-saved-tip')
    await expect(successHint).toBeVisible({ timeout: 5000 })
    await expect(successHint).toContainText('已保存')
  })
})
