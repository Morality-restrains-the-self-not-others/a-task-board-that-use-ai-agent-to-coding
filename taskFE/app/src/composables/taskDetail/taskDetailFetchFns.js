/**
 * Pure functions: data fetching and editing for TaskDetail.
 */
import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { showRequestError } from '../../utils/requestErrorDisplay.js'
import { handleActionChunkLoadError } from '../../utils/chunkLoadGuard.js'
import { blockCommentRunIfGitOauthUnbound } from '../../utils/gitOauthPushPrecheck.js'
import { collectLinkedRepoUrls } from '../../utils/commentRepoIdentity.js'
import { consumeSessionGrantTicket, sessionGrantTicketAny } from '../../utils/grantTicketSession.js'
import { stampRepoIdentityOauthGitsite } from '../../utils/createTaskGitIdentityGate.js'
import { mergeIdempotencyHeaders, newIdempotencyKey } from '../../utils/clickGuard.js'
import {
  pendingImageMention,
  getPendingImageMention,
  clearPendingImageMention,
} from './commentImageMentionState.js'
import {
  readCommentDependencyDraft,
  resetCommentDependencyDraft,
} from './commentDependencyDraft.js'
import {
  readCommentRepoIdentityDraft,
  resetCommentRepoIdentityDraft,
} from './commentRepoIdentityDraft.js'
import {
  normalizeMemberPk,
  sanitizeBranchSegment,
  resolveBranchNamePlaceholders,
} from '../../utils/taskDetailBranchAndRepoUtils.js'
import { isCommentContentTaskIdEcho } from './buildDisplayComments.js'
import { unwrapTaskDetailPayload } from '../../utils/unwrapTaskDetailPayload.js'
import { attachCommentFeeds } from './taskDetailCommentFeedApply.js'
export { fetchTranslatedTaskTitleSegment } from './taskDetailFetchTranslateTitle.js'
export {
  fetchWorkspaceCollaborators,
  fetchProgressStatusOptions,
  fetchWorkspaceTodoTitle,
  fetchParentDeliverableTitle,
  fetchForkSourceTitle,
  fetchDeliverableCategoryOptions,
  fetchWorkspaceTodosForParent,
  fetchTaskSubtree,
  fetchDeliverableTypeOptions,
  onDeliverableTypeChange,
} from './taskDetailFetchOptions.js'
export {
  onRepoReclone,
  onRepoCloneIdentityChange,
  maybeAutoApplyDefaultRepoCloneIdentities,
} from './taskDetailRepoReclone.js'

export const COMMENT_FEED_PAGE_LIMIT = 50

function appendCommentFeedQuery(path, opts = {}) {
  const params = new URLSearchParams()
  if (opts.limit != null && opts.limit !== '') {
    params.set('limit', String(opts.limit))
  }
  if (opts.cursor != null && String(opts.cursor).trim() !== '') {
    params.set('cursor', String(opts.cursor))
  }
  const qs = params.toString()
  if (!qs) return path
  return `${path}${path.includes('?') ? '&' : '?'}${qs}`
}

/**
 * 拉取容器 Agent 评论列表；失败时返回 { data, nextCursor, hasMore, error }。
 */
export async function fetchContainerAgentComments(tenantId, workspaceId, taskId, opts = {}) {
  // taskAIComment：/api/tenant/{tid}/workspace/{wid}/task/{taskId}/container-agent-comments
  const path = appendCommentFeedQuery(
    `/api/ai-comment/container-agent-comments/tenant_id/${encodeURIComponent(String(tenantId))}/workspace_id/${encodeURIComponent(String(workspaceId))}/task_id/${encodeURIComponent(String(taskId))}/`,
    opts,
  )
  return fetchCommentFeedPath('container_agent_comments', path)
}

/**
 * 拉取人类评论列表。
 */
export async function fetchHumanComments(tenantId, taskId, opts = {}) {
  const path = appendCommentFeedQuery(
    `/api/tasks/${encodeURIComponent(String(taskId))}/comments/tenant_id/${encodeURIComponent(String(tenantId))}`,
    opts,
  )
  return fetchCommentFeedPath('comments', path)
}

/**
 * 拉取 AI 评论列表。
 */
