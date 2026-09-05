<template>
  <div data-testid="referral-wechat-profit-sharing-tab">
    <div class="flex flex-wrap items-center justify-between gap-2 mb-2">
      <p class="text-xs text-gray-500" data-testid="referral-ps-receiver-status">
        接收方登记：{{ receiverStatusLabel }}
      </p>
      <button
        type="button"
        class="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-40"
        data-testid="referral-ps-sync"
        :disabled="syncBusy || loading || items.length === 0"
        :aria-busy="syncBusy ? 'true' : 'false'"
        @click="onSync"
      >
        {{ syncBusy ? '同步中…' : '同步微信状态' }}
      </button>
    </div>

    <p
      v-if="receiverStatus === 'pending_openid'"
      class="text-xs text-amber-700 mb-2"
      data-testid="referral-ps-pending-openid"
    >
      推荐人尚未绑定微信登录，无法向微信发起分账。请先让其用微信扫码登录后再重试。
    </p>

    <p
      v-if="loadError"
      class="text-sm text-red-600 mb-2"
      data-testid="referral-ps-error"
      :data-traceId="loadErrorTraceId || undefined"
    >
      {{ loadError }}
    </p>

    <p v-if="loading" class="text-sm text-gray-400 py-4 text-center">加载中…</p>

    <p
      v-else-if="items.length === 0 && !loadError"
      class="text-sm text-gray-400 py-4 text-center"
      data-testid="referral-ps-empty"
    >
      暂无微信分账记录
    </p>

    <div v-else class="space-y-3">
      <ProfitSharingShareReasonForm
        v-if="shareRow"
        :order-label="String(shareRow.order_number || shareRow.order_id || '')"
        :model-value="shareReason"
        :share-busy="shareBusy"
        form-id="referral-ps-share-form"
        test-id-prefix="referral-ps"
        @update:model-value="shareReason = $event"
        @confirm="onShareConfirm"
        @cancel="closeShare"
      />
      <div class="overflow-x-auto">
      <table class="min-w-full text-sm border border-gray-200 rounded-lg overflow-hidden">
        <thead class="bg-gray-50">
          <tr>
            <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">被推荐人</th>
            <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">AppID</th>
            <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">OpenID</th>
            <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">订单号</th>
            <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">商户单号</th>
            <th class="text-right px-2 py-1.5 border-b font-medium text-gray-500 text-xs">分账金额</th>
            <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">本地状态</th>
            <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">最早可分账</th>
            <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">失败原因</th>
            <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">微信订单号</th>
            <th
              class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs"
              title="微信分账单号优先；未返回时展示商户侧 out_order_no（同一分账记录上的幂等键，非独立实体）"
            >微信分账单号</th>
            <th
              class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs"
              data-testid="referral-ps-wechat-state-col"
              title="微信支付分账单查询状态，不是用户是否绑定微信"
            >微信分账单状态</th>
            <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in items"
            :key="row.id"
            class="border-b hover:bg-gray-50"
            :class="isShareRow(row) ? 'bg-amber-50' : ''"
          >
            <td class="px-2 py-1.5 font-mono text-xs text-gray-700">{{ row.referred_user_id || '—' }}</td>
            <td
              class="px-2 py-1.5 font-mono text-xs text-gray-700"
              data-testid="referral-ps-app-id"
            >
              {{ row.app_id || '—' }}
            </td>
            <td
              class="px-2 py-1.5 font-mono text-xs text-gray-700"
              data-testid="referral-ps-openid"
            >
              {{ row.openid || '—' }}
            </td>
            <td class="px-2 py-1.5 font-mono text-xs">
              <a
                :href="buildSystemAdminOrderRecordsHref({ tenantId: row.tenant_id, orderId: row.order_id })"
                class="text-blue-700 hover:underline"
                data-testid="referral-ps-order-link"
              >{{ row.order_number || row.order_id }}</a>
            </td>
            <td
              class="px-2 py-1.5 text-xs font-mono"
              data-testid="referral-ps-out-trade-no"
            >
              {{ row.out_trade_no || '—' }}
            </td>
            <td class="px-2 py-1.5 text-right text-xs tabular-nums">{{ row.amount_yuan || '—' }}</td>
            <td class="px-2 py-1.5 text-xs">{{ statusLabel(row.status) }}</td>
            <td class="px-2 py-1.5 text-xs whitespace-nowrap">{{ formatDate(row.settle_after) }}</td>
            <td
              class="px-2 py-1.5 text-xs min-w-[18rem] max-w-[36rem] whitespace-normal break-all align-top"
              data-testid="referral-ps-fail-reason"
              :data-traceId="row.fail_trace_id || undefined"
              :title="profitSharingFailReasonLabel(row.fail_reason)"
            >
              {{ profitSharingFailReasonLabel(row.fail_reason) }}
            </td>
            <td
              class="px-2 py-1.5 text-xs font-mono"
              data-testid="referral-ps-wechat-txn"
            >
              {{ row.wechat_transaction_id || '—' }}
            </td>
            <td
              class="px-2 py-1.5 text-xs font-mono"
              data-testid="referral-ps-wechat-order"
              :title="profitSharingBillNoTitle(row)"
            >
              {{ profitSharingBillNoLabel(row) }}
            </td>
            <td
              class="px-2 py-1.5 text-xs"
              :class="row.wechat_error ? 'text-red-600' : ''"
              data-testid="referral-ps-wechat-state"
              :title="wechatCellTitle(row)"
              :data-traceId="row.wechat_error ? (row.wechat_error_trace_id || undefined) : undefined"
            >
              {{ wechatCell(row) }}
            </td>
            <td class="px-2 py-1.5 text-xs whitespace-nowrap">
              <button
                v-if="canShare(row)"
                type="button"
                class="px-2 py-1 text-xs border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-40"
                data-testid="referral-ps-share"
                :disabled="shareBusy"
                :aria-expanded="isShareRow(row) ? 'true' : 'false'"
                :aria-controls="isShareRow(row) ? 'referral-ps-share-form' : undefined"
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
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { adminFormatDate as formatDate } from '../utils/adminFormatDate.js'
import { buildSystemAdminOrderRecordsHref } from '../utils/billingOrderDeepLink.js'
import { createClickGuard } from '../utils/clickGuard.js'
import { profitSharingFailReasonLabel } from '../utils/profitSharingFailReasonLabel.js'
import { profitSharingBillNoLabel, profitSharingBillNoTitle } from '../utils/profitSharingBillNoLabel.js'
import {
  profitSharingWechatStateLabel,
  profitSharingWechatStateTitle,
} from '../utils/profitSharingWechatStateLabel.js'
import ProfitSharingShareReasonForm from './ProfitSharingShareReasonForm.vue'

