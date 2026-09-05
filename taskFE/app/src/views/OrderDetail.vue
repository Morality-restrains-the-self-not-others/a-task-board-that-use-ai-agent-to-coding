<template>
  <div class="p-6" data-alias="view-order-detail">
    <a
      class="text-sm text-primary hover:underline"
      :href="`${tenantPath}/billing/orders/`"
    >返回订单列表</a>
    <h1 class="text-2xl font-bold mt-2 mb-6">订单详情</h1>

    <div v-if="loading" class="text-gray-500 py-8">加载中…</div>
    <div
      v-else-if="loadError"
      class="p-4 bg-red-50 text-red-800 rounded-lg text-sm"
      :data-traceId="loadErrorTraceId || undefined"
    >{{ loadError }}</div>

    <template v-else-if="order">
      <div class="bg-white rounded-xl shadow-md p-6 mb-6">
        <div class="mb-4 p-3 bg-gray-50 rounded-lg space-y-1">
          <p class="text-sm">订单号：<strong class="break-all">{{ order.order_number }}</strong></p>
          <p class="text-sm" data-testid="order-wechat-transaction-id">交易单号：<strong class="break-all">{{ order.wechat_transaction_id || '—' }}</strong></p>
          <p class="text-sm" data-testid="order-out-trade-no">商户单号：<strong class="break-all">{{ order.out_trade_no || '—' }}</strong></p>
          <p class="text-sm">金额：<strong>{{ order.total_yuan }} 元</strong></p>
          <p class="text-sm">状态：<span data-testid="order-status-label" :class="order.status === 'paid' ? 'text-green-600' : 'text-amber-600'">{{ statusLabel(order.status) }}</span></p>
          <p
            v-if="order.buyer_note"
            class="text-sm whitespace-pre-wrap"
            data-testid="order-buyer-note"
          >施工留言：{{ order.buyer_note }}</p>
        </div>

        <h2 class="text-lg font-semibold mb-3">下单资源</h2>
        <div v-if="order.items && order.items.length" data-testid="order-detail-items">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left text-gray-500 border-b border-gray-200">
                <th class="pb-2 pr-4 font-medium">资源类型</th>
                <th class="pb-2 pr-4 font-medium">区域</th>
                <th class="pb-2 pr-4 text-right font-medium">数量</th>
                <th class="pb-2 pr-4 text-right font-medium">单价（元）</th>
                <th class="pb-2 text-right font-medium">小计（元）</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="it in order.items" :key="it.id" class="border-b border-gray-100">
                <td class="py-2 pr-4 text-gray-800">{{ resourceTypeLabel(it.resource_type) }}</td>
                <td class="py-2 pr-4 text-gray-700" data-testid="order-item-region">{{ it.region || '—' }}</td>
                <td class="py-2 pr-4 text-right text-gray-700">{{ it.quantity }}</td>
                <td class="py-2 pr-4 text-right font-mono text-gray-600">{{ it.unit_price_yuan }}</td>
                <td class="py-2 text-right font-mono text-gray-900">{{ it.subtotal_yuan }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else class="text-sm text-gray-400">无行项数据</p>
        <div class="mt-4">
          <OrderResourceConsumption :consumption="order.resource_consumption" />
        </div>
      </div>

      <div class="bg-white rounded-xl shadow-md p-6 mb-6">
        <PhoneVerificationGate
          v-if="order.status === 'pending'"
          :tenant-id="tenantId"
          :active="showPhoneGate"
          :phone-status="phoneVerificationStatus"
          @verified="onPhoneVerified"
        />

        <div v-if="order.status === 'pending' && !showPhoneGate" class="space-y-3">
          <h3 class="font-medium text-gray-800">选择支付方式</h3>
          <div class="flex gap-3">
            <button
              type="button"
              class="flex-1 px-4 py-3 border-2 rounded-lg text-center transition-colors border-primary bg-primary/5"
            >
              <span class="font-medium">微信支付</span>
            </button>
          </div>
          <button
            type="button"
            class="w-full px-6 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 text-lg font-medium"
            :disabled="paying || !payMethod"
            @click="payOrder"
          >
            {{ paying ? '支付中…' : `确认支付 ${order.total_yuan} 元` }}
          </button>
          <div v-if="payError" class="text-red-600 text-sm mt-2" :data-traceId="payErrorTraceId || undefined">{{ payError }}</div>
        </div>

        <div v-else-if="order.status === 'paid'" class="p-4 bg-green-50 border border-green-200 rounded-lg">
          <p class="text-green-700 font-medium">✅ 支付成功，资源已发放到您的账户</p>
          <p class="text-sm text-green-600 mt-1">请前往「账单」页面查看资源配额。</p>
          <p
            class="text-sm text-amber-800 mt-2"
            data-testid="billing-refund-consumed-notice"
          >{{ consumedNotice }}</p>
          <div class="mt-3">
            <span
              v-if="orderRefundAppStatus(order, refundApplications) === 'pending'"
              class="text-sm text-orange-600"
              data-testid="billing-refund-pending-label"
            >退款中</span>
            <button
              v-else-if="refundEnabled && isOrderRefundable(order, refundApplications)"
              type="button"
              class="px-3 py-1.5 text-sm font-medium text-orange-700 border border-orange-300 rounded-md hover:bg-orange-50"
              data-testid="billing-refund-apply-btn"
              @click="openRefundConfirm"
            >
              <!-- Anti-Replay-OK: ui-only 打开确认弹层 -->
              申请退款
            </button>
          </div>
          <OrderInvoiceSection
            :tenant-id="String(tenantId)"
            :order="order"
            @updated="loadOrder"
          />
        </div>
        <div
          v-else-if="order.status === 'refunded'"
          class="p-4 bg-amber-50 border border-amber-200 rounded-lg"
          data-testid="order-refunded-invoice-panel"
        >
          <p class="text-amber-800 font-medium">订单已退款</p>
          <p class="text-sm text-amber-700 mt-1">若已开具电子发票，请按下方提示在微信卡包确认冲红。</p>
          <OrderInvoiceSection
            :tenant-id="String(tenantId)"
            :order="order"
            @updated="loadOrder"
          />
        </div>
      </div>
    </template>

    <PayOrderModal
      v-if="wechatQrVisible"
      :order="order"
      :code-url="wechatCodeUrl"
      :mode="wechatMode"
      :tenant-id="tenantId"
      :payment-terms-gate-active="paymentTermsGateActive"
      :payment-terms-doc="paymentTermsDoc"
      :payment-terms-consenting="paymentTermsConsenting"
      :polling-err="payError"
      :polling-err-trace-id="payErrorTraceId"
      @close="wechatQrVisible = false; paymentTermsGateActive = false"
      @payment-terms-accepted="onPaymentTermsAccepted"
    />

    <BillingRefundConfirmModal
      v-if="refundConfirmOpen"
      :order="order"
      :consumption="order?.resource_consumption || null"
      :apply-error="applyError"
      :apply-error-trace-id="applyErrorTraceId"
      :applying="applying"
      :submit="submitOrderRefund"
      @close="closeRefundConfirm"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch, extractErrorMessage } from '../utils/apiUtils'
