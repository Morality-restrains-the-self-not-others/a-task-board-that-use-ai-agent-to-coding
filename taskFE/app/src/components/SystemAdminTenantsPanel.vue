<template>
  <div
    data-testid="system-admin-tenants-panel"
    class="bg-white rounded-lg shadow-sm border border-gray-200 p-6"
  >
    <div class="flex items-center justify-between mb-4 gap-3 flex-wrap">
      <h2 class="text-lg font-semibold text-gray-900">
        租户列表
        <span v-if="total !== null" class="text-sm font-normal text-gray-500 ml-2">共 {{ total }} 个租户</span>
      </h2>
      <div class="flex items-center gap-2">
        <input
          v-model="searchQuery"
          type="text"
          class="px-3 py-2 border border-gray-300 rounded-lg text-sm w-56"
          placeholder="搜索名称、ID、邮箱、手机号"
          data-testid="system-admin-tenants-search"
          @keyup.enter="handleSearch"
        />
        <button
          class="px-3 py-2 text-sm bg-primary text-white rounded-lg hover:bg-primary/90"
          @click="handleSearch"
        >
          <!-- Anti-Replay-OK: read-search GET -->
          搜索
        </button>
        <button
          class="text-sm text-primary hover:underline"
          :disabled="loading"
          @click="loadTenants"
        >
          <!-- Anti-Replay-OK: read-refresh -->
          {{ loading ? '刷新中...' : '刷新' }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="text-sm text-gray-500 py-8 text-center">加载中...</div>
    <div
      v-else-if="loadError"
      class="p-3 bg-red-50 text-red-700 rounded-lg text-sm"
      :data-traceId="loadErrorTraceId || undefined"
      data-testid="system-admin-tenants-error"
    >
      {{ loadError }}
    </div>
    <div
      v-else-if="!tenants.length"
      class="text-sm text-gray-500 py-8 text-center"
      data-testid="system-admin-tenants-empty"
    >
      暂无租户
    </div>
    <div v-else class="overflow-x-auto">
      <table class="min-w-full text-sm" data-testid="system-admin-tenants-table">
        <thead>
          <tr class="text-left text-gray-500 border-b">
            <th class="py-2 pr-3 font-medium">ID</th>
            <th class="py-2 pr-3 font-medium">名称</th>
            <th class="py-2 pr-3 font-medium">创建者 ID</th>
            <th class="py-2 pr-3 font-medium">邮箱</th>
            <th class="py-2 pr-3 font-medium">手机号</th>
            <th class="py-2 pr-3 font-medium">创建时间</th>
            <th class="py-2 pr-3 font-medium">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in tenants"
            :key="row.id"
            class="border-b hover:bg-gray-50"
            data-testid="system-admin-tenant-row"
          >
            <td class="py-2 pr-3 font-mono text-xs">{{ row.id }}</td>
            <td class="py-2 pr-3">
              <a
                v-if="row.name"
                :href="`/system-admin/tenants/${encodeURIComponent(row.id)}/`"
                class="text-primary hover:underline"
                data-testid="system-admin-tenant-name-link"
              >
                <!-- Anti-Replay-OK: real href navigation to tenant detail -->
                {{ row.name }}
              </a>
              <span v-else>—</span>
            </td>
            <td class="py-2 pr-3 font-mono text-xs">{{ row.creator_id || '—' }}</td>
            <td class="py-2 pr-3">{{ row.email || '—' }}</td>
            <td class="py-2 pr-3">{{ row.phone || '—' }}</td>
            <td class="py-2 pr-3 text-xs">{{ formatDate(row.created_at) }}</td>
            <td class="py-2 pr-3">
              <a
                :href="`/system-admin/grant-points/?tenant_id=${encodeURIComponent(row.id)}`"
                class="text-primary hover:underline whitespace-nowrap"
                data-testid="system-admin-tenant-grant-link"
              >
                <!-- Anti-Replay-OK: real href to grant points with tenant preselected -->
                赠送资源
              </a>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="total > limit" class="mt-4 flex items-center justify-between text-sm">
      <span class="text-gray-500">第 {{ pageNum }} 页</span>
      <div class="flex gap-2">
        <button
          class="px-3 py-1 border rounded hover:bg-gray-50 disabled:opacity-50"
          :disabled="offset === 0"
          @click="prevPage"
        >
          <!-- Anti-Replay-OK: read-pagination GET -->
          上一页
        </button>
        <button
          class="px-3 py-1 border rounded hover:bg-gray-50 disabled:opacity-50"
          :disabled="offset + limit >= total"
          @click="nextPage"
        >
          <!-- Anti-Replay-OK: read-pagination GET -->
          下一页
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'

const loading = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
const tenants = ref([])
const total = ref(null)
const searchQuery = ref('')
const appliedSearch = ref('')
const limit = 50
const offset = ref(0)

const pageNum = computed(() => Math.floor(offset.value / limit) + 1)

const formatDate = (value) => {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
  return d.toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit',
  })
}

const loadTenants = async () => {
  loading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const q = new URLSearchParams()
    q.set('limit', String(limit))
    q.set('offset', String(offset.value))
    if (appliedSearch.value) q.set('search', appliedSearch.value)
    const response = await apiFetch(
      `/api/system-admin/accounts/admin/tenants/?${q.toString()}`,
      {
        method: 'GET',
        credentials: 'include',
        headers: { Accept: 'application/json' },
      },
    )
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      loadError.value = data.detail || data.message || `加载租户列表失败（${response.status}）`
      loadErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
      tenants.value = []
      total.value = 0
      return
    }
    tenants.value = Array.isArray(data.items) ? data.items : []
    total.value = typeof data.total !== 'undefined' ? Number(data.total) : tenants.value.length
  } catch (e) {
    loadError.value = e?.message || '加载租户列表失败'
    loadErrorTraceId.value = extractTraceId(e) || ''
    tenants.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  appliedSearch.value = String(searchQuery.value || '').trim()
  offset.value = 0
  loadTenants()
}

const prevPage = () => {
  if (offset.value === 0) return
  offset.value = Math.max(0, offset.value - limit)
  loadTenants()
}

const nextPage = () => {
  if (offset.value + limit >= total.value) return
  offset.value += limit
  loadTenants()
}

onMounted(() => {
  loadTenants()
})
</script>
