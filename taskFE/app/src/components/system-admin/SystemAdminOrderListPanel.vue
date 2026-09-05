<template>
  <div data-alias="panel-system-admin-order-list" class="max-w-5xl">
    <h2 class="text-xl font-semibold text-gray-900 mb-1">订单记录</h2>
    <p class="text-sm text-gray-500 mb-6">
      选择租户查看其资源订单记录，或选择「全部租户」查看全局订单。需管理员账号（is_staff）权限。
    </p>

    <AdminOrderLookupBox
      :searching="loading && !!(tradeNoQuery || wechatAccountQuery)"
      :initial-tab="wechatAccountQuery ? 'wechat' : 'trade'"
      @search-trade="onTradeNoSearch"
      @search-wechat="onWechatAccountSearch"
    />

    <!-- 租户选择 -->
    <div class="bg-white border border-gray-200 rounded-lg p-6 mb-6">
      <label class="block text-sm font-medium text-gray-700 mb-1">租户（公司）</label>
      <div class="relative">
        <input
          v-model="tenantSearch"
          type="text"
          autocomplete="off"
          class="w-full border border-gray-300 rounded-md px-3 py-2"
          placeholder="输入公司名称筛选，或选择「全部租户」"
          @focus="openDropdown"
          @blur="scheduleCloseDropdown"
          @input="onSearchInput"
        >
        <ul
          v-if="dropdownOpen && filteredTenantOptions.length"
          class="absolute z-20 mt-1 max-h-56 w-full overflow-auto rounded-md border border-gray-200 bg-white shadow-lg"
        >
          <!-- 全部租户选项 -->
          <li
            class="cursor-pointer px-3 py-2 text-sm hover:bg-primary/10 border-b border-gray-100"
            @mousedown.prevent="selectAllTenants"
          >
            <span class="font-medium text-primary">全部租户</span>
            <span class="ml-2 text-gray-400 text-xs">查看所有公司的订单</span>
          </li>
          <li
            v-for="t in tenantOptions"
            :key="t.id"
            class="cursor-pointer px-3 py-2 text-sm hover:bg-primary/10"
            @mousedown.prevent="selectTenant(t)"
          >
            <div class="flex items-center gap-2">
              <span class="font-medium text-gray-900 truncate">{{ t.name }}</span>
              <span class="text-gray-500 text-xs whitespace-nowrap">ID {{ t.id }}</span>
            </div>
            <div
              v-if="tenantContactLine(t)"
              data-testid="tenant-option-contact"
              class="mt-0.5 text-xs text-gray-500"
            >
              {{ tenantContactLine(t) }}
            </div>
          </li>
        </ul>
      </div>
      <p v-if="isAllTenants" class="mt-2 text-sm text-gray-600">
        已选：<span class="font-medium text-primary">全部租户</span>
        <span class="text-gray-400 text-xs ml-2">显示所有公司的订单</span>
      </p>
      <p v-else-if="selectedTenant" data-testid="tenant-selected-summary" class="mt-2 text-sm text-gray-600">
        已选：<span class="font-medium text-gray-900">{{ selectedTenant.name }}</span>
        <span class="text-gray-500">（ID: {{ selectedTenant.id }}）</span>
        <span v-if="tenantContactLine(selectedTenant)" class="text-gray-500"> · {{ tenantContactLine(selectedTenant) }}</span>
      </p>
      <p v-else class="mt-1 text-xs text-gray-500">请先选择租户以查看订单</p>
    </div>

    <!-- 订单列表 -->
    <div v-if="isAllTenants || selectedTenant">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-lg font-semibold text-gray-900">
          {{ isAllTenants ? '全部订单记录' : '订单记录' }}
        </h3>
        <button
          class="text-sm text-primary hover:underline"
          :disabled="loading"
          @click="fetchOrders"
        >
          {{ loading ? '刷新中…' : '刷新' }}
        </button>
      </div>

      <!-- 状态筛选 -->
      <div class="flex gap-2 mb-4" data-testid="order-status-filters">
        <button
          v-for="s in statusFilters"
          :key="s.value"
          class="px-3 py-1.5 text-xs rounded-full border transition-colors"
          :class="statusFilter === s.value
            ? 'bg-primary text-white border-primary'
            : 'bg-white text-gray-600 border-gray-300 hover:border-gray-400'"
          @click="statusFilter = s.value; fetchOrders()"
        >
          {{ s.label }}
        </button>
      </div>

      <!-- 加载中 -->
      <div v-if="loading" class="text-center py-12 text-gray-500">加载中…</div>

      <!-- 空状态 -->
      <div
        v-else-if="orders.length === 0"
        class="text-center py-12 text-gray-400"
        :data-testid="wechatAccountQuery ? 'order-wechat-account-empty' : (tradeNoQuery ? 'order-trade-no-empty' : undefined)"
      >
        {{ wechatAccountQuery ? '未找到该微信关联账号' : (tradeNoQuery ? '未找到该交易单号' : (isAllTenants ? '暂无订单记录' : '该租户暂无订单记录')) }}
      </div>

      <!-- 订单表格 -->
      <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr>
              <th class="text-left px-4 py-3 font-medium text-gray-600" style="width: 32px"></th>
              <th class="text-left px-4 py-3 font-medium text-gray-600">订单号</th>
              <th v-if="isAllTenants" class="text-left px-4 py-3 font-medium text-gray-600">租户</th>
              <th class="text-left px-4 py-3 font-medium text-gray-600">状态</th>
              <th class="text-right px-4 py-3 font-medium text-gray-600">金额（元）</th>
              <th class="text-left px-4 py-3 font-medium text-gray-600">支付方式</th>
              <th class="text-left px-4 py-3 font-medium text-gray-600">创建时间</th>
              <th class="text-left px-4 py-3 font-medium text-gray-600">支付时间</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="o in orders" :key="o.id">
              <!-- 订单摘要行 -->
              <tr
                class="border-b border-gray-100 hover:bg-gray-50 transition-colors cursor-pointer"
                :data-order-id="String(o.id)"
                @click="toggleExpand(o)"
              >
                <td class="px-4 py-3 text-gray-400">
                  <svg
                    class="w-4 h-4 transition-transform duration-200"
                    :class="isExpanded(o.id) ? 'rotate-90' : 'rotate-0'"
                    fill="none" stroke="currentColor" viewBox="0 0 24 24"
                  >
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                  </svg>
                </td>
                <td class="px-4 py-3 font-mono text-gray-900 break-all">{{ o.order_number }}</td>
                <td v-if="isAllTenants" class="px-4 py-3 text-gray-700">{{ getTenantName(o.tenant_id) }}</td>
                <td class="px-4 py-3">
                  <span
                    class="inline-block px-2 py-0.5 text-xs rounded-full font-medium"
                    :class="statusClass(o.status)"
                    data-testid="order-status-badge"
                  >
                    {{ statusLabel(o.status) }}
                  </span>
                </td>
                <td class="px-4 py-3 text-right font-mono text-gray-900">{{ o.total_yuan }}</td>
                <td class="px-4 py-3 text-gray-600">{{ paymentLabel(o.payment_method) }}</td>
                <td class="px-4 py-3 text-gray-500 text-xs">{{ formatTime(o.created_at) }}</td>
                <td class="px-4 py-3 text-gray-500 text-xs">{{ o.paid_at ? formatTime(o.paid_at) : '—' }}</td>
              </tr>
              <!-- 展开详情行 -->
              <tr v-if="isExpanded(o.id)" class="bg-gray-50 border-b border-gray-200">
                <td colspan="99" class="px-6 py-4">
                  <OrderExpandDetail
                    :loading="!!expandLoading[o.id]"
                    :items="expandItems[o.id] || []"
                    :buyer-note="expandNotes[o.id] || ''"
                    :consumption="expandConsumption[o.id]"
                    :profit-sharing="expandProfitSharing[o.id] || []"
                    :out-trade-no="expandOutTradeNo[o.id] || ''"
                    :wechat-transaction-id="expandWechatTransactionId[o.id] || ''"
                  />
                </td>
              </tr>
            </template>
          </tbody>
        </table>

        <!-- 分页 -->
        <div v-if="totalPages > 1" class="flex items-center justify-between px-4 py-3 border-t border-gray-200 bg-gray-50">
          <span class="text-sm text-gray-500">共 {{ total }} 条，第 {{ currentPage }} / {{ totalPages }} 页</span>
          <div class="flex gap-2">
            <button
              class="px-3 py-1 text-sm border border-gray-300 rounded-md hover:bg-gray-100 disabled:opacity-40 disabled:cursor-not-allowed"
              :disabled="currentPage <= 1"
              @click="goPage(currentPage - 1)"
            >
              上一页
            </button>
            <button
              class="px-3 py-1 text-sm border border-gray-300 rounded-md hover:bg-gray-100 disabled:opacity-40 disabled:cursor-not-allowed"
              :disabled="currentPage >= totalPages"
              @click="goPage(currentPage + 1)"
            >
              下一页
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="errorMessage" class="mt-4 p-4 bg-red-50 text-red-800 rounded-lg text-sm" :data-traceId="errorMessageTraceId || undefined">
      {{ errorMessage }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiFetch } from '../../utils/apiUtils'
