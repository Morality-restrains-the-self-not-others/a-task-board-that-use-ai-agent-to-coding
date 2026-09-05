import { formatYuanFromCents } from '../../utils/formatYuanCents.js'

export function refundStatusLabel(status) {
  const map = {
    pending: '待审批',
    approved: '已通过',
    rejected: '已拒绝',
  }
  return map[status] || status || '—'
}

export function formatRefundDate(value) {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** API frozen_points 为分（yuan cents）；列表按元展示。 */
export function formatFrozenYuan(points) {
  return formatYuanFromCents(points, { suffix: '元' })
}
