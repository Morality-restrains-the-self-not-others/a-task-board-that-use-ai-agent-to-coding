/**
 * 评论级执行归属与依赖契约（前端）。
 * 后端多容器编排接入前：任务级容器状态/可写层仅挂在 active 评论的「执行细节」内。
 */
import { computed, unref } from 'vue'

/**
 * $param {unknown} comments
 * $param {string} [activeContainerAgentId]
 * $param {{
 *   bindingStatusFor?: (commentId: unknown) => string,
 *   bindingCscIdFor?: (commentId: unknown) => string,
 * }} [opts]
 * $returns {string} 顶层评论 id；无候选时 ''
 */
export function resolveActiveExecutionCommentId(comments, activeContainerAgentId = '', opts = {}) {
  const list = Array.isArray(comments) ? comments : []
  const aid = String(activeContainerAgentId || '').trim()
  if (aid) {
    for (const c of list) {
      if (String(c?.id ?? '') === aid) {
        return String(c.id)
      }
      const children = Array.isArray(c?.children) ? c.children : []
      if (children.some((ch) => String(ch?.id ?? '') === aid)) {
        return String(c.id)
      }
    }
  }

  const statusOf = typeof opts.bindingStatusFor === 'function' ? opts.bindingStatusFor : null
  const cscOf = typeof opts.bindingCscIdFor === 'function' ? opts.bindingCscIdFor : null
  if (statusOf) {
    let liveWithCscId = ''
    let liveId = ''
    for (const c of list) {
      if (c?.id == null || c?.commentKind === 'container_agent') continue
      const st = String(statusOf(c.id) || '')
      if (st !== 'running' && st !== 'starting') continue
      liveId = String(c.id)
      const csc = cscOf ? String(cscOf(c.id) || '').trim() : ''
      if (csc) liveWithCscId = String(c.id)
    }
    // 优先挂接了任务级 CSC 的评论，避免并行评论抢走「当前执行」却复用同一连接面板
    if (liveWithCscId) return liveWithCscId
    if (liveId) return liveId
  }

  /** 评论是否显式 $镜像 引用（有明确的容器归属，应优先选为「当前执行」） */
  const hasImgMention = (c) => {
    const mentions = Array.isArray(c?.mentions) ? c.mentions : []
    return mentions.some((m) => m?.type === 'installed_image' && m?.id != null && String(m.id).trim() !== '')
  }

  const ais = list.filter((c) => c?.commentKind === 'ai' && c?.id != null)
  if (ais.length > 0) {
    // 优先选带 installed_image mention 的 AI 评论（容器执行归属明确，避免选错无镜像评论）
    const imgAi = ais.filter(hasImgMention)
    if (imgAi.length > 0) return String(imgAi[imgAi.length - 1].id)
    return String(ais[ais.length - 1].id)
  }

  const tops = list.filter(
    (c) =>
      c?.id != null &&
      c?.commentKind !== 'container_agent' &&
      (c?.commentKind === 'user' || !c?.commentKind),
  )
  if (tops.length > 0) {
    // 优先选带 installed_image mention 的评论
    const imgTops = tops.filter(hasImgMention)
    if (imgTops.length > 0) return String(imgTops[imgTops.length - 1].id)
    return String(tops[tops.length - 1].id)
  }

  return ''
}

/**
 * $param {object|null|undefined} comment
 * $returns {'wait_previous'|'independent'}
 */
export function resolveCommentExecutionMode(comment) {
  if (!comment || typeof comment !== 'object') {
    return 'wait_previous'
  }
  if (comment.execution_mode === 'independent' || comment.dependencyMode === 'independent') {
    return 'independent'
  }
  if (comment.wait_previous === false || comment.waitPrevious === false) {
    return 'independent'
  }
  return 'wait_previous'
}

/**
 * $param {object|null|undefined} comment
 * $returns {string[]}
 */
export function resolveCommentDependsOnIds(comment) {
  if (!comment || typeof comment !== 'object') return []
  const raw = comment.depends_on_comment_ids ?? comment.dependsOnCommentIds
  if (Array.isArray(raw)) {
    return raw.map((id) => String(id || '').trim()).filter(Boolean)
  }
  if (typeof raw === 'string' && raw.trim()) {
    return raw.split(',').map((id) => id.trim()).filter(Boolean)
  }
  return []
}

/** comment_container_bindings.status → 中文标签（执行细节 badge / 前序列表共用） */
export const COMMENT_BINDING_STATUS_LABELS = {
  pending: '待调度',
  waiting_previous: '等待前序',
  starting: '启动中',
  running: '运行中',
  completed: '已完成',
  failed: '失败',
  released: '已释放',
  cancelled: '已终止',
}

/**
 * $param {unknown} status
 * $returns {string}
 */