import { safeJson } from '@/utils/safeResponseJson.js'
import { extractTraceId } from '../../utils/traceId.js'
import { tenantContactLine } from '../../utils/tenantOptionContact.js'
import { parseOrderIdQuery } from '../../utils/billingOrderDeepLink.js'
import { useSystemAdminOrderListDeepLink } from '../../composables/useSystemAdminOrderListDeepLink.js'
import { useAdminOrderRowExpand } from '../../composables/useAdminOrderRowExpand.js'
import {
  SYSTEM_ADMIN_ORDER_STATUS_FILTERS as statusFilters,
  statusLabel,
  statusClass,
  paymentLabel,
  formatTime,
} from '../../utils/systemAdminOrderListDisplay.js'
import OrderExpandDetail from './OrderExpandDetail.vue'
import AdminOrderLookupBox from './AdminOrderLookupBox.vue'

const route = useRoute()
const router = useRouter()

const ALL_TENANTS_ID = '__all__'

const tenantSearch = ref('')
const tenantOptions = ref([])
const selectedTenant = ref(null)
const dropdownOpen = ref(false)
const fetchTimer = ref(null)
let blurCloseTimer = null

const isAllTenants = computed(() => selectedTenant.value?.id === ALL_TENANTS_ID)
const {
  expandLoading,
  expandItems,
  expandNotes,
  expandConsumption,
  expandProfitSharing,
  expandOutTradeNo,
  expandWechatTransactionId,
  isExpanded,
  clearExpand,
  toggleExpand,
  expandOrderForDeepLink,
} = useAdminOrderRowExpand({ selectedTenant, isAllTenants })
const filteredTenantOptions = computed(() => {
  if (!tenantSearch.value.trim()) return tenantOptions.value
  return tenantOptions.value
})

