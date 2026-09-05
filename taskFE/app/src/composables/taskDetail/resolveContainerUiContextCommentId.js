import { resolveActiveExecutionCommentId } from './useCommentExecutionContext.js'

function unwrapMaybeRef(v) {
  if (v == null) return v
  if (typeof v === 'object' && 'value' in v) return v.value
  return v
}

/**
 * 页面级容器转发所需 comment_id：
 * 显式 id → 当前执行评论（含绑定 CSC 优先）。
 */
export function resolveContainerUiContextCommentId(deps = {}) {
  const explicit = String(
    unwrapMaybeRef(deps.commentId) || unwrapMaybeRef(deps.effectiveCommentId) || '',
  ).trim()
  if (explicit) return explicit
  const comments = unwrapMaybeRef(deps.displayComments) || unwrapMaybeRef(deps.comments) || []
  const agentId = unwrapMaybeRef(deps.activeContainerAgentId) || ''
  return String(resolveActiveExecutionCommentId(comments, agentId, {
    bindingStatusFor: deps.bindingStatusFor,
    bindingCscIdFor: deps.bindingCscIdFor,
  }) || '').trim()
}
