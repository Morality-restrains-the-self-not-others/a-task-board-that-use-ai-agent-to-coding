import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils.js'
import { getCookie } from '../utils/cookieUtils'
import { extractTraceId } from '../utils/traceId.js'
import { mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const STATUS_PENDING = 'pending'

/**
 * 租户退款申请：加载申请列表、提交申请、同步冻结金额。
 * @returns composable API
 */
export function useBillingRefund() {
  const route = useRoute()
  const tenantId = computed(() => route.params.tenant)

  const frozenBalance = ref(0)
  const refundEnabled = ref(true)
  const applications = ref([])
  const pendingApplication = ref(null)
  const applyError = ref('')
  const applyErrorTraceId = ref('')
  const applying = ref(false)
  const loading = ref(false)

  
  const syncPending = () => {
    const pending = (applications.value || []).find((item) => item?.status === STATUS_PENDING)
    pendingApplication.value = pending || null
  }

  const fetchBalanceFrozen = async () => {
    const tid = String(tenantId.value || '').trim()
    if (!tid) return
    try {
      // taskBill handleBalance：/api/tenant/{tid}/billing/accounts/balance/
      const response = await apiFetch(`/api/tenant/${tid}/billing/accounts/balance/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' }})
      if (!response.ok) return
      const data = await response.json().catch(() => ({}))
      frozenBalance.value = Number(data.frozen_balance) || 0
      if (Object.prototype.hasOwnProperty.call(data, 'refund_enabled')) {
        refundEnabled.value = data.refund_enabled !== false
      }
    } catch (e) {
      console.error('fetchBalanceFrozen failed:', e)
    }
  }

  const loadApplications = async () => {
    const tid = String(tenantId.value || '').trim()
    if (!tid) return
    loading.value = true
    try {
      // taskBill handleTenantRefundApplications：/api/tenant/{tid}/billing/refund-applications/
      const response = await apiFetch(`/api/tenant/${tid}/billing/refund-applications/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' }})
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        applyError.value =
          (typeof data.error === 'string' && data.error) ||
          (typeof data.message === 'string' && data.message) ||
          '无法加载退款申请'
        applyErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
        return
      }
      applications.value = Array.isArray(data.results) ? data.results : []
      if (Object.prototype.hasOwnProperty.call(data, 'enabled')) {
        refundEnabled.value = data.enabled !== false
      }
      syncPending()
    } catch (e) {
      console.error('loadApplications failed:', e)
      applyError.value = e?.message || '无法加载退款申请'
      applyErrorTraceId.value = extractTraceId(e) || ''
    } finally {
      loading.value = false
    }
  }

  const refresh = async () => {
    await Promise.all([fetchBalanceFrozen(), loadApplications()])
  }

  const buildRefundBody = (reason, orderId) => {
    return { reason: String(reason || '').trim(), order_id: String(orderId) }
  }

  /**
   * 提交退款申请（冻结当前账户金额）。
   * @param {string} reason 退款原因（必填）
   * @param {string|number} orderId 关联订单 ID（必填）
   */
  const applyRefund = async (reason = '', orderId = '', idempotencyKey = '') => {
    const tid = String(tenantId.value || '').trim()
    if (!tid) {
      applyError.value = '缺少租户 ID'
      applyErrorTraceId.value = ''
      return false
    }
    applying.value = true
    applyError.value = ''
    applyErrorTraceId.value = ''
    try {
      const response = await apiFetch(`/api/tenant/${tid}/billing/refund-applications/`, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders({
          'Content-Type': 'application/json',
          Accept: 'application/json',
        }, idempotencyKey),
        body: JSON.stringify(buildRefundBody(reason, orderId))})
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        applyError.value =
          (typeof data.error === 'string' && data.error) ||
          (typeof data.message === 'string' && data.message) ||
          '提交退款申请失败'
        applyErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
        return false
      }
      await refresh()
      return true
    } catch (e) {
      applyError.value = e?.message || '提交退款申请失败'
      applyErrorTraceId.value = extractTraceId(e) || ''
      return false
    } finally {
      applying.value = false
    }
  }

  onMounted(() => {
    refresh()
  })

  return {
    frozenBalance,
    refundEnabled,
    applications,
    pendingApplication,
    applyError,
    applyErrorTraceId,
    applying,
    loading,
    loadApplications,
    fetchBalanceFrozen,
    applyRefund,
    refresh}
}
