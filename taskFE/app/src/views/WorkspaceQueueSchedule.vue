<template>
  <div class="p-8" data-testid="workspace-queue-schedule-page">
    <div class="flex flex-col space-y-6">
      <div class="flex justify-between items-center flex-wrap gap-4">
        <div>
          <h2 class="text-2xl font-bold text-text">自动调度安排</h2>
          <p class="text-text-light mt-1">
            设置自动调度时段后，工作空间内加入排队调度的任务将在时段内按需逐个运行
          </p>
        </div>
        <div v-if="tenantId" class="w-64">
          <WorkspaceSwitcher
            :tenant="tenantId"
            :initial-selected-workspace="workspaceId || ''"
            @workspace-switched="onWorkspaceSwitched"
            @workspace-load-error="onWorkspaceLoadError"
            @workspace-loaded="clearWorkspaceError"
          />
        </div>
      </div>

      <!-- 工作空间未解析：提示先选择 -->
      <div
        v-if="!workspaceId"
        class="bg-white p-6 rounded-xl shadow text-text-light"
        data-testid="no-workspace-hint"
      >
        请先选择工作空间以配置排队调度。
      </div>

      <template v-else>
        <!-- 当前调度状态条 -->
        <div
          class="bg-white p-4 rounded-xl shadow flex flex-wrap gap-x-6 gap-y-2 items-center text-sm"
          data-testid="schedule-status-bar"
        >
          <span class="font-medium" :class="snapshot?.in_window ? 'text-green-600' : 'text-text-light'">
            {{ snapshot?.in_window ? '● 允许运行时段内' : '○ 当前不在允许运行时段' }}
          </span>
          <span class="text-text-light">{{ windowMessage }}</span>
          <span class="text-text-light">
            占用槽位 <span class="font-semibold text-text">{{ queuedSlotsUsed }}</span>/{{ maxQueuedMachines }}
          </span>
        </div>

        <ScheduleHistoryCard
          :items="historyRows"
          :loading="historyLoading || historyMoreGuard.isBusy()"
          :has-more="historyHasMore"
          :load-error="historyError"
          :load-error-trace-id="historyErrorTraceId"
          :format-time="formatTime"
          @load-more="onLoadMoreHistory"
        />

        <!-- 节奏表单 -->
        <div class="bg-white p-6 rounded-xl shadow">
          <div class="flex justify-between items-center mb-6">
            <h3 class="text-xl font-bold text-text">调度设置</h3>
            <label class="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="checkbox"
                v-model="form.enabled"
                class="w-4 h-4"
                data-testid="schedule-enabled"
              >
              <span>启用自动调度</span>
            </label>
          </div>

          <div class="mb-4 max-w-xs">
            <label class="block text-sm font-medium text-text-light mb-1" for="schedule-timezone">
              时区
            </label>
            <select
              id="schedule-timezone"
              v-model="form.timezone"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary"
              data-testid="schedule-timezone"
            >
              <option v-for="tz in TIMEZONES" :key="tz" :value="tz">{{ tz }}</option>
            </select>
          </div>

          <div class="space-y-3">
            <div
              v-for="(row, i) in form.windows"
              :key="i"
              class="flex flex-wrap items-end gap-4 border border-gray-200 rounded-lg p-3"
            >
              <label class="block text-sm text-text-light">
                开始
                <input
                  type="time"
                  v-model="row.daily_start"
                  class="block mt-1 px-2 py-1.5 border border-gray-300 rounded-md"
                  :data-testid="`window-start-${i}`"
                >
              </label>
              <label class="block text-sm text-text-light">
                结束
                <input
                  type="time"
                  v-model="row.daily_end"
                  class="block mt-1 px-2 py-1.5 border border-gray-300 rounded-md"
                  :data-testid="`window-end-${i}`"
                >
              </label>
              <label class="block text-sm text-text-light">
                并发上限
                <input
                  type="number"
                  v-model.number="row.max_queued_machines"
                  min="0"
                  class="block mt-1 w-24 px-2 py-1.5 border border-gray-300 rounded-md"
                  :data-testid="`window-max-${i}`"
                >
              </label>
              <label class="flex items-center gap-1.5 text-sm pb-1.5 cursor-pointer">
                <input
                  type="checkbox"
                  v-model="row.auto_close"
                  class="w-4 h-4"
                  :data-testid="`window-auto-close-${i}`"
                >
                到点自动关闭
              </label>
              <button
                type="button"
                class="px-3 py-1.5 text-sm border border-gray-300 rounded-md text-gray-800 bg-white hover:bg-gray-50"
                :data-testid="`remove-window-${i}`"
                @click="removeWindow(i)"
              >
                删除
              </button>
            </div>
          </div>

          <div class="flex justify-end gap-3 mt-5">
            <button
              type="button"
              class="px-3 py-1.5 text-sm border border-gray-300 rounded-md text-gray-800 bg-white hover:bg-gray-50"
              data-testid="add-window"
              @click="addWindow"
            >
              + 添加时段
            </button>
            <button
              type="button"
              class="px-4 py-1.5 text-sm rounded-md text-white bg-primary hover:bg-primary/90 disabled:opacity-50"
              :disabled="saving || saveGuard.isBusy()"
              :aria-busy="saving || saveGuard.isBusy() ? 'true' : 'false'"
              data-testid="save-schedule"
              @click="save"
            >
              {{ saving ? '保存中...' : '保存设置' }}
            </button>
          </div>

          <div
            v-if="error"
            class="mt-4 text-sm text-red-600"
            :data-traceId="errorTraceId || undefined"
            data-testid="schedule-error"
          >
            {{ error }}
          </div>
          <div
            v-if="savedTip"
            class="mt-4 text-sm text-green-600"
            data-testid="schedule-saved-tip"
          >
            {{ savedTip }}
          </div>
        </div>

        <QueueMembersCard
          :tenant-id="tenantId"
          :workspace-id="workspaceId"
          :snapshot="snapshot"
          :members="members"
          :loading="loading"
          :action-error="memberActionError"
          :action-error-trace-id="memberActionErrorTraceId"
          :member-status-text="memberStatusText"
          :format-time="formatTime"
          :patch-queued-auto-run="patchQueuedAutoRun"
          @refresh="reload"
        />
      </template>
    </div>
  </div>
