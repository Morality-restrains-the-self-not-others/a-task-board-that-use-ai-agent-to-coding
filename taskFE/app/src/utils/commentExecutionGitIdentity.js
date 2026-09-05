/**
 * 评论执行细节摘要：把评论 repo_identities 解析为可展示的 Git 身份芯片。
 * 身份目录缺失时仍展示「Git 身份」（表示发评时选过），不伪造人名。
 */
import { repoCloneIdentityOptionLabel } from './taskDetailBranchAndRepoUtils.js'

/**
 * @param {unknown} repoIdentities
 * @returns {{ repo_url: string, git_identity_id: string }[]}
 */
export function normalizeCommentRepoIdentities(repoIdentities) {
  if (!Array.isArray(repoIdentities)) return []
  const out = []
  const seen = new Set()
  for (const row of repoIdentities) {
    if (!row || typeof row !== 'object') continue
    const gitIdentityId = String(row.git_identity_id || '').trim()
    if (!gitIdentityId || seen.has(gitIdentityId)) continue
    seen.add(gitIdentityId)
    out.push({
      repo_url: String(row.repo_url || '').trim(),
      git_identity_id: gitIdentityId,
    })
  }
  return out
}

/**
 * @param {unknown} identityOptions
 * @param {string} gitIdentityId
 * @returns {object|null}
 */
function findIdentityOption(identityOptions, gitIdentityId) {
  if (!Array.isArray(identityOptions) || !gitIdentityId) return null
  return identityOptions.find((item) => String(item?.id || '').trim() === gitIdentityId) || null
}

/**
 * 摘要短文案：优先身份标签（如「默认身份」），再用户名，最后通用「Git 身份」。
 * @param {object|null} identity
 * @returns {string}
 */
export function commentGitIdentityCompactLabel(identity) {
  if (!identity || typeof identity !== 'object') return 'Git 身份'
  const label = String(identity.label || '').trim()
  if (label) return label
  if (identity.is_default) return '默认身份'
  const name = String(
    identity.git_user_name || identity.git_name || identity.name || '',
  ).trim()
  if (name) return name
  const full = repoCloneIdentityOptionLabel(identity)
  if (full && full !== '未命名身份') return full
  return 'Git 身份'
}

/**
 * @param {unknown} repoIdentities 评论级 repo_identities，非空时优先
 * @param {unknown} identityOptions
 * @param {unknown} fallbackRepoIdentities 评论为空时回退的任务级仓库身份（auto_run 首评无 composer 选择）
 * @returns {{ id: string, label: string, title: string }[]}
 */
export function commentGitIdentitySummaryItems(
  repoIdentities,
  identityOptions,
  fallbackRepoIdentities = [],
) {
  const own = normalizeCommentRepoIdentities(repoIdentities)
  const rows = own.length > 0 ? own : normalizeCommentRepoIdentities(fallbackRepoIdentities)
  return rows.map((row) => {
    const identity = findIdentityOption(identityOptions, row.git_identity_id)
    const label = commentGitIdentityCompactLabel(identity)
    const title = identity
      ? (repoCloneIdentityOptionLabel(identity) || row.git_identity_id)
      : row.git_identity_id
    return {
      id: row.git_identity_id,
      label,
      text: label === 'Git 身份' ? 'Git 身份' : `Git 身份 · ${label}`,
      title,
    }
  })
}
