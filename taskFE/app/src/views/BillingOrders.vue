<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold mb-6">订单列表</h1>

    <BillingOrderNumberSearch
      :searching="loading"
      :search-query="tradeNoQuery"
      @search="onTradeNoSearch"
      @clear="onTradeNoClear"
    />

    <!-- 状态筛选 -->
    <div class="flex gap-2 mb-6" data-testid="order-status-filters">
      <button
        v-for="s in statusFilters"
        :key="s.value"
        class="px-3 py-1.5 text-xs rounded-full border transition-colors"
        :class="statusFilter === s.value
          ? 'bg-primary text-white border-primary'
          : 'bg-white text-gray-600 border-gray-300 hover:border-gray-400'"
        @click="statusFilter = s.value; currentPage = 1; fetchOrders()"
      >
        {{ s.label }}
      </button>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="text-center py-12 text-gray-500">加载中…</div>

    <!-- 空状态 -->
    <div v-else-if="orders.length === 0" class="bg-white rounded-xl shadow-md p-12 text-center text-gray-400">
      暂无订单记录
    </div>

    <!-- 订单列表 -->
    <div v-else class="bg-white rounded-xl shadow-md overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr>
            <th class="text-left px-4 py-3 font-medium text-gray-600" style="width: 32px"></th>
            <th class="text-left px-4 py-3 font-medium text-gray-600">订单号</th>
            <th class="text-left px-4 py-3 font-medium text-gray-600">状态</th>
            <th class="text-right px-4 py-3 font-medium text-gray-600">金额（元）</th>
            <th class="text-left px-4 py-3 font-medium text-gray-600">支付方式</th>
            <th class="text-left px-4 py-3 font-medium text-gray-600">创建时间</th>
            <th class="text-left px-4 py-3 font-medium text-gray-600">支付时间</th>
            <th class="text-center px-4 py-3 font-medium text-gray-600" style="width: 160px">操作</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="o in orders" :key="o.id">
            <!-- 订单摘要行 -->
            <tr
              class="border-b border-gray-100 hover:bg-gray-50 transition-colors cursor-pointer"
              :data-order-id="String(o.id)"
              @click="toggleExpand(o)"
            >
              <td class="px-4 py-3 text-gray-400">
                <svg
                  class="w-4 h-4 transition-transform duration-200"
                  :class="isExpanded(o.id) ? 'rotate-90' : 'rotate-0'"
                  fill="none" stroke="currentColor" viewBox="0 0 24 24"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
              </td>
              <td class="px-4 py-3 font-mono text-gray-900">
                <a
                  class="text-gray-900 hover:text-primary hover:underline break-all"
                  :href="`/tenant/${tenantId}/billing/orders/${o.id}/`"
                  @click.stop
                >{{ o.order_number }}</a>
              </td>
              <td class="px-4 py-3">
                <span
                  class="inline-block px-2 py-0.5 text-xs rounded-full font-medium"
                  :class="statusClass(o.status)"
                >
                  {{ statusLabel(o.status) }}
                </span>
              </td>
              <td class="px-4 py-3 text-right font-mono text-gray-900">{{ o.total_yuan }}</td>
              <td class="px-4 py-3 text-gray-600">{{ paymentLabel(o.payment_method) }}</td>
              <td class="px-4 py-3 text-gray-500 text-xs">{{ formatTime(o.created_at) }}</td>
              <td class="px-4 py-3 text-gray-500 text-xs">{{ o.paid_at ? formatTime(o.paid_at) : '—' }}</td>
              <td class="px-4 py-3 text-center" @click.stop>
                <div class="flex items-center justify-center gap-2">
                  <button
                    v-if="o.status === 'pending'"
                    class="px-3 py-1.5 text-xs font-medium text-white bg-primary rounded-md hover:bg-primary/90 transition-colors"
                    @click="openPayModal(o)"
                  >
                    支付
                  </button>
                  <button
                    v-if="o.status === 'pending'"
                    class="px-3 py-1.5 text-xs font-medium text-red-600 border border-red-300 rounded-md hover:bg-red-50 transition-colors"
                    @click="confirmCancelOrder(o)"
                  >
                    取消订单
                  </button>
                  <span v-if="o.status === 'paid' && o.payment_method === 'admin_grant'" class="text-xs text-green-600">已支付（赠送）</span>
                  <span v-else-if="o.status === 'refunded'" class="text-xs text-gray-500">已退款</span>
                  <template v-else-if="o.status === 'paid'">
                    <span
                      v-if="orderRefundAppStatus(o, refundApplications) === 'pending'"
                      class="text-xs text-orange-600"
                      data-testid="billing-refund-pending-label"
                    >退款中</span>
                    <template v-else>
                      <span class="text-xs text-green-600 mr-2">已支付</span>
                      <button
                        v-if="refundEnabled && isOrderRefundable(o, refundApplications)"
                        class="px-3 py-1.5 text-xs font-medium text-orange-600 border border-orange-300 rounded-md hover:bg-orange-50 transition-colors"
                        data-testid="billing-refund-apply-btn"
                        @click="openRefundConfirm(o)"
                      >
                        <!-- Anti-Replay-OK: ui-only 打开确认弹层 -->
                        申请退款
                      </button>
                    </template>
                  </template>
                  <span v-else-if="o.status === 'cancelled'" class="text-xs text-gray-400">已取消</span>
                  <span v-else-if="o.status === 'expired'" class="text-xs text-gray-400">已过期</span>
                </div>
              </td>
            </tr>
            <!-- 展开详情行 -->
            <tr v-if="isExpanded(o.id)" class="bg-gray-50 border-b border-gray-200">
              <td colspan="99" class="px-6 py-4">
                <OrderExpandDetail
                  :loading="!!expandLoading[o.id]"
                  :items="expandItems[o.id] || []"
                  :consumption="expandConsumption[o.id]"
                  :out-trade-no="expandOutTradeNo[o.id] || ''"
                  :wechat-transaction-id="expandWechatTransactionId[o.id] || ''"
                  out-trade-no-label="商户单号"
                  wechat-transaction-id-label="交易单号"
                />
              </td>
            </tr>
          </template>
        </tbody>
      </table>

      <!-- 分页 -->
      <div v-if="totalPages > 1" class="flex items-center justify-between px-4 py-3 border-t border-gray-200 bg-gray-50">
        <span class="text-sm text-gray-500">共 {{ total }} 条，第 {{ currentPage }} / {{ totalPages }} 页</span>
        <div class="flex gap-2">
          <button
            class="px-3 py-1 text-sm border border-gray-300 rounded-md hover:bg-gray-100 disabled:opacity-40 disabled:cursor-not-allowed"
            :disabled="currentPage <= 1"
            @click="goPage(currentPage - 1)"
          >
            上一页
          </button>
          <button
            class="px-3 py-1 text-sm border border-gray-300 rounded-md hover:bg-gray-100 disabled:opacity-40 disabled:cursor-not-allowed"
            :disabled="currentPage >= totalPages"
            @click="goPage(currentPage + 1)"
          >
            下一页
          </button>
        </div>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="errorMessage" class="mt-4 p-4 bg-red-50 text-red-800 rounded-lg text-sm" :data-traceId="errorTraceId || undefined">
      {{ errorMessage }}
    </div>

    <!-- 支付弹窗 (OPT-20260726-034: 集成手机验证门禁) -->
    <PayOrderModal
      v-if="payModalVisible"
      :order="payTarget"
      :code-url="payCodeUrl"
      :mode="payMode"
      :polling-err="payPollingErr"
      :polling-err-trace-id="payPollingErrTraceId"
      :tenant-id="tenantId"
      :phone-gate-active="phoneGateActive"
      :phone-verification-status="phoneVerificationStatus"
      :payment-terms-gate-active="paymentTermsGateActive"
      :payment-terms-doc="paymentTermsDoc"
      :payment-terms-consenting="paymentTermsConsenting"
      @close="closePayModal"
      @phone-verified="onPhoneVerified"
      @payment-terms-accepted="onPaymentTermsAccepted"
    />

    <!-- 取消确认弹窗 -->
    <CancelOrderModal
      v-if="cancelConfirmVisible"
      :order="cancelTarget"
      :cancelling="cancelling"
      @close="cancelConfirmVisible = false"
      @confirm="doCancelOrder"
    />

    <BillingRefundConfirmModal
      v-if="refundConfirmOpen"
      :order="refundTarget"
      :consumption="refundConsumption"
      :apply-error="applyError"
      :apply-error-trace-id="applyErrorTraceId"
      :applying="applying"
      :submit="submitListRefund"
      @close="closeRefundConfirm"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch } from '../utils/apiUtils'
