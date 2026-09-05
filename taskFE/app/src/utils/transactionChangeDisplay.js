/**
 * Billing transaction row display helpers.
 *
 * 配额入账/消耗的 amount_points 常为 0，不能再只展示 ±0.00 元。
 * Prefer API `change_display` / `points_source_type_display` when present.
 */

const SOURCE_LABELS = {
  user_recharge_paypal: 'PayPal 支付',
  user_recharge_wechat: '微信支付',
  user_recharge_admin: '管理员直充',
  user_recharge: '用户支付（历史）',
  admin_grant: '后台赠送',
  promotion: '活动赠送',
  adjustment: '人工调账',
  quota_consumption: '配额消耗',
  consumption: '消耗',
}

export function pointsSourceTypeDisplay(transaction) {
  if (!transaction || typeof transaction !== 'object') return ''
  const fromApi = transaction.points_source_type_display
  if (typeof fromApi === 'string' && fromApi.trim()) return fromApi.trim()
  const src = transaction.points_source_type
  if (typeof src !== 'string' || !src) return ''
  if (Object.prototype.hasOwnProperty.call(SOURCE_LABELS, src)) {
    return SOURCE_LABELS[src]
  }
  return src
}

export function pointsSourceTypeDisplayOrDash(transaction) {
  return pointsSourceTypeDisplay(transaction) || '—'
}

function centsToYuanStr(pts) {
  if (pts == null || pts === '') return '0.00'
  const n = Number(pts)
  if (!Number.isFinite(n)) return '0.00'
  return (n / 100).toFixed(2)
}

export function transactionChangeDisplay(transaction) {
  if (!transaction || typeof transaction !== 'object') return '—'
  const fromApi = transaction.change_display
  if (typeof fromApi === 'string' && fromApi.trim()) return fromApi.trim()

  const amount = Number(transaction.amount_points)
  if (Number.isFinite(amount) && amount !== 0) {
    const sign = transaction.transaction_type === 'recharge' ? '+' : '-'
    return `${sign}${centsToYuanStr(amount)} 元`
  }

  const changes = Array.isArray(transaction.resource_changes) ? transaction.resource_changes : []
  const parts = changes
    .map((c) => (c && typeof c.display === 'string' ? c.display.trim() : ''))
    .filter(Boolean)
  if (parts.length) return parts.join('；')

  return '—'
}

export function ledgerSnapshotLines(transaction) {
  if (!transaction || typeof transaction !== 'object') return []
  const snap = transaction.ledger_snapshot
  if (snap && Array.isArray(snap.display_lines)) {
    const lines = snap.display_lines
      .map((line) => (typeof line === 'string' ? line.trim() : ''))
      .filter(Boolean)
    if (lines.length) return lines
  }
  if (snap && typeof snap.display === 'string' && snap.display.trim()) {
    return snap.display.split('；').map((line) => line.trim()).filter(Boolean)
  }
  const before = Number(transaction.balance_before_points)
  const after = Number(transaction.balance_after_points)
  const hasBefore = Number.isFinite(before)
  const hasAfter = Number.isFinite(after)
  if (!hasBefore && !hasAfter) return []
  const beforeYuan = centsToYuanStr(hasBefore ? before : after)
  const afterYuan = centsToYuanStr(hasAfter ? after : before)
  if (beforeYuan === afterYuan) return [`余额 ${afterYuan} 元`]
  return [`余额 ${beforeYuan} → ${afterYuan} 元`]
}

export function ledgerSnapshotDisplay(transaction) {
  const lines = ledgerSnapshotLines(transaction)
  return lines.length ? lines.join('；') : '—'
}
