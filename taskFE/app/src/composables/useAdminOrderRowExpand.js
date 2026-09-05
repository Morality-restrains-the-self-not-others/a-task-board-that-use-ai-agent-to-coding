/**
 * 管理端订单列表行展开：拉取行项、留言、资源消耗与分账（仅管理员详情 API）。
 */
import { ref } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import { safeJson } from '@/utils/safeResponseJson.js'

/**
 * @param {{
 *   selectedTenant: import('vue').Ref<{id?: unknown}|null>,
 *   isAllTenants: import('vue').ComputedRef<boolean>,
 * }} deps
 */
export function useAdminOrderRowExpand(_deps = {}) {
  const expandedIds = ref(new Set())
  const expandLoading = ref({})
  const expandItems = ref({})
  const expandNotes = ref({})
  const expandConsumption = ref({})
  const expandProfitSharing = ref({})
  const expandOutTradeNo = ref({})
  const expandWechatTransactionId = ref({})

  const isExpanded = (id) => expandedIds.value.has(id)

  const clearExpand = () => {
    expandedIds.value = new Set()
    expandItems.value = {}
    expandNotes.value = {}
    expandConsumption.value = {}
    expandProfitSharing.value = {}
    expandOutTradeNo.value = {}
    expandWechatTransactionId.value = {}
  }

  const setExpandPayload = (id, d) => {
    expandItems.value = { ...expandItems.value, [id]: d?.items || [] }
    expandNotes.value = { ...expandNotes.value, [id]: String(d?.buyer_note || '') }
    expandConsumption.value = { ...expandConsumption.value, [id]: d?.resource_consumption || null }
    expandProfitSharing.value = {
      ...expandProfitSharing.value,
      [id]: Array.isArray(d?.profit_sharing) ? d.profit_sharing : [],
    }
    expandOutTradeNo.value = { ...expandOutTradeNo.value, [id]: String(d?.out_trade_no || '') }
    expandWechatTransactionId.value = {
      ...expandWechatTransactionId.value,
      [id]: String(d?.wechat_transaction_id || ''),
    }
  }

  const toggleExpand = async (o) => {
    const id = o.id
    if (expandedIds.value.has(id)) {
      expandedIds.value.delete(id)
      expandedIds.value = new Set(expandedIds.value)
      return
    }
    expandedIds.value.add(id)
    expandedIds.value = new Set(expandedIds.value)

    if (expandItems.value[id]) return

    expandLoading.value = { ...expandLoading.value, [id]: true }
    try {
      const r = await apiFetch(`/api/system-admin/orders/${encodeURIComponent(id)}/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (r.ok) {
        const d = await safeJson(r, {})
        setExpandPayload(id, d)
      }
    } catch {
      setExpandPayload(id, { items: [] })
    } finally {
      const next = { ...expandLoading.value }
      delete next[id]
      expandLoading.value = next
    }
  }

  const expandOrderForDeepLink = async (o) => {
    if (!o || isExpanded(o.id)) return
    await toggleExpand(o)
  }

  return {
    expandedIds,
    expandLoading,
    expandItems,
    expandNotes,
    expandConsumption,
    expandProfitSharing,
    expandOutTradeNo,
    expandWechatTransactionId,
    isExpanded,
    clearExpand,
    toggleExpand,
    expandOrderForDeepLink,
  }
}
