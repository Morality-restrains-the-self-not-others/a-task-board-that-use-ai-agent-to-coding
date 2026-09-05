/**
 * 退款审批弹层：按租户+订单拉取 resource_consumption。
 */
import { ref } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import { extractTraceId } from '../utils/traceId.js'
import { safeResponseJson } from '../utils/safeResponseJson.js'

export function useAdminRefundOrderConsumption() {
  const consumption = ref(null)
  const loading = ref(false)
  const error = ref('')
  const errorTraceId = ref('')

  const clear = () => {
    consumption.value = null
    loading.value = false
    error.value = ''
    errorTraceId.value = ''
  }

  const loadFor = async (row) => {
    consumption.value = null
    error.value = ''
    errorTraceId.value = ''
    const tenantId = String(row?.tenant_id || '').trim()
    const orderId = String(row?.order_id || '').trim()
    if (!tenantId || !orderId) {
      error.value = '该申请缺少租户或订单，无法加载资源消耗'
      loading.value = false
      return
    }
    loading.value = true
    try {
      const url = `/api/tenant/${encodeURIComponent(tenantId)}/billing/orders/${encodeURIComponent(orderId)}/`
      const response = await apiFetch(url, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      const { data, error: parseError, traceId } = await safeResponseJson(response, { fallback: {} })
      if (!response.ok) {
        error.value = data?.message || data?.error || parseError || `加载资源消耗失败（${response.status}）`
        errorTraceId.value = traceId || extractTraceId(response) || extractTraceId(data) || ''
        console.warn('[refund-order-consumption] load failed', {
          status: response.status,
          traceId: errorTraceId.value,
        })
        return
      }
      consumption.value = data?.resource_consumption || null
    } catch (e) {
      error.value = e.message || '加载资源消耗失败'
      errorTraceId.value = extractTraceId(e) || ''
      console.warn('[refund-order-consumption] load failed', { traceId: errorTraceId.value })
    } finally {
      loading.value = false
    }
  }

  return { consumption, loading, error, errorTraceId, loadFor, clear }
}
