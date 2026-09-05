/** Git 网站授权：connection 摘要归一化（供 UserGitSiteOAuthSettings 等复用） */

export const splitScope = (raw) => {
  if (raw == null || String(raw).trim() === '') return []
  return String(raw)
    .trim()
    .split(/[\s,]+/)
    .map((s) => s.trim())
    .filter(Boolean)
}

export const normalizeConnectionItem = (row = {}) => {
  const rawUid = row?.github_user_id
  const uid = rawUid == null ? '' : String(rawUid).trim()
  return {
    connected: Boolean(row?.connected),
    github_login: row?.github_login || null,
    github_user_id: uid || null,
    scope: row?.scope || null,
    updated_at: row?.updated_at || null,
  }
}

export const normalizeConnectionStatus = (payload = {}) => {
  let connections = []
  if (Array.isArray(payload.connections)) {
    connections = payload.connections.map((row) => normalizeConnectionItem(row))
  } else if (payload.github_login || payload.github_user_id) {
    connections = [
      normalizeConnectionItem({
        connected: payload.connected,
        github_login: payload.github_login,
        github_user_id: payload.github_user_id,
        scope: payload.scope,
      }),
    ]
  }
  return {
    connected: Boolean(payload.connected),
    github_login: payload.github_login,
    github_user_id: payload.github_user_id,
    scope: payload.scope,
    authorize_scope: payload.authorize_scope,
    connections,
  }
}

export const isConnectionPayloadUsable = (payload) =>
  payload && typeof payload === 'object' && Object.prototype.hasOwnProperty.call(payload, 'connected')
