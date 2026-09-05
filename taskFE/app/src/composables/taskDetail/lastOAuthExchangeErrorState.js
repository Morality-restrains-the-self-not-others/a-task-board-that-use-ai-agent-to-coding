import { ref } from 'vue'

/** 最近一次推送换票失败摘要（关联项目区旁提示，避免绿标误导） */
export const lastOAuthExchangeError = ref('')

const OAUTH_EXCHANGE_HINT_RE =
  /oauth|换票|未能换取|refresh|bridge|凭据|credential|unauthorized|token/i

/**
 * @param {unknown} message
 * @returns {boolean}
 */
export function isOAuthExchangeFailureMessage(message) {
  const text = String(message || '').trim()
  if (!text) return false
  return OAUTH_EXCHANGE_HINT_RE.test(text)
}

/**
 * @param {unknown} message
 * @param {{ maxLen?: number }} [opts]
 */
export function setLastOAuthExchangeError(message, opts = {}) {
  const maxLen = Number.isFinite(opts.maxLen) ? opts.maxLen : 240
  const text = String(message || '').trim()
  if (!text) {
    lastOAuthExchangeError.value = ''
    return
  }
  lastOAuthExchangeError.value = text.length > maxLen ? `${text.slice(0, maxLen)}…` : text
}

export function clearLastOAuthExchangeError() {
  lastOAuthExchangeError.value = ''
}
