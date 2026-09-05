/**
 * WorkPanel：按可访问工作空间的人或小组过滤看板任务（owner ∪ assignees）。
 */

/**
 * @typedef {'person' | 'group' | null | undefined} AccessFilterKind
 * @typedef {{
 *   kind?: AccessFilterKind,
 *   id?: string|number|null,
 *   label?: string,
 *   memberIds?: Array<string|number|null|undefined>,
 * }} AccessFilter
 */

/**
 * @param {unknown} filter
 * @returns {AccessFilter | null}
 */
export function normalizeAccessFilter(filter) {
  if (filter == null || typeof filter !== 'object') {
    return null
  }
  const kind = filter.kind
  if (kind !== 'person' && kind !== 'group') {
    return null
  }
  const id = filter.id != null && String(filter.id).trim() !== '' ? String(filter.id) : ''
  if (!id) {
    return null
  }
  const memberIds = Array.isArray(filter.memberIds)
    ? filter.memberIds.map((x) => String(x)).filter((x) => x !== '')
    : []
  return {
    kind,
    id,
    label: filter.label != null ? String(filter.label) : '',
    memberIds,
  }
}

/**
 * @param {AccessFilter | null | undefined} filter
 * @returns {string}
 */
export function accessFilterChipLabel(filter) {
  const f = normalizeAccessFilter(filter)
  if (!f) {
    return ''
  }
  const name = (f.label || f.id).trim()
  if (f.kind === 'person') {
    return name ? `过滤：${name}` : '过滤：人'
  }
  return name ? `过滤：小组 ${name}` : '过滤：小组'
}

/**
 * @param {{ owner?: unknown, assignees?: unknown }} todo
 * @returns {Set<string>}
 */
export function collectTaskParticipantMemberIds(todo) {
  const out = new Set()
  if (todo == null || typeof todo !== 'object') {
    return out
  }
  if (todo.owner != null && String(todo.owner).trim() !== '') {
    out.add(String(todo.owner))
  }
  const assignees = Array.isArray(todo.assignees) ? todo.assignees : []
  for (const a of assignees) {
    if (a != null && String(a).trim() !== '') {
      out.add(String(a))
    }
  }
  return out
}

/**
 * @param {Array<{ owner?: unknown, assignees?: unknown }>} todos
 * @param {AccessFilter | null | undefined} filter
 * @returns {Array}
 */
export function filterTodosByAccess(todos, filter) {
  const list = Array.isArray(todos) ? todos : []
  const f = normalizeAccessFilter(filter)
  if (!f) {
    return list
  }
  const wanted = new Set((f.memberIds || []).map((x) => String(x)).filter(Boolean))
  if (wanted.size === 0) {
    return []
  }
  return list.filter((todo) => {
    const participants = collectTaskParticipantMemberIds(todo)
    for (const id of participants) {
      if (wanted.has(id)) {
        return true
      }
    }
    return false
  })
}

/**
 * @param {unknown} value
 * @returns {string}
 */
function trimmedText(value) {
  return value != null ? String(value).trim() : ''
}

/**
 * 人员展示名：公司成员名优先（与 DisplayName 一致），再协作人池，再 username。
 * @param {{ member_name?: unknown, username?: unknown }} ui
 * @param {string} memberId
 * @param {Map<string, { member_name: string, username: string }>} collabLabelByMember
 * @returns {string}
 */
export function resolveAccessPersonLabel(ui, memberId, collabLabelByMember = new Map()) {
  const fromUiMember = trimmedText(ui?.member_name)
  if (fromUiMember) return fromUiMember
  const collab = collabLabelByMember.get(String(memberId || ''))
  if (collab?.member_name) return collab.member_name
  const fromUiUsername = trimmedText(ui?.username)
  if (fromUiUsername) return fromUiUsername
  if (collab?.username) return collab.username
  return String(memberId || '')
}

