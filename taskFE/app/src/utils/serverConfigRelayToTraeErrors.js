import { mapRelayToTraeStatusErrorMessage } from './relayToTraeUtils.js'

export const RELAY_TO_TRAE_ERROR_MESSAGE_BY_CODE = Object.freeze({
  TOKEN_ACCESS_EXPIRED: '凭证已过期，正在准备重试',
  TOKEN_ACCESS_INVALID: '凭证无效，请重新准备启动环境',
  TOKEN_PERSIST_FAILED: '换票落盘失败：请检查 ONLINE_PROJECT_STATE_ROOT 磁盘权限后重试',
  TOKEN_SCOPE_MISMATCH: '任务上下文不一致，请刷新页面后重试',
  TOKEN_EXCHANGE_ALREADY_DONE: '检测到已完成换票，正在走补偿流程',
  BUSINESS_API_ENDPOINT_INVALID: '业务端点配置异常，请检查配置后重试',
  RELAY_DOWNSTREAM_UNAVAILABLE: 'relay 服务暂不可用，请稍后重试',
  REPO_CLONE_TOKEN_REFRESH_FAILED: '仓库克隆 token 换发失败',
})

export const repoCredentialGuideMessage =
  '私有仓库克隆需要 Git OAuth。可在「提交并运行」评论区绑定；未绑定仍可发评，克隆可能失败。'

export function summarizeTokenRefreshFailuresForUi(payload) {
  const rows = Array.isArray(payload?.token_refresh_failures) ? payload.token_refresh_failures : []
  const details = rows
    .map((row) => String(row?.detail || row?.repo_url || '').trim())
    .filter(Boolean)
  if (!details.length) {
    return ''
  }
  const preview = details.slice(0, 2).join('；')
  return preview
}

export function resolveRelayToTraeErrorMessage(payload, fallback = 'relayToTrae 启动失败') {
  const obj = payload && typeof payload === 'object' ? payload : {}
  const code = String(obj.error_code || obj.errorCode || '').trim()
  if (code && RELAY_TO_TRAE_ERROR_MESSAGE_BY_CODE[code]) {
    return RELAY_TO_TRAE_ERROR_MESSAGE_BY_CODE[code]
  }
  const detail = String(obj.message || obj.detail || obj.error || '').trim()
  const mapped = mapRelayToTraeStatusErrorMessage(detail, '')
  if (mapped && mapped !== detail) {
    return mapped
  }
  if (detail && RELAY_TO_TRAE_ERROR_MESSAGE_BY_CODE[detail]) {
    return RELAY_TO_TRAE_ERROR_MESSAGE_BY_CODE[detail]
  }
  return detail || fallback
}

export function summarizeMissingRepoCredentialsForUi(payload) {
  const rows = Array.isArray(payload?.missing_repo_credentials) ? payload.missing_repo_credentials : []
  const normalized = rows
    .map((row) => String(row || '').trim())
    .filter(Boolean)
  if (!normalized.length) {
    return ''
  }
  const preview = normalized.slice(0, 3).join('，')
  return `缺失仓库(${normalized.length})：${preview}${normalized.length > 3 ? ' ...' : ''}`
}
