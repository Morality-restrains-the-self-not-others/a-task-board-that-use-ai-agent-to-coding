/**
 * 超管订单记录 Tab：从 ?tenant_id=&order_id= 选租户并定位订单行。
 */
import { watch } from 'vue'
import { parseTenantIdQuery } from '../utils/billingOrderDeepLink.js'
import { useBillingOrderIdDeepLink } from './useBillingOrderIdDeepLink.js'

/**
 * @param {{
 *   route: import('vue-router').RouteLocationNormalizedLoaded,
 *   router?: import('vue-router').Router,
 *   orders: import('vue').Ref<unknown[]>,
 *   loading: import('vue').Ref<boolean>,
 *   listOffset: import('vue').Ref<number>,
 *   pageSize: number,
 *   currentPage: import('vue').Ref<number>,
 *   statusFilter: import('vue').Ref<string>,
 *   tenantOptions: import('vue').Ref<Array<{id?: unknown, name?: string}>>,
 *   expandOrder: (order: object) => void | Promise<void>,
 *   selectTenant: (t: {id: unknown, name?: string}) => void,
 *   selectAllTenants: () => void,
 *   fetchTenantOptions: (search: string) => Promise<void>,
 *   onFocusMiss?: (orderId: string) => void,
 *   onTradeNoSearch: (q: string) => void | Promise<void>,
 *   onWechatAccountSearch: (q: string) => void | Promise<void>,
 * }} deps
 */
export function useSystemAdminOrderListDeepLink(deps) {
  const {
    route,
    router,
    orders,
    loading,
    listOffset,
    pageSize,
    currentPage,
    statusFilter,
    tenantOptions,
    expandOrder,
    selectTenant,
    selectAllTenants,
    fetchTenantOptions,
    onFocusMiss,
    onTradeNoSearch,
    onWechatAccountSearch,
  } = deps

  watch(
    () => route.query?.order_id,
    (id) => {
      if (String(id || '').trim()) statusFilter.value = ''
    },
    { immediate: true },
  )

  useBillingOrderIdDeepLink({
    route,
    router,
    orders,
    loading,
    listOffset,
    pageSize,
    currentPage,
    expandOrder,
    onFocusMiss,
  })

  const applyFromRoute = async () => {
    await fetchTenantOptions('')
    // OPT-20260821-039: 交易单号深链 — 刷新/分享 ?order_number= 自动查询。
    const wechatAccount = String(route.query?.wechat_account || '').trim()
    if (wechatAccount && typeof onWechatAccountSearch === 'function') {
      await onWechatAccountSearch(wechatAccount)
      return
    }
    const orderNumber = String(route.query?.order_number || '').trim()
    if (orderNumber && typeof onTradeNoSearch === 'function') {
      await onTradeNoSearch(orderNumber)
      return
    }
    const tid = parseTenantIdQuery(route.query)
    if (!tid) {
      selectAllTenants()
      return
    }
    const found = (tenantOptions.value || []).find((t) => String(t?.id ?? '') === tid)
    selectTenant(found || { id: tid, name: tid })
  }

  return { applyFromRoute }
}
