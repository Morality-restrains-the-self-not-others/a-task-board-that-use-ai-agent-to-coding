/**
 * OPT-20260903-002：container-layer-graph GET 在返回快照时会同步补写 git_pr 子评论
 * （taskCloudService ensureGitPrRepliesFromGraphDoc）。前端若同时并行请求 comments，
 * comments 响应可能早于补写提交，导致 Feed 暂时缺这条 git-pr-reply。
 *
 * 这里提供纯函数：当层图快照出现「评论 Feed 里还没有的 pr_html_url」时，判定应再拉一次
 * comments。配合调用侧的 taskId → 已重拉 URL 记忆，保证每个 PR URL 至多多触发一轮
 * 额外 comments GET，不会随层图心跳无限重拉。
 */

/** @param {string} u */
function isHttpUrl(u) {
  return typeof u === 'string' && /^https?:\/\//i.test(u.trim())
}

/**
 * 规约为可比较形式：协议/host 大小写不敏感、去首尾空白，路径与查询保持原样。
 * 后端回填的 git_pr.html_url 与层快照 pr_html_url 通常是同一字符串，这里额外容忍
 * scheme/host 大小写与空白差异。
 * @param {string} u
 * @returns {string}
 */
function canonicalPrUrl(u) {
  const raw = String(u || '').trim()
  if (!isHttpUrl(raw)) return ''
  try {
    const x = new URL(raw)
    const port = x.port ? `:${x.port}` : ''
    return `${x.protocol.toLowerCase()}//${x.hostname.toLowerCase()}${port}${x.pathname}${x.search}`
  } catch {
    return raw
  }
}

/**
 * 收集层图快照（layers 数组，含防御性 children / 嵌套 layers）里所有
 * `git_remote.pr_html_url` / `pr.html_url`，去重。
 * @param {unknown} layers
 * @returns {string[]}
 */
export function collectLayerPrHtmlUrls(layers) {
  const out = []
  const seen = new Set()
  const add = (raw) => {
    const u = canonicalPrUrl(raw)
    if (u && !seen.has(u)) {
      seen.add(u)
      out.push(u)
    }
  }
  const walk = (value) => {
    if (Array.isArray(value)) {
      for (const item of value) walk(item)
      return
    }
    if (!value || typeof value !== 'object') return
    const gr = value.git_remote
    if (gr && typeof gr === 'object') add(gr.pr_html_url)
    const pr = value.pr
    if (pr && typeof pr === 'object') add(pr.html_url)
    if (Array.isArray(value.children)) walk(value.children)
    if (Array.isArray(value.layers)) walk(value.layers)
  }
  walk(layers)
  return out
}

/**
 * 层图快照中已出现、但评论 Feed 尚未带回的 PR URL。
 * @param {unknown} layers 层图快照的 layers 数组
 * @param {string[]} existingCommentPrUrls 当前评论 Feed 里已有的 git_pr html_url
 * @returns {string[]}
 */
export function missingLayerPrHtmlUrls(layers, existingCommentPrUrls = []) {
  const inFeed = new Set()
  for (const raw of Array.isArray(existingCommentPrUrls) ? existingCommentPrUrls : []) {
    const u = canonicalPrUrl(raw)
    if (u) inFeed.add(u)
  }
  return collectLayerPrHtmlUrls(layers).filter((u) => !inFeed.has(u))
}

/**
 * 是否应再拉一次 comments（层图出现 Feed 缺失的 PR URL）。
 * @param {unknown} layers
 * @param {string[]} existingCommentPrUrls
 * @returns {boolean}
 */
export function shouldRefetchCommentsForLayerPrBackfill(layers, existingCommentPrUrls = []) {
  return missingLayerPrHtmlUrls(layers, existingCommentPrUrls).length > 0
}
