// @ts-check
import { expect, test } from '@playwright/test'

test.describe('Git 网站 OAuth 连接接口降级展示', () => {
  test('connection 接口 503 但返回状态载荷时，不应显示“暂时无法获取绑定状态”占位文案', async ({
    page,
    context,
    baseURL,
  }) => {
    const uid = '827923618451263488'
    const cookieDomain = new URL(baseURL || 'http://127.0.0.1:4000').hostname
    await context.addCookies([
      {
        name: 'userId',
        value: uid,
        domain: cookieDomain,
        path: '/',
      },
    ])

    await page.route(/\/api\/git-oauth\/providers\/?/, async (route) => {
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

    await page.route(/\/api\/git-oauth\/user-app-connection\/?/, async (route) => {
      if (route.request().method() !== 'GET') {
        await route.continue()
        return
      }
      await route.fulfill({
        status: 503,
        contentType: 'application/json',
        headers: { 'X-Trace-Id': 'pw-conn-trace-0001' },
        body: JSON.stringify({
          detail: '无法连接 gitOauth 或摘要响应无效；请稍后重试',
          connected: false,
          github_login: null,
          github_user_id: null,
          scope: null,
          authorize_scope: 'repo read:user',
        }),
      })
    })

    await page.goto(`/user/${uid}/profile/git-site-oauth/`, {
      waitUntil: 'networkidle',
    })

    await expect(page.getByText('尚未绑定 GitHub 账号。')).toBeVisible()
    await expect(page.getByText('无法连接 gitOauth 或摘要响应无效；请稍后重试')).toBeVisible()
    await expect(page.getByText('暂时无法获取绑定状态，请稍后重试。')).toHaveCount(0)
    // 降级错误文案必须携带响应中的 X-Trace-Id，便于按 data-traceId 检索日志
    await expect(page.locator('p.text-sm.text-danger')).toHaveAttribute(
      'data-traceId',
      'pw-conn-trace-0001',
    )
  })
})