</template>

<script>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import WorkspaceSwitcher from '../components/WorkspaceSwitcher.vue'
import QueueMembersCard from '../components/workspaceSchedule/QueueMembersCard.vue'
import ScheduleHistoryCard from '../components/workspaceSchedule/ScheduleHistoryCard.vue'
import { resolveWorkPanelInitialWorkspace } from '../utils/workPanelWorkspaceInit.js'
import { useWorkspaceQueueSchedule } from '../composables/workspaceSchedule/useWorkspaceQueueSchedule.js'
import { buildTaskDetailHref } from '../utils/navbarTaskSearch.js'
import { createClickGuard } from '../utils/clickGuard.js'

const TIMEZONES = ['Asia/Shanghai', 'UTC', 'Asia/Tokyo', 'America/Los_Angeles', 'Europe/London']

function emptyWindowRow() {
  return { id: '', daily_start: '', daily_end: '', max_queued_machines: 1, auto_close: false }
}

export default {
  name: 'WorkspaceQueueSchedule',
  components: { WorkspaceSwitcher, QueueMembersCard, ScheduleHistoryCard },
  setup() {
    const route = useRoute()
    const tenantId = computed(() => route.params.tenant)

    const workspaceId = ref('')
    const workspaceError = ref('')

    const {
      snapshot,
      loading,
      saving,
      error,
      errorTraceId,
      loadSnapshot,
      saveSchedule,
      patchQueuedAutoRun: patchQueuedAutoRunApi,
      historyItems,
      historyHasMore,
      historyLoading,
      historyError,
      historyErrorTraceId,
      loadMoreHistory,
    } = useWorkspaceQueueSchedule({
      tenantId: () => tenantId.value,
      workspaceId: () => workspaceId.value,
    })

    const form = ref({ enabled: false, timezone: 'Asia/Shanghai', windows: [] })
    const savedTip = ref('')
    const saveGuard = createClickGuard()
    const historyMoreGuard = createClickGuard()
    const memberActionError = computed(() => error.value)
    const memberActionErrorTraceId = computed(() => errorTraceId.value)

    const windowMessage = computed(() => snapshot.value?.window_message || '')
    const queuedSlotsUsed = computed(() => Number(snapshot.value?.queued_slots_used) || 0)
    const maxQueuedMachines = computed(() => Number(snapshot.value?.max_queued_machines) || 0)
    const members = computed(() => {
      const list = snapshot.value?.members
      if (!Array.isArray(list)) return []
      const tid = tenantId.value
      const wid = workspaceId.value
      return list.map((m) => {
        const href = buildTaskDetailHref(tid, {
          workspaceId: wid,
          taskId: m?.task_id,
        })
        return { ...m, href: href === '#' ? '' : href }
      })
    })
    const historyRows = computed(() => {
      const list = historyItems.value
      if (!Array.isArray(list)) return []
      const tid = tenantId.value
      const wid = workspaceId.value
      return list.map((row) => {
        const href = row?.task_id
          ? buildTaskDetailHref(tid, { workspaceId: wid, taskId: row.task_id })
          : ''
        return { ...row, href: href === '#' ? '' : href }
      })
    })

    function fillForm() {
      const rhythm = snapshot.value?.schedule_rhythm
      if (!rhythm) {
        form.value = { enabled: false, timezone: 'Asia/Shanghai', windows: [] }
        return
      }
      const wins = Array.isArray(rhythm.windows) ? rhythm.windows : []
      form.value = {
        enabled: !!rhythm.enabled,
        timezone: rhythm.timezone || 'Asia/Shanghai',
        windows: wins.map((w) => ({
          id: w.id || '',
          daily_start: w.daily_start || '',
          daily_end: w.daily_end || '',
          max_queued_machines: Number(w.max_queued_machines) || 0,
          auto_close: !!w.auto_close,
        })),
      }
    }

    async function reload() {
      savedTip.value = ''
      await loadSnapshot()
      fillForm()
    }

    async function initWorkspace() {
      const urlWs = route.query.workspace_id
      let userWorkspace = null
      const apiFetch = window.apiFetch
      if (typeof apiFetch === 'function' && tenantId.value) {
        try {
          const resp = await apiFetch(
            `/api/accounts/users/me/?tenant_id=${encodeURIComponent(tenantId.value)}`,
            { credentials: 'include', headers: { Accept: 'application/json' } },
          )
          if (resp.ok) {
            const data = await resp.json().catch(() => null)
            userWorkspace = data?.current_workspace || null
          }
        } catch (e) {
          workspaceError.value = e?.message || '加载用户工作空间失败'
        }
      }
      const resolved = resolveWorkPanelInitialWorkspace({
        urlWorkspaceId: urlWs || null,
        userWorkspace,
      })
      workspaceId.value = resolved.id ? String(resolved.id) : ''
      if (workspaceId.value) await reload()
    }

    function onWorkspaceSwitched({ id }) {
      if (!id) return
      workspaceId.value = String(id)
      reload()
    }

    function onWorkspaceLoadError(e) {
      workspaceError.value = e?.message || '工作空间列表加载失败'
    }

    function clearWorkspaceError() {
      workspaceError.value = ''
    }

    function addWindow() {
      form.value.windows.push(emptyWindowRow())
    }

    function removeWindow(i) {
      form.value.windows.splice(i, 1)
    }

    async function save() {
      await saveGuard.run(async ({ idempotencyKey }) => {
        const ok = await saveSchedule(
          {
            enabled: form.value.enabled,
            timezone: form.value.timezone,
            windows: form.value.windows.map((w) => ({
              id: w.id,
              daily_start: w.daily_start,
              daily_end: w.daily_end,
              max_queued_machines: Number(w.max_queued_machines) || 0,
              auto_close: !!w.auto_close,
            })),
          },
          { idempotencyKey },
        )
        if (ok) {
          savedTip.value = '调度设置已保存'
          fillForm()
        }
      })
    }

    async function patchQueuedAutoRun(taskId, queued, idempotencyKey) {
      return patchQueuedAutoRunApi(taskId, queued, idempotencyKey)
    }

    async function onLoadMoreHistory() {
      await historyMoreGuard.run(async () => {
        await loadMoreHistory()
      })
    }

    function memberStatusText(m) {
      const st = m?.status
      if (st === 'deferred') return '等待时段'
      if (st === 'starting') return '调度启服中'
      if (st === 'queued') return '排队中'
      return '已入队'
    }

    function formatTime(iso) {
      if (!iso) return ''
      const d = new Date(iso)
      if (Number.isNaN(d.getTime())) return ''
      return d.toLocaleString()
    }

    onMounted(initWorkspace)

    return {
      TIMEZONES,
      tenantId,
      workspaceId,
      workspaceError,
      snapshot,
      loading,
      saving,
      error,
      errorTraceId,
      savedTip,
      form,
      windowMessage,
      queuedSlotsUsed,
      maxQueuedMachines,
      members,
      historyRows,
      historyHasMore,
      historyLoading,
      historyError,
      historyErrorTraceId,
      historyMoreGuard,
      memberActionError,
      memberActionErrorTraceId,
      patchQueuedAutoRun,
      saveGuard,
      reload,
      onWorkspaceSwitched,
      onWorkspaceLoadError,
      clearWorkspaceError,
      addWindow,
      removeWindow,
      save,
      memberStatusText,
      formatTime,
      onLoadMoreHistory,
    }
  },
}
</script>
