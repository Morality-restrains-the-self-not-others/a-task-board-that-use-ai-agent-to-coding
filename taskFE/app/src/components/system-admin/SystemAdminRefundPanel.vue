<template>
  <div data-alias="panel-system-admin-refund" class="max-w-6xl">
    <h2 class="text-xl font-semibold text-gray-900 mb-1">退款审批</h2>
    <p class="text-sm text-gray-500 mb-6">
      租户提交的退款申请列表；待审批项可批准或拒绝。仅系统超级管理员可访问。
    </p>

    <div
      class="mb-6 bg-white border border-gray-200 rounded-lg p-4 flex flex-wrap items-center justify-between gap-4"
      data-testid="refund-policy-panel"
    >
      <div>
        <label for="enable-refund-applications-switch" class="block text-sm font-medium text-gray-900">
          开启退款申请
        </label>
        <p class="text-sm text-gray-500 mt-1">
          关闭后，租户账单页不显示「申请退款」按钮，且服务端拒绝新申请。
        </p>
      </div>
      <div class="flex items-center gap-3">
        <label class="relative inline-flex items-center cursor-pointer">
          <input
            id="enable-refund-applications-switch"
            v-model="policyEnabled"
            data-testid="enable-refund-applications-switch"
            type="checkbox"
            class="sr-only peer"
            :disabled="policyLoading || policySaving"
            @change="onPolicyToggle"
          >
          <div class="w-11 h-6 bg-gray-200 rounded-full peer peer-checked:bg-primary peer-focus:outline-none peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all"></div>
        </label>
        <span class="text-sm text-gray-600">{{ policyEnabled ? '已开启' : '已关闭' }}</span>
      </div>
      <div
        v-if="policyError"
        class="w-full mt-2 p-3 bg-red-50 text-red-700 rounded-lg text-sm"
        :data-traceId="policyErrorTraceId || undefined"
      >
        {{ policyError }}
      </div>
      <div v-if="policySuccess" class="w-full mt-2 p-3 bg-green-50 text-green-700 rounded-lg text-sm">
        {{ policySuccess }}
      </div>
    </div>

    <div class="mb-4 flex flex-wrap items-center gap-3">
      <label class="text-sm text-gray-700">状态筛选</label>
      <select
        v-model="statusFilter"
        class="border border-gray-300 rounded-md px-3 py-2 text-sm"
        @change="loadItems"
      >
        <option value="">全部</option>
        <option value="pending">待审批</option>
        <option value="approved">已通过</option>
        <option value="rejected">已拒绝</option>
      </select>
      <button
        type="button"
        class="px-3 py-2 text-sm border border-gray-300 rounded-md hover:bg-gray-50"
        :disabled="loading"
        @click="loadItems"
      >
        {{ loading ? '刷新中…' : '刷新' }}
      </button>
    </div>

    <div v-if="loadError" class="mb-4 p-3 bg-red-50 text-red-800 rounded-lg text-sm" :data-traceId="loadErrorTraceId || undefined">
      {{ loadError }}
    </div>

    <div class="overflow-x-auto bg-white border border-gray-200 rounded-lg">
      <table class="min-w-full text-sm">
        <thead class="bg-gray-50 text-gray-600">
          <tr>
            <th class="px-3 py-2 text-left">申请 ID</th>
            <th class="px-3 py-2 text-left">租户 ID</th>
            <th class="px-3 py-2 text-left">关联订单</th>
            <th class="px-3 py-2 text-left">申请原因</th>
            <th class="px-3 py-2 text-right">冻结金额（元）</th>
            <th class="px-3 py-2 text-left">状态</th>
            <th class="px-3 py-2 text-left">申请时间</th>
            <th class="px-3 py-2 text-left">操作</th>
          </tr>
          <!-- OPT-20260823-045: 管理端表头列过滤（复用 HeaderTextFilter，与用户表一致） -->
          <tr data-testid="refund-list-column-filters" class="bg-gray-50/60">
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2">
              <HeaderTextFilter
                :model-value="tenantFilter"
                aria-label="租户 ID"
                placeholder="租户 ID"
                data-alias="RefundFilterTenantId"
                @update:model-value="onFilterUpdate('tenant_id', $event)"
              />
            </td>
            <td class="px-3 py-2">
              <HeaderTextFilter
                :model-value="orderFilter"
                aria-label="关联订单"
                placeholder="订单 ID"
                data-alias="RefundFilterOrderId"
                @update:model-value="onFilterUpdate('order_id', $event)"
              />
            </td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2 whitespace-nowrap">
              <!-- Anti-Replay-OK: 只读列表过滤重置，发 GET 不写资源 -->
              <button
                type="button"
                data-testid="refund-list-column-filters-reset"
                class="px-2 py-1 text-xs border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
                @click="resetColumnFilters"
              >
                重置
              </button>
            </td>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <template v-for="row in items" :key="row.id">
          <tr>
            <td class="px-3 py-2 font-mono text-xs">{{ row.id }}</td>
            <td class="px-3 py-2 font-mono text-xs">{{ row.tenant_id }}</td>
            <td class="px-3 py-2 font-mono text-xs">
              <template v-if="row.order_id">
                <a
                  :href="buildSystemAdminOrderRecordsHref({ tenantId: row.tenant_id, orderId: row.order_id })"
                  class="text-blue-700 hover:underline"
                  data-testid="refund-related-order-link"
                >{{ row.order_id }}</a>
              </template>
              <span v-else class="text-gray-400">—</span>
            </td>
            <td class="px-3 py-2 max-w-xs truncate" :title="row.reason || ''">
              {{ row.reason?.trim() ? row.reason : '—' }}
            </td>
            <td class="px-3 py-2 text-right tabular-nums">{{ formatFrozenYuan(row.frozen_points) }}</td>
            <td class="px-3 py-2">{{ statusLabel(row.status) }}</td>
            <td class="px-3 py-2 whitespace-nowrap">{{ formatDate(row.created_at) }}</td>
            <td class="px-3 py-2 whitespace-nowrap">
              <button
                v-if="row.order_id"
                type="button"
                data-testid="refund-consumption-toggle"
                class="text-blue-700 hover:underline mr-3 disabled:opacity-50"
                :disabled="consumptionLoading && expandedId === row.id"
                @click="toggleConsumption(row)"
              >
                <!-- Anti-Replay-OK: ui-only 切换行内消耗预览 -->
                {{ expandedId === row.id ? '收起消耗' : '查看消耗' }}
              </button>
              <template v-if="row.status === 'pending'">
                <button
                  type="button"
                  data-testid="refund-approve-btn"
                  class="text-green-700 hover:underline mr-3 disabled:opacity-50"
                  :disabled="actionId === row.id"
                  @click="openApproveModal(row)"
                >
                  <!-- Anti-Replay-OK: ui-only 仅打开弹层 -->
                  {{ actionId === row.id && actionType === 'approve' ? '处理中…' : '批准' }}
                </button>
                <button
                  type="button"
                  data-testid="refund-reject-btn"
                  class="text-red-700 hover:underline disabled:opacity-50"
                  :disabled="actionId === row.id"
                  @click="openRejectModal(row)"
                >
                  <!-- Anti-Replay-OK: ui-only 仅打开弹层 -->
                  {{ actionId === row.id && actionType === 'reject' ? '处理中…' : '拒绝' }}
                </button>
              </template>
              <span v-else-if="!row.order_id" class="text-gray-400">—</span>
            </td>
          </tr>
          <tr
            v-if="expandedId === row.id"
            :key="`${row.id}-consumption`"
            data-testid="refund-consumption-inline"
          >
            <td colspan="8" class="px-3 py-2 bg-gray-50/60">
              <div v-if="consumptionLoading" class="text-sm text-gray-500">正在加载资源消耗…</div>
              <div
                v-else-if="consumptionError"
                class="text-sm text-red-700"
                :data-traceId="consumptionErrorTraceId || undefined"
              >
                {{ consumptionError }}
              </div>
              <OrderResourceConsumption v-else :consumption="consumption" always-show />
            </td>
          </tr>
          </template>
          <tr v-if="!loading && items.length === 0">
            <td colspan="8" class="px-3 py-8 text-center text-gray-500">暂无退款申请</td>
          </tr>
        </tbody>
      </table>
    </div>

    <SystemAdminRefundApproveModal
      :open="approveModalOpen"
      :target="approveTarget"
      :reason="approveRefundReason"
      :reason-error="approveReasonError"
      :busy="actionId === approveTarget?.id"
      :action-error="actionError"
      :action-error-trace-id="actionErrorTraceId"
      @update:reason="approveRefundReason = $event"
      @cancel="closeApproveModal"
      @confirm="confirmApprove"
    />
    <SystemAdminRefundRejectModal
      :open="rejectModalOpen"
      :target="rejectTarget"
      :note="rejectRefundNote"
      :busy="actionId === rejectTarget?.id"
      :action-error="actionError"
      :action-error-trace-id="actionErrorTraceId"
      @update:note="rejectRefundNote = $event"
      @cancel="closeRejectModal"
      @confirm="confirmReject"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { apiFetch } from '../../utils/apiUtils'
