<template>
  <div class="bg-white rounded-xl shadow-sm border border-border p-6 mt-6">
    <h2 class="text-lg font-semibold text-text mb-1">渠道分账</h2>
    <p class="text-sm text-text-light mb-4">
      按渠道汇总订单时段与金额。完成后 15 天内计入冻结，15–30 天内可手动分账，超过 30 天无法分账。不展示受推荐人订单号。
    </p>

    <p
      v-if="loadError"
      class="text-sm text-red-500 mb-4"
      data-testid="referral-ps-orders-error"
      :data-traceId="loadErrorTraceId || undefined"
    >
      {{ loadError }}
    </p>
    <p v-else-if="loading" class="text-sm text-text-light mb-4" data-testid="referral-ps-orders-loading">分账数据加载中...</p>

    <p
      v-if="shareError"
      class="text-sm text-red-500 mb-4"
      data-testid="referral-ps-share-error"
      :data-traceId="shareErrorTraceId || undefined"
    >
      {{ shareError }}
    </p>

    <p
      v-if="!loading && !loadError && channels.length === 0"
      class="text-sm text-text-light py-4 text-center"
      data-testid="referral-ps-orders-empty"
    >
      暂无渠道分账数据
    </p>

    <div v-else-if="!loading && !loadError" class="overflow-x-auto">
      <table
        class="min-w-full text-sm border border-border rounded-lg overflow-hidden"
        data-testid="referral-ps-orders-table"
      >
        <thead class="bg-gray-50">
          <tr>
            <th class="text-left px-4 py-3 border-b border-border">渠道号</th>
            <th class="text-left px-4 py-3 border-b border-border">订单时段</th>
            <th class="text-right px-4 py-3 border-b border-border">订单金额</th>
            <th class="text-right px-4 py-3 border-b border-border">冻结金额</th>
            <th class="text-right px-4 py-3 border-b border-border">失败金额</th>
            <th class="text-right px-4 py-3 border-b border-border">可分账金额</th>
            <th class="text-left px-4 py-3 border-b border-border">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in channels" :key="row.channel_code || '_'" class="border-b hover:bg-gray-50">
            <td class="px-4 py-3 font-mono text-xs" data-testid="referral-ps-channel-code">{{ channelLabel(row.channel_code) }}</td>
            <td class="px-4 py-3">{{ formatPeriod(row.period_from, row.period_to) }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ formatYuan(row.order_amount_yuan_cents) }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ formatYuan(row.frozen_amount_yuan_cents) }}</td>
            <td class="px-4 py-3 text-right tabular-nums text-red-600" data-testid="referral-ps-failed-amount">{{ formatYuan(row.failed_amount_yuan_cents) }}</td>
            <td class="px-4 py-3 text-right tabular-nums font-medium text-primary">{{ formatYuan(row.shareable_amount_yuan_cents) }}</td>
            <td class="px-4 py-3">
              <button
                v-if="row.shareable"
                type="button"
                class="px-3 py-1 text-sm bg-primary text-white rounded-lg hover:opacity-90 disabled:opacity-40"
                data-testid="referral-ps-share-btn"
                :disabled="sharing"
                :aria-busy="sharing ? 'true' : 'false'"
                @click="onShare(row)"
              >
                {{ sharing ? '分账中…' : '分账' }}
              </button>
              <span v-else class="text-xs text-text-light">—</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import { extractTraceId } from '../utils/traceId.js'
import { profitSharingFailReasonLabel } from '../utils/profitSharingFailReasonLabel.js'
import { createClickGuard } from '../utils/clickGuard.js'

const CHANNELS_URL = '/api/billing/profit-sharing/referrer-orders/'
const SHARE_URL = '/api/billing/profit-sharing/referrer-orders/share-channel/'

const channels = ref([])
const loading = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
const shareError = ref('')
const shareErrorTraceId = ref('')
const sharing = ref(false)
const shareGuard = createClickGuard()

const channelLabel = (code) => {
  const s = String(code || '').trim()
  return s || '未标注渠道'
}

const formatDay = (iso) => {
  const s = String(iso || '').trim()
  return s ? s.slice(0, 10) : ''
}

const formatPeriod = (from, to) => {
  const a = formatDay(from)
  const b = formatDay(to)
  if (!a && !b) return '—'
  if (a === b) return a
  return `${a} ~ ${b}`
}

const formatYuan = (cents) => {
  const n = Number(cents || 0)
  return `¥${(n / 100).toFixed(2)}`
}

async function loadChannels() {
  loading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const resp = await apiFetch(CHANNELS_URL, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok) {
      loadError.value = data.error || data.message || '获取渠道分账失败，请稍后重试'
      loadErrorTraceId.value = extractTraceId(resp) || ''
      channels.value = []
      return
    }
    channels.value = Array.isArray(data.channels) ? data.channels : []
  } catch (error) {
    console.error('获取渠道分账失败:', error)
    loadError.value = '网络错误，请稍后重试'
    loadErrorTraceId.value = extractTraceId(error) || ''
  } finally {
    loading.value = false
  }
}

async function onShare(row) {
  if (!row || !row.shareable) return
  const label = channelLabel(row.channel_code)
  if (!window.confirm(`确认对渠道 ${label} 的可分账金额发起分账？`)) return
  shareError.value = ''
  shareErrorTraceId.value = ''
  const result = await shareGuard.run(async ({ idempotencyKey, headers }) => {
    sharing.value = true
    try {
      const resp = await apiFetch(SHARE_URL, {
        method: 'POST',
        credentials: 'include',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
          ...headers,
        },
        body: JSON.stringify({
          channel_code: row.channel_code || '',
          idempotency_key: idempotencyKey,
        }),
      })
      const data = await resp.json().catch(() => ({}))
      if (!resp.ok) {
        shareError.value = profitSharingFailReasonLabel(data.error || data.detail || data.message || '分账失败，请稍后重试')
        shareErrorTraceId.value = extractTraceId(resp) || ''
        return
      }
      await loadChannels()
    } catch (error) {
      console.error('发起分账失败:', error)
      shareError.value = '网络错误，请稍后重试'
      shareErrorTraceId.value = extractTraceId(error) || ''
    } finally {
      sharing.value = false
    }
  })
  if (result.skipped) return
}

onMounted(loadChannels)
</script>
