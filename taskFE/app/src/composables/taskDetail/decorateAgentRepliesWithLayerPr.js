/**
 * auto_run 容器 Agent 回复：隐藏对父评论的 prompt echo，并用 ztree 已有的 PR URL 补 git_pr。
 * 后端 complete 回填失败时，Feed 仍能展示 CommentGitPrReply，而不是复制原评论。
 * 平台在启服前预创建的 pending 空气泡不展示：回复应由容器镜像写入后再出现。
 */
import { gitPrHtmlUrlOf, gitPrProviderOf } from './taskDetailGitPrReply.js'

const CONTAINER_WRITING_STATUSES = new Set(['starting', 'running', 'streaming'])

/**
 * @param {unknown} childContent
 * @param {unknown} parentContent
 * @returns {boolean}
 */
export function isContainerAgentPromptEcho(childContent, parentContent) {
  const child = String(childContent || '').replace(/\r\n/g, '\n').trim()
  const parent = String(parentContent || '').replace(/\r\n/g, '\n').trim()
  if (!child || !parent) return false
  if (child === parent) return true
  const stripped = child.replace(/^@[^\s]+\s+/, '').trim()
  return Boolean(stripped) && stripped === parent
}

/**
 * @param {unknown} nodes layerGraphZNodes
 * @returns {string[]}
 */
export function collectPrHtmlUrlsFromZNodes(nodes) {
  const urls = []
  const seen = new Set()
  const walk = (list) => {
    for (const n of Array.isArray(list) ? list : []) {
      const u = typeof n?.prHtmlUrl === 'string' ? n.prHtmlUrl.trim() : ''
      if (u && /^https?:\/\//i.test(u) && !seen.has(u)) {
        seen.add(u)
        urls.push(u)
      }
      if (Array.isArray(n?.children) && n.children.length) walk(n.children)
    }
  }
  walk(nodes)
  return urls
}

/**
 * True when the container is already writing into this agent comment
 * (status or live SSE), so the bubble must stay even if body is still empty.
 *
 * @param {object} child
 * @param {{ activeAgentId?: string, streamBusy?: boolean, streamText?: string }} [opts]
 * @returns {boolean}
 */
function isContainerActivelyWriting(child, opts = {}) {
  const st = String(child?.run_status || child?.runStatus || '').trim().toLowerCase()
  if (CONTAINER_WRITING_STATUSES.has(st)) return true
  const aid = String(opts.activeAgentId || '').trim()
  if (!aid || String(child?.id || '') !== aid) return false
  return Boolean(opts.streamBusy) || Boolean(String(opts.streamText || '').trim())
}

/**
 * Platform pre-creates a pending container_agent row as soon as the human
 * @mention / auto_run comment lands — before the VM is Running. After echo
 * strip that row is an empty "容器 Agent" bubble. Hide it until the container
 * image writes a visible reply (body, assistant_response, PR, or stream).
 *
 * @param {object} child
 * @param {{ activeAgentId?: string, streamBusy?: boolean, streamText?: string }} [opts]
 * @returns {boolean}
 */
export function isPlaceholderContainerAgentReply(child, opts = {}) {
  if (!child || typeof child !== 'object') return false
  if (child.commentKind !== 'container_agent') return false
  if (gitPrHtmlUrlOf(child)) return false
  if (String(child.assistant_response || '').trim()) return false
  if (String(child.content || '').trim()) return false
  if (isContainerActivelyWriting(child, opts)) return false
  return true
}

/**
 * @param {object} child
 * @param {object} parent
 * @param {string} firstPrUrl
 * @param {{ activeAgentId?: string, streamBusy?: boolean, streamText?: string }} [opts]
 * @returns {object}
 */
function decorateChild(child, parent, firstPrUrl, opts = {}) {
  if (!child || typeof child !== 'object') return child
  const next = { ...child }
  const isAgent = next.commentKind === 'container_agent'
  const echo = isAgent && isContainerAgentPromptEcho(next.content, parent?.content)
  if (echo) next.content = ''
  const hasPr = Boolean(gitPrHtmlUrlOf(next))
  const pendingEmpty = isAgent && !String(next.assistant_response || '').trim()
  if (!hasPr && firstPrUrl && (echo || pendingEmpty)) {
    next.git_pr = {
      html_url: firstPrUrl,
      provider: gitPrProviderOf(firstPrUrl),
    }
  }
  if (Array.isArray(next.children) && next.children.length) {
    next.children = next.children
      .map((ch) => decorateChild(ch, next, firstPrUrl, opts))
      .filter((ch) => !isPlaceholderContainerAgentReply(ch, opts))
  }
  return next
}

/**
 * @param {Array<object>|null|undefined} comments buildDisplayComments 结果
 * @param {string[]} prUrls
 * @param {{ activeAgentId?: string, streamBusy?: boolean, streamText?: string }} [opts]
 * @returns {Array<object>}
 */
export function decorateAgentRepliesWithLayerPr(comments, prUrls, opts = {}) {
  const first = (Array.isArray(prUrls) ? prUrls : [])
    .map((u) => String(u || '').trim())
    .find((u) => /^https?:\/\//i.test(u))
  const firstPrUrl = first || ''
  return (Array.isArray(comments) ? comments : [])
    .map((row) => {
      if (!row || typeof row !== 'object') return row
      const children = Array.isArray(row.children)
        ? row.children
          .map((ch) => decorateChild(ch, row, firstPrUrl, opts))
          .filter((ch) => !isPlaceholderContainerAgentReply(ch, opts))
        : []
      return { ...row, children }
    })
    .filter((row) => !isPlaceholderContainerAgentReply(row, opts))
}
