/**
 * 将注销阻断项 action_url 规范为 taskFE 已注册路由。
 * 后端若仍下发不存在的路径（如 /billing/gitlab-resources/），
 * Vue Router catch-all 会重定向到首页，用户无法「前往处理」。
 *
 * @param {{ action_url?: string, tenant_id?: string, code?: string }} item
 * @returns {string}
 */
export function resolveAccountDeletionActionUrl(item) {
  const url = String(item?.action_url || '').trim()
  const tid = String(item?.tenant_id || '').trim()
  const code = String(item?.code || '').trim()

  const gitlab = url.match(/^\/tenant\/([^/]+)\/billing\/gitlab-resources\/?$/)
  if (gitlab) {
    return `/tenant/${gitlab[1]}/settings/gitlab-connection/`
  }

  const members = url.match(/^\/tenant\/([^/]+)\/settings\/members\/?$/)
  if (members) {
    return `/tenant/${members[1]}/people/manage/`
  }

  const taskPage = url.match(/^\/tenant\/([^/]+)\/workspace\/([^/]+)\/task\/([^/]+)\/?$/)
  if (taskPage) {
    return `/tenant/${taskPage[1]}/workspace/${taskPage[2]}/task-detail/${taskPage[3]}/`
  }

  if (url === '/profile/' && code === 'BILLING_PAYMENT_PENDING' && tid) {
    return `/tenant/${tid}/billing/`
  }

  return url
}
