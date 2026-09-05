/**
 * 看板任务卡人员展示：
 * - 操作员 operator：单人
 * - 负责人 owner：单人
 * - 协作者 assignees：可多人
 */

/**
 * @param {Array<{ id?: unknown, member_name?: unknown, username?: unknown, email?: unknown }>|null|undefined} collaborators
 * @returns {Record<string, string>}
 */
export function buildCollaboratorNameById(collaborators) {
  const map = Object.create(null)
  if (!Array.isArray(collaborators)) return map
  for (const member of collaborators) {
    if (member == null) continue
    const name = String(member.member_name || member.username || member.email || '').trim()
    const memberId = member.id != null && member.id !== '' ? String(member.id) : ''
    const userId = member.user != null && member.user !== ''
      ? String(member.user)
      : (member.user_id != null && member.user_id !== '' ? String(member.user_id) : '')
    if (!memberId && !userId) continue
    const display = name || memberId || userId
    if (memberId) map[memberId] = display
    if (userId) map[userId] = display
  }
  return map
}

/**
 * 评论作者展示名：优先公司成员昵称（协作 map），再回退 created_by.username。
 * @param {{ id?: unknown, username?: unknown }|null|undefined} createdBy
 * @param {Record<string, string>|null|undefined} nameById
 * @param {string} [fallback='']
 * @returns {string}
 */
export function resolveCommentAuthorDisplayName(createdBy, nameById, fallback = '') {
  if (!createdBy || typeof createdBy !== 'object') return fallback
  const map = nameById && typeof nameById === 'object' ? nameById : {}
  const id = createdBy.id != null && createdBy.id !== '' ? String(createdBy.id) : ''
  const username = createdBy.username != null ? String(createdBy.username).trim() : ''
  if (id && map[id]) return map[id]
  if (username && map[username]) return map[username]
  if (username) return username
  return fallback
}

/**
 * 协作成员头像：按成员 id / user id 双索引。
 * 优先公司 member_avatar_url，其次个人 avatar_url。
 * @param {Array<{ id?: unknown, user?: unknown, user_id?: unknown, member_avatar_url?: unknown, avatar_url?: unknown }>|null|undefined} collaborators
 * @returns {Record<string, string>}
 */
export function buildCollaboratorAvatarById(collaborators) {
  const map = Object.create(null)
  if (!Array.isArray(collaborators)) return map
  for (const member of collaborators) {
    if (member == null) continue
    const url = String(
      member.member_avatar_url || member.avatar_url || '',
    ).trim()
    if (!url) continue
    const memberId = member.id != null && member.id !== '' ? String(member.id) : ''
    const userId = member.user != null && member.user !== ''
      ? String(member.user)
      : (member.user_id != null && member.user_id !== '' ? String(member.user_id) : '')
    if (memberId) map[memberId] = url
    if (userId) map[userId] = url
  }
  return map
}

/**
 * 评论作者头像：优先 created_by.avatar_url，再协作 map，空则返回 ''（调用方回退 initials）。
 * @param {{ id?: unknown, username?: unknown, avatar_url?: unknown }|null|undefined} createdBy
 * @param {Record<string, string>|null|undefined} avatarById
 * @returns {string}
 */
export function resolveCommentAuthorAvatar(createdBy, avatarById) {
  if (!createdBy || typeof createdBy !== 'object') return ''
  const direct = createdBy.avatar_url != null ? String(createdBy.avatar_url).trim() : ''
  if (direct) return direct
  const map = avatarById && typeof avatarById === 'object' ? avatarById : {}
  const id = createdBy.id != null && createdBy.id !== '' ? String(createdBy.id) : ''
  if (id && map[id]) return map[id]
  const username = createdBy.username != null ? String(createdBy.username).trim() : ''
  if (username && map[username]) return map[username]
  return ''
}

/**
 * @param {unknown} memberId
 * @param {Record<string, string>|null|undefined} nameById
 * @returns {string}
 */
export function resolveMemberDisplayName(memberId, nameById) {
  if (memberId == null || memberId === '') return ''
  const id = String(memberId)
  const map = nameById && typeof nameById === 'object' ? nameById : {}
  return map[id] || `ID:${id}`
}

/**
 * 单人角色（操作员 / 负责人）；空则「未指派」。
 * @param {unknown} memberId
 * @param {Record<string, string>|null|undefined} nameById
 * @returns {string}
 */
export function formatSingleMemberDisplay(memberId, nameById) {
  if (memberId == null || memberId === '') return '未指派'
  return resolveMemberDisplayName(memberId, nameById) || '未指派'
}

/** @deprecated 使用 formatSingleMemberDisplay；保留别名避免旧 import 断裂 */
export function formatOperatorDisplay(operatorId, nameById) {
  return formatSingleMemberDisplay(operatorId, nameById)
}

/** @deprecated 使用 formatSingleMemberDisplay */
export function formatOwnerDisplay(ownerId, nameById) {
  return formatSingleMemberDisplay(ownerId, nameById)
}

/**
 * 协作者：最多展示 2 名，超出用「+N」；空则「未指派」。
 * 全量名单可经返回对象的 `title`（顿号拼接）做悬停提示。
 * @param {unknown} assignees
 * @param {Record<string, string>|null|undefined} nameById
 * @param {{ maxVisible?: number }} [opts]
 * @returns {string}
 */
export function formatCollaboratorsDisplay(assignees, nameById, opts = {}) {
  const ids = Array.isArray(assignees) ? assignees : []
  if (ids.length === 0) return '未指派'
  const names = ids.map((raw) => resolveMemberDisplayName(raw, nameById)).filter(Boolean)
  if (!names.length) return '未指派'
  const maxVisible = Number.isFinite(opts.maxVisible) ? Math.max(0, Number(opts.maxVisible)) : 2
  if (names.length <= maxVisible) return names.join('、')
  const head = names.slice(0, maxVisible).join('、')
  const extra = names.length - maxVisible
  return `${head} +${extra}`
}

/**
 * 协作者全量顿号文案（供 title / aria）。
 * @param {unknown} assignees
 * @param {Record<string, string>|null|undefined} nameById
 * @returns {string}
 */
export function formatCollaboratorsDisplayFull(assignees, nameById) {
  const ids = Array.isArray(assignees) ? assignees : []
  if (ids.length === 0) return '未指派'
  return ids.map((raw) => resolveMemberDisplayName(raw, nameById)).filter(Boolean).join('、') || '未指派'
}
