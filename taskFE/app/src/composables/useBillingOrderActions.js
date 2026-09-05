import { ref, onUnmounted } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import {
  fetchCurrentPaymentTerms,
  submitPaymentTermsConsent,
  isPaymentTermsConsentRequired,
} from '../utils/paymentTermsConsent.js'


/**
 * 订单操作（支付/取消）composable。
 * 复用 BillingOrders.vue 与 PayOrderModal/CancelOrderModal 之间的共享状态和逻辑。
 *
 * OPT-20260726-034: 支付前手机号验证门禁 — openPayModal 先检查手机验证状态，
 * 若策略要求且用户未验证则先展示 PhoneVerificationGate，验证通过后再调支付 API。
 * 2026-08-19: 支付服务条款签署门禁 — 服务端返回 PAYMENT_TERMS_CONSENT_REQUIRED 时弹条款确认。
 */
export function useBillingOrderActions({ tenantId, onOrderChanged }) {
  // ---- 支付状态 ----
  const payModalVisible = ref(false)
  const payTarget = ref(null)
  const payCodeUrl = ref('')
  const payOutTradeNo = ref('')
  const payMode = ref('')
  const payPollingErr = ref('')
  const payPollingErrTraceId = ref('')
  let payPollTimer = null
  const paymentConsentId = ref('')

  // OPT-20260726-034: 手机号验证门禁状态
  const phoneGateActive = ref(false)
  const phoneVerificationStatus = ref({
    has_phone: false,
    phone_masked: '',
    phone_e164: '',
    sms_verified: false,
    required: false})

  // 支付服务条款签署门禁
  const paymentTermsGateActive = ref(false)
  const paymentTermsDoc = ref(null)
  const paymentTermsConsenting = ref(false)

  // ---- 取消状态 ----
  const cancelConfirmVisible = ref(false)
  const cancelTarget = ref(null)
  const cancelling = ref(false)

  // ---- 支付逻辑 ----

  /**
   * OPT-20260726-034: 检查手机验证状态，决定是否展示验证门禁
   */
  const checkPhoneVerification = async () => {
    try {
      const tid = tenantId.value
      const r = await apiFetch(`/api/tenant/${encodeURIComponent(tid)}/billing/phone-verification-status/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' }})
      if (r.ok) {
        const d = await r.json()
        phoneVerificationStatus.value = {
          has_phone: !!d.has_phone,
          phone_masked: d.phone_masked || '',
          phone_e164: d.phone_e164 || '',
          sms_verified: !!d.sms_verified,
          required: !!d.required}
        // 需要手机验证且用户尚未完成验证
        return !!(d.required && (!d.has_phone || !d.sms_verified))
      }
    } catch (e) {
      console.error('检查手机验证状态失败', e)
    }
    return false
  }

  /**
   * OPT-20260726-034: 实际发起支付 API 请求（获取二维码）
   */
  const initiatePayment = async () => {
    if (!payTarget.value) return
    try {
      const tid = tenantId.value
      const body = { payment_method: 'wechat' }
      if (paymentConsentId.value) body.consent_id = paymentConsentId.value
      const r = await apiFetch(`/api/tenant/${encodeURIComponent(tid)}/billing/orders/${payTarget.value.id}/pay/`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify(body)})
      const d = await r.json().catch(() => ({}))
      if (!r.ok) {
        payPollingErrTraceId.value = d.trace_id || d._traceId || r.traceId || ''
        if (isPaymentTermsConsentRequired(d)) {
          try {
            paymentTermsDoc.value = await fetchCurrentPaymentTerms()
          } catch (e) {
            payPollingErr.value = e.message || '加载支付服务条款失败'
            return
          }
          paymentTermsGateActive.value = true
          payPollingErr.value = ''
          return
        }
        payPollingErr.value = d.error || '支付请求失败'
        return
      }
      paymentTermsGateActive.value = false
      payCodeUrl.value = d.code_url
      payOutTradeNo.value = d.out_trade_no
      payMode.value = d.mode || ''
      startPayPolling()
    } catch (e) {
      payPollingErr.value = '支付请求失败: ' + (e.message || '网络错误')
      payPollingErrTraceId.value = e.traceId || ''
    }
  }

  const openPayModal = async (order) => {
    payTarget.value = order
    payCodeUrl.value = ''
    payOutTradeNo.value = ''
    payMode.value = ''
    payPollingErr.value = ''
    payPollingErrTraceId.value = ''
    phoneGateActive.value = false
    paymentTermsGateActive.value = false
    paymentTermsDoc.value = null
    paymentConsentId.value = ''

    // OPT-20260726-034: 先检查手机验证状态
    const needPhoneGate = await checkPhoneVerification()
    if (needPhoneGate) {
      // 需要手机验证 → 先展示验证门禁，暂不调支付 API
      phoneGateActive.value = true
      payModalVisible.value = true
      return
    }

    // 不需要手机验证 → 直接调支付 API
    await initiatePayment()
    payModalVisible.value = true
  }

  /**
   * OPT-20260726-034: 手机验证通过后，发起支付
   */
  const onPhoneVerified = async () => {
    phoneGateActive.value = false
    phoneVerificationStatus.value.sms_verified = true
    // 验证通过后调支付 API 获取二维码
    await initiatePayment()
  }

  const onPaymentTermsAccepted = async () => {
    if (!paymentTermsDoc.value?.id) {
      payPollingErr.value = '暂无支付服务条款，请联系管理员'
      return
    }
    paymentTermsConsenting.value = true
    payPollingErr.value = ''
    try {
      paymentConsentId.value = await submitPaymentTermsConsent(paymentTermsDoc.value.id)
      paymentTermsGateActive.value = false
      await initiatePayment()
    } catch (e) {
      payPollingErr.value = e.message || '签署失败'
      payPollingErrTraceId.value = e.traceId || ''
    } finally {
      paymentTermsConsenting.value = false
    }
  }

  const closePayModal = () => {
    stopPayPolling()
    payModalVisible.value = false
    payTarget.value = null
    payCodeUrl.value = ''
    payOutTradeNo.value = ''
    payPollingErr.value = ''
    payPollingErrTraceId.value = ''
    paymentTermsGateActive.value = false
    paymentTermsDoc.value = null
    paymentConsentId.value = ''
  }

  const startPayPolling = () => {
    stopPayPolling()
    let attempts = 0
    const maxAttempts = 60
    payPollTimer = setInterval(async () => {
      attempts++
      if (attempts > maxAttempts) {
        stopPayPolling()
        payPollingErr.value = '支付超时，请检查支付状态'
        return
      }
      try {
        const tid = tenantId.value
        const r = await apiFetch(`/api/tenant/${encodeURIComponent(tid)}/billing/orders/${payTarget.value.id}/`, {
          credentials: 'include',
          headers: { Accept: 'application/json' }})
        if (r.ok) {
          const d = await r.json()
          if (d.status === 'paid') {
            closePayModal()
            if (onOrderChanged) onOrderChanged()
            return
          }
        }
      } catch { /* continue polling */ }
    }, 2000)
  }

  const stopPayPolling = () => {
    if (payPollTimer) {
      clearInterval(payPollTimer)
      payPollTimer = null
    }
  }

  // ---- 取消逻辑 ----

  const confirmCancelOrder = (order) => {
    cancelTarget.value = order
    cancelConfirmVisible.value = true
  }

  const doCancelOrder = async (onError) => {
    if (!cancelTarget.value) return
    cancelling.value = true
    try {
      const tid = tenantId.value
      const r = await apiFetch(
        `/api/tenant/${encodeURIComponent(tid)}/billing/orders/${cancelTarget.value.id}/cancel/`,
        {
          method: 'POST',
          credentials: 'include',
          headers: { Accept: 'application/json' }}
      )
      const d = await r.json()
      if (!r.ok) {
        if (onError) onError(d.error || '取消订单失败')
        cancelConfirmVisible.value = false
        return
      }
      cancelConfirmVisible.value = false
      cancelTarget.value = null
      if (onOrderChanged) onOrderChanged()
    } catch (e) {
      if (onError) onError('取消订单失败: ' + (e.message || '网络错误'))
      cancelConfirmVisible.value = false
    } finally {
      cancelling.value = false
    }
  }

  onUnmounted(() => {
    stopPayPolling()
  })

  return {
    // pay
    payModalVisible, payTarget, payCodeUrl, payOutTradeNo, payMode, payPollingErr, payPollingErrTraceId,
    openPayModal, closePayModal,
    // OPT-20260726-034: 手机号验证门禁
    phoneGateActive, phoneVerificationStatus, onPhoneVerified,
    // 支付服务条款签署门禁
    paymentTermsGateActive, paymentTermsDoc, paymentTermsConsenting, onPaymentTermsAccepted,
    // cancel
    cancelConfirmVisible, cancelTarget, cancelling,
    confirmCancelOrder, doCancelOrder}
}
