/**
 * BillingOrders：根据路由 query.order_id，在列表加载后展开并定位到目标订单。
 */
import { watch, nextTick } from 'vue'
import {
  parseOrderIdQuery,
  findOrderInList,
  pageFromListOffset,
  scrollAndHighlightOrderRow,
} from '../utils/billingOrderDeepLink.js'

/**
 * @param {{
 *   route: import('vue-router').RouteLocationNormalizedLoaded,
 *   router?: import('vue-router').Router,
 *   orders: import('vue').Ref<unknown[]>,
 *   loading: import('vue').Ref<boolean>,
 *   listOffset: import('vue').Ref<number>,
 *   pageSize: number,
 *   currentPage: import('vue').Ref<number>,
 *   expandOrder: (order: object) => void | Promise<void>,
 *   onFocusMiss?: (orderId: string) => void,
 * }} deps
 */
export function useBillingOrderIdDeepLink(deps) {
  const {
    route,
    router,
    orders,
    loading,
    listOffset,
    pageSize,
    currentPage,
    expandOrder,
    onFocusMiss,
  } = deps

  let lastFocused = ''
  let lastMissed = ''

  const clearOrderIdQuery = () => {
    if (!router || !route.query?.order_id) return
    const next = { ...route.query }
    delete next.order_id
    router.replace({ query: next }).catch(() => {})
  }

  const tryFocus = async () => {
    if (loading.value) return
    const orderId = parseOrderIdQuery(route.query)
    if (!orderId) {
      lastFocused = ''
      lastMissed = ''
      return
    }
    if (orderId === lastFocused) return

    const page = pageFromListOffset(listOffset?.value ?? 0, pageSize)
    if (currentPage && currentPage.value !== page) {
      currentPage.value = page
    }

    const hit = findOrderInList(orders.value || [], orderId)
    if (!hit) {
      if (orderId !== lastMissed) {
        lastMissed = orderId
        onFocusMiss?.(orderId)
      }
      return
    }

    lastFocused = orderId
    lastMissed = ''
    await expandOrder(hit)
    await nextTick()
    const ok = scrollAndHighlightOrderRow(orderId)
    if (!ok && typeof window !== 'undefined') {
      window.setTimeout(() => scrollAndHighlightOrderRow(orderId), 80)
    }
    // 定位完成后清掉 query，避免用户翻页时被 order_id 再次强制对齐
    clearOrderIdQuery()
  }

  watch(
    () => [route.query?.order_id, orders.value, loading.value, listOffset?.value],
    () => {
      void tryFocus()
    },
    { deep: true, immediate: true },
  )

  return { tryFocus, clearOrderIdQuery }
}
