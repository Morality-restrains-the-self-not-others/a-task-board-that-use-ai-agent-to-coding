<template>
  <div class="px-4 py-3 bg-white shadow-xl w-full box-border">
    <!-- 单行工具栏：标题/搜索/调度 + 机器摘要 与 右侧操作并排；窄屏 flex-wrap。 -->
    <!-- 标题区 / 摘要区仍分两个 data-alias，以隔离 workspace 与机器轮询两路 data-traceId。 -->
    <div
      class="flex flex-wrap items-center gap-x-4 gap-y-2"
      data-alias="header-toolbar-row"
    >
    <!-- data-traceId：工作空间列表加载失败时挂载失败请求 traceId，成功后清除（对齐全站约定） -->
    <div
      class="flex flex-nowrap items-center gap-x-4 min-w-0"
      data-alias="header-title-row"
      :data-traceId="workspaceLoadErrorTraceId || undefined"
    >
      <h2 class="text-lg font-bold text-text shrink-0">工作面板</h2>
      <!-- OPT-20260902-021：搜索根节点自带定宽，去掉固定宽度包裹层；父级直接放行即可 -->
      <WorkPanelTaskSearch
        :visible="Boolean(tenantId)"
        :tenant-id="tenantId == null ? '' : String(tenantId)"
        :access-code="accessCode"
      />
      <!-- 自动调度安排入口（OPT-20260824-067：排队调度入口自侧栏移至搜索框与工作空间选择器之间） -->
      <router-link
        v-if="autoScheduleRoute"
        :to="autoScheduleRoute"
        class="text-sm text-primary hover:underline whitespace-nowrap"
        data-testid="auto-schedule-link"
        data-alias="auto-schedule-link"
      >
        自动调度安排
      </router-link>
    </div>
    <!-- data-traceId：机器摘要/运行态 15s 轮询失败时挂载失败请求 traceId，成对成功后清除（对齐全站约定） -->
    <div
      class="flex flex-nowrap items-center gap-x-4 min-w-0 flex-1"
      data-alias="header-summary-row"
      :data-traceId="machineSummaryErrorTraceId || undefined"
    >
      <div class="flex flex-wrap items-center gap-x-3 gap-y-1 min-w-0">
        <WorkspaceMachineSummary
          v-if="showMachineSummary"
          :machine-summary="machineSummary"
          :machine-runtime-filter="machineRuntimeFilter"
          @machine-filter="$emit('machine-filter', $event)"
        />
        <span
          v-if="machineSummaryStaleSince"
          data-alias="machine-summary-stale-badge"
          class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs text-gray-500 bg-gray-100 border border-gray-200 shrink-0"
          :title="`数据可能已过期（起始于 ${new Date(machineSummaryStaleSince).toLocaleString()}）`"
        >数据可能已过期</span>
        <button
          v-if="machineRuntimeFilterChipLabel"
          type="button"
          data-alias="machine-status-filter-chip"
          class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-primary/10 text-primary border border-primary/20 hover:bg-primary/15 focus:outline-none focus-visible:ring-1 focus-visible:ring-primary shrink-0"
          :aria-label="`清除${machineRuntimeFilterChipLabel}`"
          @click="$emit('clear-machine-filter')"
        >
          <span>{{ machineRuntimeFilterChipLabel }}</span>
          <span aria-hidden="true">×</span>
        </button>
        <button
          v-if="accessFilterChipLabel"
          type="button"
          data-alias="access-filter-chip"
          class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-primary/10 text-primary border border-primary/20 hover:bg-primary/15 focus:outline-none focus-visible:ring-1 focus-visible:ring-primary shrink-0"
          :aria-label="`清除${accessFilterChipLabel}`"
          @click="$emit('clear-access-filter')"
        >
          <span>{{ accessFilterChipLabel }}</span>
          <span aria-hidden="true">×</span>
        </button>
        <p
          v-if="showEmptyTaskHint || todosLoadError"
          class="text-sm text-gray-500"
          data-alias="work-panel-empty-task-hint"
          :data-traceId="todosLoadErrorTraceId || undefined"
        >
          <template v-if="todosLoadError">
            任务列表加载失败：{{ todosLoadError }}
          </template>
          <template v-else-if="machineSummary && machineSummary.startedCount > 0">
            任务列表为空，但机器节点显示已启动 {{ machineSummary.startedCount }} 台；请刷新页面或检查任务是否加载失败。
          </template>
          <template v-else>
            当前工作空间暂无任务，点击「创建任务」开始
          </template>
        </p>
      </div>
      <div class="ml-auto flex items-center gap-2 shrink-0" data-alias="header-actions-row">
        <div class="flex items-center gap-2 shrink-0" data-alias="header-workspace-switcher-row">
          <WorkspaceSwitcher
            v-if="tenantId"
            :tenant="tenantId"
            :refresh-trigger="refreshTrigger"
            @workspace-switched="handleWorkspaceSwitched"
            @workspace-created="$emit('workspace-created')"
            @workspace-load-error="workspaceLoadErrorTraceId = $event"
            @workspace-loaded="workspaceLoadErrorTraceId = ''"
          />
          <button v-else class="btn-secondary" disabled>加载中...</button>
        </div>
        <button id="create-task-btn" class="btn-primary" @click="$emit('create-task')">创建任务</button>
        <div class="relative" data-alias="access-filter-root">
          <button
            type="button"
            data-alias="access-filter-toggle"
            class="btn-secondary"
            :aria-expanded="accessFilterPanelOpen ? 'true' : 'false'"
            @click="$emit('toggle-access-filter-panel')"
            @dblclick.prevent="$emit('close-access-filter-panel')"
          >人/小组</button>
          <div
            v-if="accessFilterPanelOpen"
            data-alias="access-filter-panel"
            class="absolute right-0 mt-1 w-64 max-h-80 overflow-hidden rounded-md border border-gray-200 bg-white shadow-lg z-20 flex flex-col"
          >
            <div class="flex border-b border-gray-100 shrink-0">
              <button
                type="button"
                data-alias="access-filter-tab-person"
                class="flex-1 px-3 py-2 text-sm"
                :class="accessFilterTab === 'person' ? 'text-primary font-medium border-b-2 border-primary' : 'text-gray-600'"
                @click="$emit('access-filter-tab', 'person')"
              >人</button>
              <button
                type="button"
                data-alias="access-filter-tab-group"
                class="flex-1 px-3 py-2 text-sm"
                :class="accessFilterTab === 'group' ? 'text-primary font-medium border-b-2 border-primary' : 'text-gray-600'"
                @click="$emit('access-filter-tab', 'group')"
              >小组</button>
            </div>
            <div class="overflow-y-auto flex-1 min-h-0">
              <p v-if="accessSubjectsLoading" class="px-3 py-3 text-sm text-gray-500">加载中…</p>
              <template v-else-if="accessFilterTab === 'person'">
                <p v-if="accessPeople.length === 0" class="px-3 py-3 text-sm text-gray-500">暂无可访问成员</p>
                <button
                  v-for="p in accessPeople"
                  :key="`person-${p.id}`"
                  type="button"
                  data-alias="access-filter-person-option"
                  class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50"
                  :class="isAccessSelected('person', p.id) ? 'bg-primary/5 text-primary font-medium' : 'text-gray-800'"
                  @click="$emit('access-filter-select', { kind: 'person', id: p.id, label: p.label })"
                >{{ p.label }}</button>
              </template>
              <template v-else>
                <p v-if="accessGroups.length === 0" class="px-3 py-3 text-sm text-gray-500">暂无可访问小组</p>
                <button
                  v-for="g in accessGroups"
                  :key="`group-${g.id}`"
                  type="button"
                  data-alias="access-filter-group-option"
                  class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50"
                  :class="isAccessSelected('group', g.id) ? 'bg-primary/5 text-primary font-medium' : 'text-gray-800'"
                  @click="$emit('access-filter-select', { kind: 'group', id: g.id, label: g.label })"
                >{{ g.label }}</button>
              </template>
            </div>
          </div>
        </div>
        <button class="btn-secondary" @click="$emit('filter')">筛选</button>
      </div>
    </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import WorkspaceSwitcher from '../components/WorkspaceSwitcher.vue'