export function commentBindingStatusLabel(status) {
  const s = String(status || '').trim()
  if (!s) return '未知'
  return COMMENT_BINDING_STATUS_LABELS[s] || s
}

export function isReleasedRuntimeStatus(runtimeStatus) {
  const raw = String(runtimeStatus || '').trim()
  if (!raw) return false
  if (raw === '已释放') return true
  const lower = raw.toLowerCase()
  return lower === 'released' || lower === 'terminated'
}

/**
 * 执行细节 summary 容器状态徽章。云实例已释放时 binding 可能仍为 running，
 * 必须以运行态快照为准，不得继续显示「容器 运行中」。
 * $param {unknown} bindingStatus comment_container_bindings.status
 * $param {unknown} serverRuntimeStatus 该评论 runtime 快照 status 或展示文案
 * $returns {string}
 */
export function commentExecutionBindingBadgeText(bindingStatus, serverRuntimeStatus = '') {
  const binding = String(bindingStatus || '').trim()
  if (isReleasedRuntimeStatus(serverRuntimeStatus) || binding === 'released') {
    return '服务器已释放'
  }
  if (!binding) return ''
  return `容器 ${commentBindingStatusLabel(binding)}`
}

/** 与 summary 徽章同一判定：已释放时任务关联交互节点必须换成空态。 */
export function isCommentExecutionReleased(bindingStatus, serverRuntimeStatus = '') {
  return commentExecutionBindingBadgeText(bindingStatus, serverRuntimeStatus) === '服务器已释放'
}

/**
 * 非当前执行评论在「执行细节」面板内的提示。
 * OPT-20260822-062：云实例 Released/Terminated 时即使 binding 仍为 running，
 * 也优先提示「服务器已释放」，不再声称实例仍挂接。
 * $param {{
 *   bindingStatus?: string,
 *   ownsSharedContainer?: boolean,
 *   dependencyMode?: string,
 *   serverRuntimeStatus?: string,
 * }} [input]
 */
export function commentExecutionInactiveHint({
  bindingStatus = '',
  ownsSharedContainer = false,
  dependencyMode = 'wait_previous',
  serverRuntimeStatus = '',
} = {}) {
  if (isReleasedRuntimeStatus(serverRuntimeStatus) || String(bindingStatus || '') === 'released') {
    return '服务器已释放，本评论容器不再运行。'
  }
  const localMode = dependencyMode === 'independent' ? 'independent' : 'wait_previous'
  if (bindingStatus === 'waiting_previous') {
    return '调度器等待前序评论容器完成后再启动本评论容器。'
  }
  if (bindingStatus === 'starting' && !ownsSharedContainer) {
    return localMode === 'independent'
      ? '可并行（不等待前序）已调度；正在为本评论分配独立 CSC/容器。'
      : '串行调度中；正在为本评论分配独立实例。'
  }
  if (bindingStatus === 'running' || bindingStatus === 'starting') {
    return ownsSharedContainer
      ? '本评论已挂接独立实例；连接与运行状态见本面板。'
      : '本评论已独立调度，独立实例分配中。'
  }
  return localMode === 'independent'
    ? '该评论可独立展开执行（不等待前序）；每个并行评论使用独立 CSC。'
    : '等待前序评论执行完成后再展开；本评论的容器面板见下方。'
}

function isFinishedBindingStatus(status) {
  const s = String(status || '')
  return s === 'completed' || s === 'failed' || s === 'released' || s === 'cancelled'
}

function predecessorSummary(comment, fallbackId) {
  if (!comment) return `前序评论 ${fallbackId}（不在当前页）`
  const content = String(comment.content || '').replace(/\s+/g, ' ').trim()
  if (content) return content.length > 72 ? `${content.slice(0, 72)}…` : content
  return `评论 ${String(comment.id || fallbackId)}`
}

/**
 * $param {object|null|undefined} comment
 * $param {(id: string) => string} bindingStatusFor
 * $param {string} fallbackId
 */
function predecessorRowFromComment(comment, bindingStatusFor, fallbackId) {
  const id = String(comment?.id || fallbackId || '')
  const status = String(
    bindingStatusFor(id) || comment?.status || comment?.comment_status || '',
  )
  const missing = !comment
  return {
    id,
    summary: predecessorSummary(comment, id),
    status,
    statusLabel: missing ? '未加载' : commentBindingStatusLabel(status),
    finished: isFinishedBindingStatus(status),
    missing,
  }
}

