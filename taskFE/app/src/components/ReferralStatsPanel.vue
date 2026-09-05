<template>
  <div class="bg-white rounded-xl shadow-sm border border-border p-6 mt-6">
    <h2 class="text-lg font-semibold text-text mb-4">推荐获取收益</h2>

    <div v-if="!hasActiveQualification" class="bg-amber-50 border border-amber-200 rounded-lg p-4 mb-4">
      <p class="text-sm text-amber-700" data-testid="referral-stats-ineligible-notice">
        <strong>注意：</strong>仅有推荐资格的用户才可获得收益分成。无资格时仍会计推荐人数；是否分账以被推荐用户下单时是否具备资格为准，获资后可从已推荐用户的后续下单获得分成。
      </p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-3 mb-4">
      <label class="text-sm text-text-light">
        渠道
        <select v-model="channelCode" class="mt-1 w-full px-3 py-2 border border-border rounded-lg text-sm text-text" data-testid="referral-stats-channel">
          <option value="">全部渠道</option>
          <option v-for="ch in channels" :key="ch.code" :value="ch.code">{{ ch.name }}</option>
        </select>
      </label>
      <label class="text-sm text-text-light">
        开始日期
        <input v-model="fromDate" type="date" class="mt-1 w-full px-3 py-2 border border-border rounded-lg text-sm text-text" data-testid="referral-stats-from" />
      </label>
      <label class="text-sm text-text-light">
        结束日期
        <input v-model="toDate" type="date" class="mt-1 w-full px-3 py-2 border border-border rounded-lg text-sm text-text" data-testid="referral-stats-to" />
      </label>
    </div>
    <!-- Anti-Replay-OK: read-only GET filter -->
    <button
      type="button"
      class="mb-4 text-sm bg-gray-100 px-3 py-2 rounded-lg"
      data-testid="referral-stats-apply"
      @click="fetchReferralStats"
    >
      筛选
    </button>

    <p
      v-if="statsError"
      class="text-sm text-red-500 mb-4"
      :data-traceId="statsErrorTraceId || undefined"
    >
      {{ statsError }}
    </p>
    <p v-else-if="statsLoading" class="text-sm text-text-light mb-4">统计加载中...</p>

    <div v-if="referralStats" class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
      <div class="rounded-lg border border-border bg-gray-50 p-4">
        <p class="text-sm text-text-light">推荐人数</p>
        <p class="text-2xl font-semibold text-text mt-1">{{ referralStats.referral_count }}</p>
      </div>
      <div class="rounded-lg border border-border bg-gray-50 p-4">
        <p class="text-sm font-semibold text-text" data-testid="referral-rate-label">当前的分账比例</p>
        <p class="text-2xl font-semibold text-primary mt-1" data-testid="referral-rate-display">{{ referralStats.referral_rate_display || '—' }}</p>
        <ReferralRateChangeNotice class="mt-2" />
      </div>
    </div>

    <div v-if="referralStats" class="overflow-x-auto mb-6">
      <p class="text-sm font-medium text-text mb-2">分渠道人数与分账</p>
      <table class="min-w-full text-sm border border-border rounded-lg overflow-hidden" data-testid="referral-channel-stats">
        <thead class="bg-gray-50">
          <tr>
            <th class="text-left px-4 py-3 border-b border-border">渠道</th>
            <th class="text-left px-4 py-3 border-b border-border">推荐人数</th>
            <th class="text-left px-4 py-3 border-b border-border">消费</th>
            <th class="text-left px-4 py-3 border-b border-border">分账</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in referralStats.channels" :key="item.channel_code || item.name">
            <td class="px-4 py-3 border-b border-border">{{ item.name }}</td>
            <td class="px-4 py-3 border-b border-border">{{ item.referral_count }}</td>
            <td class="px-4 py-3 border-b border-border">¥{{ item.consumption }}</td>
            <td class="px-4 py-3 border-b border-border font-medium text-primary">¥{{ item.commission }}</td>
          </tr>
          <tr v-if="!referralStats.channels.length">
            <td class="px-4 py-4 text-text-light" colspan="4">暂无分渠道数据</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="referralStats" class="overflow-x-auto">
      <table class="min-w-full text-sm border border-border rounded-lg overflow-hidden">
        <thead class="bg-gray-50">
          <tr>
            <th class="text-left px-4 py-3 border-b border-border">月份</th>
            <th class="text-left px-4 py-3 border-b border-border">消费</th>
            <th class="text-left px-4 py-3 border-b border-border">收益</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in referralStats.monthlyEarnings" :key="item.month">
            <td class="px-4 py-3 border-b border-border">{{ item.month }}</td>
            <td class="px-4 py-3 border-b border-border">¥{{ item.consumption }}</td>
            <td class="px-4 py-3 border-b border-border font-medium text-primary">¥{{ item.commission }}</td>
          </tr>
          <tr v-if="referralStats.monthlyEarnings.length === 0">
            <td class="px-4 py-4 text-text-light" colspan="3">暂无推荐收益数据</td>
          </tr>
        </tbody>
      </table>
      <p class="text-xs text-text-light mt-3 leading-relaxed">
        收益规则：当前的分账比例为 <strong class="text-text">{{ referralStats.referral_rate_display || '—' }}</strong>，收益 = 推荐用户当月消费 × {{ referralStats.referral_rate_display || '—' }}（金额四舍五入到分）。
        <strong class="text-text" data-testid="referral-earnings-rule">仅推荐用户已消费的金额可产生收益；下单时推荐人无推荐资格则不分账。推荐时尚未获资格、之后申请通过的，仍可从已推荐用户的后续下单获得分成。</strong>
      </p>
      <ReferralRateChangeNotice class="mt-2" />
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import { extractTraceId } from '../utils/traceId.js'
import ReferralRateChangeNotice from './ReferralRateChangeNotice.vue'