import WorkPanelTaskSearch from '../components/WorkPanelTaskSearch.vue'
import WorkspaceMachineSummary from '../components/WorkspaceMachineSummary.vue'
import { machineRuntimeFilterChipLabel as chipLabelForFilter } from '../utils/workPanelMachineRuntimeFilter.js'
import { accessFilterChipLabel as accessChipLabelForFilter } from '../utils/workPanelAccessFilter.js'

/** 工作空间列表加载失败的请求 traceId（header-title-row data-traceId 数据源） */
const workspaceLoadErrorTraceId = ref('')

/**
 * 当前选中工作空间 ID（OPT-20260824-067）。
 * WorkspaceSwitcher 加载成功/切换时上抛 workspace-switched 事件，此处记录
 * 以构造「自动调度安排」入口链接（路由 queue-schedule 依赖 workspace_id query）。
 */
const currentWorkspaceId = ref('')

/** 自动调度安排入口路由：tenantId + workspaceId 就绪后可见 */
const autoScheduleRoute = computed(() => {
  if (props.tenantId == null || props.tenantId === '' || !currentWorkspaceId.value) return ''
  return `/tenant/${props.tenantId}/queue-schedule/?workspace_id=${encodeURIComponent(currentWorkspaceId.value)}`
})

/** 记录当前工作空间并转发事件（WorkspaceSwitcher 挂载后自动上抛初始选中项） */
function handleWorkspaceSwitched(workspace) {
  if (workspace?.id != null) {
    currentWorkspaceId.value = String(workspace.id)
  }
  emit('workspace-switched', workspace)
}

