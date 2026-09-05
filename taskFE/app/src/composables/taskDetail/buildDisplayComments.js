/**
 * 从评论的 mentions 数组中提取 installed_image 引用，派生 container_image_id / container_image_label。
 * 评论级镜像优先于任务级 imageId 回退，使每条评论的容器面板绑定自己的镜像上下文。
 * @param {object|null|undefined} comment
 * @returns {{ container_image_id: string, container_image_label: string }}
 */
export function extractImageMentionFromComment(comment) {
  if (!comment || typeof comment !== 'object') return {}
  const mentions = Array.isArray(comment.mentions) ? comment.mentions : []
  const img = mentions.find(
    (m) => m?.type === 'installed_image' && m?.id != null && String(m.id).trim() !== '',
  )
  if (img) {
    return {
      container_image_id: String(img.id),
      container_image_label: String(img.name || ''),
    }
  }
  // 回退：comment 本身可能已有 container_image_id（如 AI comment 或后端直给）
  if (comment.container_image_id != null && String(comment.container_image_id).trim() !== '') {
    return {
      container_image_id: String(comment.container_image_id),
      container_image_label: String(comment.container_image_label || comment.containerImageLabel || ''),
    }
  }
  return {}
}

/**
 * True when comment body is only the task id (accidental paste of URL / 辅助信息).
 * @param {unknown} content
 * @param {unknown} taskId
 * @returns {boolean}
 */
export function isCommentContentTaskIdEcho(content, taskId) {
  const c = content != null ? String(content).trim() : ''
  const t = taskId != null ? String(taskId).trim() : ''
  return Boolean(c && t && c === t)
}

/**
 * @param {object} c
 * @returns {string}
 */
function parentCommentIdOf(c) {
  return String(c?.parent_comment_id || c?.parentCommentId || '').trim()
}

/**
 * Nest container_agent rows under their human parent; orphans stay top-level.
 * Top-level order remains chronological by created_at.
 *
 * @param {Array<object>} flat
 * @returns {Array<object>}
 */
export function nestDisplayCommentsByParent(flat) {
  const list = Array.isArray(flat) ? flat : []
  const byId = new Map()
  for (const row of list) {
    if (row?.id != null) byId.set(String(row.id), row)
  }
  /** @type {Map<string, object[]>} */
  const childrenByParent = new Map()
  /** @type {object[]} */
  const roots = []

  for (const row of list) {
    const parentId = parentCommentIdOf(row)
    if (parentId && byId.has(parentId) && parentId !== String(row?.id || '')) {
      const bucket = childrenByParent.get(parentId) || []
      bucket.push({ ...row, children: [] })
      childrenByParent.set(parentId, bucket)
      continue
    }
    roots.push({ ...row, children: [] })
  }

  for (const root of roots) {
    const kids = childrenByParent.get(String(root.id)) || []
    kids.sort(
      (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
    )
    root.children = kids
  }

  roots.sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
  )
  return roots
}

/**
 * Merge human / AI / container-agent comments into one Feed timeline,
 * nesting container_agent under parent_comment_id when present.
 * @param {object|null|undefined} task
 * @returns {Array<object>}
 */
export function buildDisplayComments(task) {
  if (!task) return []
  const taskId = task.id != null ? String(task.id) : ''
  const keep = (c) => !isCommentContentTaskIdEcho(c?.content, taskId)
  const regular = (Array.isArray(task.comments) ? task.comments : [])
    .filter(keep)
    .map((c) => ({
      ...c,
      ...extractImageMentionFromComment(c),
      commentKind: 'user',
    }))
  const ai = (Array.isArray(task.ai_comments) ? task.ai_comments : [])
    .filter(keep)
    .map((c) => ({
      ...c,
      ...extractImageMentionFromComment(c),
      commentKind: 'ai',
    }))
  const containerAgent = (
    Array.isArray(task.container_agent_comments) ? task.container_agent_comments : []
  )
    .filter(keep)
    .map((c) => ({
      ...c,
      commentKind: 'container_agent',
    }))
  const flat = [...regular, ...ai, ...containerAgent].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
  )
  return nestDisplayCommentsByParent(flat)
}
