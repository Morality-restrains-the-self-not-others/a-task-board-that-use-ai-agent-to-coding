/**
 * PR/MR 回复评论：推送成功后落一条人类评论，并查询/合并远端 PR。
 */
import { apiFetch } from '../../utils/apiUtils.js'
import { mergeIdempotencyHeaders } from '../../utils/clickGuard.js'
import { messageFromFailedResponse } from '../../utils/httpError.js'
import { extractTraceId } from '../../utils/traceId.js'
import { resolveActiveExecutionCommentId } from './useCommentExecutionContext.js'

export function gitPrHtmlUrlOf(comment) {
  const gp = comment?.git_pr
  if (gp && typeof gp.html_url === 'string' && gp.html_url.trim()) {
    return gp.html_url.trim()
  }
  if (typeof comment?.git_pr_html_url === 'string' && comment.git_pr_html_url.trim()) {
    return comment.git_pr_html_url.trim()
  }
  return ''
}

export function gitPrProviderOf(htmlUrl, fallback = '') {
  const hinted = String(fallback || '').trim().toLowerCase()
  if (hinted === 'gitlab' || hinted === 'github') return hinted
  const u = String(htmlUrl || '')
  if (/\/-\/merge_requests\//i.test(u) || /gitlab/i.test(u)) return 'gitlab'
  return 'github'
}

export function collectGitPrHtmlUrls(comments) {
  const urls = []
  const seen = new Set()
  const walk = (list) => {
    for (const row of Array.isArray(list) ? list : []) {
      const u = gitPrHtmlUrlOf(row)
      if (u && !seen.has(u)) {
        seen.add(u)
        urls.push(u)
      }
      if (Array.isArray(row?.children) && row.children.length) walk(row.children)
    }
  }
  walk(comments)
  return urls
}

export function gitPrReplyCommentPath({ tenantId, workspaceId, taskId, parentCommentId }) {
  return `/api/tenant_id/${encodeURIComponent(tenantId)}/workspaceId/${encodeURIComponent(workspaceId)}/tasks/${encodeURIComponent(taskId)}/comments/${encodeURIComponent(parentCommentId)}/`
}

export async function recordGitPrReplyComment(deps, htmlUrl) {
  const url = typeof htmlUrl === 'string' ? htmlUrl.trim() : ''
  const tenantId = String(deps?.effectiveTenantId?.value || '').trim()
  const workspaceId = String(deps?.effectiveWorkspaceId?.value || '').trim()
  const taskId = String(deps?.effectiveTaskId?.value || '').trim()
  const parentCommentId = resolveActiveExecutionCommentId(
    deps?.displayComments?.value,
    deps?.activeContainerAgentId?.value || '',
    {
      bindingStatusFor: deps?.bindingStatusFor,
      bindingCscIdFor: deps?.bindingCscIdFor,
    },
  )
  if (!url || !tenantId || !workspaceId || !taskId || !parentCommentId) {
    console.warn('[LayerPush] PR 回复评论跳过：缺少 tenant/workspace/task/parent')
    return
  }
  const body = {
    content: url,
    execution_mode: 'independent',
    git_pr: { html_url: url, provider: gitPrProviderOf(url) },
  }
  try {
    const resp = await apiFetch(
      gitPrReplyCommentPath({ tenantId, workspaceId, taskId, parentCommentId }),
      {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      },
    )
    if (!resp?.ok) {
      console.warn('[LayerPush] PR 回复评论失败 HTTP', resp?.status)
      return
    }
    // POST 成功后本地立刻重拉；异步路径仍靠 task_git_pr_reply_created SSE
    if (typeof deps.fetchTaskDetail === 'function') {
      void deps.fetchTaskDetail()
    }
  } catch (err) {
    console.warn('[LayerPush] PR 回复评论失败', err)
  }
}

export async function fetchGitPrStatuses(tenantId, htmlUrls) {
  const tid = String(tenantId || '').trim()
  const urls = (Array.isArray(htmlUrls) ? htmlUrls : []).filter((u) => typeof u === 'string' && u.trim())
  if (!tid || !urls.length) return {}
  const resp = await apiFetch(
    `/api/git-oauth/merge-request-status/tenant_id/${encodeURIComponent(tid)}/`,
    {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ html_urls: urls.slice(0, 20) }),
    },
  )
  if (!resp?.ok) return {}
  const data = await resp.json().catch(() => ({}))
  const out = {}
  for (const row of Array.isArray(data?.results) ? data.results : []) {
    const u = typeof row?.html_url === 'string' ? row.html_url.trim() : ''
    if (!u) continue
    out[u] = {
      state: String(row?.state || 'unknown'),
      title: String(row?.title || ''),
      error: String(row?.error || ''),
      // 审计回显：谁在何时点击了一键合并（服务端从 git_oauth 审计表解析）
      merged_by: row?.merged_by != null ? String(row.merged_by) : '',
      merged_at: row?.merged_at != null ? String(row.merged_at) : '',
    }
  }
  return out
}

/** 须大于 taskGitOauth 合并 PUT（60s），小于 APISIX read（120s）。 */
export const MERGE_GIT_PR_TIMEOUT_MS = 90000

export const MERGE_GIT_PR_TIMEOUT_MESSAGE =
  'Git 站点合并未在时限内完成，请打开 MR 确认是否已合并后再试'

function isMergeTimeoutMessage(message) {
  const msg = String(message || '')
  if (/请求超时（\d+(\.\d+)? 秒）/.test(msg)) return true
  return /client\.timeout|context deadline exceeded|timeout exceeded while awaiting headers/i.test(msg)
}

function isMergeTimeoutFailure(err) {
  if (err?.name === 'TimeoutError') return true
  return isMergeTimeoutMessage(err?.message)
}

function wrapMergeFailure(errLike, extra = {}) {
  const raw = errLike instanceof Error ? errLike : new Error(String(errLike || '合并失败'))
  const timedOut = isMergeTimeoutFailure(raw)
  const next = timedOut ? new Error(MERGE_GIT_PR_TIMEOUT_MESSAGE) : raw
  if (timedOut) next.name = 'TimeoutError'
  const traceId = extra.traceId || raw.traceId
  if (traceId) next.traceId = traceId
  const status = extra.status ?? raw.status
  if (status != null) next.status = status
  return next
}

export async function mergeGitPullRequest({ tenantId, htmlUrl, taskId, commentId, headers }) {
  const tid = String(tenantId || '').trim()
  const url = String(htmlUrl || '').trim()
  if (!tid || !url) throw new Error('缺少租户或 PR 链接')
  let resp
  try {
    resp = await apiFetch(
      `/api/git-oauth/merge-request-merge/tenant_id/${encodeURIComponent(tid)}/`,
      {
        method: 'POST',
        credentials: 'include',
        timeout: MERGE_GIT_PR_TIMEOUT_MS,
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, headers?.['Idempotency-Key']),
        body: JSON.stringify({
          html_url: url,
          task_id: String(taskId || ''),
          comment_id: String(commentId || ''),
        }),
      },
    )
  } catch (err) {
    throw wrapMergeFailure(err)
  }
  if (!resp?.ok) {
    const msg = messageFromFailedResponse(resp, `HTTP ${resp.status}`)
    throw wrapMergeFailure(new Error(msg), {
      status: resp.status,
      traceId: resp?.traceId || extractTraceId(resp),
    })
  }
  return resp.json().catch(() => ({ ok: true, merged: true }))
}
