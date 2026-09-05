<template>
  <div
    v-if="visible"
    ref="rootEl"
    class="relative w-64 max-w-sm min-w-[12rem] shrink-0"
    data-testid="work-panel-task-search"
    data-alias="work-panel-task-search"
  >
    <label class="sr-only" for="work-panel-task-search-input">搜索任务</label>
    <input
      id="work-panel-task-search-input"
      v-model="query"
      type="search"
      autocomplete="off"
      class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-md bg-white text-gray-800 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
      placeholder="任务编号 / 标题 / 负责人 / 操作员 / 评论容器"
      data-testid="work-panel-task-search-input"
      @focus="openDropdown"
      @keydown.escape.prevent="closeDropdown"
    />
    <div
      v-if="open && (loading || results.length || errorMessage || query.trim())"
      class="absolute left-0 right-0 mt-1 bg-white border border-gray-200 rounded-lg shadow-lg z-[60] max-h-80 overflow-auto"
      data-testid="work-panel-task-search-dropdown"
      role="listbox"
    >
      <div v-if="loading" class="px-3 py-2 text-sm text-gray-500">搜索中…</div>
      <div
        v-else-if="errorMessage"
        class="px-3 py-2 text-sm text-red-600"
        data-testid="work-panel-task-search-error"
        :data-traceId="errorTraceId || undefined"
      >
        {{ errorMessage }}
      </div>
      <div v-else-if="!results.length" class="px-3 py-2 text-sm text-gray-500">无匹配任务</div>
      <div
        v-for="hit in results"
        :key="String(hit.id)"
        class="px-3 py-2 border-b border-gray-50 last:border-0 hover:bg-gray-50"
        role="option"
        data-testid="work-panel-task-search-hit"
      >
        <div class="text-sm text-gray-800 truncate font-medium">{{ labelFor(hit) }}</div>
        <div class="mt-1 flex flex-wrap gap-2">
          <a
            :href="panelHref(hit)"
            class="text-xs text-primary hover:underline"
            data-testid="work-panel-task-search-open-panel"
            @click="onNavClick('open-panel', hit)"
          >
            工作面板
          </a>
          <a
            :href="taskHref(hit)"
            class="text-xs text-primary hover:underline"
            data-testid="work-panel-task-search-open-task"
            @click="onNavClick('open-task', hit)"
          >
            打开任务
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { apiFetch, parseCompanyMembersResponse } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import {
  buildTaskDetailHref,
  buildTaskSearchQuery,
  buildWorkPanelHref,
  formatSearchHitLabel,
  matchMemberIdsByQuery,
  normalizeSearchText,
} from '../utils/navbarTaskSearch.js'

const props = defineProps({
  tenantId: { type: String, default: '' },
  visible: { type: Boolean, default: false },
  accessCode: { type: String, default: '' },
})

const rootEl = ref(null)
const query = ref('')
const open = ref(false)
const loading = ref(false)
const results = ref([])
const errorMessage = ref('')
const errorTraceId = ref('')
const membersCache = ref([])
const nameById = ref(new Map())

let debounceTimer = null
let abortCtrl = null

const tenantReady = computed(() => Boolean(String(props.tenantId || '').trim()))

function labelFor(hit) {
  return formatSearchHitLabel(hit, nameById.value)
}

function closeDropdown() {
  open.value = false
}

function openDropdown() {
  if (tenantReady.value) open.value = true
}

async function ensureMembers() {
  if (!tenantReady.value || membersCache.value.length) return
  try {
    const res = await apiFetch(
      `/api/tenant/${props.tenantId}/accounts/members/company_members/`,
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    if (!res.ok) return
    const data = await res.json()
    const { members } = parseCompanyMembersResponse(data)
    const mapped = members
      .map((m) => {
        const id = String(m.id || m.company_member_id || '').trim()
        if (!id) return null
        const name = m.member_name || m.username || m.email || id
        return { id, name: String(name) }
      })
      .filter(Boolean)
    membersCache.value = mapped
    const map = new Map()
    for (const m of mapped) map.set(m.id, m.name)
    nameById.value = map
  } catch (e) {
    console.warn('[WorkPanelTaskSearch] load members failed', e)
  }
}

async function runSearch() {
    const raw = String(query.value ?? '').trim()
    const q = normalizeSearchText(query.value)
    if (q && q !== raw) {
      console.info('[WorkPanelTaskSearch] query_normalized', { origLen: raw.length, newLen: q.length })
    }
    errorMessage.value = ''
  errorTraceId.value = ''
  if (!tenantReady.value) {
    results.value = []
    return
  }
  if (!q) {
    results.value = []
    return
  }
  open.value = true
  loading.value = true
  if (abortCtrl) abortCtrl.abort()
  abortCtrl = new AbortController()
  try {
    await ensureMembers()
    const assigneeIds = matchMemberIdsByQuery(membersCache.value, q)
    const qs = buildTaskSearchQuery({ q, assigneeIds, limit: 20 })
    console.info('[WorkPanelTaskSearch] search', { tenant: props.tenantId, qLen: q.length, assigneeIds: assigneeIds.length })
    const res = await apiFetch(`/api/tasks/search/tenant_id/${props.tenantId}/?${qs}`, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
      signal: abortCtrl.signal,
    })
    if (!res.ok) {
      errorTraceId.value = extractTraceId(res) || ''
      errorMessage.value = `搜索失败（${res.status}）`
      results.value = []
      console.warn('[WorkPanelTaskSearch] search non-ok', res.status, errorTraceId.value)
      return
    }
    const data = await res.json()
    results.value = Array.isArray(data?.results) ? data.results : []
  } catch (e) {
    if (e?.name === 'AbortError') return
    errorTraceId.value = extractTraceId(e) || ''
    errorMessage.value = e?.message || '搜索失败'
    results.value = []
    console.error('[WorkPanelTaskSearch] search error', e)
  } finally {
    loading.value = false
  }
}

function scheduleSearch() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(runSearch, 250)
}

function panelHref(hit) {
  return buildWorkPanelHref(props.tenantId, {
    workspaceId: hit?.workspace_id,
    accessCode: props.accessCode,
  })
}

function taskHref(hit) {
  return buildTaskDetailHref(props.tenantId, {
    workspaceId: hit?.workspace_id,
    taskId: hit?.id,
    accessCode: props.accessCode,
    // OPT-20260817-013: 评论容器名 / cmt_ 命中时 hit 携带 comment_id，
    // 详情页据此滚到该评论并高亮。
    commentId: hit?.comment_id,
  })
}

function onNavClick(kind, hit) {
  const href = kind === 'open-task' ? taskHref(hit) : panelHref(hit)
  console.info('[WorkPanelTaskSearch] navigate', { kind, href, taskId: hit?.id })
  // 不在 click 里收起下拉：先卸载 <a> 会取消浏览器默认跳转。
}

function onDocClick(ev) {
  if (!rootEl.value) return
  if (!rootEl.value.contains(ev.target)) closeDropdown()
}

watch(query, () => {
  scheduleSearch()
})

watch(
  () => props.tenantId,
  () => {
    membersCache.value = []
    nameById.value = new Map()
    results.value = []
  },
)

onMounted(() => {
  document.addEventListener('click', onDocClick)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick)
  if (debounceTimer) clearTimeout(debounceTimer)
  if (abortCtrl) abortCtrl.abort()
})
</script>