// 从已获取的租户选项中构建 id→name 映射，用于全部租户视图显示公司名
const tenantNameMap = computed(() => {
  const map = {}
  for (const t of tenantOptions.value) {
    if (t && t.id && t.name) map[String(t.id)] = t.name
  }
  return map
})

const getTenantName = (tid) => {
  if (!tid) return '—'
  const id = String(tid)
  return tenantNameMap.value[id] || id
}

const orders = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = 15
const listOffset = ref(0)
const loading = ref(false)
const errorMessage = ref('')
const errorMessageTraceId = ref('')
const statusFilter = ref('')
const tradeNoQuery = ref('')
const wechatAccountQuery = ref('')

const resetOrderListState = () => {
  dropdownOpen.value = false
  orders.value = []
  total.value = 0
  currentPage.value = 1
  statusFilter.value = ''
  tradeNoQuery.value = ''
  wechatAccountQuery.value = ''
  clearExpand()
}

const syncLookupToUrl = ({ orderNumber, wechatAccount }) => {
  const next = { ...route.query }
  delete next.order_number
  delete next.wechat_account
  if (wechatAccount) next.wechat_account = wechatAccount
  else if (orderNumber) next.order_number = orderNumber
  router.replace({ query: next }).catch(() => {})
}

const runGlobalLookup = async (kind, query) => {
  resetOrderListState()
  if (kind === 'wechat') wechatAccountQuery.value = query
  else tradeNoQuery.value = query
  selectedTenant.value = { id: ALL_TENANTS_ID, name: '全部租户' }
  tenantSearch.value = '全部租户'
  syncLookupToUrl({
    orderNumber: kind === 'trade' ? query : '',
    wechatAccount: kind === 'wechat' ? query : '',
  })
  await fetchOrders()
  if (orders.value.length === 1) {
    await expandOrderForDeepLink(orders.value[0])
  }
}

const onTradeNoSearch = async (q) => {
  const query = String(q || '').trim()
  await runGlobalLookup('trade', query)
}