const props = defineProps({
  userId: { type: String, default: '' },
  hasActiveQualification: { type: Boolean, default: false },
  channels: { type: Array, default: () => [] },
})

const statsLoading = ref(false)
const statsError = ref('')
const statsErrorTraceId = ref('')
const referralStats = ref(null)
const channelCode = ref('')
const fromDate = ref('')
const toDate = ref('')

const normalizeMonthlyEarnings = (rows) => {
  if (!Array.isArray(rows)) return []
  return rows.map((item) => ({
    month: item.month,
    consumption: item.consumption ?? item.first_level_consumption ?? '0.00',
    commission: item.commission ?? item.first_level_commission ?? item.total_commission ?? '0.00',
  }))
}

const normalizeChannels = (rows) => {
  if (!Array.isArray(rows)) return []
  return rows.map((item) => ({
    channel_code: item.channel_code || '',
    name: item.name || item.channel_code || '历史（未分渠道）',
    referral_count: Number(item.referral_count ?? 0),
    consumption: item.consumption ?? '0.00',
    commission: item.commission ?? '0.00',
  }))
}

const fetchReferralStats = async () => {
  statsLoading.value = true
  statsError.value = ''
  statsErrorTraceId.value = ''
  try {
    const userId = String(props.userId || '').trim()
    const query = new URLSearchParams()
    if (channelCode.value) query.set('channel_code', channelCode.value)
    if (fromDate.value) query.set('from', fromDate.value)
    if (toDate.value) query.set('to', toDate.value)
    const qs = query.toString()
    const response = await apiFetch(
      `/api/referral/stats/user_id/${userId}/${qs ? `?${qs}` : ''}`,
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    if (!response.ok) {
      statsErrorTraceId.value = extractTraceId(response)
      statsError.value = '获取推荐收益统计失败，请稍后重试'
      return
    }
    const data = await response.json()
    referralStats.value = {
      referral_count: Number(data.referral_count ?? data.first_level_count ?? '0'),
      referral_rate_display: String(data.referral_rate_display || '').trim(),
      monthlyEarnings: normalizeMonthlyEarnings(data.monthly_earnings),
      channels: normalizeChannels(data.channels),
    }
  } catch (error) {
    console.error('获取推荐收益统计失败:', error)
    statsErrorTraceId.value = extractTraceId(error)
    statsError.value = '获取推荐收益统计失败，请稍后重试'
  } finally {
    statsLoading.value = false
  }
}

watch(() => props.userId, (id) => {
  if (id) fetchReferralStats()
}, { immediate: true })

defineExpose({ fetchReferralStats })
</script>