/**
 * 从 workspace-permissions 行解析可选人/小组。
 * 人必须有 company_member_id（或能经 collaborators 用 user_id 映射）。
 *
 * @param {unknown} permissionsPayload
 * @param {Array<{ id?: unknown, user?: unknown, member_name?: unknown, username?: unknown }>} [collaborators]
 * @returns {{ people: Array<{ id: string, label: string, userId: string }>, groups: Array<{ id: string, label: string }> }}
 */
export function parseAccessFilterSubjects(permissionsPayload, collaborators = []) {
  const people = []
  const groups = []
  const seenPeople = new Set()
  const seenGroups = new Set()
  const collabByUser = new Map()
  /** @type {Map<string, { member_name: string, username: string }>} */
  const collabLabelByMember = new Map()
  for (const c of Array.isArray(collaborators) ? collaborators : []) {
    if (c == null || typeof c !== 'object') continue
    const uid = c.user != null ? String(c.user) : ''
    const mid = c.id != null ? String(c.id) : ''
    if (uid && mid) {
      collabByUser.set(uid, mid)
    }
    if (mid) {
      collabLabelByMember.set(mid, {
        member_name: trimmedText(c.member_name),
        username: trimmedText(c.username),
      })
    }
  }
  const rows = Array.isArray(permissionsPayload) ? permissionsPayload : []
  for (const perm of rows) {
    if (perm == null || typeof perm !== 'object') continue
    const ui = perm.user_info
    if (ui && typeof ui === 'object') {
      const userId = ui.id != null ? String(ui.id) : ''
      let memberId =
        ui.company_member_id != null && String(ui.company_member_id).trim() !== ''
          ? String(ui.company_member_id)
          : ''
      if (!memberId && userId && collabByUser.has(userId)) {
        memberId = collabByUser.get(userId)
      }
      if (!memberId || seenPeople.has(memberId)) continue
      seenPeople.add(memberId)
      const label = resolveAccessPersonLabel(ui, memberId, collabLabelByMember)
      people.push({ id: memberId, label: String(label), userId })
      continue
    }
    const gi = perm.group_info
    if (gi && typeof gi === 'object') {
      const gid = gi.id != null ? String(gi.id) : ''
      if (!gid || seenGroups.has(gid)) continue
      seenGroups.add(gid)
      const label = gi.name != null && String(gi.name).trim() ? String(gi.name) : gid
      groups.push({ id: gid, label })
    }
  }
  return { people, groups }
}

/**
 * 小组成员 user_id → company_member_id（经 collaborators）。
 *
 * @param {Array<{ user?: unknown }>} groupMembers
 * @param {Array<{ id?: unknown, user?: unknown }>} collaborators
 * @returns {string[]}
 */
export function resolveCompanyMemberIdsFromGroupMembers(groupMembers, collaborators) {
  const byUser = new Map()
  for (const c of Array.isArray(collaborators) ? collaborators : []) {
    if (c == null || typeof c !== 'object') continue
    const uid = c.user != null ? String(c.user) : ''
    const mid = c.id != null ? String(c.id) : ''
    if (uid && mid) byUser.set(uid, mid)
  }
  const out = []
  const seen = new Set()
  for (const m of Array.isArray(groupMembers) ? groupMembers : []) {
    if (m == null || typeof m !== 'object') continue
    const uid = m.user != null ? String(m.user) : ''
    if (!uid) continue
    const mid = byUser.get(uid)
    if (!mid || seen.has(mid)) continue
    seen.add(mid)
    out.push(mid)
  }
  return out
}

/**
 * 切换：同 kind+id 再点清除；否则切换到新过滤。
 * @param {AccessFilter | null | undefined} current
 * @param {AccessFilter | null | undefined} next
 * @returns {AccessFilter | null}
 */
export function toggleAccessFilter(current, next) {
  const cur = normalizeAccessFilter(current)
  const n = normalizeAccessFilter(next)
  if (!n) {
    return cur
  }
  if (cur && cur.kind === n.kind && cur.id === n.id) {
    return null
  }
  return n
}