export async function fetchAIComments(tenantId, workspaceId, taskId, opts = {}) {
  // taskAIComment：/api/tenant/{tid}/workspace/{wid}/task-detail/{taskId}/ai-comments
  const path = appendCommentFeedQuery(
    `/api/ai-comment/task-detail/tenant_id/${encodeURIComponent(String(tenantId))}/workspace_id/${encodeURIComponent(String(workspaceId))}/task_id/${encodeURIComponent(String(taskId))}/ai-comments/`,
    opts,
  )
  return fetchCommentFeedPath('ai_comments', path)
}

function extractCommentList(data) {
  if (Array.isArray(data)) return data
  if (Array.isArray(data?.results)) return data.results
  if (Array.isArray(data?.comments)) return data.comments
  return []
}

function mergeCommentsById(existing, incoming) {
  const seen = new Set((Array.isArray(existing) ? existing : []).map((c) => String(c?.id)))
  const merged = [...(Array.isArray(existing) ? existing : [])]
  for (const c of Array.isArray(incoming) ? incoming : []) {
    const id = c?.id != null ? String(c.id) : ''
    if (!id || seen.has(id)) continue
    seen.add(id)
    merged.push(c)
  }
  return merged
}

export async function fetchCommentFeedPath(label, path) {
  try {
    const response = await apiFetch(path, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (!response.ok) {
      console.warn(`获取 ${label} 失败`, response.status)
      return { data: [], nextCursor: null, hasMore: false, error: `${label}: HTTP ${response.status}` }
    }
    const data = await response.json()
    if (Array.isArray(data)) {
      return { data, nextCursor: null, hasMore: false, error: null }
    }
    const results = extractCommentList(data)
    const nextCursor = data?.next_cursor ?? null
    const hasMore = data?.has_more === true || (nextCursor != null && nextCursor !== '')
    return { data: results, nextCursor, hasMore, error: null }
  } catch (error) {
    console.warn(`获取 ${label} 出错:`, error)
    return { data: [], nextCursor: null, hasMore: false, error: `${label}: ${error?.message || 'request failed'}` }
  }
}

export async function fetchTaskDetail(deps) {
  const {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId, localTask,
    taskDetailLoading, taskDetailLoadError, taskDetailLoadErrorTraceId, syncRepoCloneIdentityMapFromTask,
  } = deps
  const taskId = effectiveTaskId.value
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  if (!taskId || !tenantId || !workspaceId) {
    if (taskDetailLoadError) taskDetailLoadError.value = '缺少租户、工作区或任务 ID，无法加载任务详情'
    if (taskDetailLoadErrorTraceId) taskDetailLoadErrorTraceId.value = ''
    return
  }
  if (taskDetailLoading) taskDetailLoading.value = true
  if (taskDetailLoadError) taskDetailLoadError.value = ''
  if (taskDetailLoadErrorTraceId) taskDetailLoadErrorTraceId.value = ''
  try {
    const response = await apiFetch(`/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${workspaceId}/${taskId}/`, { credentials: 'include', headers: { 'Accept': 'application/json' } })
    if (response.ok) {
      const raw = await response.json()
      const data = unwrapTaskDetailPayload(raw, taskId)
      if (!data) {
        if (taskDetailLoadError) {
          taskDetailLoadError.value = '获取任务详情失败：响应不是任务对象'
        }
        console.warn('[task-detail] GET payload is not a task record', {
          taskId,
          payloadType: Array.isArray(raw) ? 'array' : typeof raw,
        })
        return
      }
      if (Array.isArray(raw)) {
        console.warn('[task-detail] GET returned a task list; selected by taskId', {
          taskId,
          count: raw.length,
        })
      }
      const [humanFeed, aiFeed, agentFeed] = await Promise.all([
        fetchHumanComments(tenantId, taskId, { limit: COMMENT_FEED_PAGE_LIMIT }),
        fetchAIComments(tenantId, workspaceId, taskId, { limit: COMMENT_FEED_PAGE_LIMIT }),
        fetchContainerAgentComments(tenantId, workspaceId, taskId, { limit: COMMENT_FEED_PAGE_LIMIT }),
      ])
      attachCommentFeeds(data, humanFeed, aiFeed, agentFeed)
      localTask.value = data
      syncRepoCloneIdentityMapFromTask()
      return
    }
    const errData = response._errorData
    const detail = errData && typeof errData.detail === 'string' ? errData.detail : ''
    if (taskDetailLoadError) {
      taskDetailLoadError.value = detail || `获取任务详情失败（HTTP ${response.status}）`
    }
    if (taskDetailLoadErrorTraceId) {
      taskDetailLoadErrorTraceId.value = extractTraceId(response) || ''
    }
    console.error('获取任务详情失败', response.status, detail)
  } catch (error) {
    if (taskDetailLoadError) {
      taskDetailLoadError.value = error?.message || '获取任务详情出错'
    }
    if (taskDetailLoadErrorTraceId) {
      taskDetailLoadErrorTraceId.value = extractTraceId(error) || ''
    }
    console.error('获取任务详情出错:', error)
  } finally {
    if (taskDetailLoading) taskDetailLoading.value = false
  }
}

export async function loadMoreCommentFeeds(deps) {
  const {
    effectiveTenantId,
    effectiveWorkspaceId,
    effectiveTaskId,
    localTask,
    commentsFeedsLoadingMore,
  } = deps
  const task = localTask?.value
  if (!task) return
  if (commentsFeedsLoadingMore?.value) return
  const tenantId = effectiveTenantId.value
  const workspaceId = effectiveWorkspaceId.value
  const taskId = effectiveTaskId.value
  if (!tenantId || !workspaceId || !taskId) return

  const pending = []
  if (task.comments_has_more && task.comments_next_cursor) {
    pending.push({
      field: 'comments',
      cursorKey: 'comments_next_cursor',
      hasMoreKey: 'comments_has_more',
      promise: fetchHumanComments(tenantId, taskId, {
        limit: COMMENT_FEED_PAGE_LIMIT,
        cursor: task.comments_next_cursor,
      }),
    })
  }
  if (task.ai_comments_has_more && task.ai_comments_next_cursor) {
    pending.push({
      field: 'ai_comments',
      cursorKey: 'ai_comments_next_cursor',
      hasMoreKey: 'ai_comments_has_more',
      promise: fetchAIComments(tenantId, workspaceId, taskId, {
        limit: COMMENT_FEED_PAGE_LIMIT,
        cursor: task.ai_comments_next_cursor,
      }),
    })
  }
  if (task.container_agent_comments_has_more && task.container_agent_comments_next_cursor) {
    pending.push({
      field: 'container_agent_comments',
      cursorKey: 'container_agent_comments_next_cursor',
      hasMoreKey: 'container_agent_comments_has_more',
      promise: fetchContainerAgentComments(tenantId, workspaceId, taskId, {
        limit: COMMENT_FEED_PAGE_LIMIT,
        cursor: task.container_agent_comments_next_cursor,
      }),
    })
  }
  if (!pending.length) return

  if (commentsFeedsLoadingMore) commentsFeedsLoadingMore.value = true
  try {
    const results = await Promise.all(pending.map((item) => item.promise))
    const nextTask = { ...localTask.value }
    const feedErrors = [
      ...(Array.isArray(nextTask.comments_feed_errors) ? nextTask.comments_feed_errors : []),
    ]
    pending.forEach((item, idx) => {
      const feed = results[idx]
      if (feed.error) {
        feedErrors.push(feed.error)
        return
      }
      nextTask[item.field] = mergeCommentsById(nextTask[item.field], feed.data)
      nextTask[item.cursorKey] = feed.nextCursor ?? null
      nextTask[item.hasMoreKey] = Boolean(feed.hasMore)
    })
    if (feedErrors.length > 0) {
      nextTask.comments_feed_errors = feedErrors
    } else {
      delete nextTask.comments_feed_errors
    }
    localTask.value = nextTask
  } finally {
    if (commentsFeedsLoadingMore) commentsFeedsLoadingMore.value = false
  }
}










export async function onProgressStatusChange(event, deps) {
  const {
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    progressStatusOptions, isUpdatingProgressStatus, progressStatusError,
    progressStatusErrorTraceId, localTask, fetchTaskSubtree,
  } = deps
  const nextStatusId = event?.target?.value ? String(event.target.value) : ''
  const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value; const taskId = effectiveTaskId.value
  if (!tenantId || !workspaceId || !taskId || !nextStatusId) { progressStatusError.value = '更新进度状态失败'; return }
  isUpdatingProgressStatus.value = true
  progressStatusError.value = ''
  if (progressStatusErrorTraceId) progressStatusErrorTraceId.value = ''
  try {
    const response = await apiFetch(`/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${workspaceId}/${taskId}/`, { method: 'PATCH', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ progress_column_id: nextStatusId }) })
    if (!response.ok) {
      let msg = '更新进度状态失败'
      try {
        const errBody = await response.json()
        if (errBody?.code === 'DESCENDANTS_NOT_TERMINAL' || errBody?.error) {
          msg = String(errBody.detail || errBody.error || msg)
        }
      } catch { /* ignore */ }
      progressStatusError.value = msg
      if (progressStatusErrorTraceId) progressStatusErrorTraceId.value = extractTraceId(response) || ''
      return
    }
    const selected = progressStatusOptions.value.find((item) => item.id === nextStatusId)
    if (localTask.value) { localTask.value.progress_column_id = nextStatusId; if (selected) { localTask.value.progress_column = { id: selected.id, name: selected.name } } }
    if (typeof fetchTaskSubtree === 'function') void fetchTaskSubtree()
  } catch (error) {
    console.error('更新进度状态失败:', error)
    progressStatusError.value = '更新进度状态失败'
    if (progressStatusErrorTraceId) progressStatusErrorTraceId.value = extractTraceId(error) || ''
  } finally { isUpdatingProgressStatus.value = false }
}