const props = defineProps({
  userId: { type: [String, Number], default: '' },
})

const STATUS_LABEL = {
  pending: '待分账',
  processing: '分账中',
  finished: '已完成',
  failed: '失败',
}

const RECEIVER_LABEL = {
  registered: '已登记',
  failed: '登记失败',
  pending_openid: '未绑定',
  skipped_not_live: '未绑定',
  deleted: '未绑定',
}

const items = ref([])
const loading = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
const receiverStatus = ref('')
const syncBusy = ref(false)
const syncGuard = createClickGuard()
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

const receiverStatusLabel = computed(() => RECEIVER_LABEL[receiverStatus.value] || '未绑定')

const statusLabel = (status) => STATUS_LABEL[String(status || '').trim()] || status || '—'

function wechatCell(row) {
  if (row?.wechat_error) return row.wechat_error
  return profitSharingWechatStateLabel(row?.wechat_state)
}

function wechatCellTitle(row) {
  return profitSharingWechatStateTitle(row?.wechat_state, row?.wechat_error)
}

async function loadItems() {
  const uid = String(props.userId || '').trim()
  if (!uid) return
  loading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const q = new URLSearchParams()
    q.set('referrer_user_id', uid)
    q.set('status', 'all')
    q.set('limit', '50')
    q.set('offset', '0')
    // Anti-Replay-OK: 只读 GET 列表
    const resp = await apiFetch(`/api/system-admin/profit-sharing/?${q}`, {
      method: 'GET',
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok) {
      loadError.value = profitSharingFailReasonLabel(data.error || data.message || '加载失败')
      loadErrorTraceId.value = extractTraceId(resp) || extractTraceId(data) || ''
      items.value = []
      return
    }
    items.value = Array.isArray(data.items) ? data.items : []
    receiverStatus.value = String(data.receiver_registration_status || '').trim()
  } catch (e) {
    loadError.value = e.message || '加载失败'
    loadErrorTraceId.value = extractTraceId(e) || ''
    items.value = []
  } finally {
    loading.value = false
  }
}

async function onSync() {
  const uid = String(props.userId || '').trim()
  const result = await syncGuard.run(async () => {
    syncBusy.value = true
    loadError.value = ''
    loadErrorTraceId.value = ''
    try {
      const ids = items.value.map((row) => String(row.id || '').trim()).filter(Boolean)
      const resp = await apiFetch('/api/system-admin/profit-sharing/refresh-wechat/', {
        method: 'POST',
        credentials: 'include',
        headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
        body: JSON.stringify({ referrer_user_id: uid, ids }),
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) {
        loadError.value = profitSharingFailReasonLabel(data.error || data.message || '同步失败')
        loadErrorTraceId.value = extractTraceId(resp) || extractTraceId(data) || ''
        return
      }
      const byID = new Map()
      for (const row of Array.isArray(data.items) ? data.items : []) {
        byID.set(String(row.id), row)
      }
      items.value = items.value.map((row) => {
        const next = byID.get(String(row.id))
        if (!next) return row
        return {
          ...row,
          wechat_state: next.wechat_state || '',
          wechat_error: next.wechat_error || '',
          wechat_error_trace_id: next.wechat_error_trace_id || '',
        }
      })
    } catch (e) {
      loadError.value = e.message || '同步失败'
      loadErrorTraceId.value = extractTraceId(e) || ''
    } finally {
      syncBusy.value = false
    }
  })
  if (result?.skipped) {
    return
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

watch(
  () => String(props.userId || ''),
  (uid) => {
    if (uid) loadItems()
  },
  { immediate: true },
)
</script>