import { buildSystemAdminOrderRecordsHref } from '../../utils/billingOrderDeepLink.js'
import { extractTraceId } from '../../utils/traceId.js'
import { humanizePaymentProviderError } from '../../utils/humanizePaymentProviderError.js'
import modalService from '../../utils/modalService.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../../utils/clickGuard.js'
import { refundStatusLabel as statusLabel, formatRefundDate as formatDate, formatFrozenYuan } from './refundPanelFormat.js'
import SystemAdminRefundApproveModal from './SystemAdminRefundApproveModal.vue'
import SystemAdminRefundRejectModal from './SystemAdminRefundRejectModal.vue'
import OrderResourceConsumption from '../OrderResourceConsumption.vue'
import HeaderTextFilter from '../HeaderTextFilter.vue'
import { useAdminRefundOrderConsumption } from '../../composables/useAdminRefundOrderConsumption.js'

const DEFAULT_APPROVE_REFUND_REASON = '订单退款'

const items = ref([])
const loading = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
// OPT-20260823-045: 表头列过滤（租户 ID / 关联订单），debounce 后自动拉取。
const tenantFilter = ref('')
const orderFilter = ref('')
let columnFilterTimer = null

const onFilterUpdate = (key, value) => {
  if (key === 'tenant_id') tenantFilter.value = value
  else if (key === 'order_id') orderFilter.value = value
  clearTimeout(columnFilterTimer)
  columnFilterTimer = setTimeout(() => {
    loadItems()
  }, 400)
}