import { safeResponseJson } from '../utils/safeResponseJson.js'
import PhoneVerificationGate from '../components/PhoneVerificationGate.vue'
import PayOrderModal from '../components/PayOrderModal.vue'
import { usePaymentPoll } from '../composables/usePaymentPoll'
import { useOrderPaySse } from '../composables/useOrderPaySse'
import { billingResourceTypeLabel as resourceTypeLabel } from '../utils/billingResourceTypeLabel.js'
import OrderResourceConsumption from '../components/OrderResourceConsumption.vue'
import OrderInvoiceSection from '../components/OrderInvoiceSection.vue'
import BillingRefundConfirmModal from '../components/BillingRefundConfirmModal.vue'
import { useBillingRefund } from '../composables/useBillingRefund.js'
import { BILLING_REFUND_CONSUMED_NOTICE } from '../utils/billingRefundCopy.js'
import {
  billingOrderStatusLabel as statusLabel,
  isBillingOrderRefundable as isOrderRefundable,
  billingOrderRefundApplicationStatus as orderRefundAppStatus,
} from '../utils/billingOrderDisplay.js'
import {
  fetchCurrentPaymentTerms,
  submitPaymentTermsConsent,
  isPaymentTermsConsentRequired,
} from '../utils/paymentTermsConsent.js'

/* @alias:view-order-detail */

const route = useRoute()
const tenantId = computed(() => route.params.tenant)
const orderId = computed(() => route.params.orderId)
const tenantPath = computed(() => `/tenant/${tenantId.value}`)

const loading = ref(true)
const loadError = ref('')
const loadErrorTraceId = ref('')
const order = ref(null)

const payMethod = ref('wechat')
const paying = ref(false)
const payError = ref('')
const payErrorTraceId = ref('')
const wechatQrVisible = ref(false)
const wechatCodeUrl = ref('')
const wechatOutTradeNo = ref('')
const wechatMode = ref('')
const paymentConsentId = ref('')
const paymentTermsGateActive = ref(false)
const paymentTermsDoc = ref(null)
const paymentTermsConsenting = ref(false)

const consumedNotice = BILLING_REFUND_CONSUMED_NOTICE
const {
  refundEnabled,
  applications: refundApplications,
  applyError,
  applyErrorTraceId,
  applying,
  applyRefund,
  refresh: refreshRefund,
} = useBillingRefund()
const refundConfirmOpen = ref(false)

const openRefundConfirm = () => {
  applyError.value = ''
  applyErrorTraceId.value = ''
  refundConfirmOpen.value = true
}

const closeRefundConfirm = () => {
  if (applying.value) return
  refundConfirmOpen.value = false
}

const submitOrderRefund = async (reason, idempotencyKey) => {
  const ok = await applyRefund(reason, order.value?.id, idempotencyKey)
  if (ok) await refreshRefund()
  return ok
}

const showPhoneGate = ref(false)
const phoneVerificationStatus = ref({
  has_phone: false,
  phone_masked: '',
  phone_e164: '',
  sms_verified: false,
  required: false,
})

