<template>
  <div class="bg-white p-6 rounded-xl shadow" data-testid="queue-members-card">
    <div class="flex justify-between items-center mb-4 flex-wrap gap-2">
      <h3 class="text-xl font-bold text-text">排队任务</h3>
      <div class="flex items-center gap-2">
        <button
          type="button"
          class="px-3 py-1.5 text-sm rounded-md text-white bg-primary hover:bg-primary/90 disabled:opacity-50"
          data-testid="queue-join-toggle"
          :aria-expanded="joinPanelOpen ? 'true' : 'false'"
          @click="toggleJoinPanel"
        >
          <!-- Anti-Replay-OK: ui-only 展开/收起搜索面板 -->
          {{ joinPanelOpen ? '收起加入' : '加入队列' }}
        </button>
        <button
          type="button"
          class="px-3 py-1.5 text-sm border border-gray-300 rounded-md text-gray-800 bg-white hover:bg-gray-50 disabled:opacity-50"
          data-testid="refresh-members"
          :disabled="refreshGuard.isBusy() || loading"
          :aria-busy="refreshGuard.isBusy() ? 'true' : 'false'"
          @click="onRefreshClick"
        >
          <!-- Anti-Replay-OK: read-refresh 只读刷新队列快照 -->
          {{ refreshGuard.isBusy() || loading ? '刷新中...' : '刷新' }}
        </button>
      </div>
    </div>

    <div
      v-if="joinPanelOpen"
      class="mb-4 border border-gray-200 rounded-lg p-3 space-y-2"
      data-testid="queue-join-panel"
    >
      <div class="flex flex-wrap gap-2 items-end">
        <label class="block text-sm text-text-light flex-1 min-w-[12rem]">
          搜索本工作空间任务
          <input
            v-model="query"
            type="search"
            autocomplete="off"
            class="mt-1 w-full px-3 py-1.5 border border-gray-300 rounded-md"
            placeholder="任务编号 / 标题"
            data-testid="queue-join-search-input"
            @keydown.enter.prevent="runButtonSearch"
          >
        </label>
        <button
          type="button"
          class="px-3 py-1.5 text-sm border border-gray-300 rounded-md text-gray-800 bg-white hover:bg-gray-50 disabled:opacity-50"
          data-testid="queue-join-search-submit"
          :disabled="searchLoading || searchButtonGuard.isBusy()"
          :aria-busy="searchLoading ? 'true' : 'false'"
          @click="runButtonSearch"
        >
          <!-- Anti-Replay-OK: read-refresh 只读搜索 -->
          {{ searchLoading ? '搜索中...' : '搜索' }}
        </button>
      </div>
      <p
        v-if="searchError"
        class="text-sm text-red-600"
        data-testid="queue-join-search-error"
        :data-traceId="searchErrorTraceId || undefined"
      >
        {{ searchError }}
      </p>
      <div v-else-if="searchLoading" class="text-sm text-text-light" data-testid="queue-join-search-loading">
        搜索中...
      </div>
      <div
        v-else-if="searchRan && !searchHits.length"
        class="text-sm text-text-light"
        data-testid="queue-join-search-empty"
      >
        无匹配且未入队的任务
      </div>
      <ul v-else-if="searchHits.length" class="divide-y divide-gray-100" data-testid="queue-join-search-results">
        <li
          v-for="hit in searchHits"
          :key="String(hit.id)"
          class="py-2 flex justify-between items-center gap-3"
        >
          <span class="text-sm text-text truncate">{{ hit.title || '（无标题）' }}</span>
          <button
            type="button"
            class="shrink-0 px-3 py-1 text-sm rounded-md text-white bg-primary hover:bg-primary/90 disabled:opacity-50"
            :data-testid="`queue-join-hit-${hit.id}`"
            :disabled="joinGuard.isBusy() || busyTaskId === hit.id"
            :aria-busy="busyTaskId === hit.id ? 'true' : 'false'"
            @click="joinHit(hit)"
          >
            {{ busyTaskId === hit.id ? '加入中...' : '加入队列' }}
          </button>
        </li>
      </ul>
    </div>

    <p
      v-if="actionError"
      class="mb-3 text-sm text-red-600"
      data-testid="queue-members-error"
      :data-traceId="actionErrorTraceId || undefined"
    >
      {{ actionError }}
    </p>

    <div v-if="loading" class="text-text-light text-sm py-4" data-testid="members-loading">
      加载中...
    </div>
    <div v-else-if="!members.length" class="text-text-light text-sm py-4" data-testid="members-empty">
      暂无加入排队调度的任务。可点击「加入队列」搜索任务。
    </div>
    <table v-else class="w-full text-sm">
      <thead>
        <tr class="text-left text-text-light border-b border-gray-200">
          <th class="py-2 pr-4 font-medium">任务</th>
          <th class="py-2 pr-4 font-medium">状态</th>
          <th class="py-2 pr-4 font-medium">层级</th>
          <th class="py-2 pr-4 font-medium">入队时间</th>
          <th class="py-2 font-medium">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="m in members" :key="m.task_id" class="border-b border-gray-100">
          <td class="py-2 pr-4">
            <!-- Anti-Replay-OK: 真实 a[href] 打开任务详情，无写请求 -->
            <a
              v-if="m.href"
              :href="m.href"
              class="font-medium text-text hover:text-primary hover:underline"
              :data-testid="`queue-member-title-${m.task_id}`"
            >{{ m.title || '（无标题）' }}</a>
            <span
              v-else
              class="font-medium text-text"
              :data-testid="`queue-member-title-${m.task_id || 'unknown'}`"
            >{{ m.title || '（无标题）' }}</span>
            <span class="text-text-light ml-2 text-xs">{{ m.task_id }}</span>
          </td>
          <td class="py-2 pr-4" :data-testid="`member-status-${m.task_id}`">
            {{ memberStatusText(m) }}
          </td>
          <td class="py-2 pr-4">{{ m.depth }}</td>
          <td class="py-2 pr-4 text-text-light">{{ formatTime(m.enqueued_at) }}</td>
          <td class="py-2">
            <button
              type="button"
              class="px-3 py-1 text-sm border border-gray-300 rounded-md text-gray-800 bg-white hover:bg-gray-50 disabled:opacity-50"
              :data-testid="`queue-leave-${m.task_id}`"
              :disabled="leaveGuard.isBusy() || busyTaskId === m.task_id || !m.task_id"
              :aria-busy="busyTaskId === m.task_id ? 'true' : 'false'"
              @click="leaveMember(m)"
            >
              {{ busyTaskId === m.task_id ? '处理中...' : '离开队列' }}
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import { createClickGuard } from '../../utils/clickGuard.js'
import { extractTraceId } from '../../utils/traceId.js'
import { buildTaskSearchQuery } from '../../utils/navbarTaskSearch.js'
import modalService from '../../utils/modalService.js'

