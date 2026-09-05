<template>
  <div data-alias="panel-system-admin-invoice" class="max-w-6xl">
    <h2 class="text-xl font-semibold text-gray-900 mb-1">开票申请</h2>
    <p class="text-sm text-gray-500 mb-6">此处用于知晓租户提交的开票请求；请在微信商户平台手动开具发票后，点击「已开具」登记。</p>

    <div class="mb-4 flex flex-wrap items-center gap-3">
      <label class="text-sm text-gray-700">状态筛选</label>
      <select v-model="statusFilter" class="border border-gray-300 rounded-md px-3 py-2 text-sm" @change="loadItems">
        <option value="pending">待审批</option>
        <option value="">全部</option>
        <option value="approved">已通过</option>
        <option value="rejected">已拒绝</option>
      </select>
      <button
        type="button"
        class="px-3 py-2 text-sm border border-gray-300 rounded-md hover:bg-gray-50"
        :disabled="loading"
        @click="loadItems"
      >{{ loading ? '刷新中…' : '刷新' }}</button>
    </div>

    <div v-if="loadError" class="mb-4 p-3 bg-red-50 text-red-800 rounded-lg text-sm" :data-traceId="loadErrorTraceId || undefined">
      {{ loadError }}
    </div>

    <div class="overflow-x-auto bg-white border border-gray-200 rounded-lg">
      <table class="min-w-full text-sm">
        <thead class="bg-gray-50 text-gray-600">
          <tr>
            <th class="px-3 py-2 text-left">申请 ID</th>
            <th class="px-3 py-2 text-left">租户</th>
            <th class="px-3 py-2 text-left">订单</th>
            <th class="px-3 py-2 text-left">抬头</th>
            <th class="px-3 py-2 text-left">状态</th>
            <th class="px-3 py-2 text-left">操作</th>
          </tr>
          <!-- OPT-20260823-045: 管理端表头列过滤（复用 HeaderTextFilter，与退款/用户表一致） -->
          <tr data-testid="invoice-list-column-filters" class="bg-gray-50/60">
            <td class="px-3 py-2">
              <HeaderTextFilter
                :model-value="idFilter"
                aria-label="申请 ID"
                placeholder="申请 ID"
                data-alias="InvoiceFilterId"
                @update:model-value="onFilterUpdate('id', $event)"
              />
            </td>
            <td class="px-3 py-2">
              <HeaderTextFilter
                :model-value="tenantFilter"
                aria-label="租户 ID"
                placeholder="租户 ID"
                data-alias="InvoiceFilterTenantId"
                @update:model-value="onFilterUpdate('tenant_id', $event)"
              />
            </td>
            <td class="px-3 py-2">
              <HeaderTextFilter
                :model-value="orderFilter"
                aria-label="订单 ID"
                placeholder="订单 ID"
                data-alias="InvoiceFilterOrderId"
                @update:model-value="onFilterUpdate('order_id', $event)"
              />
            </td>
            <td class="px-3 py-2">
              <HeaderTextFilter
                :model-value="buyerNameFilter"
                aria-label="抬头名称"
                placeholder="抬头名称"
                data-alias="InvoiceFilterBuyerName"
                @update:model-value="onFilterUpdate('buyer_name', $event)"
              />
            </td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2 whitespace-nowrap">
              <!-- Anti-Replay-OK: 只读列表过滤重置，发 GET 不写资源 -->
              <button
                type="button"
                data-testid="invoice-list-column-filters-reset"
                class="px-2 py-1 text-xs border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
                @click="resetColumnFilters"
              >
                重置
              </button>
            </td>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="row in items" :key="row.id">
            <td class="px-3 py-2 font-mono text-xs">{{ row.id }}</td>
            <td class="px-3 py-2 font-mono text-xs">{{ row.tenant_id }}</td>
            <td class="px-3 py-2 font-mono text-xs">{{ row.order_id }}</td>
            <td class="px-3 py-2">
              <div>
                {{ row.buyer_name }}（{{ row.buyer_type === 'ORGANIZATION' ? '企业' : '个人' }}）
                <span
                  class="ml-1 inline-block px-1.5 py-0.5 text-xs rounded"
                  :class="row.invoice_type === 'special' ? 'bg-yellow-100 text-yellow-800' : 'bg-gray-100 text-gray-600'"
                  data-testid="invoice-type-badge"
                >{{ row.invoice_type === 'special' ? '专票' : '普票' }}</span>
              </div>
              <div v-if="row.invoice_type === 'special'" class="mt-1 text-xs text-gray-500 leading-5" data-testid="invoice-special-info">
                税号：{{ row.taxpayer_id }}<br>
                地址：{{ row.address }}<br>
                电话：{{ row.telephone }}<br>
                开户行：{{ row.bank_name }}<br>
                账号：{{ row.bank_account }}
              </div>
              <div v-else-if="row.buyer_type === 'ORGANIZATION'" class="mt-1 text-xs text-gray-500">税号：{{ row.taxpayer_id }}</div>
            </td>
            <td class="px-3 py-2">{{ statusLabel(row.status) }}</td>
            <td class="px-3 py-2">
              <template v-if="row.status === 'pending'">
                <button
                  type="button"
                  class="mr-2 px-2 py-1 text-xs border border-gray-300 rounded disabled:opacity-50"
                  data-testid="invoice-upload-btn"
                  :disabled="uploadingId === row.id"
                  @click="pickFile(row)"
                >{{ row.invoice_file_path ? '重新上传发票' : '上传发票' }}</button>
                <a
                  v-if="row.invoice_file_url"
                  :href="row.invoice_file_url"
                  target="_blank"
                  rel="noopener"
                  class="mr-2 text-xs text-blue-600 underline"
                  data-testid="invoice-file-link"
                >查看已上传</a>
                <input
                  v-model="fapiaoNumberInput[row.id]"
                  type="text"
                  maxlength="32"
                  placeholder="微信发票号码（可选）"
                  class="mr-2 mb-1 w-40 px-2 py-1 text-xs border border-gray-300 rounded"
                  data-testid="invoice-fapiao-number-input"
                >
                <button
                  type="button"
                  class="mr-2 px-2 py-1 text-xs text-white bg-primary rounded disabled:opacity-50"
                  data-testid="invoice-issued-btn"
                  :disabled="actionId === row.id || uploadingId === row.id"
                  :aria-busy="actionId === row.id ? 'true' : 'false'"
                  @click="act(row, 'approve')"
                >已开具</button>
                <button
                  type="button"
                  class="px-2 py-1 text-xs border border-gray-300 rounded"
                  data-testid="invoice-reject-btn"
                  :disabled="actionId === row.id || uploadingId === row.id"
                  @click="act(row, 'reject')"
                >拒绝</button>
              </template>
              <span v-else class="text-gray-400">—</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <input
      ref="fileInputRef"
      type="file"
      accept=".pdf,.jpg,.jpeg,.png"
      class="hidden"
      data-testid="invoice-file-input"
      @change="onFileChange"
    >
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { createClickGuard } from '../../utils/clickGuard.js'
import HeaderTextFilter from '../HeaderTextFilter.vue'

