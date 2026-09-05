/**
 * 支付渠道（微信/PayPal）失败文案：禁止把 SDK HTTP 原文（含签名头）展示给用户。
 */

const DUMP_MARKERS = [
  'error http response',
  'statuscode:',
  'wechatpay-signature',
  'wechatpay-serial',
  'wechatpay-nonce',
  'wechatpay-timestamp',
]

/**
 * @param {unknown} text
 * @returns {boolean}
 */
export function looksLikePaymentProviderDump(text) {
  const s = String(text || '').toLowerCase()
  return DUMP_MARKERS.some((m) => s.includes(m))
}

/**
 * 解析支付渠道 SDK dump 为 {detail, notEnough, isDump}。
 *
 * 与退款共用同一层 dump 清洗（OPT-20260824-084）：退款面板 humanizePaymentProviderError
 * 与分账失败原因 profitSharingFailReasonLabel 都消费本函数，避免两处各写一套解析。
 * 非 dump 且非余额不足时返回 null。
 *
 * @param {unknown} raw
 * @returns {{ detail: string, notEnough: boolean, isDump: boolean } | null}
 */
export function parsePaymentProviderError(raw) {
  const text = String(raw || '').trim()
  if (!text) return null
  const messageLine = text.match(/Message:\s*([^\n\]]+)/)
  const detail = messageLine ? messageLine[1].trim() : ''
  const notEnough = /NOT_ENOUGH/i.test(text) || /余额不足/.test(text)
  const isDump = looksLikePaymentProviderDump(text)
  if (!isDump && !notEnough) return null
  return { detail, notEnough, isDump }
}

/**
 * @param {unknown} raw
 * @param {string} [fallback='操作失败']
 * @returns {string}
 */
export function humanizePaymentProviderError(raw, fallback = '操作失败') {
  const text = String(raw || '').trim()
  if (!text) return fallback
  const parsed = parsePaymentProviderError(text)
  if (!parsed) return text
  if (parsed.notEnough) {
    return `微信退款失败：${parsed.detail || '基本账户余额不足，请充值后重新发起'}`
  }
  if (parsed.detail) return `微信退款失败：${parsed.detail}`
  return '微信退款失败，请稍后重试或联系财务为商户号充值'
}
