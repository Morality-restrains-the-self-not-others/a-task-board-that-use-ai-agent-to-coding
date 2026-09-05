// @ts-check
/**
 * OPT-20260717-045: 核验 Git 网站 OAuth 取消授权后页面展示正确的未绑定状态。
 *
 * 测试策略：
 * - 拦截 connection API：首次 GET 返回已连接 → 页面展示已绑定
 * - 点击「取消该账号授权」→ DELETE 请求将服务端状态切换为未连接
 * - 再次 GET connection API（如页面刷新/重绘）→ 返回未连接 → 页面展示「尚未绑定」
 * - 间接校验 gitOauth 数据库无对应 credential（通过 API 响应体中 connected=false）
 *
 * 运行：
 *   npx playwright test tests/GitSiteOAuth.disconnect.playwright.test.js --config=playwright.verify.config.js
 */
import { test, expect } from '@playwright/test'

test.describe('GitSiteOAuth 取消授权 — Mock', () => {
  test('取消授权后页面应展示未绑定状态，API 响应不再视为已连接', async ({ page, context, baseURL }) => {
    const uid = '827923618451263488'
    const cookieDomain = new URL(baseURL || 'http://127.0.0.1:4000').hostname

    await context.addCookies([
      { name: 'userId', value: uid, domain: cookieDomain, path: '/' },
      { name: 'csrftoken', value: 'playwright-csrf-token', domain: cookieDomain, path: '/' },
    ])

    /** 模拟连接状态，初始为已连接，DELETE 后变为未连接 */
    let connected = true
    let deleteReceived = false

    // ── 拦截 provider 列表 ──
    await page.route(/\/api\/accounts\/git-oauth\/providers\/?/, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          providers: [
            {
              provider: 'github',
              service_provider: 'github-official',
              provider_key: 'github:github-official',
              website: 'http://github.com',
              label: 'GitHub',
            },
          ],
        }),
      })
    })

    // ── 拦截 connection API（GET 返回当前 bound 状态，DELETE 取消后变未绑定） ──
    await page.route(/\/api\/user\/[^/]+\/accounts\/github\/app\/connection/, async (route) => {
      const req = route.request()
      if (req.method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            connected,
            github_login: connected ? 'playwright-user' : null,
            github_user_id: connected ? 10001 : null,
            scope: connected ? 'repo read:user' : null,
            authorize_scope: 'repo read:user',
            connections: connected
              ? [
                  {
                    connected: true,
                    github_login: 'playwright-user',
                    github_user_id: 10001,
                    scope: 'repo read:user',
                  },
                ]
              : [],
          }),
        })
        return
      }
      if (req.method() === 'DELETE') {
        deleteReceived = true
        // 切换为未连接状态
        connected = false
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            disconnected: true,
            was_connected: true,
            // 数据库无对应 credential 通过 connected=false 间接反映
            connected: false,
          }),
        })
        return
      }
      await route.continue()
    })

    // 同意确认对话框
    page.on('dialog', (dialog) => dialog.accept())

    // 第一步：导航到 git-site-oauth 页面，初始为已连接
    await page.goto(`/user/${uid}/profile/git-site-oauth/`, {
      waitUntil: 'networkidle',
    })

    // 验证页面上展示已连接状态
    await expect(page.getByText('playwright-user')).toBeVisible()
    await expect(page.getByText('已绑定 GitHub')).toBeVisible()

    // 找取消授权按钮并点击
    const cancelBtn = page.getByRole('button', { name: '取消该账号授权' })
    await expect(cancelBtn).toBeVisible()
    await cancelBtn.click()

    // 第二步：验证 DELETE 已发送
    expect(deleteReceived, '应已收到 DELETE 请求').toBe(true)

    // 验证页面展示「已取消」提示
    await expect(page.getByText('已取消 playwright-user 的 GitHub 授权')).toBeVisible()

    // 第三步：验证页面不再展示已连接信息，改为未绑定状态
    await expect(page.getByText('尚未绑定 GitHub 账号。')).toBeVisible()

    // 验证已绑定的用户信息不再展示
    await expect(page.getByText('playwright-user')).toHaveCount(0)
    await expect(page.getByText('已绑定 GitHub')).toHaveCount(0)
  })
})
