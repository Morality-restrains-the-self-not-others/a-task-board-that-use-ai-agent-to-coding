/**
 * Display helper for profit-sharing bill numbers.
 *
 * WeChat APIv3 uses two identifiers on the same ProfitSharingRecord:
 * - wechat_profit_sharing_id = QueryOrder `order_id`（微信分账单号）
 * - out_profit_sharing_no = CreateOrder `out_order_no`（商户侧幂等键）
 *
 * out_order_no is a value object on the existing aggregate, not a separate
 * entity. UI shows one 「微信分账单号」cell: WeChat id first, merchant key as fallback.
 *
 * @param {{ wechat_profit_sharing_id?: unknown, out_profit_sharing_no?: unknown } | null | undefined} row
 * @returns {string}
 */
export function profitSharingBillNoLabel(row) {
  const wechat = String(row?.wechat_profit_sharing_id || '').trim()
  if (wechat) return wechat
  const merchant = String(row?.out_profit_sharing_no || '').trim()
  if (merchant) return merchant
  return '—'
}

/**
 * Tooltip for the folded bill-number cell.
 *
 * @param {{ wechat_profit_sharing_id?: unknown, out_profit_sharing_no?: unknown } | null | undefined} row
 * @returns {string}
 */
export function profitSharingBillNoTitle(row) {
  const wechat = String(row?.wechat_profit_sharing_id || '').trim()
  const merchant = String(row?.out_profit_sharing_no || '').trim()
  if (wechat && merchant && wechat !== merchant) {
    return `商户分账单号 ${merchant}（微信支付 out_order_no，非独立实体）`
  }
  if (merchant && !wechat) {
    return '商户分账单号（微信支付 out_order_no，尚未返回微信分账单号）'
  }
  return ''
}
