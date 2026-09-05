/**
 * Map billing_profit_sharing wechat_state (QueryOrder `state` or local sentinel)
 * to zh-CN label + tooltip.
 *
 * Distinct from:
 * - receiver registration（推荐人是否绑定微信登录 / openid）
 * - local `status`（待分账 / 分账中 / 已完成 / 失败）
 *
 * Sentinel not_submitted means wechat_profit_sharing_id is empty — we have not
 * POSTed /v3/profitsharing/orders yet (expected during freeze).
 * Official QueryOrder states: PROCESSING / FINISHED
 * https://pay.weixin.qq.com/doc/v3/merchant/4012525210
 *
 * @param {unknown} state
 * @returns {string}
 */

export const WECHAT_PS_STATE_NOT_SUBMITTED = 'not_submitted'

// OPT-20260826-009: 旧二进制把哨兵写成中文「尚未提交微信」，新后端改机器码
// not_submitted。展示层同时识别新旧值，避免旧数据/旧进程被当成未知状态。
const WECHAT_PS_STATE_NOT_SUBMITTED_LEGACY = '尚未提交微信'

const NOT_SUBMITTED_DISPLAY = {
  label: '未向微信发起分账',
  title:
    '微信分账单状态（不是用户是否绑定微信）。未向微信发起分账：本地台账尚未向微信提交分账单；冻结期内这是预期状态。冻结结束或点击「分账」后才会出现处理中/已完成。',
}

const STATE_DISPLAY = {
  [WECHAT_PS_STATE_NOT_SUBMITTED]: NOT_SUBMITTED_DISPLAY,
  [WECHAT_PS_STATE_NOT_SUBMITTED_LEGACY]: NOT_SUBMITTED_DISPLAY,
  PROCESSING: {
    label: '微信处理中',
    title:
      '微信分账单状态：处理中（非终态）。可稍后点「同步微信状态」再查，直到变为已完成。',
  },
  FINISHED: {
    label: '微信已分账完成',
    title:
      '微信分账单状态：分账完成（终态）。仅代表微信侧动账执行完毕；各接收方结果请看本地状态与失败原因。',
  },
}

export function profitSharingWechatStateLabel(state) {
  const key = String(state || '').trim()
  if (!key) return '—'
  return STATE_DISPLAY[key]?.label || key
}

export function profitSharingWechatStateTitle(state, wechatError) {
  const err = String(wechatError || '').trim()
  if (err) return err
  const key = String(state || '').trim()
  if (!key) return ''
  return STATE_DISPLAY[key]?.title || `微信分账单状态：${key}`
}
