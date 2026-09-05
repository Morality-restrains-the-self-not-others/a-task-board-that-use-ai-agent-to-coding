/**
 * 本机单账号槽 + 活跃账号（auth token 来源）。
 * OPT-20260807-015：多账号槽位已彻底下线，仅保留当前会话单账号；写入即整体替换。
 * 仅存浏览器 localStorage。跨 Tab 同步通过原生 storage 事件（onAccountStateChanged），无外部依赖。
 */

export const SAVED_ACCOUNTS_STORAGE_KEY = 'savedAccounts'
export const ACTIVE_ACCOUNT_STORAGE_KEY = 'activeAccountUserId'
export const MAX_SAVED_ACCOUNTS = 5

function readStorage() {
  if (typeof localStorage === 'undefined') return []
  const raw = localStorage.getItem(SAVED_ACCOUNTS_STORAGE_KEY)
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function writeStorage(list) {
  if (typeof localStorage === 'undefined') {
    throw new Error('localStorage unavailable')
  }
  localStorage.setItem(SAVED_ACCOUNTS_STORAGE_KEY, JSON.stringify(list))
}

function normalizeSlot(slot) {
  const userId = String(slot?.userId ?? '').trim()
  const token = String(slot?.token ?? '').trim()
  if (!userId) {
    throw new Error('AccountSlot.userId is required')
  }
  if (!token) {
    throw new Error('AccountSlot.token is required')
  }
  return {
    userId,
    username: String(slot?.username ?? '').trim(),
    avatarUrl: slot?.avatarUrl != null && String(slot.avatarUrl).trim() !== ''
      ? String(slot.avatarUrl).trim()
      : null,
    token,
    addedAt: Number.isFinite(Number(slot?.addedAt)) ? Number(slot.addedAt) : Date.now(),
  }
}

function dedupeByUserId(list) {
  const seen = new Set()
  return list.filter((s) => {
    if (seen.has(s.userId)) return false
    seen.add(s.userId)
    return true
  })
}

/**
 * 按 username（大小写不敏感）二次去重：同名账号仅保留最近添加的槽位。
 * 仅对 username 非空的条目生效 —— 空 username 条目保留 userId 维度的唯一性。
 */
function dedupeByUsername(list) {
  const seen = new Map()
  for (const s of list) {
    if (s.username) {
      const key = s.username.toLowerCase()
      const existing = seen.get(key)
      if (!existing || s.addedAt > existing.addedAt) {
        seen.set(key, s)
      }
    } else {
      // 无 username 的条目按 userId 保留，避免空串碰撞合并
      seen.set(`__uid__${s.userId}`, s)
    }
  }
  return Array.from(seen.values())
}

/**
 * 列出所有已保存账号，自动去重并在存储损坏/冗余时自愈。
 * - 逐条容错：单条 normalizeSlot 失败不会导致整列崩溃
 * - userId 去重 + username 去重：解决多槽位同账号重复显示
 * - 读时自愈：若清理后列表与原始数据不同，写回 localStorage
 */
export function listSavedAccounts() {
  const raw = readStorage()
  const normalized = raw.map((s) => {
    try {
      return normalizeSlot(s)
    } catch {
      return null
    }
  }).filter(Boolean)

  const deduped = dedupeByUsername(dedupeByUserId(normalized))

  // 自愈：去重后条目数减少或内容变化时写回干净副本
  if (deduped.length !== raw.length) {
    try {
      writeStorage(deduped)
    } catch {
      // 静默失败；下次 upsert 仍会清理
    }
  }

  return deduped
}

/**
 * 插入或更新同 userId 槽。
 * OPT-20260807-015：多账号槽位已彻底下线，本机仅保留当前会话单账号 ——
 * 写入即整体替换（登录/切换新账号时旧槽位一并清除，历史多账号数据在下一次写入时收敛）。
 * @returns {{ list: object[], upserted: object, isNew: boolean }}
 */
export function upsertSavedAccount(slot) {
  const next = normalizeSlot(slot)
  const existing = listSavedAccounts().find((s) => s.userId === next.userId)
  const isNew = !existing
  const upserted = isNew ? next : { ...existing, ...next, addedAt: existing.addedAt }
  writeStorage([upserted])
  return { list: [upserted], upserted, isNew }
}

export function removeSavedAccount(userId) {
  const uid = String(userId ?? '').trim()
  if (!uid) {
    throw new Error('userId is required')
  }
  const list = listSavedAccounts().filter((s) => s.userId !== uid)
  writeStorage(list)
  return list
}

export function getSavedAccount(userId) {
  const uid = String(userId ?? '').trim()
  return listSavedAccounts().find((s) => s.userId === uid) || null
}

export function clearSavedAccounts() {
  writeStorage([])
}

// ---- 活跃账号 ----

function readActiveUserId() {
  if (typeof localStorage === 'undefined') return ''
  try {
    return String(localStorage.getItem(ACTIVE_ACCOUNT_STORAGE_KEY) || '').trim()
  } catch {
    return ''
  }
}

function writeActiveUserId(userId) {
  if (typeof localStorage === 'undefined') return
  try {
    if (userId) {
      localStorage.setItem(ACTIVE_ACCOUNT_STORAGE_KEY, String(userId))
    } else {
      localStorage.removeItem(ACTIVE_ACCOUNT_STORAGE_KEY)
    }
  } catch { /* ignore */ }
}

/**
 * 获取当前活跃账号。
 * 活跃槽不存在时（登出清理 / 旧数据无活跃标记）回退最近添加的账号，
 * 保持既有登录态连续（新近登录的账号即最可能仍有效的会话）。
 */
export function getActiveAccount() {
  const uid = readActiveUserId()
  if (uid) {
    const slot = getSavedAccount(uid)
    if (slot) return slot
  }
  const list = listSavedAccounts()
  if (list.length === 0) return null
  return list.reduce((a, b) => ((a.addedAt || 0) >= (b.addedAt || 0) ? a : b))
}

/**
 * 获取活跃账号的 token（apiUtils 认证头来源）。
 */
export function getActiveToken() {
  const acc = getActiveAccount()
  return acc ? acc.token : null
}

/**
 * 获取活跃账号的 userId。
 */
export function getCurrentUserId() {
  const acc = getActiveAccount()
  return acc ? acc.userId : null
}

/**
 * 设置活跃账号（upsert + 切换），用于登录成功后。
 */
export function setActiveAccount(slot) {
  const result = upsertSavedAccount(slot)
  writeActiveUserId(String(slot.userId))
  return result
}

/**
 * 监听账号状态变更（跨 Tab 同步：storage 事件仅在其他 Tab 触发）。
 *
 * @param {Function} callback — ({ event: string }) => void
 * @returns {Function} 取消监听的函数
 */
export function onAccountStateChanged(callback) {
  if (typeof window === 'undefined') return () => {}
  function handler(event) {
    if (event.key !== SAVED_ACCOUNTS_STORAGE_KEY && event.key !== ACTIVE_ACCOUNT_STORAGE_KEY) {
      return
    }
    try { callback({ event: 'authStateChanged' }) } catch { /* ignore */ }
  }
  window.addEventListener('storage', handler, false)
  return () => window.removeEventListener('storage', handler)
}
