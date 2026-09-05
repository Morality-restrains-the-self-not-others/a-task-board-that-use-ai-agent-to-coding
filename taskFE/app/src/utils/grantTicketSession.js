const STORAGE_KEY = 'gitOauthGrantTickets'

function readStore() {
  if (typeof sessionStorage === 'undefined') return {}
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    const parsed = raw ? JSON.parse(raw) : {}
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function writeStore(store) {
  if (typeof sessionStorage === 'undefined') return
  sessionStorage.setItem(STORAGE_KEY, JSON.stringify(store || {}))
}

export function gitsiteFromRepoUrl(repoUrl) {
  const raw = String(repoUrl || '').trim()
  if (!raw) return ''
  try {
    if (raw.includes('://')) {
      const parsed = new URL(raw)
      if (parsed.protocol === 'ssh:') return String(parsed.hostname || '').toLowerCase()
      return parsed.host.toLowerCase()
    }
  } catch {
    return ''
  }
  const at = raw.indexOf('@')
  if (at >= 0) {
    const rest = raw.slice(at + 1)
    const colon = rest.indexOf(':')
    if (colon > 0) return rest.slice(0, colon).toLowerCase()
  }
  return ''
}

export function rememberGrantTicket(ticket, repoUrl) {
  const id = String(ticket || '').trim()
  const site = gitsiteFromRepoUrl(repoUrl)
  if (!id) return
  const store = readStore()
  if (site) store[site] = id
  store['*'] = id
  writeStore(store)
}

/**
 * grant_ticket 服务端单次消费：create-project / create-task / fork 携带并成功后，
 * 该 ticket 即失效，不能再作为「已授权」证据（OPT-20260902-025）。
 * 只删除匹配的 ticket id（site 键与通配 * 均移除），不影响其他仍待用的 ticket。
 * @param {string|string[]} tickets
 */
export function consumeSessionGrantTickets(tickets) {
  const ids = new Set(
    (Array.isArray(tickets) ? tickets : [tickets])
      .map((ticket) => String(ticket || '').trim())
      .filter(Boolean),
  )
  if (ids.size === 0) return
  const store = readStore()
  let changed = false
  for (const key of Object.keys(store)) {
    if (ids.has(String(store[key] || '').trim())) {
      delete store[key]
      changed = true
    }
  }
  if (changed) writeStore(store)
}

/** 单 ticket 便捷版，见 {@link consumeSessionGrantTickets}。 */
export function consumeSessionGrantTicket(ticket) {
  consumeSessionGrantTickets([ticket])
}

export function rememberGrantTicketFromSearch(search, repoUrl) {
  const q = new URLSearchParams(String(search || '').replace(/^\?/, ''))
  const ticket = String(q.get('grant_ticket') || '').trim()
  if (!ticket) return ''
  rememberGrantTicket(ticket, repoUrl || q.get('repo_url') || '')
  return ticket
}

/**
 * 判断该仓库是否持有「仍待用」的 grant_ticket。
 * gitsite 可解析时只认该 site 的 ticket：通配 * 只是最近一张、可能属于别的站点，
 * 若源任务创建已把本 site 的 ticket 消费掉，不能再用它冒充已绑定（OPT-20260902-025）。
 */
export function hasSessionGrantForRepo(repoUrl) {
  const store = readStore()
  const site = gitsiteFromRepoUrl(repoUrl)
  if (site) return Boolean(store[site])
  return Boolean(store['*'])
}

/**
 * 取该仓库 site 对应的待用 ticket（site 不可解析时才回退通配）。
 */
export function sessionGrantTicketForRepo(repoUrl) {
  const store = readStore()
  const site = gitsiteFromRepoUrl(repoUrl)
  if (site) return String(store[site] || '')
  return String(store['*'] || '')
}

export function sessionGrantTicketAny() {
  const store = readStore()
  if (store['*']) return String(store['*'])
  for (const [key, value] of Object.entries(store)) {
    if (key === '*') continue
    const id = String(value || '').trim()
    if (id) return id
  }
  return ''
}

export function gitRepoUrlsFromPayload(repos) {
  const out = []
  for (const item of Array.isArray(repos) ? repos : []) {
    if (typeof item === 'string') {
      const url = item.trim()
      if (url) out.push(url)
      continue
    }
    if (item && typeof item === 'object') {
      const url = String(item.url || item.repo_url || '').trim()
      if (url) out.push(url)
    }
  }
  return out
}

export function collectSessionGrantTickets(repoUrls, extraTickets = []) {
  const seen = new Set()
  const out = []
  const add = (raw) => {
    const id = String(raw || '').trim()
    if (!id || seen.has(id)) return
    seen.add(id)
    out.push(id)
  }
  for (const url of Array.isArray(repoUrls) ? repoUrls : []) {
    add(sessionGrantTicketForRepo(url))
  }
  add(sessionGrantTicketAny())
  for (const extra of Array.isArray(extraTickets) ? extraTickets : []) {
    add(extra)
  }
  return out
}