export async function submitComment(deps) {
  const {
    effectiveTenantId, effectiveTaskId, newComment, commentComposerChips,
    buildCommentBodyWithChips, fetchTaskDetail,
  } = deps
  try {
    const tenantId = effectiveTenantId.value; const taskId = effectiveTaskId.value
    if (!tenantId || !taskId) { console.error('提交评论失败: 缺少 tenantId 或 taskId'); return }
    const content = buildCommentBodyWithChips()
    if (!content) return
    if (isCommentContentTaskIdEcho(content, taskId)) {
      showRequestError('评论内容不能仅为任务编号')
      return
    }
    const depDraft = readCommentDependencyDraft()
    const executionMode = deps.defaultExecutionMode === 'independent'
      ? 'independent'
      : depDraft.executionMode
    const body = {
      content,
      execution_mode: executionMode === 'independent' ? 'independent' : 'wait_previous',
      depends_on_comment_ids: executionMode === 'independent' ? [] : depDraft.dependsOnCommentIds,
      auto_commit: !!depDraft.autoCommit,
    }
    // 优先读 taskId 槽，回退默认槽（兼容尚未传 taskId 的编辑器）
    const mention = getPendingImageMention(taskId) || pendingImageMention.value
    if (mention && mention.id) {
      if (await blockCommentRunIfGitOauthUnbound(deps.taskProjectsWithDetails?.value || [])) return
      body.mentions = [{ type: 'installed_image', id: String(mention.id), name: String(mention.name || '') }]
      if (mention.skill) {
        body.mentions[0].skill = String(mention.skill)
      }
      const {
        readRegisteredCommentHardwarePanel,
        resolveCommentServerRunTemplate,
      } = await import('./commentRunHardwareTemplate.js')
      const panel = readRegisteredCommentHardwarePanel()
      const override = resolveCommentServerRunTemplate(panel)
      if (override) {
        body.server_run_template = override
      }
      const requiredRepos = collectLinkedRepoUrls(deps.taskProjectsWithDetails?.value || [])
      const identities = stampRepoIdentityOauthGitsite(readCommentRepoIdentityDraft())
      if (requiredRepos.length) {
        body.repo_identities = identities
      }
      const grantTicket = sessionGrantTicketAny()
      if (grantTicket) body.grant_ticket = grantTicket
    }
    const idempotencyKey = deps.idempotencyKey || newIdempotencyKey()
    const response = await apiFetch(`/api/tasks/${encodeURIComponent(String(taskId))}/comments/tenant_id/${encodeURIComponent(String(tenantId))}`, {
      method: 'POST',
      credentials: 'include',
      headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
      body: JSON.stringify(body),
    })
    if (response.ok) {
      // OPT-20260902-025：随 @镜像 评论提交的 grant_ticket 已消费，从 session 移除
      if (body.grant_ticket) consumeSessionGrantTicket(body.grant_ticket)
      newComment.value = ''
      commentComposerChips.value = []
      clearPendingImageMention(taskId)
      clearPendingImageMention() // 同时清默认槽（兼容无 taskId 的编辑器设置的值）
      resetCommentDependencyDraft()
      resetCommentRepoIdentityDraft()
      await fetchTaskDetail()
    }
    else { console.error('提交评论失败') }
  } catch (error) {
    console.error('提交评论出错:', error)
    if (handleActionChunkLoadError(error, { showError: showRequestError })) return
    showRequestError(error?.message || '提交评论失败', error)
  }
}

