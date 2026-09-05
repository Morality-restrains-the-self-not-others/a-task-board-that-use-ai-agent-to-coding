<template>
  <div data-alias="view-system-admin-tenant-detail" class="p-6 max-w-6xl" data-testid="system-admin-tenant-detail">
    <p class="text-sm mb-4">
      <a
        href="/system-admin/users/?tab=tenants"
        class="text-primary hover:underline"
        data-testid="system-admin-tenant-detail-back"
      >
        <!-- Anti-Replay-OK: real href back to tenant list -->
        ← 返回租户列表
      </a>
    </p>

    <div v-if="headerLoading" class="text-sm text-gray-500 mb-6">加载租户信息...</div>
    <div
      v-else-if="headerError"
      class="p-3 bg-red-50 text-red-700 rounded-lg text-sm mb-6"
      :data-traceId="headerErrorTraceId || undefined"
      data-testid="system-admin-tenant-header-error"
    >
      {{ headerError }}
    </div>
    <div v-else class="bg-white rounded-lg border border-gray-200 p-6 mb-6" data-testid="system-admin-tenant-header">
      <h1 class="text-2xl font-semibold text-gray-900 mb-1">{{ tenant.name || '—' }}</h1>
      <p class="text-sm text-gray-500 font-mono mb-3">{{ tenant.id }}</p>
      <dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-2 text-sm">
        <div><dt class="text-gray-500">创建者 ID</dt><dd class="font-mono">{{ tenant.creator_id || '—' }}</dd></div>
        <div><dt class="text-gray-500">邮箱</dt><dd>{{ tenant.email || '—' }}</dd></div>
        <div><dt class="text-gray-500">手机号</dt><dd>{{ tenant.phone || '—' }}</dd></div>
        <div><dt class="text-gray-500">创建时间</dt><dd>{{ formatDate(tenant.created_at) }}</dd></div>
      </dl>
    </div>

    <section class="bg-white rounded-xl shadow-md p-6 mb-6" data-testid="system-admin-tenant-quotas">
      <h2 class="text-lg font-semibold mb-4">剩余资源</h2>
      <div v-if="quotasLoading" class="text-sm text-gray-500">加载配额...</div>
      <div
        v-else-if="quotasError"
        class="p-3 bg-red-50 text-red-700 rounded-lg text-sm"
        :data-traceId="quotasErrorTraceId || undefined"
        data-testid="system-admin-tenant-quotas-error"
      >
        {{ quotasError }}
      </div>
      <template v-else>
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
            </div>
            <div class="bg-amber-50 rounded-lg p-4" data-testid="gitlab-aggregate-traffic-card">
              <p class="text-sm text-gray-500 mb-1">GitLab 流量</p>
              <p class="text-2xl font-bold text-amber-700">
                {{ formatUsedGb(quotas.gitlab_traffic_used_gb) }} / {{ quotas.gitlab_traffic_prepaid_gb ?? 0 }}
                <span class="text-sm font-normal">GB</span>
              </p>
            </div>
          </template>
        </div>
        <GitlabRegionQuotaCards v-if="gitlabRegionRows.length" :rows="gitlabRegionRows" />
      </template>
    </section>

    <section class="bg-white rounded-xl shadow-md p-6 mb-6" data-testid="system-admin-tenant-workspaces">
      <h2 class="text-lg font-semibold mb-4">
        工作空间
        <span v-if="workspacesTotal !== null" class="text-sm font-normal text-gray-500 ml-2">共 {{ workspacesTotal }} 个</span>
      </h2>
      <div v-if="workspacesLoading" class="text-sm text-gray-500">加载工作空间...</div>
      <div
        v-else-if="workspacesError"
        class="p-3 bg-red-50 text-red-700 rounded-lg text-sm"
        :data-traceId="workspacesErrorTraceId || undefined"
        data-testid="system-admin-tenant-workspaces-error"
      >
        {{ workspacesError }}
      </div>
      <div
        v-else-if="!workspaces.length"
        class="text-sm text-gray-500 py-4"
        data-testid="system-admin-tenant-workspaces-empty"
      >
        暂无工作空间
      </div>
      <div v-else class="overflow-x-auto">
        <table class="min-w-full text-sm" data-testid="system-admin-tenant-workspaces-table">
          <thead>
            <tr class="text-left text-gray-500 border-b">
              <th class="py-2 pr-3 font-medium">名称</th>
              <th class="py-2 pr-3 font-medium">ID</th>
              <th class="py-2 pr-3 font-medium">默认</th>
              <th class="py-2 pr-3 font-medium">创建时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="ws in workspaces" :key="ws.id" class="border-b">
              <td class="py-2 pr-3">{{ ws.name || '—' }}</td>
              <td class="py-2 pr-3 font-mono text-xs">{{ ws.id }}</td>
              <td class="py-2 pr-3">{{ ws.is_default ? '是' : '—' }}</td>
              <td class="py-2 pr-3 text-xs">{{ formatDate(ws.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div
        v-if="!workspacesLoading && workspacesTotal > workspacesLimit"
        class="mt-4 flex items-center justify-between text-sm"
        data-testid="system-admin-tenant-workspaces-pager"
      >
        <span class="text-gray-500" data-testid="system-admin-tenant-workspaces-page">
          第 {{ Math.floor(workspacesOffset / workspacesLimit) + 1 }} 页
        </span>
        <div class="flex gap-2">
          <button
            class="px-3 py-1 border rounded hover:bg-gray-50 disabled:opacity-50"
            :disabled="workspacesOffset === 0"
            data-testid="system-admin-tenant-workspaces-prev"
            @click="workspacesPrev"
          >
            <!-- Anti-Replay-OK: read-pagination GET -->
            上一页
          </button>
          <button
            class="px-3 py-1 border rounded hover:bg-gray-50 disabled:opacity-50"
            :disabled="workspacesOffset + workspacesLimit >= workspacesTotal"
            data-testid="system-admin-tenant-workspaces-next"
            @click="workspacesNext"
          >
            <!-- Anti-Replay-OK: read-pagination GET -->
            下一页
          </button>
        </div>
      </div>
    </section>

    <section class="bg-white rounded-xl shadow-md p-6" data-testid="system-admin-tenant-orders">
      <div class="flex items-center justify-between mb-4 gap-3 flex-wrap">
        <h2 class="text-lg font-semibold">
          订单情况
          <span v-if="ordersTotal !== null" class="text-sm font-normal text-gray-500 ml-2">共 {{ ordersTotal }} 笔</span>
        </h2>
        <a
          :href="`/system-admin/order-records/?tenant_id=${encodeURIComponent(tenantId)}`"
          class="text-sm text-primary hover:underline"
          data-testid="system-admin-tenant-orders-all"
        >
          <!-- Anti-Replay-OK: real href to admin order list filtered by tenant -->
          查看全部
        </a>
      </div>
      <div v-if="ordersLoading" class="text-sm text-gray-500">加载订单...</div>
      <div
        v-else-if="ordersError"
        class="p-3 bg-red-50 text-red-700 rounded-lg text-sm"
        :data-traceId="ordersErrorTraceId || undefined"
        data-testid="system-admin-tenant-orders-error"
      >
        {{ ordersError }}
      </div>
      <div
        v-else-if="!orders.length"
        class="text-sm text-gray-500 py-4"
        data-testid="system-admin-tenant-orders-empty"
      >
        暂无订单
      </div>
      <div v-else class="overflow-x-auto">
        <table class="min-w-full text-sm" data-testid="system-admin-tenant-orders-table">
          <thead>
            <tr class="text-left text-gray-500 border-b">
              <th class="py-2 pr-3 font-medium">订单号</th>
              <th class="py-2 pr-3 font-medium">状态</th>
              <th class="py-2 pr-3 font-medium">金额（元）</th>
              <th class="py-2 pr-3 font-medium">支付方式</th>
              <th class="py-2 pr-3 font-medium">创建时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in orders" :key="o.id" class="border-b">
              <td class="py-2 pr-3 font-mono text-xs">{{ o.order_number }}</td>
              <td class="py-2 pr-3">{{ statusLabel(o.status) }}</td>
              <td class="py-2 pr-3 font-mono">{{ o.total_yuan }}</td>
              <td class="py-2 pr-3">{{ paymentLabel(o.payment_method) }}</td>
              <td class="py-2 pr-3 text-xs">{{ formatDate(o.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { formatUsedGb } from '../utils/formatUsedGb.js'
import {
  billingOrderStatusLabel as statusLabel,
  billingOrderPaymentLabel as paymentLabel,
} from '../utils/billingOrderDisplay.js'
import GitlabRegionQuotaCards from '../components/GitlabRegionQuotaCards.vue'

const route = useRoute()
const tenantId = computed(() => String(route.params.tenantId || ''))

const tenant = ref({})
const headerLoading = ref(false)
const headerError = ref('')
const headerErrorTraceId = ref('')

const quotas = ref({})
const quotasLoading = ref(false)
const quotasError = ref('')
const quotasErrorTraceId = ref('')

const workspaces = ref([])
const workspacesTotal = ref(null)
const workspacesLoading = ref(false)
const workspacesError = ref('')
const workspacesErrorTraceId = ref('')
const workspacesLimit = 50
const workspacesOffset = ref(0)

const orders = ref([])
const ordersTotal = ref(null)
const ordersLoading = ref(false)
const ordersError = ref('')
const ordersErrorTraceId = ref('')

const hasTaskPostQuotaSplit = computed(() => {
  const q = quotas.value || {}
  return q.task_post_quota_gifted != null && q.task_post_quota_purchased != null
})

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
    diskGiftedGb: r.disk_gifted_gb ?? null,
    diskPurchasedGb: r.disk_purchased_gb ?? null,
    trafficGiftedGb: r.traffic_gifted_gb ?? null,
    trafficPurchasedGb: r.traffic_purchased_gb ?? null,
    webUrl: String(r.gitlab_web_url || ''),
  }))
})

const formatDate = (value) => {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
  return d.toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit',
  })
}

