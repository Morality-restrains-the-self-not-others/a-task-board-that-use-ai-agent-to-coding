/**
 * Merge a task-detail payload into current localTask.
 *
 * GET/PATCH taskJSON 常把 comments / projects 写成 []（评论改走独立 feed；
 * 工作区列表或 feature-params 部分更新也会带空 projects）。
 * 同一任务上不得用空数组覆盖已加载的评论或关联项目；换任务时必须用新 payload。
 * 无 id 的非任务对象（如 relay 状态）不得整对象替换已加载任务。
 */
export const TASK_DETAIL_COMMENT_FEED_KEYS = [
  'comments',
  'ai_comments',
  'container_agent_comments',
]

export const TASK_DETAIL_PRESERVE_ARRAY_KEYS = [
  ...TASK_DETAIL_COMMENT_FEED_KEYS,
  'projects',
]

function preserveNonEmptyArray(next, current, incoming, key) {
  const incomingList = incoming[key]
  const currentList = current[key]
  const incomingHasItems = Array.isArray(incomingList) && incomingList.length > 0
  if (!incomingHasItems && Array.isArray(currentList) && currentList.length > 0) {
    next[key] = currentList
  }
}

export function mergeTaskDetailUpdate(current, incoming) {
  if (incoming == null) return incoming
  if (typeof incoming !== 'object') return current
  if (!current || typeof current !== 'object') return { ...incoming }

  const currentId = String(current.id || '').trim()
  const incomingId = String(incoming.id || '').trim()
  if (!incomingId) {
    return current
  }
  const sameTask = currentId !== '' && currentId === incomingId
  if (!sameTask) return { ...incoming }

  const next = { ...current, ...incoming }
  for (const key of TASK_DETAIL_PRESERVE_ARRAY_KEYS) {
    preserveNonEmptyArray(next, current, incoming, key)
  }
  if (current.comments_feeds_loaded === true && incoming.comments_feeds_loaded == null) {
    next.comments_feeds_loaded = true
  }
  return next
}
