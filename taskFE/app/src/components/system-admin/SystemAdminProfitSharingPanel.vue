<template>
  <div data-alias="panel-system-admin-profit-sharing" class="max-w-6xl">
    <h2 class="text-xl font-semibold text-gray-900 mb-1">待分账订单</h2>
    <p class="text-sm text-gray-500 mb-6">
      已支付且需向推荐人分账的订单。默认显示待处理（待分账 / 分账中 / 失败）。平台员工可填写审计缘由后向微信发起分账。
      微信支付商户后台「分账动账通知」请配置
      https://www.daydaymoney.com/api/billing/profitsharing/change-notify/ 。
    </p>

    <div class="mb-4 flex flex-wrap items-center gap-2">
      <button
        v-for="s in statusFilters"
        :key="s.value"
        type="button"
        class="px-3 py-1.5 text-xs rounded-full border transition-colors"
        :class="statusFilter === s.value
          ? 'bg-primary text-white border-primary'
          : 'bg-white text-gray-600 border-gray-300 hover:border-gray-400'"
        @click="setStatus(s.value)"
      >
        <!-- Anti-Replay-OK: 只读筛选，无写操作 -->
        {{ s.label }}
      </button>
      <button
        type="button"
        class="ml-auto px-3 py-2 text-sm border border-gray-300 rounded-md hover:bg-gray-50"
        :disabled="loading"
        @click="loadItems"
      >
        <!-- Anti-Replay-OK: 只读刷新 -->
        {{ loading ? '刷新中…' : '刷新' }}
      </button>
    </div>

    <div
      v-if="loadError"
      class="mb-4 p-3 bg-red-50 text-red-800 rounded-lg text-sm"
      data-testid="profit-sharing-error"
      :data-traceId="loadErrorTraceId || undefined"
    >
      {{ loadError }}
    </div>

    <div v-if="loading" class="text-center py-12 text-gray-500">加载中…</div>

    <div v-else class="space-y-3">
      <ProfitSharingShareReasonForm
        v-if="shareRow"
        :order-label="String(shareRow.order_number || shareRow.order_id || '')"
        :model-value="shareReason"
        :share-busy="shareBusy"
        form-id="profit-sharing-share-form"
        test-id-prefix="profit-sharing"
        @update:model-value="shareReason = $event"
        @confirm="onShareConfirm"
        @cancel="closeShare"
      />
      <div class="overflow-x-auto bg-white border border-gray-200 rounded-lg">
      <table class="min-w-full text-sm">
        <thead class="bg-gray-50 text-gray-600">
          <tr>
            <th class="px-3 py-2 text-left">订单号</th>
            <th class="px-3 py-2 text-left">商户单号</th>
            <th class="px-3 py-2 text-left">租户 ID</th>
            <th class="px-3 py-2 text-left">分账接收方</th>
            <th class="px-3 py-2 text-left">AppID</th>
            <th class="px-3 py-2 text-left">OpenID</th>
            <th class="px-3 py-2 text-right">订单金额（元）</th>
            <th class="px-3 py-2 text-right">分账金额（元）</th>
            <th class="px-3 py-2 text-left">状态</th>
            <th class="px-3 py-2 text-left">最早分账时间</th>
            <th class="px-3 py-2 text-left">失败原因</th>
            <th class="px-3 py-2 text-left">微信订单号</th>
            <th
              class="px-3 py-2 text-left"
              title="微信分账单号优先；未返回时展示商户侧 out_order_no（同一分账记录上的幂等键，非独立实体）"
            >微信分账单号</th>
            <th class="px-3 py-2 text-left">操作</th>
          </tr>
          <!-- OPT-20260823-045: 管理端表头列过滤（订单号 / 租户 ID / 接收方 ID，复用 HeaderTextFilter） -->
          <tr data-testid="profit-sharing-list-column-filters" class="bg-gray-50/60">
            <td class="px-3 py-2">
              <HeaderTextFilter
                :model-value="orderNumberFilter"
                aria-label="订单号"
                placeholder="订单号"
                data-alias="ProfitSharingFilterOrderNumber"
                @update:model-value="onFilterUpdate('order_number', $event)"
              />
            </td>
            <td class="px-3 py-2">
              <HeaderTextFilter
                :model-value="outTradeNoFilter"
                aria-label="商户单号"
                placeholder="商户单号"
                data-alias="ProfitSharingFilterOutTradeNo"
                @update:model-value="onFilterUpdate('out_trade_no', $event)"
              />
            </td>
            <td class="px-3 py-2">
              <HeaderTextFilter
                :model-value="tenantIdFilter"
                aria-label="租户 ID"
                placeholder="租户 ID"
                data-alias="ProfitSharingFilterTenantId"
                @update:model-value="onFilterUpdate('tenant_id', $event)"
              />
            </td>
            <td class="px-3 py-2">
              <HeaderTextFilter
                :model-value="receiverFilter"
                aria-label="接收方 ID"
                placeholder="接收方 ID"
                data-alias="ProfitSharingFilterReceiver"
                @update:model-value="onFilterUpdate('receiver_user_id', $event)"
              />
            </td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2"></td>
            <td class="px-3 py-2 whitespace-nowrap">
              <!-- Anti-Replay-OK: 只读列表过滤重置，发 GET 不写资源 -->
              <button
                type="button"
                data-testid="profit-sharing-list-column-filters-reset"
                class="px-2 py-1 text-xs border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
                @click="resetColumnFilters"
              >
                重置
              </button>
            </td>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-if="items.length === 0">
            <td
              colspan="14"
              data-testid="profit-sharing-empty"
              class="text-center py-12 text-gray-400"
            >
              暂无待分账订单
            </td>
          </tr>
          <tr v-for="row in items" :key="row.id" :class="isShareRow(row) ? 'bg-amber-50' : ''">
            <td class="px-3 py-2 font-mono text-xs">
              <a
                :href="buildSystemAdminOrderRecordsHref({ tenantId: row.tenant_id, orderId: row.order_id })"
                class="text-blue-700 hover:underline"
                data-testid="profit-sharing-order-link"
              >{{ row.order_number || row.order_id }}</a>
            </td>
            <td
              class="px-3 py-2 font-mono text-xs"
              data-testid="profit-sharing-out-trade-no"
            >
              {{ row.out_trade_no || '—' }}
            </td>
            <td class="px-3 py-2 font-mono text-xs">{{ row.tenant_id }}</td>
            <td class="px-3 py-2 font-mono text-xs">{{ row.receiver_display || row.receiver_user_id }}</td>
            <td
              class="px-3 py-2 font-mono text-xs"
              data-testid="profit-sharing-app-id"
            >
              {{ row.app_id || '—' }}
            </td>
            <td
              class="px-3 py-2 font-mono text-xs"
              data-testid="profit-sharing-openid"
            >
              {{ row.openid || '—' }}
            </td>
            <td class="px-3 py-2 text-right tabular-nums">{{ row.total_yuan }}</td>
            <td class="px-3 py-2 text-right tabular-nums">{{ row.amount_yuan }}</td>
            <td class="px-3 py-2">{{ statusLabel(row.status) }}</td>
            <td class="px-3 py-2 whitespace-nowrap">{{ formatDate(row.settle_after) }}</td>
            <td
              class="px-3 py-2 text-xs min-w-[18rem] max-w-[36rem] whitespace-normal break-all align-top"
              data-testid="profit-sharing-fail-reason"
              :data-traceId="row.fail_trace_id || undefined"
              :title="profitSharingFailReasonLabel(row.fail_reason)"
            >
              {{ profitSharingFailReasonLabel(row.fail_reason) }}
            </td>
            <td
              class="px-3 py-2 font-mono text-xs"
              data-testid="profit-sharing-wechat-txn"
            >
              {{ row.wechat_transaction_id || '—' }}
            </td>
            <td
              class="px-3 py-2 font-mono text-xs"
              data-testid="profit-sharing-wechat-order"
              :title="profitSharingBillNoTitle(row)"
            >
              {{ profitSharingBillNoLabel(row) }}
            </td>
            <td class="px-3 py-2 whitespace-nowrap">
              <button
                v-if="canShare(row)"
                type="button"
                class="px-2 py-1 text-xs border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-40"
                data-testid="profit-sharing-share"
                :disabled="shareBusy"
                :aria-expanded="isShareRow(row) ? 'true' : 'false'"
                :aria-controls="isShareRow(row) ? 'profit-sharing-share-form' : undefined"
                :aria-busy="shareBusy && isShareRow(row) ? 'true' : 'false'"
                @click="openShare(row)"
              >
                <!-- Anti-Replay-OK: 仅打开缘由表单，写出站在确认按钮 -->
                {{ isShareRow(row) ? '取消' : '分账' }}
              </button>
              <span v-else>—</span>
            </td>
          </tr>
        </tbody>
      </table>
      </div>
    </div>

    <div v-if="total > limit" class="mt-4 flex items-center justify-end gap-3 text-sm text-gray-600">
      <span>共 {{ total }} 条</span>
      <button
        type="button"
        class="px-3 py-1 border rounded disabled:opacity-40"
        :disabled="offset === 0 || loading"
        @click="prevPage"
      >
        上一页
      </button>
      <button
        type="button"
        class="px-3 py-1 border rounded disabled:opacity-40"
        :disabled="offset + items.length >= total || loading"
        @click="nextPage"
      >
        下一页
      </button>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { buildSystemAdminOrderRecordsHref } from '../../utils/billingOrderDeepLink.js'
