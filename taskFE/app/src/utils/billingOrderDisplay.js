/** 订单列表展示与退款资格判断（BillingOrders / 超管订单列表 / 订单详情共用） */

export const BILLING_ORDER_STATUS_LABELS = {
  pending: '待支付',
  paid: '已支付',
  cancelled: '已取消',
  expired: '已过期',
  refunded: '已退款',
}

export const BILLING_ORDER_STATUS_FILTERS = [
  { value: '', label: '全部' },
  ...Object.entries(BILLING_ORDER_STATUS_LABELS).map(([value, label]) => ({ value, label })),
]

export function billingOrderStatusLabel(s) {
  return BILLING_ORDER_STATUS_LABELS[s] || s
}

export function billingOrderStatusClass(s) {
  const map = {
    pending: 'bg-yellow-100 text-yellow-800',
    paid: 'bg-green-100 text-green-800',
    cancelled: 'bg-gray-100 text-gray-500',
    expired: 'bg-red-100 text-red-800',
    refunded: 'bg-gray-100 text-gray-500',
  }
  return map[s] || 'bg-gray-100 text-gray-600'
}

export function billingOrderPaymentLabel(m) {
  const map = { wechat: '微信支付', paypal: 'PayPal', balance: '账户支付', admin_grant: '系统赠送' }
  return map[m] || m || '—'
}

/**
 * 与后端 taskBill orderRefundChannel 对齐：订单是否为微信渠道实付。
 * 开票仅允许微信渠道（taskBill applyInvoiceApplication 的 channel=="wechat" 校验）。
 * 按 JSON 已有字段（payment_method / payment_ref）判断，勿复制后端私有函数。
 */
export function billingOrderIsWechatChannel(order) {
  if (!order) return false
  const method = String(order.payment_method || '').toLowerCase()
  const ref = String(order.payment_ref || '')
  let channel = ''
  if (method === 'wechat' || method === 'paypal') {
    channel = method
  } else if (ref.startsWith('wechat:')) {
    channel = 'wechat'
  } else if (ref.startsWith('paypal:')) {
    channel = 'paypal'
  }
  if (!channel) return false
  const providerRef = channel === 'wechat'
    ? ref.replace(/^wechat:/, '')
    : ref.replace(/^paypal:/, '')
  return channel === 'wechat' && Boolean(providerRef.trim())
}

export function billingOrderFormatTime(t) {
  if (!t) return '—'
  try {
    const d = new Date(t)
    if (Number.isNaN(d.getTime())) return t
    return d.toLocaleString('zh-CN', { hour12: false })
  } catch {
    return t
  }
}

/** 与后端 orderHasActiveRefund 一致：pending/approved 阻塞再次申请 */
const ACTIVE_REFUND_APP_STATUSES = new Set(['pending', 'approved'])

/**
 * 返回该订单关联的活跃退款申请状态（pending|approved），无则 null。
 * @param {object} order
 * @param {Array<{order_id?: string, status?: string}>} [applications]
 */
export function billingOrderRefundApplicationStatus(order, applications = []) {
  if (!order) return null
  const oid = String(order.id ?? '')
  if (!oid) return null
  const hit = (applications || []).find(
    (a) =>
      String(a?.order_id ?? '') === oid &&
      ACTIVE_REFUND_APP_STATUSES.has(String(a?.status || '')),
  )
  return hit ? String(hit.status) : null
}

/**
 * 仅微信/PayPal 实付订单可申请退款（赠送/零金额不可）。
 * 另：订单已有 pending/approved 申请，或租户已有任意 pending 申请时不可再申请
 *（与 taskBill tenantHasPendingRefund / orderHasActiveRefund 对齐）。
 * @param {object} order
 * @param {Array<{order_id?: string, status?: string}>} [applications]
 */
export function isBillingOrderRefundable(order, applications = []) {
  if (!order || order.status !== 'paid') return false
  if (billingOrderRefundApplicationStatus(order, applications)) return false
  if ((applications || []).some((a) => String(a?.status || '') === 'pending')) return false
  const method = String(order.payment_method || '').toLowerCase()
  if (method === 'admin_grant') return false
  let cents = Number(order.total_yuan_cents)
  if (!Number.isFinite(cents)) {
    const yuan = Number(order.total_yuan)
    cents = Number.isFinite(yuan) ? Math.round(yuan * 100) : NaN
  }
  if (Number.isFinite(cents) && cents <= 0) return false
  if (method === 'wechat' || method === 'paypal') return true
  const ref = String(order.payment_ref || '')
  return ref.startsWith('wechat:') || ref.startsWith('paypal:')
}
