<template>
  <div data-alias="panel-system-admin-recharge-consumption">
    <p class="text-sm text-gray-500 mb-4">
      按用户汇总支付金额与资源消费情况。「可分成引流分成金额」等于「已消费金额」——仅已消费部分可计入下线分销佣金。
    </p>

    <form class="flex flex-wrap items-end gap-3 mb-4" @submit.prevent="search">
      <div class="flex-1 min-w-[200px]">
        <label class="block text-sm text-gray-700 mb-1">搜索用户</label>
        <input
          v-model="searchQuery"
          type="search"
          placeholder="邮箱 / 用户名 / 手机号 / 用户 ID"
          class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
        />
      </div>
      <button
        type="submit"
        class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 text-sm"
        :disabled="loading"
      >
        搜索
      </button>
      <button
        v-if="searchQuery"
        type="button"
        class="px-4 py-2 border border-gray-300 rounded-lg text-sm text-gray-700 hover:bg-gray-50"
        @click="clearSearch"
      >
        清除
      </button>
    </form>

    <div v-if="loadError" class="text-red-600 text-sm mb-4" :data-traceId="loadErrorTraceId || undefined">
      {{ loadError }}
    </div>

    <div v-if="loading" class="text-sm text-gray-500 mb-4">加载中…</div>

    <div v-else class="overflow-x-auto border border-gray-200 rounded-lg">
      <table class="min-w-full divide-y divide-gray-200 text-sm" data-testid="recharge-consumption-table">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-3 py-2 text-left">用户 ID</th>
            <th class="px-3 py-2 text-left">邮箱</th>
            <th class="px-3 py-2 text-right">支付金额（元）</th>
            <th class="px-3 py-2 text-right">可分成引流分成金额</th>
            <th class="px-3 py-2 text-right">未消费金额</th>
            <th class="px-3 py-2 text-right">已消费金额</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="row in rows" :key="row.user_id">
            <td class="px-3 py-2 font-mono text-xs">{{ row.user_id }}</td>
            <td class="px-3 py-2">
              <div>{{ row.email || '—' }}</div>
              <div v-if="row.username" class="text-xs text-gray-500">{{ row.username }}</div>
            </td>
            <td class="px-3 py-2 text-right tabular-nums">{{ row.recharge_amount_yuan ?? '—' }}</td>
            <td class="px-3 py-2 text-right tabular-nums">{{ formatYuanCents(row.commissionable_points) }}</td>
            <td class="px-3 py-2 text-right tabular-nums">{{ formatYuanCents(row.unconsumed_points) }}</td>
            <td class="px-3 py-2 text-right tabular-nums">{{ formatYuanCents(row.consumed_points) }}</td>
          </tr>
          <tr v-if="rows.length === 0">
            <td class="px-3 py-4 text-gray-500 text-center" colspan="6">暂无数据</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="total !== null" class="flex items-center justify-between mt-4 text-sm text-gray-600">
      <span>共 {{ total }} 条</span>
      <div class="flex items-center gap-2">
        <button
          type="button"
          class="px-3 py-1.5 border border-gray-300 rounded-md disabled:opacity-50"
          :disabled="offset === 0 || loading"
          @click="goPrev"
        >
          上一页
        </button>
        <span>{{ pageLabel }}</span>
        <button
          type="button"
          class="px-3 py-1.5 border border-gray-300 rounded-md disabled:opacity-50"
          :disabled="!hasNext || loading"
          @click="goNext"
        >
          下一页
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import { extractTraceId } from '../utils/traceId.js'
import { safeJson } from '../utils/safeResponseJson.js'
import { formatYuanFromCents } from '../utils/formatYuanCents.js'

/* @alias:panel-system-admin-recharge-consumption */

const PAGE_LIMIT = 50

const rows = ref([])
const total = ref(null)
const offset = ref(0)
const loading = ref(false)
const loadError = ref('')
const loadErrorTraceId = ref('')
const searchQuery = ref('')
const activeQuery = ref('')

const hasNext = computed(() => {
  if (total.value === null) return false
  return offset.value + PAGE_LIMIT < total.value
})

const pageLabel = computed(() => {
  if (total.value === null || total.value === 0) return '第 1 页'
  const page = Math.floor(offset.value / PAGE_LIMIT) + 1
  const pages = Math.max(1, Math.ceil(total.value / PAGE_LIMIT))
  return `第 ${page} / ${pages} 页`
})

const formatYuanCents = (value) => formatYuanFromCents(value)

const loadRows = async () => {
  loading.value = true
  loadError.value = ''
  loadErrorTraceId.value = ''
  let requestTraceId = ''
  try {
    const params = new URLSearchParams({
      limit: String(PAGE_LIMIT),
      offset: String(offset.value),
    })
    if (activeQuery.value) {
      // 后端 q 参数：纯数字视为精确用户 ID 列表，否则按邮箱/用户名/手机号模糊搜索
      params.set('q', activeQuery.value)
    }
    const response = await apiFetch(`/api/system-admin/user-recharge-consumption/?${params.toString()}`, {
      method: 'GET',
    })
    requestTraceId = extractTraceId(response) || ''
    if (!response.ok) {
      loadErrorTraceId.value = requestTraceId
      let message = `加载失败 (${response.status})`
      try {
        const err = await safeJson(response, {})
        // 兼容后端 writeErrorJSON ("error") 和旧 Django 风格 ("detail") 两种键名
        if (err.detail) message = err.detail
        else if (err.error) message = err.error
      } catch {
        /* ignore */
      }
      loadError.value = message
      rows.value = []
      total.value = 0
      return
    }
    // 安全解析 JSON：响应体可能为 HTML（网关异常等）
    const data = await safeJson(response, null)
    if (data === null) {
      loadErrorTraceId.value = requestTraceId
      loadError.value = '服务器返回了无法解析的响应，请稍后重试'
      rows.value = []
      total.value = 0
      return
    }
    rows.value = Array.isArray(data.results) ? data.results : []
    total.value = typeof data.total === 'number' ? data.total : rows.value.length
  } catch (error) {
    console.error('加载消费情况失败:', error)
    loadErrorTraceId.value = error.traceId || requestTraceId
    loadError.value = '加载消费情况失败，请稍后重试'
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const search = async () => {
  activeQuery.value = searchQuery.value.trim()
  offset.value = 0
  await loadRows()
}

const clearSearch = async () => {
  searchQuery.value = ''
  activeQuery.value = ''
  offset.value = 0
  await loadRows()
}

const goPrev = async () => {
  if (offset.value <= 0) return
  offset.value = Math.max(0, offset.value - PAGE_LIMIT)
  await loadRows()
}

const goNext = async () => {
  if (!hasNext.value) return
  offset.value += PAGE_LIMIT
  await loadRows()
}

onMounted(() => {
  loadRows()
})
</script>