import { useBillingOrderActions } from '../composables/useBillingOrderActions'
import { useBillingRefund } from '../composables/useBillingRefund.js'
import { useBillingOrderIdDeepLink } from '../composables/useBillingOrderIdDeepLink.js'
import { parseOrderIdQuery } from '../utils/billingOrderDeepLink.js'
import { safeJson, safeResponseJson } from '@/utils/safeResponseJson.js'
import PayOrderModal from '../components/PayOrderModal.vue'
import CancelOrderModal from '../components/CancelOrderModal.vue'
import BillingRefundConfirmModal from '../components/BillingRefundConfirmModal.vue'
import OrderExpandDetail from '../components/system-admin/OrderExpandDetail.vue'
import BillingOrderNumberSearch from '../components/BillingOrderNumberSearch.vue'
import {
  BILLING_ORDER_STATUS_FILTERS as statusFilters,
  billingOrderStatusLabel as statusLabel,
  billingOrderStatusClass as statusClass,
  billingOrderPaymentLabel as paymentLabel,
  billingOrderFormatTime as formatTime,
  isBillingOrderRefundable as isOrderRefundable,
  billingOrderRefundApplicationStatus as orderRefundAppStatus,
} from '@/utils/billingOrderDisplay.js'

const route = useRoute()
const router = useRouter()
const tenantId = computed(() => route.params.tenant)
const orders = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = 15
const listOffset = ref(0)
const loading = ref(false)
const errorMessage = ref('')
const errorTraceId = ref('')
const statusFilter = ref('')
const tradeNoQuery = ref('')
// 展开状态
const expandedIds = ref(new Set())
const expandLoading = ref({})
const expandItems = ref({})
const expandConsumption = ref({})
const expandOutTradeNo = ref({})
const expandWechatTransactionId = ref({})

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const isExpanded = (id) => expandedIds.value.has(id)

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
    const tid = tenantId.value
    const r = await apiFetch(`/api/tenant/${encodeURIComponent(tid)}/billing/orders/${id}/`, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (r.ok) {
      const d = await safeJson(r, {})
      expandItems.value = { ...expandItems.value, [id]: d.items || [] }
      expandConsumption.value = { ...expandConsumption.value, [id]: d.resource_consumption || null }
      expandOutTradeNo.value = { ...expandOutTradeNo.value, [id]: String(d.out_trade_no || '') }
      expandWechatTransactionId.value = {
        ...expandWechatTransactionId.value,
        [id]: String(d.wechat_transaction_id || ''),
      }
    }
  } catch {
    expandItems.value = { ...expandItems.value, [id]: [] }
  } finally {
    const next = { ...expandLoading.value }
    delete next[id]
    expandLoading.value = next
  }
}