const resetColumnFilters = () => {
  tenantFilter.value = ''
  orderFilter.value = ''
  clearTimeout(columnFilterTimer)
  loadItems()
}
const statusFilter = ref('pending')
const actionId = ref('')
const actionType = ref('')
const approveModalOpen = ref(false)
const approveTarget = ref(null)
const approveRefundReason = ref(DEFAULT_APPROVE_REFUND_REASON)
const approveReasonError = ref('')
const actionError = ref('')
const actionErrorTraceId = ref('')
// OPT-20260819-017: 拒绝也用自定义模态框（不再 window.prompt）
const rejectModalOpen = ref(false)
const rejectTarget = ref(null)
const rejectRefundNote = ref('')
// OPT-20260820-042: 行内资源消耗预览，复用批准/拒绝弹层同一数据源
const {
  consumption,
  loading: consumptionLoading,
  error: consumptionError,
  errorTraceId: consumptionErrorTraceId,
  loadFor: loadConsumption,
  clear: clearConsumption,
} = useAdminRefundOrderConsumption()
const expandedId = ref('')

const policyEnabled = ref(true)
const policyLoading = ref(false)
const policySaving = ref(false)
const policyError = ref('')
const policyErrorTraceId = ref('')
const policySuccess = ref('')


const loadPolicy = async () => {
  policyLoading.value = true
  policyError.value = ''
  policyErrorTraceId.value = ''
  try {
    const response = await apiFetch('/api/system-admin/refund-policy/', {
      credentials: 'include',
      headers: { Accept: 'application/json' }})
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      policyError.value = data.message || data.error || `策略加载失败（${response.status}）`
      policyErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
      return
    }
    policyEnabled.value = data.enabled !== false
  } catch (e) {
    policyError.value = e.message || '策略加载失败'
    policyErrorTraceId.value = extractTraceId(e) || ''
  } finally {
    policyLoading.value = false
  }
}

const savePolicy = async () => {
  policySaving.value = true
  policyError.value = ''
  policyErrorTraceId.value = ''
  policySuccess.value = ''
  try {
    const response = await apiFetch('/api/system-admin/refund-policy/', {
      method: 'PUT',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json'},
      body: JSON.stringify({ enabled: !!policyEnabled.value })})
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      policyError.value = data.message || data.error || `保存失败（${response.status}）`
      policyErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
      await loadPolicy()
      return
    }
    policyEnabled.value = data.enabled !== false
    policySuccess.value = policyEnabled.value ? '已开启退款申请' : '已关闭退款申请'
  } catch (e) {
    policyError.value = e.message || '保存失败'
    policyErrorTraceId.value = extractTraceId(e) || ''
    await loadPolicy()
  } finally {
    policySaving.value = false
  }
}

/** v-model 已翻转到目标值；取消确认时回滚，确认后才 PUT。 */
const onPolicyToggle = async () => {
  const nextEnabled = !!policyEnabled.value
  const prevEnabled = !nextEnabled
  const message = nextEnabled
    ? '确定开启退款申请？开启后租户可在账单页提交退款申请。'
    : '确定关闭退款申请？关闭后租户账单页将隐藏「申请退款」按钮，且服务端拒绝新申请。'
  try {
    await modalService.confirm(message, '确认更改退款策略', '确定', '取消')
  } catch {
    policyEnabled.value = prevEnabled
    return
  }
  await savePolicy()
}

