/**
 * System-admin「支付与签署」抽屉的展示辅助。
 * admin_grant 流水金额为 0、无支付渠道，不得误标为「无关联签署记录」。
 *
 * 协议签署与支付流水不是 1:1 外键：后端只把最新「支付服务条款」挂到用户支付行。
 * 注册时的服务协议、历史版本支付条款因此会出现在下方独立区块——这是数据模型，不是缺失。
 */

export const ORPHAN_CONSENT_HEADING = '其他协议签署'

export const ORPHAN_CONSENT_HINT =
  '支付记录只展示当前有效的支付服务条款。注册时签署的服务协议、以及条款更新前的历史版本会出现在这里，不表示签署缺失。'

export function isPaymentTermsDocumentKind(kind) {
  const k = String(kind || '').trim()
  return k === 'recharge_cents' || k === 'recharge_points'
}

export function licenseDocumentKindLabel(kind) {
  if (isPaymentTermsDocumentKind(kind)) return '支付服务条款'
  if (String(kind || '').trim() === 'service') return '服务协议'
  return String(kind || '').trim() || '协议'
}

export function unboundPaymentConsents(recharges, consents) {
  const boundIds = new Set(
    (Array.isArray(recharges) ? recharges : [])
      .map((r) => r?.consent?.id)
      .filter(Boolean)
      .map(String)
  )
  return (Array.isArray(consents) ? consents : []).filter((c) => !boundIds.has(String(c?.id)))
}

export function isAdminGrantRecharge(row) {
  if (!row || typeof row !== 'object') return false
  if (row.consent_required === false) return true
  const src = String(row.points_source_type || '').trim()
  const channel = String(row.channel || row.payment_channel || '').trim()
  return src === 'admin_grant' || channel === 'admin_grant'
}

export function formatRechargeChannel(row) {
  if (isAdminGrantRecharge(row)) {
    const ch = String(row.channel || row.payment_channel || '').trim()
    if (ch && ch !== 'admin_grant') return ch
    return '系统赠送'
  }
  return row?.channel || row?.payment_channel || '—'
}

export function consentStatusLabel(row) {
  if (isAdminGrantRecharge(row)) {
    return row?.consent_note || '系统赠送，无需支付签署'
  }
  if (row?.consent) return '已关联签署'
  return '无关联签署记录'
}