export function startEdit(deps) {
  const { localTask, workspaceProjects, editingTask, isEditing, editError, getProjectRepos, inferWorkBranchPreset, inferMergeTargetPreset } = deps
  if (!localTask.value) return
  const branchStrategy = localTask.value?.branch_strategy || {}
  const apiProjects = Array.isArray(localTask.value.projects) ? localTask.value.projects : []
  const linkedFromApi = []; const byPid = new Map()
  for (const p of apiProjects) {
    const pid = p?.project_id != null ? String(p.project_id) : ''
    if (!pid) continue
    if (!byPid.has(pid)) byPid.set(pid, { project_id: pid, repo_branches: {} })
    const urls = getProjectRepos(pid)
    const idx = Number.isFinite(Number(p.repo_index)) ? Number(p.repo_index) : 0
    const url = urls[idx] || ''
    if (url) { byPid.get(pid).repo_branches[url] = p.base_branch ? String(p.base_branch) : '' }
  }
  linkedFromApi.push(...byPid.values())
  // 单项目约束：编辑态至少保留一行项目选择（无关联项目时展示空行供用户选择）
  const singleLinkedProject = linkedFromApi.length > 0
    ? [linkedFromApi[0]]
    : [{ project_id: '', repo_branches: {} }]
  const workBranchNameRaw = branchStrategy.work_branch_name || branchStrategy.target_branch_name || ''
  const mergeTargetNameRaw = branchStrategy.merge_target_branch_name || ''
  const titleSegment = sanitizeBranchSegment(localTask.value.title || '', 'task')
  const workBranchName = resolveBranchNamePlaceholders(workBranchNameRaw, {
    taskId: String(localTask.value.id || ''),
    taskTitleSegment: titleSegment,
  })
  const mergeTargetName = resolveBranchNamePlaceholders(mergeTargetNameRaw, {
    taskId: String(localTask.value.id || ''),
    taskTitleSegment: titleSegment,
  })
  const deliverableObjId = (() => {
    const raw =
      localTask.value.deliverable_obj_id ??
      localTask.value.deliverable_obj?.id ??
      localTask.value.task_type?.id
    if (raw == null || raw === '') return ''
    return String(raw)
  })()
  const parentTaskId = (() => {
    const raw = localTask.value.parent_task ?? localTask.value.parent_task_id
    if (raw == null || raw === '') return ''
    if (typeof raw === 'object' && raw != null && raw.id != null) return String(raw.id)
    return String(raw)
  })()
  editingTask.value = {
    id: localTask.value.id,
    title: localTask.value.title || '',
    description: localTask.value.description || '',
    priority: localTask.value.priority ?? 2,
    task_kind: localTask.value.task_kind || '',
    code_lang: localTask.value.code_lang || '',
    deliverable_obj_id: deliverableObjId,
    parent_task: parentTaskId,
    linkedProjects: singleLinkedProject,
    owner: normalizeMemberPk(localTask.value.owner),
    operator: normalizeMemberPk(localTask.value.operator),
    assignees: Array.isArray(localTask.value.assignees)
      ? localTask.value.assignees.map(normalizeMemberPk).filter(Boolean)
      : [],
    workBranchPreset: inferWorkBranchPreset(workBranchName),
    workBranchName,
    mergeTargetPreset: inferMergeTargetPreset(mergeTargetName),
    mergeTargetName,
  }
  isEditing.value = true; editError.value = ''
}
