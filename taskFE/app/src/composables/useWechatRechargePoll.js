import { ref, getCurrentInstance, onUnmounted } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { openBillingRechargeSse } from './useBillingRechargeSse.js'

/**
 * 微信支付弹层：taskSSE 自动确认到账；保留手动「查询支付状态」/ mock 兜底。禁止长轮询。
 */
export function useWechatRechargePoll({ tenantId, router, route, finalAmount, expectedYuanCents }) {
  const wechatModalVisible = ref(false)
  const wechatCodeUrl = ref('')
  const wechatOutTradeNo = ref('')
  const wechatPolling = ref(false)
  const wechatModalError = ref('')
  const wechatMode = ref('mock')
  let hooksRef = null
  let closeSse = null
  let handledOutTradeNo = ''

  const stopSse = () => {
    if (closeSse) {
      closeSse()
      closeSse = null
    }
  }

  const closeWechatModal = () => {
    stopSse()
    wechatModalVisible.value = false
    wechatPolling.value = false
    wechatModalError.value = ''
  }

  const markWechatSuccess = (data, { rechargeSuccess, lastRechargeSummary }) => {
    const outTradeNo = String(data?.out_trade_no || wechatOutTradeNo.value || '').trim()
    if (outTradeNo && handledOutTradeNo === outTradeNo) return
    if (outTradeNo) handledOutTradeNo = outTradeNo
    rechargeSuccess.value = true
    lastRechargeSummary.value = {
      yuan: String(data.recharge_yuan ?? finalAmount.value),
      cents: data.recharge_cents ?? expectedYuanCents.value,
      totalYuanCents: data.cents ?? '—'
    }
    closeWechatModal()
    setTimeout(() => {
      router.push({ name: 'billing_dashboard', params: { tenant: String(route.params.tenant) } })
    }, 3000)
  }

  const startSse = (hooks) => {
    stopSse()
    const { close } = openBillingRechargeSse({
      tenantId: tenantId.value,
      match: (data) => {
        const want = String(wechatOutTradeNo.value || '').trim()
        const got = String(data.out_trade_no || '').trim()
        if (!want) return true
        return !got || want === got
      },
      onCompleted: (data) => markWechatSuccess(data, hooks || hooksRef)
    })
    closeSse = close
  }

  const checkWechatStatusOnce = async ({
    simulate = false,
    rechargeSuccess,
    lastRechargeSummary
  } = {}) => {
    const outTradeNo = wechatOutTradeNo.value
    if (!outTradeNo) return
    wechatPolling.value = true
    wechatModalError.value = ''
    try {
      const q = new URLSearchParams({ out_trade_no: outTradeNo })
      if (simulate) q.set('simulate', '1')
      const response = await apiFetch(
        `/api/tenant/${tenantId.value}/billing/accounts/recharge_wechat_status/?${q.toString()}`,
        { credentials: 'include', headers: { Accept: 'application/json' } }
      )
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        wechatModalError.value = data.error || '无法查询支付状态'
        return
      }
      if (data.status === 'completed') {
        markWechatSuccess(data, { rechargeSuccess, lastRechargeSummary })
        return
      }
      wechatModalError.value = '尚未确认到账，若已支付请稍后再次点「查询支付状态」'
    } catch {
      wechatModalError.value = '网络错误，请稍后重试'
    } finally {
      wechatPolling.value = false
    }
  }

  const openWechatModalFromCreate = (data, hooks) => {
    wechatCodeUrl.value = data.code_url
    wechatOutTradeNo.value = data.out_trade_no
    wechatMode.value = data.mode || 'mock'
    wechatModalVisible.value = true
    wechatModalError.value = ''
    handledOutTradeNo = ''
    hooksRef = hooks
    startSse(hooks)
  }

  const simulateWechatPay = (hooks) => {
    checkWechatStatusOnce({ ...(hooks || hooksRef), simulate: true })
  }

  const queryWechatPayStatus = (hooks) => {
    checkWechatStatusOnce(hooks || hooksRef)
  }

  const abortWechatPoll = () => {
    wechatPolling.value = false
    stopSse()
  }

  if (getCurrentInstance()) {
    onUnmounted(() => stopSse())
  }

  return {
    wechatModalVisible,
    wechatCodeUrl,
    wechatOutTradeNo,
    wechatPolling,
    wechatModalError,
    wechatMode,
    closeWechatModal,
    openWechatModalFromCreate,
    simulateWechatPay,
    queryWechatPayStatus,
    abortWechatPoll
  }
}