const getJSON = async (url, errRef, traceRef) => {
  const response = await apiFetch(url, {
    method: 'GET',
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    errRef.value = data.detail || data.message || data.error || `加载失败（${response.status}）`
    traceRef.value = extractTraceId(response) || extractTraceId(data) || ''
    return null
  }
  return data
}

const loadHeader = async () => {
  headerLoading.value = true
  headerError.value = ''
  headerErrorTraceId.value = ''
  try {
    const data = await getJSON(
      `/api/system-admin/accounts/admin/tenants/${encodeURIComponent(tenantId.value)}/`,
      headerError,
      headerErrorTraceId,
    )
    tenant.value = data || {}
  } catch (e) {
    headerError.value = e?.message || '加载租户信息失败'
    headerErrorTraceId.value = extractTraceId(e) || ''
    tenant.value = {}
  } finally {
    headerLoading.value = false
  }
}

const loadQuotas = async () => {
  quotasLoading.value = true
  quotasError.value = ''
  quotasErrorTraceId.value = ''
  try {
    const data = await getJSON(
      `/api/system-admin/tenant-quotas/tenant_id/${encodeURIComponent(tenantId.value)}/`,
      quotasError,
      quotasErrorTraceId,
    )
    quotas.value = data || {}
  } catch (e) {
    quotasError.value = e?.message || '加载配额失败'
    quotasErrorTraceId.value = extractTraceId(e) || ''
    quotas.value = {}
  } finally {
    quotasLoading.value = false
  }
}

const loadWorkspaces = async () => {
  workspacesLoading.value = true
  workspacesError.value = ''
  workspacesErrorTraceId.value = ''
  try {
    const q = new URLSearchParams({ limit: String(workspacesLimit), offset: String(workspacesOffset.value) })
    const data = await getJSON(
      `/api/system-admin/tenant-workspaces/tenant_id/${encodeURIComponent(tenantId.value)}/?${q}`,
      workspacesError,
      workspacesErrorTraceId,
    )
    workspaces.value = Array.isArray(data?.items) ? data.items : []
    workspacesTotal.value = typeof data?.total !== 'undefined' ? Number(data.total) : workspaces.value.length
  } catch (e) {
    workspacesError.value = e?.message || '加载工作空间失败'
    workspacesErrorTraceId.value = extractTraceId(e) || ''
    workspaces.value = []
    workspacesTotal.value = 0
  } finally {
    workspacesLoading.value = false
  }
}

const workspacesPrev = () => {
  if (workspacesOffset.value === 0) return
  workspacesOffset.value = Math.max(0, workspacesOffset.value - workspacesLimit)
  loadWorkspaces()
}

const workspacesNext = () => {
  if (workspacesOffset.value + workspacesLimit >= workspacesTotal.value) return
  workspacesOffset.value += workspacesLimit
  loadWorkspaces()
}

const loadOrders = async () => {
  ordersLoading.value = true
  ordersError.value = ''
  ordersErrorTraceId.value = ''
  try {
    const q = new URLSearchParams({
      tenant_id: tenantId.value,
      limit: '20',
      offset: '0',
    })
    const data = await getJSON(
      `/api/system-admin/orders/?${q}`,
      ordersError,
      ordersErrorTraceId,
    )
    orders.value = Array.isArray(data?.orders) ? data.orders : []
    ordersTotal.value = typeof data?.total !== 'undefined' ? Number(data.total) : orders.value.length
  } catch (e) {
    ordersError.value = e?.message || '加载订单失败'
    ordersErrorTraceId.value = extractTraceId(e) || ''
    orders.value = []
    ordersTotal.value = 0
  } finally {
    ordersLoading.value = false
  }
}

onMounted(() => {
  Promise.all([loadHeader(), loadQuotas(), loadWorkspaces(), loadOrders()])
})
</script>
