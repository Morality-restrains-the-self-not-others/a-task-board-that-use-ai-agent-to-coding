import { getCurrentInstance, onUnmounted } from 'vue'
import { openBillingRechargeSse } from './useBillingRechargeSse.js'

/**
 * 资源订单支付完成 SSE — OPT-20260808-016
 *
 * taskBill markOrderPaid 落地后经 taskSSE 推送 order_paid 事件（billing:user hub）。
 * 本 composable 订阅同一 recharge-events 通道，按 order_id 匹配当前订单；
 * 与 useWechatRechargePoll 同款模式：SSE 即时通知 + 手动查询兜底，禁止长轮询。
 *
 * 用法：
 *   const { startOrderSse, stopOrderSse } = useOrderPaySse({
 *     tenantId: () => tenantId.value,
 *     getOrderId: () => currentOrder.value?.id,
 *     isActive: (orderId) => wechatQrVisible.value && currentOrder.value?.id === orderId,
 *     onPaid: async (orderId) => { /* 重新拉单并关闭弹窗 *\/ },
 *   })
 */
export function useOrderPaySse({ tenantId, getOrderId, isActive, onPaid }) {
  let closeSse = null

  const stopOrderSse = () => {
    if (closeSse) {
      closeSse()
      closeSse = null
    }
  }

  const startOrderSse = () => {
    stopOrderSse()
    const orderId = getOrderId()
    const tid = String(typeof tenantId === 'function' ? tenantId() : tenantId || '').trim()
    if (!orderId || !tid || typeof EventSource === 'undefined') return
    const { close } = openBillingRechargeSse({
      tenantId: tid,
      match: (data) => {
        if (typeof isActive === 'function' && !isActive(orderId)) return false
        // order_paid 事件携带 order_id；匹配当前订单（兼容字符串/数字）
        const got = String(data.order_id ?? '').trim()
        return got === String(orderId)
      },
      onCompleted: () => onPaid?.(orderId),
    })
    closeSse = close
  }

  if (getCurrentInstance()) {
    onUnmounted(stopOrderSse)
  }

  return { startOrderSse, stopOrderSse }
}
