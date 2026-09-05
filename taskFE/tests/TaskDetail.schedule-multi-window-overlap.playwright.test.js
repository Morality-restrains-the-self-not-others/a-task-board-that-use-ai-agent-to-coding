// @ts-check
/**
 * OPT-20260725-045: 排队调度多窗口 — 窗口重叠检测 E2E 验证。
 * 设置已自任务详情拆出至「工作空间的排队调度」页面（OPT-20260824-005），
 * 交叠校验由后端 PUT 执行（applyWorkspaceScheduleRhythmFromBody → 400）。
 *
 * 覆盖：
 * 1) 保存 2 个交叠时段 → 后端 400 → 页面 schedule-error 显示交叠错误
 * 2) 修改为不交叠 → 保存成功 → schedule-saved-tip 可见
 *
 * 运行：
 *   npx playwright test tests/TaskDetail.schedule-multi-window-overlap.playwright.test.js --config=playwright.verify.config.js
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
      { id: 'w0', daily_start: '09:00', daily_end: '12:00', max_queued_machines: 1, auto_close: false },
      { id: 'w1', daily_start: '13:00', daily_end: '16:00', max_queued_machines: 1, auto_close: false },
    ],
    max_queued_machines: 2,
    auto_close: false,
    in_window: false,
  },
  in_window: false,
  window_message: '等待时段 09:00–12:00, 13:00–16:00 (Asia/Shanghai)',
  queued_slots_used: 0,
  max_queued_machines: 2,
  members: [],
}

test.describe('工作空间排队调度多窗口 — 重叠检测', () => {
  test('交叠时段被后端拒绝并显示错误，修改为不交叠后保存成功', async ({ page }) => {
    test.setTimeout(60_000)
    /** 记录 PUT 请求次数与 body */
    let putBodies = []

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ])

    await page.route('**/api/**', async (route) => {
      const req = route.request()
      const url = req.url()
      const method = req.method()

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

      const qsPath = `/api/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/queue-schedule/`
      if (url.includes(qsPath) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(SNAPSHOT),
        })
        return
      }

      if (url.includes(qsPath) && method === 'PUT') {
        const body = req.postDataJSON()
        putBodies.push(body)
        // 模拟后端交叠校验：两个窗口时段存在交叠 → 400
        const overlap = (a, b) => {
          const m = (s) => {
            const [h, min] = String(s || '').split(':').map(Number)
            return (h || 0) * 60 + (min || 0)
          }
          const a1 = m(a.daily_start); const a2 = m(a.daily_end)
          const b1 = m(b.daily_start); const b2 = m(b.daily_end)
          if (a1 === a2 || b1 === b2) return false
          return a1 < b2 && b1 < a2
        }
        const wins = (body.windows || []).filter((w) => w.daily_start && w.daily_end)
        const hasOverlap = wins.some((w, i) => wins.slice(i + 1).some((w2) => overlap(w, w2)))
        if (hasOverlap) {
          await route.fulfill({
            status: 400,
            contentType: 'application/json',
            body: JSON.stringify({
              status: 'error',
              error: '时间段 1（09:00–12:00）与时间段 2（11:00–14:00）存在交叠',
              message: '时间段 1（09:00–12:00）与时间段 2（11:00–14:00）存在交叠',
              trace_id: 'tr-overlap',
            }),
          })
          return
        }
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ ...SNAPSHOT }),
        })
        return
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    })

    await page.goto(QUEUE_SCHEDULE_PATH, { waitUntil: 'domcontentloaded' })
    await expect(page.getByText('自动调度安排')).toBeVisible({ timeout: 30000 })

    // 已有 2 个窗口行（快照回显），将第 2 个改为交叠（11:00–14:00 与 09:00–12:00 交叠）
    const start2 = page.getByTestId('window-start-1')
    const end2 = page.getByTestId('window-end-1')
    await expect(start2).toBeVisible({ timeout: 10000 })
    await start2.fill('11:00')
    await end2.fill('14:00')

    // 保存 → 后端 400 → 页面错误块可见（含交叠文案）
    await page.getByTestId('save-schedule').click()
    const errEl = page.getByTestId('schedule-error')
    await expect(errEl).toBeVisible({ timeout: 10000 })
    await expect(errEl).toContainText(/重叠|交叠/i)

    // 修改为不交叠（13:00–16:00）→ 保存成功
    await start2.fill('13:00')
    await end2.fill('16:00')
    await page.getByTestId('save-schedule').click()
    const tip = page.getByTestId('schedule-saved-tip')
    await expect(tip).toBeVisible({ timeout: 10000 })
    await expect(tip).toContainText('已保存')

    // 两次 PUT 均真实发出（第一次交叠被拒，第二次成功）
    await expect.poll(() => putBodies.length, { timeout: 10000 }).toBeGreaterThanOrEqual(2)
  })
})
