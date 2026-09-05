<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold mb-6">账单</h1>

    <!-- 资源配额卡片 -->
    <div class="bg-white rounded-xl shadow-md p-6 mb-6">
      <h2 class="text-lg font-semibold mb-4">资源配额</h2>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
        <div class="bg-blue-50 rounded-lg p-4" data-testid="task-post-quota-card">
          <p class="text-sm text-gray-500 mb-1">任务帖配额</p>
          <p class="text-2xl font-bold text-blue-700">{{ quotas.task_post_quota ?? 0 }} <span class="text-sm font-normal">帖</span></p>
          <p
            v-if="hasTaskPostQuotaSplit"
            class="text-xs text-gray-600 mt-1"
            data-testid="task-post-quota-split"
          >
            赠送 {{ quotas.task_post_quota_gifted }} 帖 · 购买 {{ quotas.task_post_quota_purchased }} 帖
          </p>
        </div>
        <template v-if="!gitlabRegionRows.length">
          <div class="bg-green-50 rounded-lg p-4" data-testid="gitlab-aggregate-disk-card">
            <p class="text-sm text-gray-500 mb-1">GitLab 磁盘</p>
            <p class="text-2xl font-bold text-green-700">
              {{ formatUsedGb(quotas.gitlab_disk_used_gb) }} / {{ quotas.gitlab_disk_gb ?? 0 }}
              <span class="text-sm font-normal">GB</span>
            </p>
            <div
              class="mt-2 h-1.5 rounded-full bg-green-100 overflow-hidden"
              role="progressbar"
              :aria-valuenow="usagePercent(quotas.gitlab_disk_used_gb, quotas.gitlab_disk_gb)"
              aria-valuemin="0"
              aria-valuemax="100"
              aria-label="GitLab 磁盘已用"
            >
              <div
                class="h-full bg-green-600 rounded-full"
                :style="{ width: usagePercent(quotas.gitlab_disk_used_gb, quotas.gitlab_disk_gb) + '%' }"
              />
            </div>
            <p v-if="quotas.gitlab_disk_expires_at" class="text-xs text-gray-500 mt-1">到期：{{ quotas.gitlab_disk_expires_at?.slice(0, 10) }}</p>
          </div>
          <div class="bg-amber-50 rounded-lg p-4" data-testid="gitlab-aggregate-traffic-card">
            <p class="text-sm text-gray-500 mb-1">GitLab 流量</p>
            <p class="text-2xl font-bold text-amber-700">
              {{ formatUsedGb(quotas.gitlab_traffic_used_gb) }} / {{ quotas.gitlab_traffic_prepaid_gb ?? 0 }}
              <span class="text-sm font-normal">GB</span>
            </p>
            <div
              class="mt-2 h-1.5 rounded-full bg-amber-100 overflow-hidden"
              role="progressbar"
              :aria-valuenow="usagePercent(quotas.gitlab_traffic_used_gb, quotas.gitlab_traffic_prepaid_gb)"
              aria-valuemin="0"
              aria-valuemax="100"
              aria-label="GitLab 流量已用"
            >
              <div
                class="h-full bg-amber-600 rounded-full"
                :style="{ width: usagePercent(quotas.gitlab_traffic_used_gb, quotas.gitlab_traffic_prepaid_gb) + '%' }"
              />
            </div>
          </div>
        </template>
      </div>

      <GitlabRegionQuotaCards v-if="gitlabRegionRows.length" :rows="gitlabRegionRows" />

      <p class="text-sm text-gray-500 mt-3 max-w-xl">
        说明：创建任务帖消耗 1 次配额并获 12 个月存续期，续存再消耗 1 次配额、不另扣费；GitLab 磁盘和流量从对应配额扣减。配额不足时请购买更多资源。
      </p>
      <div class="mt-4 flex flex-wrap gap-3">
        <a
          :href="`${tenantPath}/billing/orders/create/`"
          class="inline-flex items-center px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
        >
          <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
          </svg>
          购买资源
        </a>
      </div>
    </div>

    <!-- 概览卡片：消耗=用量/订单扣费；支付=实付入账，二者口径不同 -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
      <div class="bg-white rounded-xl shadow-md p-6">
        <h3 class="text-sm font-medium text-gray-500 mb-2">本月消耗（元）</h3>
        <div class="text-2xl font-bold">{{ monthlyYuan }}</div>
      </div>

      <div
        data-testid="stat-total-consumption"
        class="bg-white rounded-xl shadow-md p-6"
      >
        <h3 class="text-sm font-medium text-gray-500 mb-2">累计消耗（元）</h3>
        <div class="text-2xl font-bold">{{ totalYuan }}</div>
        <p class="text-xs text-gray-500 mt-1">用量扣费与仍有效的资源订单合计（不含已退款、已取消订单）</p>
      </div>

      <div
        data-testid="stat-total-payment"
        class="bg-white rounded-xl shadow-md p-6"
      >
        <h3 class="text-sm font-medium text-gray-500 mb-2">累计支付（元）</h3>
        <div class="text-2xl font-bold">{{ totalRechargeYuan }}</div>
        <p class="text-xs text-gray-500 mt-1">含 PayPal、微信、管理员直充、资源订单实付及历史支付入账（不含后台赠送）</p>
      </div>
    </div>

    <!-- 最近交易 -->
    <div class="bg-white rounded-xl shadow-md p-6">
      <div class="flex justify-between items-center mb-4">
        <h2 class="text-lg font-semibold">最近交易</h2>
        <a
          :href="`${tenantPath}/billing/transactions/`"
          class="text-primary hover:underline text-sm"
        >
          查看全部
        </a>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full">
          <thead>
            <tr class="border-b">
              <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">交易时间</th>
              <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">类型</th>
              <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">来源</th>
              <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">变动明细</th>
              <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">瞬时账目</th>
              <th class="text-left py-3 px-4 text-sm font-medium text-gray-500">描述</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="transaction in recentRowsWithLines" :key="transaction.id" class="border-b hover:bg-gray-50">
              <td class="py-3 px-4 text-sm">{{ formatDate(transaction.created_at) }}</td>
              <td class="py-3 px-4 text-sm">
                <span
                  :class="{
                    'text-green-600': transaction.transaction_type === 'recharge',
                    'text-red-600': transaction.transaction_type === 'consumption',
                    'text-blue-600': transaction.transaction_type === 'refund'
                  }"
                >
                  {{ getTransactionTypeText(transaction.transaction_type) }}
                </span>
              </td>
              <td class="py-3 px-4 text-sm text-gray-700">{{ pointsSourceTypeDisplayOrDash(transaction) }}</td>
              <td class="py-3 px-4 text-sm font-medium">
                <span
                  :class="{
                    'text-green-600': transaction.transaction_type === 'recharge',
                    'text-red-600': transaction.transaction_type === 'consumption',
                    'text-blue-600': transaction.transaction_type === 'refund'
                  }"
                >
                  {{ transactionChangeDisplay(transaction) }}
                </span>
              </td>
              <td class="py-3 px-4 text-sm text-gray-800">
                <div
                  v-for="(line, idx) in transaction._lines"
                  :key="idx"
                >
                  {{ line }}
                </div>
                <span v-if="transaction._lines.length === 0">—</span>
              </td>
              <td class="py-3 px-4 text-sm">{{ transaction.description || '-' }}</td>
            </tr>
            <tr v-if="recentRowsWithLines.length === 0">
              <td colspan="6" class="py-8 text-center text-gray-500">暂无交易记录</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useBillingDashboard } from '../composables/useBillingDashboard.js'
