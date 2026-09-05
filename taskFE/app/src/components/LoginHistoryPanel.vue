<template>
  <div>
    <div
      v-if="loadError"
      class="bg-white rounded-xl border border-red-200 p-4 text-red-600"
      data-testid="login-history-error"
      :data-traceId="loadErrorTraceId || undefined"
    >
      {{ loadError }}
    </div>
    <div v-else-if="loading && items.length === 0" class="bg-white rounded-xl shadow-sm border border-border p-6 text-text-light">
      加载中...
    </div>
    <div v-else class="bg-white rounded-xl shadow-sm border border-border overflow-hidden">
      <div class="px-4 py-3 border-b border-gray-100 flex items-center gap-2">
        <button
          type="button"
          class="px-3 py-1.5 text-sm rounded-lg border transition-colors disabled:opacity-50"
          :class="!showFailures ? 'bg-primary/10 border-primary text-primary' : 'border-border text-text-light hover:bg-gray-50'"
          :disabled="loading"
          data-testid="login-history-filter-success"
          @click="setIncludeFailures(false)"
        >仅成功</button>
        <button
          type="button"
          class="px-3 py-1.5 text-sm rounded-lg border transition-colors disabled:opacity-50"
          :class="showFailures ? 'bg-primary/10 border-primary text-primary' : 'border-border text-text-light hover:bg-gray-50'"
          :disabled="loading"
          data-testid="login-history-filter-all"
          @click="setIncludeFailures(true)"
        >含失败尝试</button>
      </div>
      <div
        v-if="items.length === 0"
        class="bg-white p-6 text-text-light"
        data-testid="login-history-empty"
      >
        暂无登录记录
      </div>
      <div v-else class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">时间</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">结果</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">入口</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">方式</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">IP</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">设备</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr
            v-for="row in items"
            :key="row.id"
            data-testid="login-history-row"
          >
            <td class="px-4 py-3 text-sm text-gray-700 whitespace-nowrap">{{ formatTime(row.logged_in_at) }}</td>
            <td class="px-4 py-3 text-sm">
              <span
                v-if="row.outcome && row.outcome !== 'success'"
                class="inline-flex items-center rounded-full bg-red-50 px-2 py-0.5 text-xs font-medium text-red-600"
                data-testid="login-history-outcome-failure"
              >{{ row.outcome_label || row.outcome }}</span>
              <span v-else class="text-gray-500">{{ row.outcome_label || '成功' }}</span>
            </td>
            <td class="px-4 py-3 text-sm text-gray-900">{{ row.entry_label || row.entry }}</td>
            <td class="px-4 py-3 text-sm text-gray-700">{{ row.method_label || row.method_type || '—' }}</td>
            <td class="px-4 py-3 text-sm font-mono text-gray-700">{{ row.client_ip || '—' }}</td>
            <td class="px-4 py-3 text-sm text-gray-500 max-w-xs truncate" :title="row.user_agent || ''">{{ row.user_agent || '—' }}</td>
          </tr>
        </tbody>
      </table>
      </div>
      <div v-if="hasMore" class="p-4 border-t border-gray-100">
        <!-- Anti-Replay-OK: read-only pagination GET; in-flight lock only -->
        <button
          type="button"
          class="px-4 py-2 text-sm rounded-lg border border-border text-text hover:bg-primary/5 disabled:opacity-50"
          data-testid="login-history-load-more"
          :disabled="loading"
          :aria-busy="loading ? 'true' : 'false'"
          @click="loadMore"
        >
          {{ loading ? '加载中...' : '加载更多' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { createClickGuard } from '../utils/clickGuard.js'

const props = defineProps({
  apiUrl: {
    type: String,
    required: true,
  },
})

const PAGE_SIZE = 20
const loading = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
const items = ref([])
const total = ref(0)
const showFailures = ref(false)
const loadMoreGuard = createClickGuard()

const hasMore = computed(() => items.value.length < total.value)

function formatTime(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString('zh-CN')
}

function listUrl(offset) {
  const base = String(props.apiUrl || '')
  const joiner = base.includes('?') ? '&' : '?'
  const params = [`limit=${PAGE_SIZE}`, `offset=${offset}`]
  if (showFailures.value) {
    params.push('include_failures=1')
  }
  return `${base}${joiner}${params.join('&')}`
}

async function setIncludeFailures(include) {
  if (include === showFailures.value) return
  showFailures.value = include
  items.value = []
  total.value = 0
  await fetchPage(0, false)
}

async function fetchPage(offset, append) {
  loading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  try {
    const response = await apiFetch(listUrl(offset), {
      method: 'GET',
      headers: { Accept: 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
      credentials: 'include',
    })
    const traceId = response.headers?.get?.('X-Trace-Id') || response.headers?.get?.('x-trace-id') || ''
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      loadError.value = data.detail || data.error || `加载登录历史失败 (${response.status})`
      loadErrorTraceId.value = data.trace_id || traceId || ''
      return
    }
    const rows = Array.isArray(data.results) ? data.results : []
    total.value = Number(data.total) || 0
    items.value = append ? items.value.concat(rows) : rows
  } catch (err) {
    loadError.value = err?.message || '加载登录历史失败'
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  await loadMoreGuard.run(async () => {
    await fetchPage(items.value.length, true)
  })
}

onMounted(() => {
  fetchPage(0, false)
})
</script>