const items = ref([])
const loading = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
const statusFilter = ref('pending')
// OPT-20260823-045: 表头列过滤（申请 ID / 租户 / 订单 / 抬头），debounce 后自动拉取。
const idFilter = ref('')
const tenantFilter = ref('')
const orderFilter = ref('')
const buyerNameFilter = ref('')
let columnFilterTimer = null
const actionId = ref('')
const uploadingId = ref('')
const uploadTarget = ref(null)
const fileInputRef = ref(null)
const fapiaoNumberInput = ref({})
const guard = createClickGuard()

const statusLabel = (s) => ({ pending: '待审批', approved: '已通过', rejected: '已拒绝', cancelled: '已取消' }[s] || s)

const onFilterUpdate = (key, value) => {
  if (key === 'id') idFilter.value = value
  else if (key === 'tenant_id') tenantFilter.value = value
  else if (key === 'order_id') orderFilter.value = value
  else if (key === 'buyer_name') buyerNameFilter.value = value
  clearTimeout(columnFilterTimer)
  columnFilterTimer = setTimeout(() => {
    loadItems()
  }, 400)
}

const resetColumnFilters = () => {
  idFilter.value = ''
  tenantFilter.value = ''
  orderFilter.value = ''
  buyerNameFilter.value = ''
  clearTimeout(columnFilterTimer)
  loadItems()
}