const props = defineProps({
  tenantId: {
    type: [String, Number],
    default: null
  },
  refreshTrigger: {
    type: Number,
    default: 0
  },
  showEmptyTaskHint: {
    type: Boolean,
    default: false
  },
  /** 任务列表 HTTP/网络失败文案；非空时优先于空看板提示，避免与机器摘要打架 */
  todosLoadError: {
    type: String,
    default: ''
  },
  todosLoadErrorTraceId: {
    type: String,
    default: ''
  },
  /** @type {import('vue').PropType<import('../utils/workPanelMachineSummary.js').WorkspaceMachineSummary | null>} */
  machineSummary: {
    type: Object,
    default: null
  },
  /** 兼容旧调用：有非空字符串时即使 summary 为空也展示占位行 */
  machineSummaryLabel: {
    type: String,
    default: ''
  },
  /** 机器摘要/运行态轮询失败请求的 traceId（header-summary-row data-traceId 数据源，成功清除） */
  machineSummaryErrorTraceId: {
    type: String,
    default: ''
  },
  /** OPT-20260809-029: 数据开始陈旧的时间戳（ms，null = 当前快照新鲜）。连续轮询失败超过阈值置位，成对成功恢复即清除 */
  machineSummaryStaleSince: {
    type: Number,
    default: null
  },
  machineRuntimeFilter: {
    type: String,
    default: null
  },
  accessFilter: {
    type: Object,
    default: null
  },
  accessPeople: {
    type: Array,
    default: () => []
  },
  accessGroups: {
    type: Array,
    default: () => []
  },
  accessSubjectsLoading: {
    type: Boolean,
    default: false
  },
  accessFilterPanelOpen: {
    type: Boolean,
    default: false
  },
  accessFilterTab: {
    type: String,
    default: 'person'
  },
  /** 透传到任务搜索跳转 URL，避免丢失 accessCode */
  accessCode: {
    type: String,
    default: ''
  }
})

const emit = defineEmits([
  'create-task',
  'filter',
  'workspace-switched',
  'workspace-created',
  'machine-filter',
  'clear-machine-filter',
  'toggle-access-filter-panel',
  'close-access-filter-panel',
  'access-filter-tab',
  'access-filter-select',
  'clear-access-filter',
])

const showMachineSummary = computed(
  () => Boolean(props.machineSummary) || Boolean(props.machineSummaryLabel),
)

const machineRuntimeFilterChipLabel = computed(() =>
  chipLabelForFilter(props.machineRuntimeFilter),
)

const accessFilterChipLabel = computed(() =>
  accessChipLabelForFilter(props.accessFilter),
)

function isAccessSelected(kind, id) {
  const f = props.accessFilter
  if (!f || f.kind !== kind) return false
  return String(f.id) === String(id)
}
</script>

<style scoped>
/* 组件样式继承自原WorkPanel组件 */
</style>
