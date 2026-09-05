/**
 * Map billing_profit_sharing.fail_reason machine codes to zh-CN labels.
 *
 * Shared by system-admin pending queue and referral performance 微信分账 Tab
 * so a newly stored code gets one label instead of drifting across copies.
 * Unknown / already-Chinese reasons fall back to the trimmed raw string.
 *
 * processPendingProfitSharings 失败时会把 SDK err.Error()（含 error http response /
 * Wechatpay-Signature 头）写入 fail_reason —— 经 parsePaymentProviderError 复用退款同款
 * dump 清洗，确保签名头不泄漏到 UI（OPT-20260824-084）。
 *
 * @param {unknown} failReason
 * @returns {string}
 */
import { parsePaymentProviderError } from './humanizePaymentProviderError.js'

export function profitSharingFailReasonLabel(failReason) {
  const key = String(failReason || '').trim()
  if (!key) return '—'
  if (FAIL_REASON_LABEL[key]) return FAIL_REASON_LABEL[key]
  if (isEmptyReceiverAccountError(key)) {
    return FAIL_REASON_LABEL.referrer_openid_missing
  }
  const parsed = parsePaymentProviderError(key)
  if (parsed) {
    if (parsed.notEnough) {
      return `分账失败：${parsed.detail || '基本账户余额不足，请充值后重新发起'}`
    }
    if (parsed.detail) return `分账失败：${parsed.detail}`
    return '分账失败，请稍后重试或联系财务'
  }
  return key
}

function isEmptyReceiverAccountError(text) {
  if (text.includes('/body/receivers/0/account')) return true
  return text.includes('PARAM_ERROR') && text.includes('分账接收方帐号')
}

const FAIL_REASON_LABEL = {
  referrer_openid_missing: '推荐人未绑定微信收款账号',
  qualification_revoked: '推荐资格已撤销',
  ACCOUNT_ABNORMAL: '分账接收账户异常',
  NO_RELATION: '分账关系已解除',
  RECEIVER_HIGH_RISK: '高风险接收方',
  RECEIVER_REAL_NAME_NOT_VERIFIED: '接收方未实名',
  NO_AUTH: '分账权限已解除',
  RECEIVER_RECEIPT_LIMIT: '超出用户月收款限额',
  PAYER_ACCOUNT_ABNORMAL: '分出方账户异常',
  INVALID_REQUEST: '描述参数设置失败',
  BALANCE_NOT_ENOUGH: '余额不足',
  TIME_OUT_CLOSED: '超时关单',
}