const loadItems = async () => {
  loading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const q = new URLSearchParams()
    if (statusFilter.value) q.set('status', statusFilter.value)
    if (String(idFilter.value || '').trim()) q.set('id', String(idFilter.value).trim())
    if (String(tenantFilter.value || '').trim()) q.set('tenant_id', String(tenantFilter.value).trim())
    if (String(orderFilter.value || '').trim()) q.set('order_id', String(orderFilter.value).trim())
    if (String(buyerNameFilter.value || '').trim()) q.set('buyer_name', String(buyerNameFilter.value).trim())
    const url = `/api/system-admin/invoice-applications/${q.toString() ? `?${q}` : ''}`
    const r = await apiFetch(url, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await r.json().catch(() => ({}))
    if (!r.ok) {
      loadError.value = data.error || data.message || '加载失败'
      loadErrorTraceId.value = extractTraceId(r) || extractTraceId(data) || ''
      return
    }
    items.value = Array.isArray(data.results) ? data.results : []
  } catch (e) {
    loadError.value = e.message || '加载失败'
    loadErrorTraceId.value = extractTraceId(e) || ''
  } finally {
    loading.value = false
  }
}

const pickFile = (row) => {
  uploadTarget.value = row
  fileInputRef.value?.click()
}

// 上传管理员手动开具的发票文件（PDF/JPEG/PNG，≤10MiB）；仅 pending 申请可挂载。
const onFileChange = async (e) => {
  const file = e.target.files?.[0]
  const row = uploadTarget.value
  e.target.value = ''
  uploadTarget.value = null
  if (!file || !row) return
  uploadingId.value = row.id
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const fd = new FormData()
    fd.append('file', file)
    const r = await apiFetch(`/api/system-admin/invoice-applications/${row.id}/invoice-file/`, {
      method: 'POST',
      credentials: 'include',
      body: fd,
    })
    const data = await r.json().catch(() => ({}))
    if (!r.ok) {
      loadError.value = data.error || data.message || '上传失败'
      loadErrorTraceId.value = extractTraceId(r) || extractTraceId(data) || ''
      return
    }
    await loadItems()
  } catch (err) {
    loadError.value = err.message || '上传失败'
    loadErrorTraceId.value = extractTraceId(err) || ''
  } finally {
    uploadingId.value = ''
  }
}

const act = async (row, action) => {
  await guard.run(async ({ idempotencyKey }) => {
    actionId.value = row.id
    try {
      // OPT-20260823-053：手动开具登记可附带微信发票号码，作为跨系统对账凭证。
      const fapiaoNumber = action === 'approve' ? (fapiaoNumberInput.value[row.id] || '').trim() : ''
      const r = await apiFetch(`/api/system-admin/invoice-applications/${row.id}/${action}/`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
          'Idempotency-Key': idempotencyKey,
        },
        body: JSON.stringify({
          note: action === 'approve' ? '已手动开具' : '资料不符',
          fapiao_number: fapiaoNumber,
        }),
      })
      const data = await r.json().catch(() => ({}))
      if (!r.ok) {
        loadError.value = data.error || data.message || '操作失败'
        loadErrorTraceId.value = extractTraceId(r) || extractTraceId(data) || ''
        return
      }
      if (action === 'approve') delete fapiaoNumberInput.value[row.id]
      await loadItems()
    } catch (e) {
      loadError.value = e.message || '操作失败'
      loadErrorTraceId.value = extractTraceId(e) || ''
    } finally {
      actionId.value = ''
    }
  })
}

onMounted(loadItems)
onUnmounted(() => {
  clearTimeout(columnFilterTimer)
})
</script>