import { apiFetch } from '../utils/apiUtils'
import { safeJson } from '@/utils/safeResponseJson.js'
import { formatYuanFromCents } from '../utils/formatYuanCents.js'
import { formatUsedGb, usagePercent } from '../utils/formatUsedGb.js'
import { ledgerSnapshotLines, pointsSourceTypeDisplayOrDash, transactionChangeDisplay } from '../utils/transactionChangeDisplay.js'
import GitlabRegionQuotaCards from '../components/GitlabRegionQuotaCards.vue'

const {
  tenantPath,
  monthlyConsumption,
  totalConsumption,
  totalRecharge,
  recentTransactions,
  formatDate,
  getTransactionTypeText,
} = useBillingDashboard()

const route = useRoute()
const tenantId = computed(() => route.params.tenant)
const monthlyYuan = computed(() => formatYuanFromCents(monthlyConsumption.value, { empty: '0.00' }))
const totalYuan = computed(() => formatYuanFromCents(totalConsumption.value, { empty: '0.00' }))
const totalRechargeYuan = computed(() => formatYuanFromCents(totalRecharge.value, { empty: '0.00' }))
// OPT-20260819-016: 行级 ledgerSnapshotLines 只计算一次（模板 v-for 与空态 v-if 双调用）
const recentRowsWithLines = computed(() =>
  (recentTransactions.value || []).map((txn) => ({ ...txn, _lines: ledgerSnapshotLines(txn) })),
)

const quotas = ref({})
const hasTaskPostQuotaSplit = computed(() => {
  const q = quotas.value || {}
  return q.task_post_quota_gifted != null && q.task_post_quota_purchased != null
})
// OPT-20260818-022：多区域租户按区展示磁盘/流量（后端 gitlab_resources[]）。
const gitlabRegionRows = computed(() => {
  const list = Array.isArray(quotas.value.gitlab_resources) ? quotas.value.gitlab_resources : []
  return list.map((r) => ({
    regionSlug: String(r.region || ''),
    region: String(r.region_name || r.region || '—'),
    diskGb: r.disk_gb ?? 0,
    diskUsedGb: r.disk_used_gb ?? 0,
    diskExpiresAt: r.disk_expires_at ? String(r.disk_expires_at).slice(0, 10) : '',
    trafficPrepaidGb: r.traffic_prepaid_gb ?? 0,
    trafficUsedGb: r.traffic_used_gb ?? 0,
    // OPT-20260820-041: 赠送/购买拆分（后端 quotas 接口按区下发；旧版本无字段时为 null）
    diskGiftedGb: r.disk_gifted_gb ?? null,
    diskPurchasedGb: r.disk_purchased_gb ?? null,
    trafficGiftedGb: r.traffic_gifted_gb ?? null,
    trafficPurchasedGb: r.traffic_purchased_gb ?? null,
    webUrl: String(r.gitlab_web_url || ''),
  }))
})

const fetchQuotas = async () => {
  try {
    const r = await apiFetch(`/api/tenant/${tenantId.value}/billing/quotas/`, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (r.ok) {
      quotas.value = await safeJson(r, {})
    }
  } catch (e) {
    console.error('获取资源配额失败', e)
  }
}

onMounted(() => {
  fetchQuotas()
})
</script>

<style scoped>
</style>
