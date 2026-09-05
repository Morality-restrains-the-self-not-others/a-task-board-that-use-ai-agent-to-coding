/** Git 仓库 OAuth 授权状态展示（与后端 TokenStatus 对齐） */

export const GIT_REPO_OAUTH_STATUS = {
  TOKEN_AVAILABLE: 'token_available',
  NOT_BOUND: 'not_bound',
  TOKEN_ERROR: 'token_error',
  NOT_APPLICABLE: 'not_applicable',
}

const STATUS_LABELS = {
  [GIT_REPO_OAUTH_STATUS.TOKEN_AVAILABLE]: '已授权',
  [GIT_REPO_OAUTH_STATUS.NOT_BOUND]: '需要授权',
  [GIT_REPO_OAUTH_STATUS.TOKEN_ERROR]: '授权异常',
  [GIT_REPO_OAUTH_STATUS.NOT_APPLICABLE]: '无需 OAuth',
}

const STATUS_BADGE_CLASSES = {
  [GIT_REPO_OAUTH_STATUS.TOKEN_AVAILABLE]:
    'bg-green-50 text-green-700 border-green-200',
  [GIT_REPO_OAUTH_STATUS.NOT_BOUND]:
    'bg-amber-50 text-amber-800 border-amber-200',
  [GIT_REPO_OAUTH_STATUS.TOKEN_ERROR]:
    'bg-red-50 text-red-700 border-red-200',
  [GIT_REPO_OAUTH_STATUS.NOT_APPLICABLE]:
    'bg-gray-50 text-gray-600 border-gray-200',
}

export function resolveGitRepoOAuthStatusLabel(tokenStatus, { loading = false } = {}) {
  if (loading) return '检查中…'
  const key = String(tokenStatus || '').trim()
  if (!key) return '检查中…'
  return STATUS_LABELS[key] || '未知'
}

export const GIT_REPO_OAUTH_TOKEN_ERROR_HINT =
  '当前授权无法访问该仓库或已失效，请点击「重试」重新绑定'

export function gitRepoOAuthStatusHint(tokenStatus) {
  const key = String(tokenStatus || '').trim()
  if (key === GIT_REPO_OAUTH_STATUS.TOKEN_ERROR) {
    return GIT_REPO_OAUTH_TOKEN_ERROR_HINT
  }
  return ''
}

export function gitRepoOAuthStatusBadgeClass(tokenStatus, { loading = false } = {}) {
  if (loading) return 'bg-gray-100 text-gray-500 border-gray-200'
  const key = String(tokenStatus || '').trim()
  return STATUS_BADGE_CLASSES[key] || 'bg-gray-50 text-gray-600 border-gray-200'
}

export function isPlaceholderGitRepoOAuthStatus(tokenStatus) {
  const key = String(tokenStatus || '').trim()
  return !key || key === GIT_REPO_OAUTH_STATUS.NOT_APPLICABLE
}

/**
 * 排序权重：未授权 / 授权异常置顶，便于用户优先处理。
 * 数值越小越靠前。
 */
export function gitRepoOAuthNeedsAttentionRank(tokenStatus) {
  const key = String(tokenStatus || '').trim()
  if (key === GIT_REPO_OAUTH_STATUS.NOT_BOUND) return 0
  if (key === GIT_REPO_OAUTH_STATUS.TOKEN_ERROR) return 1
  if (!key) return 2
  if (key === GIT_REPO_OAUTH_STATUS.NOT_APPLICABLE) return 3
  if (key === GIT_REPO_OAUTH_STATUS.TOKEN_AVAILABLE) return 4
  return 5
}

/**
 * 按 OAuth 关注优先级稳定排序（未授权置顶）。
 * @param {Array} repos
 * @param {(row: any) => string} getTokenStatus
 */
export function sortReposByOAuthAttention(repos, getTokenStatus) {
  const list = Array.isArray(repos) ? repos : []
  const getStatus = typeof getTokenStatus === 'function' ? getTokenStatus : () => ''
  return list
    .map((row, index) => ({ row, index, rank: gitRepoOAuthNeedsAttentionRank(getStatus(row)) }))
    .sort((a, b) => (a.rank !== b.rank ? a.rank - b.rank : a.index - b.index))
    .map((item) => item.row)
}