const checkPhoneRequired = async () => {
  try {
    const r = await apiFetch(`/api/tenant/${tenantId.value}/billing/phone-verification-status/`, {
      credentials: 'include',
      headers: { Accept: 'application/json' }})
    if (r.ok) {
      const d = await r.json().catch(() => ({}))
      phoneVerificationStatus.value = {
        has_phone: !!d.has_phone,
        phone_masked: d.phone_masked || '',
        phone_e164: d.phone_e164 || '',
        sms_verified: !!d.sms_verified,
        required: !!d.required,
      }
      showPhoneGate.value = !!(d.required && (!d.has_phone || !d.sms_verified))
    }
  } catch (e) {
    console.error('检查手机验证状态失败', e)
  }
}

const onPhoneVerified = () => {
  showPhoneGate.value = false
}

const loadOrder = async () => {
  loading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const r = await apiFetch(`/api/tenant/${tenantId.value}/billing/orders/${orderId.value}/`, {
      credentials: 'include',
      headers: { Accept: 'application/json' }})
    const d = await r.json().catch(() => ({}))
    if (!r.ok) {
      loadError.value = extractErrorMessage(d, r) || d.error || '加载订单失败'
      loadErrorTraceId.value = d.trace_id || d._traceId || r.traceId || ''
      order.value = null
      return
    }
    order.value = d
  } catch (e) {
    loadError.value = '加载订单失败: ' + e.message
    loadErrorTraceId.value = e.traceId || ''
    order.value = null
  } finally {
    loading.value = false
  }
}

const payOrder = async () => {
  payError.value = ''
  payErrorTraceId.value = ''
  paying.value = true
  try {
    const body = { payment_method: payMethod.value }
    if (paymentConsentId.value) body.consent_id = paymentConsentId.value
    const r = await apiFetch(`/api/tenant/${tenantId.value}/billing/orders/${order.value.id}/pay/`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify(body)})
    const { data: d, traceId } = await safeResponseJson(r, { fallback: {} })
    if (!r.ok) {
      payErrorTraceId.value = traceId
      if (isPaymentTermsConsentRequired(d)) {
        try {
          paymentTermsDoc.value = await fetchCurrentPaymentTerms()
          paymentTermsGateActive.value = true
          wechatQrVisible.value = true
          payError.value = ''
        } catch (e) {
          payError.value = e.message || '加载支付服务条款失败'
        }
        return
      }
      payError.value = extractErrorMessage(d, r) || d.error || '支付失败'
      return
    }
    paymentTermsGateActive.value = false
    wechatCodeUrl.value = d.code_url
    wechatOutTradeNo.value = d.out_trade_no
    wechatMode.value = d.mode || ''
    wechatQrVisible.value = true
    startOrderSse()
    startPaymentPoll()
  } catch (e) {
    payErrorTraceId.value = e.traceId || ''
    payError.value = '支付失败: ' + e.message
  } finally {
    paying.value = false
  }
}

const onPaymentTermsAccepted = async () => {
  if (!paymentTermsDoc.value?.id) {
    payError.value = '暂无支付服务条款，请联系管理员'
    return
  }
  paymentTermsConsenting.value = true
  try {
    paymentConsentId.value = await submitPaymentTermsConsent(paymentTermsDoc.value.id)
    paymentTermsGateActive.value = false
    await payOrder()
  } catch (e) {
    payError.value = e.message || '签署失败'
    payErrorTraceId.value = e.traceId || ''
  } finally {
    paymentTermsConsenting.value = false
  }
}

const applyOrderPaid = async (paidOrderId) => {
  const r = await apiFetch(`/api/tenant/${tenantId.value}/billing/orders/${paidOrderId}/`, {
    credentials: 'include',
    headers: { Accept: 'application/json' }})
  if (!r.ok) return
  const d = await r.json().catch(() => ({}))
  if (d.status === 'paid') {
    order.value = d
    wechatQrVisible.value = false
  }
}

const { startOrderSse, stopOrderSse } = useOrderPaySse({
  tenantId: () => tenantId.value,
  getOrderId: () => order.value?.id,
  isActive: (id) => wechatQrVisible.value && order.value?.id === id,
  onPaid: applyOrderPaid,
})

const { startPoll: startPaymentPoll, stopPoll: stopPaymentPoll } = usePaymentPoll({
  getOrderId: () => order.value?.id,
  isActive: (id) => wechatQrVisible.value && order.value?.id === id,
  intervalMs: 10000,
  maxAttempts: 12,
  onPoll: async (id) => {
    const r = await apiFetch(`/api/tenant/${tenantId.value}/billing/orders/${id}/`, {
      credentials: 'include',
      headers: { Accept: 'application/json' }})
    if (!r.ok) return false
    const d = await r.json().catch(() => ({}))
    if (d.status === 'paid') {
      order.value = d
      wechatQrVisible.value = false
      return true
    }
    return false
  },
})

watch(wechatQrVisible, (visible) => { if (!visible) { stopOrderSse(); stopPaymentPoll() } })
onUnmounted(() => { stopOrderSse(); stopPaymentPoll() })

onMounted(() => {
  loadOrder()
  checkPhoneRequired()
})
</script>
