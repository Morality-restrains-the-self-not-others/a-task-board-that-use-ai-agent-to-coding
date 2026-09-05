<template>
  <div ref="rootEl" class="entity-revision-panel relative">
    <!-- Anti-Replay-OK: ui-only 单击展开、双击收起；GET 由 fetchGuard 防请求风暴 -->
    <button
      type="button"
      class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50"
      data-testid="entity-revision-open"
      :aria-busy="loading ? 'true' : undefined"
      :aria-expanded="open ? 'true' : 'false'"
      @click="onTriggerClick"
      @dblclick.prevent="onTriggerDblClick"
    >
      历史版本
    </button>
    <div
      v-if="open"
      class="absolute right-0 z-20 mt-2 w-[min(28rem,calc(100vw-2rem))] rounded-lg border border-gray-200 bg-white p-3 shadow-lg"
      data-testid="entity-revision-panel"
    >
      <div class="flex items-start justify-between gap-2">
        <p class="text-xs text-gray-500 mb-2">历史从创建或下次修改标题/正文起记录；不可还原到热数据。</p>
        <!-- Anti-Replay-OK: ui-only 收起历史面板 -->
        <button
          type="button"
          class="shrink-0 text-xs text-gray-600 hover:text-gray-900 px-1 py-0.5 rounded hover:bg-gray-100"
          data-testid="entity-revision-close"
          aria-label="收起历史版本"
          @click="close"
        >
          收起
        </button>
      </div>
      <div
        v-if="loadError"
        class="text-sm text-red-600"
        data-testid="entity-revision-error"
        :data-traceId="loadErrorTraceId || undefined"
      >
        {{ loadError }}
      </div>
      <p v-else-if="loading && items.length === 0" class="text-sm text-gray-500">加载中...</p>
      <p
        v-else-if="items.length === 0"
        class="text-sm text-gray-500"
        data-testid="entity-revision-empty"
      >
        尚无历史版本（创建后或下次修改标题/正文才会记录）
      </p>
      <ul v-else class="divide-y divide-gray-100 max-h-64 overflow-y-auto">
        <li
          v-for="row in items"
          :key="row.id"
          data-testid="entity-revision-row"
        >
          <button
            type="button"
            class="w-full text-left px-2 py-2 hover:bg-gray-50"
            @click="selectRow(row)"
          >
            <!-- Anti-Replay-OK: GET revision detail; in-flight via selectGuard -->
            <span class="text-sm font-medium text-gray-900">v{{ row.version_num }} · {{ headline(row) }}</span>
            <span class="block text-xs text-gray-500">{{ formatTime(row.created_at) }} · {{ row.changed_fields || '—' }}</span>
          </button>
        </li>
      </ul>
      <div
        v-if="selected"
        class="mt-3 rounded-md bg-gray-50 p-3 text-sm whitespace-pre-wrap"
        data-testid="entity-revision-detail"
      >
        <p class="font-medium text-gray-900 mb-1">{{ headline(selected) }}</p>
        <p class="text-gray-700">{{ selected.description || '（无正文）' }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { createClickGuard } from '../../utils/clickGuard.js'

const props = defineProps({
  listUrl: { type: String, default: '' },
  titleField: { type: String, default: 'title' },
})

const rootEl = ref(null)
const open = ref(false)
const loading = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
const items = ref([])
const selected = ref(null)
const fetched = ref(false)
const clickOpenedAt = ref(0)
const fetchGuard = createClickGuard({ debounceMs: 0 })
const selectGuard = createClickGuard()

const canFetch = computed(() => Boolean(String(props.listUrl || '').trim()))

function headline(row) {
  if (!row) return ''
  if (props.titleField === 'name') return row.name || ''
  return row.title || ''
}

function formatTime(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString('zh-CN')
}

function close() {
  open.value = false
}

function onTriggerClick() {
  const now = Date.now()
  if (now - clickOpenedAt.value < 350 && open.value) {
    return
  }
  open.value = true
  clickOpenedAt.value = now
  if (!fetched.value) {
    fetchList()
  }
}

function onTriggerDblClick() {
  close()
  clickOpenedAt.value = 0
}

function onDocClick(event) {
  if (!open.value) return
  const el = rootEl.value
  if (el && !el.contains(event.target)) {
    close()
  }
}

onMounted(() => {
  document.addEventListener('click', onDocClick)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick)
})

async function fetchList() {
  if (!canFetch.value) return
  await fetchGuard.run(async () => {
    if (fetched.value) return
    loading.value = true
    loadError.value = ''
    loadErrorTraceId.value = ''
    try {
      const response = await apiFetch(props.listUrl, {
        method: 'GET',
        headers: { Accept: 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
        credentials: 'include',
      })
      const traceId = response.headers?.get?.('X-Trace-Id') || response.headers?.get?.('x-trace-id') || ''
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        loadError.value = data.detail || data.error || `加载历史版本失败 (${response.status})`
        loadErrorTraceId.value = data.trace_id || traceId || ''
        return
      }
      items.value = Array.isArray(data.results) ? data.results : []
      fetched.value = true
    } catch (err) {
      loadError.value = err?.message || '加载历史版本失败'
      loadErrorTraceId.value = err?.traceId || err?.trace_id || ''
    } finally {
      loading.value = false
    }
  })
}

function detailUrl(revisionId) {
  const base = String(props.listUrl || '').replace(/\/?$/, '/')
  return `${base}${encodeURIComponent(revisionId)}/`
}

async function selectRow(row) {
  if (!row?.id) return
  selected.value = row
  await selectGuard.run(async () => {
    try {
      const response = await apiFetch(detailUrl(row.id), {
        method: 'GET',
        headers: { Accept: 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
        credentials: 'include',
      })
      const traceId = response.headers?.get?.('X-Trace-Id') || response.headers?.get?.('x-trace-id') || ''
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        loadError.value = data.detail || data.error || `加载版本详情失败 (${response.status})`
        loadErrorTraceId.value = data.trace_id || traceId || ''
        return
      }
      selected.value = data
    } catch (err) {
      loadError.value = err?.message || '加载版本详情失败'
      loadErrorTraceId.value = err?.traceId || err?.trace_id || ''
    }
  })
}
</script>