const onWechatAccountSearch = async (q) => {
  const query = String(q || '').trim()
  await runGlobalLookup('wechat', query)
}

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// ---- 租户搜索 ----
const fetchTenantOptions = async (search) => {
  try {
    const q = new URLSearchParams()
    if (search && search.trim()) {
      q.set('search', search.trim())
    }
    const url = `/api/system-admin/accounts/admin/tenant-options/?${q.toString()}`
    const response = await apiFetch(url, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await safeJson(response, [])
    if (!response.ok) {
      tenantOptions.value = []
      return
    }
    tenantOptions.value = Array.isArray(data) ? data : []
  } catch {
    tenantOptions.value = []
  }
}

const openDropdown = () => {
  if (blurCloseTimer) { clearTimeout(blurCloseTimer); blurCloseTimer = null }
  dropdownOpen.value = true
}

const scheduleCloseDropdown = () => {
  if (blurCloseTimer) clearTimeout(blurCloseTimer)
  blurCloseTimer = setTimeout(() => { dropdownOpen.value = false; blurCloseTimer = null }, 200)
}

const onSearchInput = () => {
  openDropdown()
  if (fetchTimer.value) clearTimeout(fetchTimer.value)
  fetchTimer.value = setTimeout(() => fetchTenantOptions(tenantSearch.value), 280)
}

const selectAllTenants = () => {
  if (blurCloseTimer) { clearTimeout(blurCloseTimer); blurCloseTimer = null }
  selectedTenant.value = { id: ALL_TENANTS_ID, name: '全部租户' }
  tenantSearch.value = '全部租户'
  resetOrderListState()
  fetchOrders()
}

const selectTenant = (t) => {
  if (blurCloseTimer) { clearTimeout(blurCloseTimer); blurCloseTimer = null }
  selectedTenant.value = t
  tenantSearch.value = t.name
  resetOrderListState()
  fetchOrders()
}

// ---- 订单获取 ----
const fetchOrders = async () => {
  if (!selectedTenant.value) return

  loading.value = true
  errorMessage.value = ''
  errorMessageTraceId.value = ''

  try {
    const params = new URLSearchParams()
    params.set('limit', String(pageSize))
    params.set('offset', String((currentPage.value - 1) * pageSize))
    if (statusFilter.value) {
      params.set('status', statusFilter.value)
    }
    if (tradeNoQuery.value) {
      params.set('order_number', tradeNoQuery.value)
    }
    if (wechatAccountQuery.value) {
      params.set('wechat_account', wechatAccountQuery.value)
    }
    // OPT-20260819-033：全部租户模式也透传 order_id，
    // 让 doAdminListOrders 做 offset 对齐（focus_order_id）而非静默落在第一页。
    const focusId = parseOrderIdQuery(route.query)
    if (focusId && !tradeNoQuery.value && !wechatAccountQuery.value) {
      params.set('order_id', focusId)
    }

    let url
    if (tradeNoQuery.value || wechatAccountQuery.value || isAllTenants.value) {
      url = `/api/system-admin/orders/?${params.toString()}`
    } else {
      const tid = selectedTenant.value.id
      url = `/api/tenant/${encodeURIComponent(tid)}/billing/orders/?${params.toString()}`
    }

    const response = await apiFetch(url, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await safeJson(response, {})
    if (!response.ok) {
      errorMessage.value = data.error || data.detail || `请求失败（${response.status}）`
      errorMessageTraceId.value = extractTraceId(response) || ''
      orders.value = []
      total.value = 0
      return
    }
    orders.value = data.orders || []
    total.value = data.total || 0
    const returnedOffset = Number(data.offset)
    listOffset.value = Number.isFinite(returnedOffset)
      ? returnedOffset
      : (currentPage.value - 1) * pageSize
  } catch (e) {
    errorMessage.value = e.message || '网络错误'
    errorMessageTraceId.value = extractTraceId(e) || ''
    orders.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const goPage = (p) => {
  if (p < 1 || p > totalPages.value) return
  currentPage.value = p
  clearExpand()
  fetchOrders()
}

const { applyFromRoute } = useSystemAdminOrderListDeepLink({
  route,
  router,
  orders,
  loading,
  listOffset,
  pageSize,
  currentPage,
  statusFilter,
  tenantOptions,
  expandOrder: expandOrderForDeepLink,
  selectTenant,
  selectAllTenants,
  fetchTenantOptions,
  onFocusMiss: (orderId) => {
    errorMessage.value = `未找到订单 ${orderId}（可能已被删除或不属于当前租户）`
  },
  onTradeNoSearch,
  onWechatAccountSearch,
})

onMounted(() => {
  applyFromRoute()
})

onBeforeUnmount(() => {
  if (fetchTimer.value) clearTimeout(fetchTimer.value)
  if (blurCloseTimer) clearTimeout(blurCloseTimer)
})
</script>
