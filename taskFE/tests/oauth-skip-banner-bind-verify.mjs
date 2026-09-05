/**
 * 验收：skip 横幅在 user-app-connection 已绑定后不再显示「去绑定 Git OAuth」。
 * CDP 9222 + 公网任务详情。
 */
import { chromium } from 'playwright'
import { mkdirSync, writeFileSync } from 'fs'

const CDP = 'http://127.0.0.1:9222'
const ORIGIN = 'https://www.daydaymoney.com'
const TASK_URL =
  `${ORIGIN}/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878902048790179840/?accessCode=DR2AKvP9J9`
const EMAIL = 'contact@daydaymoney.com'
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD
const OUT = '/tmp/ram-work/playwright/oauth-skip-banner-bind-verify'
mkdirSync(OUT, { recursive: true })

const browser = await chromium.connectOverCDP(CDP, { timeout: 15_000 })
const context = browser.contexts()[0] || (await browser.newContext())
const page = context.pages()[0] || (await context.newPage())
page.setDefaultTimeout(30_000)

await page.goto(`${ORIGIN}/auth/login/`, { waitUntil: 'domcontentloaded', timeout: 45_000 })
const login = await page.evaluate(async ({ email, password }) => {
  const h = { Origin: location.origin, Accept: 'application/json' }
  const privacy = await (await fetch('/api/privacy-policy/public/current/', { headers: h })).json()
  const license = await (await fetch('/api/license-agreement/public/current/', { headers: h })).json()
  const loginRes = await fetch('/api/auth/', {
    method: 'POST',
    credentials: 'include',
    headers: { ...h, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      username: email,
      password,
      rememberMe: true,
      accepted_privacy_policy_id: String(privacy?.id || ''),
      accepted_license_agreement_id: String(license?.id || ''),
    }),
  })
  const data = await loginRes.json().catch(() => ({}))
  const token = String(data?.token || '')
  const userId = String(data?.user?.id || '')
  if (loginRes.ok && token && userId) {
    await fetch('/api/accounts/users/activate-session/', {
      method: 'POST',
      credentials: 'include',
      headers: { ...h, 'Content-Type': 'application/json', Authorization: `Token ${token}` },
      body: JSON.stringify({ user_id: userId }),
    })
  }
  return { status: loginRes.status, userId, hasToken: Boolean(token) }
}, { email: EMAIL, password: PASSWORD })

await page.goto(`${TASK_URL}&_cb=${Date.now()}`, { waitUntil: 'domcontentloaded', timeout: 60_000 })
await page.waitForTimeout(2500)
const consent = page.getByRole('button', { name: '同意并继续' })
if (await consent.isVisible({ timeout: 3000 }).catch(() => false)) {
  await consent.click()
  await page.waitForTimeout(1500)
}
await page.getByTestId('auto-run-start-skipped-banner').waitFor({ state: 'visible', timeout: 25_000 })
const dump = await page.evaluate(async () => {
  const banner = document.querySelector('[data-testid="auto-run-start-skipped-banner"]')
  const bind = document.querySelector('[data-testid="auto-run-skip-oauth-bind"]')
  const restart = document.querySelector('[data-testid="auto-run-force-restart"]')
  const path = location.pathname
  const m = path.match(/tenant\/([^/]+)\/workspace\/([^/]+)\/task-detail\/([^/]+)/)
  let connection = null
  let task = null
  if (m) {
    const [, tenantId, workspaceId, taskId] = m
    const taskResp = await fetch(
      `/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${workspaceId}/${taskId}/`,
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    const taskBody = await taskResp.json().catch(() => ({}))
    task = {
      status: taskResp.status,
      auto_run_start_skipped: taskBody.auto_run_start_skipped,
      auto_run_start_skip_reason: taskBody.auto_run_start_skip_reason,
    }
    const connResp = await fetch(
      '/api/git-oauth/user-app-connection/?repo_url=' +
        encodeURIComponent('https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad'),
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    connection = { status: connResp.status, body: await connResp.json().catch(() => ({})) }
  }
  return {
    href: location.href,
    banner: Boolean(banner),
    bind: Boolean(bind),
    restart: Boolean(restart),
    reason: (document.querySelector('[data-testid="auto-run-start-skipped-reason"]')?.textContent || '').trim(),
    task,
    connected: connection?.body?.connected === true,
    connectionStatus: connection?.status,
  }
})
await page.screenshot({ path: `${OUT}/after.png`, fullPage: true, timeout: 8000 }).catch(() => {})
writeFileSync(`${OUT}/after.json`, JSON.stringify({ login, dump }, null, 2))
console.log(JSON.stringify({ login, dump }, null, 2))
if (!dump.banner) {
  console.error('FAIL: skip banner missing')
  process.exit(1)
}
if (!dump.restart) {
  console.error('FAIL: force restart missing')
  process.exit(1)
}
if (dump.connected && dump.bind) {
  console.error('FAIL: OAuth already connected but bind button still visible')
  process.exit(1)
}
if (dump.connected && /重新绑定/.test(dump.reason || '')) {
  console.error('FAIL: connected but skip reason still asks to rebind:', dump.reason)
  process.exit(1)
}
if (!dump.connected && !dump.bind) {
  console.error('WARN: not connected and bind hidden — unexpected')
}
console.log('PASS')
process.exit(0)