const props = defineProps({
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  snapshot: { type: Object, default: null },
  members: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  actionError: { type: String, default: '' },
  actionErrorTraceId: { type: String, default: '' },
  memberStatusText: { type: Function, required: true },
  formatTime: { type: Function, required: true },
  patchQueuedAutoRun: { type: Function, required: true },
})

const emit = defineEmits(['refresh'])

function isWorkspaceAutoScheduleEnabled(snapshot) {
  return snapshot?.schedule_rhythm?.enabled === true
}

const joinPanelOpen = ref(false)
const query = ref('')
const searchHits = ref([])
const searchLoading = ref(false)
const searchError = ref('')
const searchErrorTraceId = ref('')
const searchRan = ref(false)
const busyTaskId = ref('')

const joinGuard = createClickGuard()
const leaveGuard = createClickGuard()
const refreshGuard = createClickGuard()
const searchGuard = createClickGuard()
// 搜索按钮/回车走无 debounce 的 in-flight 锁（OPT-20260827-044）：
// 输入框 watch 已做 250ms 自动搜索，按钮再套 300ms guard debounce
// 会让「改关键字后立刻点搜索」的第二次被静默跳过。
const searchButtonGuard = createClickGuard({ debounceMs: 0 })

const memberIdSet = computed(() => {
  const set = new Set()
  for (const m of props.members || []) {
    const id = String(m?.task_id || '').trim()
    if (id) set.add(id)
  }
  return set
})