import { profitSharingFailReasonLabel } from '../../utils/profitSharingFailReasonLabel.js'
import { profitSharingBillNoLabel, profitSharingBillNoTitle } from '../../utils/profitSharingBillNoLabel.js'
import { createClickGuard } from '../../utils/clickGuard.js'
import HeaderTextFilter from '../HeaderTextFilter.vue'
import ProfitSharingShareReasonForm from '../ProfitSharingShareReasonForm.vue'

const STATUS_LABEL = {
  pending: '待分账',
  processing: '分账中',
  finished: '已完成',
  failed: '失败',
}

const statusFilters = [
  { value: 'open', label: '待处理' },
  { value: 'pending', label: '待分账' },
  { value: 'processing', label: '分账中' },
  { value: 'failed', label: '失败' },
  { value: 'finished', label: '已完成' },
  { value: 'all', label: '全部' },
]

const items = ref([])
const loading = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
const statusFilter = ref('open')
const total = ref(0)
const limit = 20
const offset = ref(0)
// OPT-20260823-045: 表头列过滤（订单号 / 租户 ID / 接收方 ID），debounce 后自动拉取。
const orderNumberFilter = ref('')
const tenantIdFilter = ref('')
const receiverFilter = ref('')
// OPT-20260825-023: 商户单号表头列过滤（微信 out_trade_no / payment_ref 去前缀）。
const outTradeNoFilter = ref('')
let columnFilterTimer = null
const shareGuard = createClickGuard()
const shareRow = ref(null)
const shareReason = ref('')
const shareBusy = ref(false)

