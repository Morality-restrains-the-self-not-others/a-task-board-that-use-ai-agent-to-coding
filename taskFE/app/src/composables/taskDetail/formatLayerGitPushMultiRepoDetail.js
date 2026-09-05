/**
 * 从 container-layer-git-push 失败响应拼装多仓成败明细（父仓/子仓同逻辑）。
 * 与 onlineServiceJS formatOauthMultiRepoPushDetail 语义对齐。
 */

/** @param {object} r */
export function labelLayerGitPushRepo(r) {
  const slug = String(r?.github_slug || '').trim()
  const prefix = String(r?.rel_prefix || '').trim()
  if (slug && prefix) return `${slug}（路径 ${prefix}）`
  if (slug) return slug
  if (prefix) return prefix
  return 'unknown'
}

/**
 * @param {object[]} repos
 * @returns {string}
 */
export function formatOauthMultiRepoPushDetail(repos) {
  const list = Array.isArray(repos) ? repos.filter(Boolean) : []
  if (!list.length) return ''
  const ok = list.filter((r) => r.push_ok)
  const bad = list.filter((r) => !r.push_ok)
  const lines = []
  if (bad.length && ok.length) {
    lines.push(`部分仓库推送未成功（成功 ${ok.length}，失败 ${bad.length}）`)
  } else if (bad.length) {
    lines.push(`推送失败（${bad.length}/${list.length}）`)
  } else {
    lines.push(`推送成功（${ok.length}）`)
  }
  for (const r of list) {
    const lab = labelLayerGitPushRepo(r)
    if (r.push_ok) lines.push(`成功：${lab}`)
    else {
      const why = String(r.detail || '未推送').trim().slice(0, 240)
      lines.push(`失败：${lab} — ${why}`)
    }
  }
  return lines.join('\n')
}

/**
 * @param {object} body - 推送 API JSON
 * @returns {string} 展示用文案（优先多仓明细，否则 detail）
 */
export function formatLayerGitPushFailureMessage(body) {
  const repos = body?.github_oauth_multirepo?.repos
  const multi = formatOauthMultiRepoPushDetail(repos)
  if (multi) return multi
  if (body?.detail !== undefined) {
    return typeof body.detail === 'string' ? body.detail : JSON.stringify(body.detail)
  }
  return ''
}