function toggleJoinPanel() {
  joinPanelOpen.value = !joinPanelOpen.value
}

async function onRefreshClick() {
  await refreshGuard.run(async () => {
    emit('refresh')
  })
}

function clearPendingDebounce() {
  if (debounceTimer) {
    clearTimeout(debounceTimer)
    debounceTimer = null
  }
}

async function doSearch() {
  const q = String(query.value || '').trim()
  searchError.value = ''
  searchErrorTraceId.value = ''
  searchRan.value = true
  if (!q) {
    searchHits.value = []
    return
  }
  const apiFetch = window.apiFetch
  if (typeof apiFetch !== 'function') {
    searchError.value = 'apiFetch unavailable'
    searchHits.value = []
    return
  }
  searchLoading.value = true
  try {
    const qs = buildTaskSearchQuery({
      q,
      workspaceId: props.workspaceId,
      limit: 20,
    })
    const resp = await apiFetch(`/api/tasks/search/tenant_id/${props.tenantId}/?${qs}`, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok) {
      const err = new Error(data?.error || data?.message || `搜索失败（${resp.status}）`)
      err.traceId = extractTraceId(resp) || data?.trace_id || ''
      throw err
    }
    const rows = Array.isArray(data?.results) ? data.results : []
    const wid = String(props.workspaceId || '')
    searchHits.value = rows.filter((hit) => {
      const id = String(hit?.id || '').trim()
      if (!id || memberIdSet.value.has(id)) return false
      const hitWs = String(hit?.workspace_id || '')
      return !hitWs || hitWs === wid
    })
  } catch (e) {
    searchError.value = e?.message || '搜索失败'
    searchErrorTraceId.value = e?.traceId || extractTraceId(e) || ''
    searchHits.value = []
    console.warn('[QueueMembersCard] search_error', { message: searchError.value })
  } finally {
    searchLoading.value = false
  }
}

async function runSearch() {
  await searchGuard.run(async () => {
    await doSearch()
  })
}

// 搜索按钮 / 回车显式提交：清掉 pending debounce 并绕过 guard 的 300ms
// debounce（仅保留 in-flight 锁），保证连续两次不同关键字都发出 GET。
async function runButtonSearch() {
  clearPendingDebounce()
  await searchButtonGuard.run(async () => {
    await doSearch()
  })
}

async function joinHit(hit) {
  const id = String(hit?.id || '').trim()
  if (!id) return
  if (!isWorkspaceAutoScheduleEnabled(props.snapshot)) {
    await modalService.alert('请先在上方「调度设置」中勾选「启用自动调度」并保存，再加入队列。')
    return
  }
  await joinGuard.run(async ({ idempotencyKey }) => {
    busyTaskId.value = id
    try {
      const ok = await props.patchQueuedAutoRun(id, true, idempotencyKey)
      if (ok) {
        console.info('[QueueMembersCard] queue_members_join', { task_id: id })
        searchHits.value = searchHits.value.filter((h) => String(h.id) !== id)
      }
    } finally {
      busyTaskId.value = ''
    }
  })
}

async function leaveMember(m) {
  const id = String(m?.task_id || '').trim()
  if (!id) return
  try {
    await modalService.confirm(`确定将「${m.title || '（无标题）'}」移出自动调度队列？`)
  } catch {
    return
  }
  await leaveGuard.run(async ({ idempotencyKey }) => {
    busyTaskId.value = id
    try {
      const ok = await props.patchQueuedAutoRun(id, false, idempotencyKey)
      if (ok) {
        console.info('[QueueMembersCard] queue_members_leave', { task_id: id })
      }
    } finally {
      busyTaskId.value = ''
    }
  })
}

let debounceTimer = null
watch(query, () => {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    if (joinPanelOpen.value && String(query.value || '').trim()) {
      runSearch()
    }
  }, 250)
})

onBeforeUnmount(() => {
  if (debounceTimer) clearTimeout(debounceTimer)
})
</script>
