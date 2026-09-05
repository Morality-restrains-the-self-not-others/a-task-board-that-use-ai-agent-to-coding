<template>
  <div
    v-if="visible"
    class="fixed inset-0 z-50 flex justify-end bg-black/40"
    @click.self="$emit('close')"
  >
    <div class="h-full w-full max-w-2xl bg-white shadow-xl flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-gray-100 px-5 py-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900">推荐绩效</h3>
          <p class="text-xs text-gray-500 mt-0.5">用户 ID：{{ userId }}</p>
        </div>
        <button type="button" class="text-gray-500 hover:text-gray-700" @click="$emit('close')">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Body -->
      <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4 space-y-6">
        <!-- Loading -->
        <div v-if="loading" class="flex items-center justify-center py-16 text-gray-500">
          <div class="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary mr-3" />
          加载中…
        </div>

        <!-- Error -->
        <div v-else-if="error" class="space-y-2">
          <p class="text-sm text-red-600" :data-traceId="errorTraceId || undefined">{{ error }}</p>
          <!-- Anti-Replay-OK: GET 只读重试 -->
          <button
            type="button"
            class="text-sm text-primary hover:underline"
            data-testid="referral-load-retry"
            @click="loadData"
          >
            重试
          </button>
        </div>

        <template v-else-if="data">
          <!-- 概览卡片 -->
          <div class="grid grid-cols-3 gap-4">
            <div class="bg-blue-50 rounded-lg p-4 text-center">
              <p class="text-2xl font-bold text-blue-700">{{ data.referral_count }}</p>
              <p class="text-xs text-blue-600 mt-1">推荐人数</p>
            </div>
            <div class="bg-green-50 rounded-lg p-4 text-center">
              <p class="text-2xl font-bold text-green-700">{{ fmtPoints(data.own_recharge_total) }}</p>
              <p class="text-xs text-green-600 mt-1">自身支付（元）</p>
            </div>
            <div class="bg-purple-50 rounded-lg p-4 text-center">
              <p class="text-2xl font-bold text-purple-700">{{ fmtPoints(data.referred_recharge_total) }}</p>
              <p class="text-xs text-purple-600 mt-1">推荐用户支付（元）</p>
            </div>
          </div>

          <!-- 佣金概览卡片 -->
          <div v-if="data.commission" class="grid grid-cols-2 gap-4">
            <div class="bg-amber-50 rounded-lg p-4 text-center">
              <p class="text-2xl font-bold text-amber-700">{{ fmtPoints(data.commission.pending_points) }}</p>
              <p class="text-xs text-amber-600 mt-1" data-testid="referral-pending-rate">待入账佣金（{{ commissionRateDisplay }}）</p>
            </div>
            <div class="bg-emerald-50 rounded-lg p-4 text-center">
              <p class="text-2xl font-bold text-emerald-700">{{ fmtPoints(data.commission.settled_points) }}</p>
              <p class="text-xs text-emerald-600 mt-1" data-testid="referral-settled-rate">已入账佣金（{{ commissionRateDisplay }}）</p>
            </div>
          </div>

          <!-- 时间筛选 -->
          <div class="flex items-center gap-3 text-sm">
            <label class="text-gray-600 whitespace-nowrap">时间范围：</label>
            <input
              type="date"
              v-model="startDate"
              class="px-2 py-1 border border-gray-300 rounded text-sm"
            >
            <span class="text-gray-400">至</span>
            <input
              type="date"
              v-model="endDate"
              class="px-2 py-1 border border-gray-300 rounded text-sm"
            >
            <button
              class="px-3 py-1 bg-primary text-white rounded text-sm hover:bg-primary/90"
              @click="loadData"
            >
              查询
            </button>
          </div>

          <!-- 自身支付记录 -->
          <div>
            <h4 class="text-sm font-semibold text-gray-800 mb-2">
              自身支付（{{ data.own_recharge_count }} 笔，共 {{ fmtPoints(data.own_recharge_total) }} 元）
            </h4>
            <p v-if="ownPayments.length === 0" class="text-sm text-gray-400 py-2 text-center">
              暂无自身支付记录
            </p>
            <div v-else class="overflow-x-auto">
              <table
                class="min-w-full text-sm border border-gray-200 rounded-lg overflow-hidden"
                data-testid="referral-own-payments-table"
              >
                <thead class="bg-gray-50">
                  <tr>
                    <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">时间</th>
                    <th class="text-right px-2 py-1.5 border-b font-medium text-gray-500 text-xs">金额（元）</th>
                    <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">来源</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="(p, idx) in ownPayments"
                    :key="`own-${p.paid_at}-${idx}`"
                    class="border-b hover:bg-gray-50"
                  >
                    <td class="px-2 py-1.5 text-xs text-gray-600">{{ formatDate(p.paid_at) }}</td>
                    <td class="px-2 py-1.5 text-right text-xs font-medium text-gray-900">{{ fmtPoints(p.amount) }}</td>
                    <td class="px-2 py-1.5 text-xs">
                      <span
                        class="px-1.5 py-0.5 rounded text-xs font-medium"
                        :class="resourceClass(p.points_source_type)"
                      >
                        {{ resourceLabel(p.points_source_type) }}
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- 推荐用户支付明细 / 微信分账 -->
          <div>
            <div class="flex items-center gap-2 mb-2">
              <button
                type="button"
                data-testid="referral-tab-payments"
                class="px-3 py-1 text-sm rounded-full border"
                :class="detailTab === 'payments'
                  ? 'bg-primary text-white border-primary'
                  : 'bg-white text-gray-600 border-gray-300'"
                @click="detailTab = 'payments'"
              >
                <!-- Anti-Replay-OK: 只切只读视图 -->
                支付明细
              </button>
              <button
                type="button"
                data-testid="referral-tab-wechat-ps"
                class="px-3 py-1 text-sm rounded-full border"
                :class="detailTab === 'wechat'
                  ? 'bg-primary text-white border-primary'
                  : 'bg-white text-gray-600 border-gray-300'"
                @click="detailTab = 'wechat'"
              >
                <!-- Anti-Replay-OK: 只切只读视图 -->
                微信分账
              </button>
            </div>
            <div v-show="detailTab === 'payments'">
              <h4 class="text-sm font-semibold text-gray-800 mb-2">
                推荐用户支付明细（{{ paymentDetailRows.length }} 人）
              </h4>
              <p v-if="asReferredNote" class="text-xs text-gray-500 mb-2">{{ asReferredNote }}</p>
              <div v-if="paymentDetailRows.length === 0" class="text-sm text-gray-400 py-4 text-center">
                暂无推荐用户支付记录
              </div>
              <table v-else class="min-w-full text-sm border border-gray-200 rounded-lg overflow-hidden">
                <thead class="bg-gray-50">
                  <tr>
                    <th class="text-left px-3 py-2 border-b font-medium text-gray-500">用户 ID</th>
                    <th class="text-right px-3 py-2 border-b font-medium text-gray-500">支付笔数</th>
                    <th class="text-right px-3 py-2 border-b font-medium text-gray-500">支付总额（元）</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="ru in paymentDetailRows"
                    :key="ru.user_id"
                    class="border-b hover:bg-gray-50"
                  >
                    <td class="px-3 py-2 font-mono text-xs text-gray-700">{{ ru.user_id }}</td>
                    <td class="px-3 py-2 text-right text-gray-600">{{ ru.recharge_count }}</td>
                    <td class="px-3 py-2 text-right font-medium text-gray-900">{{ fmtPoints(ru.recharge_total) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <ReferralWechatProfitSharingTab
              v-if="detailTab === 'wechat'"
              :user-id="userId"
            />
          </div>

          <!-- 佣金明细（pending/settled + 资源类型） -->
          <div v-if="data.commission && data.commission.items.length">
            <h4 class="text-sm font-semibold text-gray-800 mb-2">
              佣金明细（{{ data.commission.items.length }} 笔，入账周期 {{ data.commission.settle_delay_days }} 天）
            </h4>
            <div class="overflow-x-auto">
              <table class="min-w-full text-sm border border-gray-200 rounded-lg overflow-hidden">
                <thead class="bg-gray-50">
                  <tr>
                    <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">来源用户</th>
                    <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">资源类型</th>
                    <th class="text-right px-2 py-1.5 border-b font-medium text-gray-500 text-xs">消费金额（元）</th>
                    <th class="text-right px-2 py-1.5 border-b font-medium text-gray-500 text-xs">佣金</th>
                    <th class="text-center px-2 py-1.5 border-b font-medium text-gray-500 text-xs">状态</th>
                    <th class="text-left px-2 py-1.5 border-b font-medium text-gray-500 text-xs">消费时间</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="item in data.commission.items"
                    :key="item.id"
                    class="border-b"
                    :class="item.status === 'settled' ? 'bg-emerald-50' : 'bg-amber-50'"
                  >
                    <td class="px-2 py-1.5 font-mono text-xs text-gray-700">{{ truncateId(item.referred_user_id) }}</td>
                    <td class="px-2 py-1.5 text-xs">
                      <span class="px-1.5 py-0.5 rounded text-xs font-medium" :class="resourceClass(item.resource_type)">
                        {{ resourceLabel(item.resource_type) }}
                      </span>
                    </td>
                    <td class="px-2 py-1.5 text-right text-xs text-gray-600">{{ fmtPoints(item.consumption_points) }}</td>
                    <td class="px-2 py-1.5 text-right text-xs font-medium text-gray-900">{{ fmtPoints(item.commission_points) }}</td>
                    <td class="px-2 py-1.5 text-center text-xs">
                      <span
                        class="px-1.5 py-0.5 rounded-full text-xs font-medium"
                        :class="item.status === 'settled' ? 'bg-green-100 text-green-700' : 'bg-amber-100 text-amber-700'"
                      >
                        {{ item.status === 'settled' ? '已入账' : '待入账' }}
                      </span>
                    </td>
                    <td class="px-2 py-1.5 text-xs text-gray-500">{{ formatDate(item.consumed_at) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { messageFromFailedResponse } from '../utils/httpError.js'
import { adminFormatDate as formatDate } from '../utils/adminFormatDate.js'
import ReferralWechatProfitSharingTab from './ReferralWechatProfitSharingTab.vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  userId: { type: [String, Number], default: '' },
})

defineEmits(['close'])

const loading = ref(false)
const error = ref('')
const errorTraceId = ref('')
const data = ref(null)
const startDate = ref('')
const endDate = ref('')
const detailTab = ref('payments')

const ownPayments = computed(() => {
  const rows = data.value?.own_payments
  return Array.isArray(rows) ? rows : []
})

const paymentDetailRows = computed(() => {
  const rows = data.value?.referred_users
  if (Array.isArray(rows) && rows.length > 0) return rows
  const incoming = data.value?.as_referred
  if (!incoming) return []
  return [{
    user_id: String(props.userId || ''),
    recharge_count: incoming.recharge_count,
    recharge_total: incoming.recharge_total,
  }]
})

const asReferredNote = computed(() => {
  const incoming = data.value?.as_referred
  if (!incoming || (data.value?.referred_users || []).length > 0) return ''
  return `该用户由 ${incoming.referrer_user_id} 推荐`
})

const commissionRateDisplay = computed(() => {
  const raw = String(data.value?.commission?.commission_rate_display || '').trim()
  return raw || '—'
})

function fmtPoints(pts) {
  if (pts == null) return '—'
  return Number(pts).toLocaleString()
}

function truncateId(id) {
  if (!id) return '—'
  const s = String(id)
  return s.length > 12 ? s.slice(0, 12) + '...' : s
}

const RESOURCE_LABELS = {
  task_post: '任务帖',
  gitlab_disk: 'GitLab 磁盘',
  gitlab_traffic: 'GitLab 流量',
  server_instance: '云服务器',
  user_recharge_admin: '管理员直充',
  user_recharge_wechat: '微信支付',
  user_recharge_paypal: 'PayPal 支付',
  referral_commission: '推荐佣金',
}

function resourceLabel(type) {
  return RESOURCE_LABELS[type] || type || '—'
}

function resourceClass(type) {
  switch (type) {
    case 'task_post': return 'bg-blue-100 text-blue-700'
    case 'gitlab_disk': return 'bg-purple-100 text-purple-700'
    case 'gitlab_traffic': return 'bg-cyan-100 text-cyan-700'
    case 'server_instance': return 'bg-orange-100 text-orange-700'
    default: return 'bg-gray-100 text-gray-600'
  }
}

async function loadData() {
  const uid = String(props.userId || '').trim()
  if (!uid) return
  loading.value = true
  error.value = ''
  errorTraceId.value = ''
  data.value = null
  try {
    const params = new URLSearchParams()
    if (startDate.value) params.set('start_date', startDate.value)
    if (endDate.value) params.set('end_date', endDate.value)

    const url = `/api/system-admin/users/${encodeURIComponent(uid)}/referral-performance/${
      params.toString() ? '?' + params.toString() : ''
    }`
    const resp = await apiFetch(url, {
      method: 'GET',
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (!resp.ok) {
      error.value = messageFromFailedResponse(resp, '加载失败')
      errorTraceId.value = extractTraceId(resp) || extractTraceId(resp._errorData) || ''
      return
    }
    const json = await resp.json().catch(() => ({}))
    json.referred_users = json.referred_users || []
    data.value = json
  } catch (e) {
    error.value = '网络错误，请稍后重试'
    errorTraceId.value = extractTraceId(e) || ''
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.visible, props.userId],
  ([v]) => {
    if (v) {
      startDate.value = ''
      endDate.value = ''
      detailTab.value = 'payments'
      loadData()
    }
  }
)
</script>