function canShare(row) {
  const st = String(row?.status || '').trim()
  return st === 'pending' || st === 'failed'
}

function isShareRow(row) {
  return !!shareRow.value && String(shareRow.value.id) === String(row?.id)
}

function openShare(row) {
  if (shareBusy.value) return
  if (isShareRow(row)) {
    closeShare()
    return
  }
  shareRow.value = row
  shareReason.value = ''
}

function closeShare() {
  if (shareBusy.value) return
  shareRow.value = null
  shareReason.value = ''
}

function patchShareFailure(rowId, failReason, traceId) {
  items.value = items.value.map((it) => {
    if (String(it.id) !== String(rowId)) return it
    return {
      ...it,
      status: 'failed',
      fail_reason: failReason || it.fail_reason,
      fail_trace_id: traceId || '',
    }
  })
}

const statusLabel = (status) => STATUS_LABEL[String(status || '').trim()] || status || '—'

const formatDate = (value) => {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const setStatus = (value) => {
  statusFilter.value = value
  offset.value = 0
  loadItems()
}

const onFilterUpdate = (key, value) => {
  if (key === 'order_number') orderNumberFilter.value = value
  else if (key === 'tenant_id') tenantIdFilter.value = value
  else if (key === 'receiver_user_id') receiverFilter.value = value
  else if (key === 'out_trade_no') outTradeNoFilter.value = value
  clearTimeout(columnFilterTimer)
  columnFilterTimer = setTimeout(() => {
    offset.value = 0
    loadItems()
  }, 400)
}

const resetColumnFilters = () => {
  orderNumberFilter.value = ''
  tenantIdFilter.value = ''
  receiverFilter.value = ''
  outTradeNoFilter.value = ''
  clearTimeout(columnFilterTimer)
  offset.value = 0
  loadItems()
}

const prevPage = () => {
  offset.value = Math.max(0, offset.value - limit)
  loadItems()
}

const nextPage = () => {
  offset.value += limit
  loadItems()
}

const loadItems = async () => {
  loading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const q = new URLSearchParams()
    q.set('status', statusFilter.value)
    q.set('limit', String(limit))
    q.set('offset', String(offset.value))
    if (String(orderNumberFilter.value || '').trim()) q.set('order_number', String(orderNumberFilter.value).trim())
    if (String(tenantIdFilter.value || '').trim()) q.set('tenant_id', String(tenantIdFilter.value).trim())
    if (String(receiverFilter.value || '').trim()) q.set('receiver_user_id', String(receiverFilter.value).trim())
    if (String(outTradeNoFilter.value || '').trim()) q.set('out_trade_no', String(outTradeNoFilter.value).trim())
    const url = `/api/system-admin/profit-sharing/?${q}`
    const response = await apiFetch(url, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      loadError.value = data.message || data.error || `加载失败（${response.status}）`
      loadErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
      items.value = []
      total.value = 0
      return
    }
    items.value = Array.isArray(data.items) ? data.items : []
    total.value = Number(data.total) || 0
  } catch (e) {
    loadError.value = e.message || '加载失败'
    loadErrorTraceId.value = extractTraceId(e) || ''
    items.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

async function onShareConfirm() {
  const row = shareRow.value
  if (!row?.id) return
  const reason = String(shareReason.value || '').trim()
  const n = Array.from(reason).length
  if (n < 8 || n > 500) {
    loadError.value = '缘由须为 8～500 字'
    loadErrorTraceId.value = ''
    return
  }
  const result = await shareGuard.run(async ({ idempotencyKey }) => {
    shareBusy.value = true
    loadError.value = ''
    loadErrorTraceId.value = ''
    try {
      const resp = await apiFetch(`/api/system-admin/profit-sharing/${encodeURIComponent(row.id)}/share/`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
          'Idempotency-Key': idempotencyKey,
        },
        body: JSON.stringify({ reason }),
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) {
        const raw = data.error || data.message || '分账失败'
        const tid = extractTraceId(resp) || extractTraceId(data) || ''
        loadError.value = profitSharingFailReasonLabel(raw)
        loadErrorTraceId.value = tid
        patchShareFailure(row.id, raw, tid)
        return
      }
      shareRow.value = null
      shareReason.value = ''
      await loadItems()
    } catch (e) {
      loadError.value = e.message || '分账失败'
      loadErrorTraceId.value = extractTraceId(e) || ''
      patchShareFailure(row.id, loadError.value, loadErrorTraceId.value)
    } finally {
      shareBusy.value = false
    }
  })
  if (result?.skipped) {
    return
  }
}

onMounted(loadItems)
onUnmounted(() => {
  clearTimeout(columnFilterTimer)
})
</script>
