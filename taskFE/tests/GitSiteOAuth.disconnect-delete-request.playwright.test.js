// @ts-check
import { expect, test } from '@playwright/test'

test.describe('Git 网站 OAuth 取消授权', () => {
  test('点击取消授权应发起 DELETE，并在成功后展示已取消文案', async ({ page, context, baseURL }) => {
    const uid = '827923618451263488'
    const cookieDomain = new URL(baseURL || 'http://127.0.0.1:4000').hostname
    await context.addCookies([
      { name: 'userId', value: uid, domain: cookieDomain, path: '/' },
      { name: 'csrftoken', value: 'playwright-csrf-token', domain: cookieDomain, path: '/' },
    ])

    let deleteHeaders = null
    let connected = true

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
        deleteHeaders = req.headers()
        connected = false
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            disconnected: true,
            was_connected: true,
          }),
        })
        return
      }
      await route.continue()
    })

    page.on('dialog', (dialog) => dialog.accept())

    await page.goto(`/user/${uid}/profile/git-site-oauth/`, {
      waitUntil: 'networkidle',
    })

    const cancelBtn = page.getByRole('button', { name: '取消该账号授权' })
    await expect(cancelBtn).toBeVisible()
    await cancelBtn.click()

    await expect(page.getByText('已取消 playwright-user 的 GitHub 授权')).toBeVisible()
    await expect(page.getByText('尚未绑定 GitHub 账号。')).toBeVisible()

    expect(deleteHeaders, '应发起 DELETE 请求').toBeTruthy()
    expect(deleteHeaders?.accept).toBe('application/json')
    expect(deleteHeaders?.['x-csrftoken'], 'DELETE 应附带 X-CSRFToken').toBeTruthy()
  })
})
