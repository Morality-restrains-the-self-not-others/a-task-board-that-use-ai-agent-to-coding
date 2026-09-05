import { GIT_REPO_OAUTH_STATUS } from './gitRepoOAuthStatusUtils.js'

/**
 * 项目级「自动克隆子仓库」开关。缺省 true，与 project_entries / CreateProject 默认一致。
 * @param {unknown} value
 * @returns {boolean}
 */
export function isAutoCloneNestedReposEnabled(value) {
  if (value === false || value === 0 || value === '0' || value === 'false') return false
  return true
}

/**
 * 项目详情「是否允许自动运行」的 Git 授权门禁。
 * 父仓授权异常始终阻断。子仓授权异常 / 子仓列表失败仅在「自动克隆子仓库」开启时阻断。
 *
 * @param {{
 *   tokenStatuses?: string[],
 *   nestedTokenStatuses?: string[],
 *   nestedError?: string,
 *   nestedErrorTraceId?: string,
 *   nestedLoading?: boolean,
 *   autoCloneNestedRepos?: unknown,
 * }} [opts]
 * @returns {{
 *   blocked: boolean,
 *   code: string,
 *   message: string,
 *   displayLabel: string,
 *   traceId: string,
 * }}
 */
export function resolveAutoRunGitAuthBlock({
  tokenStatuses = [],
  nestedTokenStatuses = [],
  nestedError = '',
  nestedErrorTraceId = '',
  nestedLoading = false,
  autoCloneNestedRepos = true,
} = {}) {
  const cloneNested = isAutoCloneNestedReposEnabled(autoCloneNestedRepos)
  const statuses = [
    ...(Array.isArray(tokenStatuses) ? tokenStatuses : []),
    ...(cloneNested && Array.isArray(nestedTokenStatuses) ? nestedTokenStatuses : []),
  ]
  const hasTokenError = statuses.some(
    (status) => String(status || '').trim() === GIT_REPO_OAUTH_STATUS.TOKEN_ERROR,
  )
  if (hasTokenError) {
    return {
      blocked: true,
      code: 'AUTO_RUN_GIT_AUTH_ERROR',
      message: '存在 Git 仓库授权异常，自动运行无法启动；请先完成授权后再启用',
      displayLabel: '无法启动',
      // 授权态聚合阻断，无单次请求可对齐 — 省略 data-traceId
      traceId: '',
    }
  }

  const err = String(nestedError || '').trim()
  if (cloneNested && !nestedLoading && err) {
    return {
      blocked: true,
      code: 'AUTO_RUN_NESTED_REPOS_UNAVAILABLE',
      message: '无法获取子 Git 仓库列表，自动运行无法启动；请先解决授权或网络问题后重试',
      displayLabel: '无法启动',
      // 与子仓列表请求同源，透传其 traceId 供门禁提示挂 data-traceId
      traceId: String(nestedErrorTraceId || '').trim(),
    }
  }

  return {
    blocked: false,
    code: '',
    message: '',
    displayLabel: '',
    traceId: '',
  }
}

/** 展示文案：门禁阻断时固定「无法启动」，否则按是否启用 */
export function resolveDefaultAutoRunDisplayLabel({
  enabled = false,
  gitAuthBlock = null,
} = {}) {
  if (gitAuthBlock?.blocked) {
    return gitAuthBlock.displayLabel || '无法启动'
  }
  return enabled ? '已启用' : '未启用'
}