/**
 * 列出 wait_previous 评论的前序：显式 depends_on_comment_ids，否则按任务序取
 * 本评论之前的非 container_agent 评论。independent 返回空数组。
 *
 * $param {object|null|undefined} comment
 * $param {unknown} allComments
 * $param {{ bindingStatusFor?: (commentId: string) => string }} [opts]
 * $returns {Array<{
 *   id: string,
 *   summary: string,
 *   status: string,
 *   statusLabel: string,
 *   finished: boolean,
 *   missing: boolean,
 * }>}
 */
export function listCommentPredecessors(comment, allComments, opts = {}) {
  if (!comment || typeof comment !== 'object') return []
  const cid = String(comment.id || '')
  if (!cid) return []
  if (resolveCommentExecutionMode(comment) === 'independent') return []
  const list = Array.isArray(allComments) ? allComments : []
  const bindingStatusFor = typeof opts?.bindingStatusFor === 'function' ? opts.bindingStatusFor : () => ''
  const deps = resolveCommentDependsOnIds(comment)
  if (deps.length > 0) {
    return deps.map((id) => {
      const found = list.find((x) => String(x?.id) === String(id))
      return predecessorRowFromComment(found, bindingStatusFor, id)
    })
  }
  const idx = list.findIndex((x) => String(x?.id) === cid)
  if (idx < 0) return []
  return list
    .slice(0, idx)
    .filter((c) => String(c?.commentKind || '') !== 'container_agent')
    .map((c) => predecessorRowFromComment(c, bindingStatusFor, String(c?.id || '')))
}

/**
 * 判断评论是否存在「有效前序」：wait_previous 模式下，若存在尚未完成的
 * 前序评论（显式 depends_on_comment_ids 或按任务序的前一非 container_agent 评论），
 * 则该评论确需「等待前序完成」；否则展示「串行（无前序）」避免误导
 * （OPT-20260811-085：首条/无前序评论不应再显示「等待前序完成」）。
 *
 * $param {object|null|undefined} comment
 * $param {unknown} allComments
 * $param {{ bindingStatusFor?: (commentId: string) => string }} [opts]
 * $returns {boolean}
 */
export function commentHasEffectivePredecessors(comment, allComments, opts = {}) {
  return listCommentPredecessors(comment, allComments, opts).some((p) => !p.finished)
}

/**
 * 供「添加评论」依赖多选：顶层非 container_agent 评论。
 * $param {unknown} comments
 * $param {(c: object) => string} [authorLabelFn]
 * $returns {Array<{ id: string, authorLabel: string, summary: string }>}
 */
export function buildCommentPredecessorOptions(comments, authorLabelFn) {
  const list = Array.isArray(comments) ? comments : []
  const labelOf = typeof authorLabelFn === 'function' ? authorLabelFn : () => ''
  return list
    .filter((c) => c?.id != null && c?.commentKind !== 'container_agent')
    .map((c) => {
      const content = String(c.content || '').replace(/\s+/g, ' ').trim()
      return {
        id: String(c.id),
        authorLabel: String(labelOf(c) || c?.created_by?.username || '评论'),
        summary: content.length > 72 ? `${content.slice(0, 72)}…` : content,
      }
    })
}

/**
 * $param {unknown} comments
 * $param {string} [activeContainerAgentId]
 * $returns {Array<{ commentId: string, dependencyMode: 'wait_previous'|'independent', isActive: boolean, waitPrevious: boolean }>}
 */
export function buildCommentExecutionBindings(comments, activeContainerAgentId = '') {
  const list = Array.isArray(comments) ? comments : []
  const activeId = resolveActiveExecutionCommentId(list, activeContainerAgentId)
  return list
    .filter((c) => c?.id != null && c?.commentKind !== 'container_agent')
    .map((c) => {
      const dependencyMode = resolveCommentExecutionMode(c)
      const commentId = String(c.id)
      return {
        commentId,
        dependencyMode,
        waitPrevious: dependencyMode === 'wait_previous',
        isActive: Boolean(activeId) && commentId === activeId,
      }
    })
}

/**
 * $param {object} opts
 * $param {import('vue').MaybeRefOrGetter|unknown} opts.comments
 * $param {import('vue').MaybeRefOrGetter|string} [opts.activeContainerAgentId]
 */
export function useCommentExecutionContext(opts = {}) {
  const activeExecutionCommentId = computed(() =>
    resolveActiveExecutionCommentId(unref(opts.comments), unref(opts.activeContainerAgentId)),
  )
  const bindings = computed(() =>
    buildCommentExecutionBindings(unref(opts.comments), unref(opts.activeContainerAgentId)),
  )
  function isActiveExecutionComment(commentId) {
    return String(commentId ?? '') === String(activeExecutionCommentId.value || '')
  }
  function dependencyModeFor(comment) {
    return resolveCommentExecutionMode(comment)
  }
  return {
    activeExecutionCommentId,
    bindings,
    isActiveExecutionComment,
    dependencyModeFor,
  }
}
