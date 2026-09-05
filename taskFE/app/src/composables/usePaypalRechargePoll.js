import { apiFetch } from '../utils/apiUtils.js'
import { openBillingRechargeSse } from './useBillingRechargeSse.js'

/**
 * PayPal 回跳：单次 REST 查询 + 同一支付 SSE 通道等待 webhook 入账（禁止长轮询）。
 */
export function usePaypalRechargePoll({
  tenantId,
  router,
  route,
  isRecharging,
  rechargeSuccess,
  rechargeError,
  lastRechargeSummary
}) {
  let closeSse = null
  let handledOrderId = ''

  const stopSse = () => {
    if (closeSse) {
      closeSse()
      closeSse = null
    }
  }

  const markPaypalSuccess = (data) => {
    const orderId = String(data?.order_id || '').trim()
    if (orderId && handledOrderId === orderId) return
    if (orderId) handledOrderId = orderId
    stopSse()
    rechargeSuccess.value = true
    lastRechargeSummary.value = {
      yuan: String(data.recharge_yuan ?? ''),
      cents: data.recharge_cents ?? '—',
      totalYuanCents: data.cents ?? '—'
    }
    isRecharging.value = false
    setTimeout(() => {
      router.push({
        name: 'billing_dashboard',
        params: { tenant: String(route.params.tenant) }
      })
    }, 3000)
  }

  const queryPaypalStatusOnce = async (orderId) => {
    const response = await apiFetch(
      `/api/tenant/${tenantId.value}/billing/accounts/recharge_paypal_status/?order_id=${encodeURIComponent(orderId)}`,
      { credentials: 'include', headers: { Accept: 'application/json' } }
    )
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      rechargeError.value = data.error || '无法查询支付状态，请稍后在交易记录中查看'
      return false
    }
    if (data.status === 'completed') {
      markPaypalSuccess({ ...data, order_id: orderId })
      return true
    }
    return false
  }

  const handlePaypalReturnStatusPoll = async () => {
    const q = route.query
    const isReturn = q.paypal_return === '1' || q.paypal_return === 1
    const token = q.token
    if (!isReturn) return
    if (!token) {
      rechargeError.value = '支付回调缺少订单信息'
      await router.replace({ path: route.path })
      return
    }
    const orderId = String(token)
    handledOrderId = ''
    await router.replace({ path: route.path })
    isRecharging.value = true
    rechargeError.value = null
    rechargeSuccess.value = false

    // 先挂 SSE：webhook 入账后自动确认（与微信共用通道）
    stopSse()
    const { close } = openBillingRechargeSse({
      tenantId: tenantId.value,
      match: (data) => {
        const got = String(data.order_id || '').trim()
        if (got) return got === orderId
        return String(data.transaction_id || '') === `paypal:${orderId}`
      },
      onCompleted: (data) => markPaypalSuccess({ ...data, order_id: data.order_id || orderId })
    })
    closeSse = close

    try {
      const done = await queryPaypalStatusOnce(orderId)
      if (done) return
      rechargeError.value =
        '支付处理中。若已扣款，页面将自动确认到账；也可稍后在交易记录中查看。'
      // 保持 isRecharging，直到 SSE 成功或用户离开（onUnmounted 由 openBillingRechargeSse 处理）
    } catch {
      rechargeError.value = '网络错误，请稍后重试或查看交易记录'
      isRecharging.value = false
      stopSse()
    }
  }

  const handlePaypalCancelFromQuery = () => {
    if (route.query.paypal_cancel === '1' || route.query.paypal_cancel === 1) {
      stopSse()
      rechargeError.value = '已取消 PayPal 支付'
      router.replace({ path: route.path })
      return true
    }
    return false
  }

  return {
    handlePaypalReturnStatusPoll,
    handlePaypalCancelFromQuery
  }
}