const loadItems = async () => {
  loading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const q = new URLSearchParams()
    if (statusFilter.value) q.set('status', statusFilter.value)
    if (String(tenantFilter.value || '').trim()) q.set('tenant_id', String(tenantFilter.value).trim())
    if (String(orderFilter.value || '').trim()) q.set('order_id', String(orderFilter.value).trim())
    const url = `/api/system-admin/refund-applications/${q.toString() ? `?${q}` : ''}`
    const response = await apiFetch(url, {
      credentials: 'include',
      headers: { Accept: 'application/json' }})
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      loadError.value = data.message || data.error || `加载失败（${response.status}）`
      loadErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
      items.value = []
      return
    }
    items.value = Array.isArray(data.results) ? data.results : []
  } catch (e) {
    loadError.value = e.message || '加载失败'
    loadErrorTraceId.value = extractTraceId(e) || ''
    items.value = []
  } finally {
    loading.value = false
  }
}

const openApproveModal = (row) => {
  approveTarget.value = row
  approveRefundReason.value = DEFAULT_APPROVE_REFUND_REASON
  approveReasonError.value = ''
  actionError.value = ''
  actionErrorTraceId.value = ''
  approveModalOpen.value = true
}

const toggleConsumption = (row) => {
  if (expandedId.value === row.id) {
    expandedId.value = ''
    clearConsumption()
    return
  }
  expandedId.value = row.id
  loadConsumption(row)
}

const closeApproveModal = () => {
  if (actionId.value) return
  approveModalOpen.value = false
  approveTarget.value = null
  approveRefundReason.value = DEFAULT_APPROVE_REFUND_REASON
  approveReasonError.value = ''
  actionError.value = ''
  actionErrorTraceId.value = ''
}

const approveGuard = createClickGuard()
const rejectGuard = createClickGuard()

const confirmApprove = async () => {
  const row = approveTarget.value
  if (!row) return
  const note = String(approveRefundReason.value || '').trim()
  if (!note) {
    approveReasonError.value = '退款原因不能为空'
    return
  }
  approveReasonError.value = ''
  await approveGuard.run(async ({ idempotencyKey }) => {
    const ok = await postAction(row, 'approve', note, idempotencyKey)
    if (ok) closeApproveModal()
  })
}

const openRejectModal = (row) => {
  rejectTarget.value = row
  rejectRefundNote.value = ''
  actionError.value = ''
  actionErrorTraceId.value = ''
  rejectModalOpen.value = true
}

const closeRejectModal = () => {
  if (actionId.value) return
  rejectModalOpen.value = false
  rejectTarget.value = null
  rejectRefundNote.value = ''
  actionError.value = ''
  actionErrorTraceId.value = ''
}

const confirmReject = async () => {
  const row = rejectTarget.value
  if (!row) return
  await rejectGuard.run(async ({ idempotencyKey }) => {
    const ok = await postAction(
      row,
      'reject',
      String(rejectRefundNote.value || '').trim(),
      idempotencyKey,
    )
    if (ok) closeRejectModal()
  })
}

const postAction = async (row, action, noteOverride, idempotencyKey) => {
  actionId.value = row.id
  actionType.value = action
  loadError.value = ''
  loadErrorTraceId.value = ''
  actionError.value = ''
  actionErrorTraceId.value = ''
  try {
    const note =
      noteOverride !== undefined
        ? String(noteOverride || '').trim()
        : DEFAULT_APPROVE_REFUND_REASON
    const response = await apiFetch(
      `/api/system-admin/refund-applications/${encodeURIComponent(row.id)}/${action}/`,
      {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            Accept: 'application/json',
          },
          idempotencyKey,
        ),
        body: JSON.stringify({ note: String(note || '').trim() })}
    )
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      const raw = data.message || data.error || `操作失败（${response.status}）`
      const msg = humanizePaymentProviderError(raw, `操作失败（${response.status}）`)
      const tid = extractTraceId(response) || extractTraceId(data) || ''
      actionError.value = msg
      actionErrorTraceId.value = tid
      loadError.value = msg
      loadErrorTraceId.value = tid
      return false
    }
    await loadItems()
    return true
  } catch (e) {
    const msg = humanizePaymentProviderError(e.message || '操作失败', '操作失败')
    const tid = extractTraceId(e) || ''
    actionError.value = msg
    actionErrorTraceId.value = tid
    loadError.value = msg
    loadErrorTraceId.value = tid
    return false
  } finally {
    actionId.value = ''
    actionType.value = ''
  }
}

onMounted(() => {
  loadPolicy()
  loadItems()
})

onUnmounted(() => {
  clearTimeout(columnFilterTimer)
})
</script>