/** 深链展开：若已展开则跳过，否则走 toggleExpand */
const expandOrderForDeepLink = async (o) => {
  if (!o || isExpanded(o.id)) return
  await toggleExpand(o)
}

const fetchOrders = async () => {
  loading.value = true
  errorMessage.value = ''
  errorTraceId.value = ''

  try {
    const params = new URLSearchParams()
    params.set('limit', String(pageSize))
    params.set('offset', String((currentPage.value - 1) * pageSize))
    if (statusFilter.value) {
      params.set('status', statusFilter.value)
    }
    if (tradeNoQuery.value) {
      params.set('order_number', tradeNoQuery.value)
    }
    const focusId = parseOrderIdQuery(route.query)
    if (focusId && !tradeNoQuery.value) {
      params.set('order_id', focusId)
    }

    const tid = tenantId.value
    const url = `/api/tenant/${encodeURIComponent(tid)}/billing/orders/?${params.toString()}`

    const response = await apiFetch(url, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const { data, traceId } = await safeResponseJson(response, { fallback: {} })
    if (!response.ok) {
      errorTraceId.value = traceId
      errorMessage.value = data.error || data.detail || `请求失败（${response.status}）`
      orders.value = []
      total.value = 0
      listOffset.value = 0
      return
    }
    orders.value = data.orders || []
    total.value = data.total || 0
    const returnedOffset = Number(data.offset)
    listOffset.value = Number.isFinite(returnedOffset) ? returnedOffset : (currentPage.value - 1) * pageSize
    if (Number.isFinite(returnedOffset) && returnedOffset !== (currentPage.value - 1) * pageSize) {
      currentPage.value = Math.floor(returnedOffset / pageSize) + 1
    }
  } catch (e) {
    errorTraceId.value = e.traceId || ''
    errorMessage.value = e.message || '网络错误'
    orders.value = []
    total.value = 0
    listOffset.value = 0
  } finally {
    loading.value = false
  }
}

const resetTradeNoSearchState = () => {
  currentPage.value = 1
  expandedIds.value = new Set()
  expandItems.value = {}
  expandConsumption.value = {}
  expandOutTradeNo.value = {}
  expandWechatTransactionId.value = {}
}

const onTradeNoSearch = (q) => {
  tradeNoQuery.value = q
  resetTradeNoSearchState()
  fetchOrders()
  // OPT-20260829-006: 查询写入 URL query 便于分享/刷新保持；与 order_id 深链互斥
  const query = { ...route.query }
  const normalized = String(q || '').trim()
  if (normalized) {
    query.order_number = normalized
  } else {
    delete query.order_number
  }
  delete query.order_id
  router.replace({ query })
}

const onTradeNoClear = () => {
  tradeNoQuery.value = ''
  resetTradeNoSearchState()
  fetchOrders()
  const query = { ...route.query }
  delete query.order_number
  delete query.order_id
  router.replace({ query })
}

const goPage = (p) => {
  if (p < 1 || p > totalPages.value) return
  currentPage.value = p
  expandedIds.value = new Set()
  expandItems.value = {}
  expandConsumption.value = {}
  expandOutTradeNo.value = {}
  expandWechatTransactionId.value = {}
  fetchOrders()
}

// 订单操作（支付/取消）— 通过 composable 管理
const {
  payModalVisible, payTarget, payCodeUrl, payMode, payPollingErr, payPollingErrTraceId,
  openPayModal, closePayModal,
  phoneGateActive, phoneVerificationStatus, onPhoneVerified,
  paymentTermsGateActive, paymentTermsDoc, paymentTermsConsenting, onPaymentTermsAccepted,
  cancelConfirmVisible, cancelTarget, cancelling,
  confirmCancelOrder, doCancelOrder: _rawDoCancel,
} = useBillingOrderActions({
  tenantId,
  onOrderChanged: fetchOrders,
})

// 包装 cancel，注入 errorMessage 回调
const doCancelOrder = () => _rawDoCancel((msg) => { errorMessage.value = msg; errorTraceId.value = '' })

// ---- 退款申请逻辑 ----
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
const refundTarget = ref(null)
const refundConsumption = ref(null)

const openRefundConfirm = (order) => {
  refundTarget.value = order
  refundConsumption.value = expandConsumption.value[order?.id] || null
  applyError.value = ''
  applyErrorTraceId.value = ''
  refundConfirmOpen.value = true
}

const closeRefundConfirm = () => {
  if (applying.value) return
  refundConfirmOpen.value = false
  refundTarget.value = null
  refundConsumption.value = null
}

const submitListRefund = async (reason, idempotencyKey) => {
  const ok = await applyRefund(reason, refundTarget.value?.id, idempotencyKey)
  if (ok) {
    refundConfirmOpen.value = false
    refundTarget.value = null
    refundConsumption.value = null
    await refreshRefund()
  }
  return ok
}

useBillingOrderIdDeepLink({
  route,
  router,
  orders,
  loading,
  listOffset,
  pageSize,
  currentPage,
  expandOrder: expandOrderForDeepLink,
  onFocusMiss: (orderId) => {
    errorMessage.value = `未找到订单 ${orderId}（可能已被删除或不属于当前租户）`
  },
})

// 深链进入时清掉状态筛选，避免目标订单被滤掉
watch(
  () => route.query?.order_id,
  (id) => {
    if (String(id || '').trim()) statusFilter.value = ''
  },
  { immediate: true },
)

// OPT-20260829-006: 深链进入 —— URL 带 order_number 时预填查询框并请求
onMounted(() => {
  const q = String(route.query?.order_number || '').trim()
  if (q) {
    tradeNoQuery.value = q
  }
  fetchOrders()
})

// URL 变化（分享链接 / 浏览器前进后退）时同步查询；本组件自写的 query 已被
// tradeNoQuery 占用，直接短路避免二次请求。
watch(
  () => route.query?.order_number,
  (val) => {
    const q = String(val || '').trim()
    if (q === tradeNoQuery.value) return
    tradeNoQuery.value = q
    resetTradeNoSearchState()
    fetchOrders()
  },
)
</script>
