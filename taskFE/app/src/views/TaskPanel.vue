<template>
  <!-- 过滤栏与其条件分区紧挨；「其他」置底撑满；分区可折叠；pb-0 让看板横向滚动条贴浏览器底 -->
  <div class="flex-1 min-h-0 px-6 pt-2 pb-0 overflow-hidden flex flex-col">
    <div
      v-if="!hasKanbanColumns"
      class="rounded-lg border border-dashed border-gray-300 bg-white px-4 py-3 text-sm text-gray-500"
    >
      未加载进度列，请刷新页面或在工作空间设置中配置进度体系
    </div>
    <div
      v-else-if="!hasTaskTypes"
      class="rounded-lg border border-dashed border-gray-300 bg-white px-4 py-3 text-sm text-gray-500"
    >
      未加载交付物类别，请刷新或在工作空间设置中配置交付物体系
    </div>
    <div
      v-else
      id="task-panel-container"
      class="flex-1 min-h-0 overflow-y-auto space-y-4 flex flex-col"
      data-alias="deliverable-grouped-task-panel"
    >
      <div
        v-for="bar in deliverableFilterBars"
        :key="bar.id"
        class="deliverable-filter-block space-y-2"
        data-alias="deliverable-filter-block"
        :data-bar-id="bar.id"
      >
        <DeliverableBreadcrumb
          :path="bar.path"
          :categories="taskTypes"
          :todos="todos"
          :removable="deliverableFilterBars.length > 1"
          @clear="handleBarClear(bar.id)"
          @select-content="(payload) => handleBarSelectContent(bar.id, payload)"
          @add-bar="handleAddFilterBar(bar.id)"
          @remove-bar="handleRemoveFilterBar(bar.id)"
        />
        <DeliverableBoardSection
          v-if="sectionByBarId[bar.id]"
          :title="sectionByBarId[bar.id].label"
          title-class="text-indigo-800"
          :expanded="isSectionExpanded(bar.id)"
          :column-counts="countTodosByKanbanColumns(sectionByBarId[bar.id].todos, taskStatuses)"
          data-alias="deliverable-section-filter"
          :bar-id="bar.id"
          @toggle="toggleSection(bar.id)"
        >
          <DeliverableKanbanBoard
            :section-key="`filter-${bar.id}`"
            :container-id="`task-panel-board-${bar.id}`"
            :todos="sectionByBarId[bar.id].todos"
            :task-types="taskTypes"
            :task-statuses="taskStatuses"
            :tenant-id="tenantId"
            :workspace-id="workspaceId"
            :runtime-indicators="runtimeIndicators"
            :collaborator-name-by-id="collaboratorNameById"
            @task-clicked="$emit('task-clicked', $event)"
            @create-comment="handleCreateComment"
            @update-comment="handleUpdateComment"
            @progress-column-change="handleProgressColumnChange"
          />
        </DeliverableBoardSection>
      </div>

      <DeliverableBoardSection
        title="其他（未过滤）"
        title-class="text-gray-800"
        :expanded="isSectionExpanded(OTHER_SECTION_KEY)"
        :column-counts="countTodosByKanbanColumns(partitioned.other, taskStatuses)"
        :fill-remaining="true"
        data-alias="deliverable-section-other"
        @toggle="toggleSection(OTHER_SECTION_KEY)"
      >
        <DeliverableKanbanBoard
          section-key="other"
          container-id="task-panel-board-other"
          :todos="partitioned.other"
          :task-types="taskTypes"
          :task-statuses="taskStatuses"
          :tenant-id="tenantId"
          :workspace-id="workspaceId"
          :runtime-indicators="runtimeIndicators"
          :collaborator-name-by-id="collaboratorNameById"
          @task-clicked="$emit('task-clicked', $event)"
          @create-comment="handleCreateComment"
          @update-comment="handleUpdateComment"
          @progress-column-change="handleProgressColumnChange"
        />
      </DeliverableBoardSection>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, computed } from 'vue'
import DeliverableBreadcrumb from '../components/DeliverableBreadcrumb.vue'
import DeliverableBoardSection from '../components/DeliverableBoardSection.vue'
import DeliverableKanbanBoard from '../components/DeliverableKanbanBoard.vue'
import Sortable from 'sortablejs'
import {
  countTodosByKanbanColumns,
  filterTodosByKanbanColumn,
  resolveTodoKanbanColumnId,
} from '../utils/workPanelKanbanUtils.js'
import {
  pathSelectingDeliverableContent,
  readFilterFromPath,
  rootDeliverablePath,
} from '../utils/workPanelDeliverableAggregation.js'
import {
  createDeliverableFilterBar,
  defaultDeliverableFilterBars,
  insertDeliverableFilterBarBefore,
  partitionTodosByFilterBars,
} from '../utils/workPanelDeliverableFilterBars.js'
import {
  createTaskComment,
  patchTodo,
  resolveCommentUpdateArgs,
  updateTaskComment,
} from '../utils/taskPanelTodoApi.js'

const OTHER_SECTION_KEY = 'other'

const props = defineProps({
  tenantId: { type: [String, Number], default: null },
  workspaceId: { type: [String, Number], default: null },
  currentWorkspace: {
    type: Object,
    default: () => ({ id: null, name: '默认工作空间' }),
  },
  taskStatuses: {
    type: Array,
    default: () => [
      { id: 0, name: '待处理', color: '#ff6b6b' },
      { id: 1, name: '进行中', color: '#4dabf7' },
      { id: 2, name: '已完成', color: '#51cf66' },
    ],
  },
  taskTypes: { type: Array, default: () => [] },
  todos: { type: Array, default: () => [] },
  runtimeIndicators: { type: Object, default: () => ({}) },
  collaboratorNameById: { type: Object, default: () => ({}) },
  filterOptions: {
    type: Object,
    default: () => ({ priority: null, assignee: null, search: '' }),
  },
  deliverableFilterBars: {
    type: Array,
    default: () => defaultDeliverableFilterBars(),
  },
})

const emit = defineEmits([
  'task-clicked',
  'tasks-updated',
  'update:deliverableFilterBars',
])

const sortableInstances = ref([])
/** @type {import('vue').Ref<Record<string, boolean>>} key → collapsed */
const sectionCollapsed = ref({})

const hasKanbanColumns = computed(() => props.taskStatuses.length > 0)
const hasTaskTypes = computed(() => Array.isArray(props.taskTypes) && props.taskTypes.length > 0)

const partitioned = computed(() =>
  partitionTodosByFilterBars(props.todos, props.taskTypes, props.deliverableFilterBars),
)

const sectionByBarId = computed(() => {
  const map = Object.create(null)
  for (const section of partitioned.value.sections) {
    map[section.barId] = section
  }
  return map
})

function isSectionExpanded(key) {
  return sectionCollapsed.value[String(key)] !== true
}

function toggleSection(key) {
  const k = String(key)
  sectionCollapsed.value = {
    ...sectionCollapsed.value,
    [k]: isSectionExpanded(k),
  }
  scheduleDragInit()
}

const todoApiDeps = () => ({
  tenantId: props.tenantId,
  workspaceId: props.workspaceId,
  currentWorkspace: props.currentWorkspace,
})

function emitBars(next) {
  console.log('[work-panel] deliverable filter bars →', next)
  emit('update:deliverableFilterBars', next)
}

function updateBarPath(barId, path) {
  const next = (Array.isArray(props.deliverableFilterBars) ? props.deliverableFilterBars : []).map(
    (bar) => (String(bar.id) === String(barId) ? { ...bar, path } : bar),
  )
  emitBars(next)
}

function handleBarClear(barId) {
  updateBarPath(barId, rootDeliverablePath())
}

function handleBarSelectContent(barId, { category, content }) {
  if (!content?.id || !category?.id) return
  const bar = (props.deliverableFilterBars || []).find((b) => String(b.id) === String(barId))
  const { taskId } = readFilterFromPath(bar?.path)
  if (taskId != null && String(taskId) === String(content.id)) {
    handleBarClear(barId)
    return
  }
  updateBarPath(
    barId,
    pathSelectingDeliverableContent({ id: category.id, name: category.name }, content),
  )
}

function handleAddFilterBar(beforeBarId) {
  emitBars(
    insertDeliverableFilterBarBefore(
      props.deliverableFilterBars,
      beforeBarId,
      createDeliverableFilterBar(),
    ),
  )
}

function handleRemoveFilterBar(barId) {
  const list = Array.isArray(props.deliverableFilterBars) ? props.deliverableFilterBars : []
  if (list.length <= 1) return
  emitBars(list.filter((bar) => String(bar.id) !== String(barId)))
}

const updateTask = (taskId, updateData) => patchTodo(todoApiDeps(), taskId, updateData)

const initDragDrop = () => {
  sortableInstances.value.forEach((instance) => {
    try {
      instance.destroy()
    } catch (e) {
      console.warn('销毁 Sortable 实例时出错:', e)
    }
  })
  sortableInstances.value = []

  const containers = document.querySelectorAll(
    '#task-panel-container .task-cards-container[data-progress-column-id]',
  )
  if (containers.length === 0) return

  containers.forEach((container) => {
    try {
      const sortable = new Sortable(container, {
        group: 'shared',
        animation: 150,
        // headless / Playwright 下原生 HTML5 DnD 不稳定，强制 fallback 鼠标拖拽
        forceFallback: true,
        fallbackOnBody: true,
        draggable: '.task-card',
        filter: 'select, option, .task-card-no-drag',
        preventOnFilter: false,
        ghostClass: 'sortable-ghost',
        chosenClass: 'sortable-chosen',
        dragClass: 'sortable-drag',
        onEnd: async function (evt) {
          const taskId = evt.item.dataset.taskId
          const fromContainer = evt.from
          const toContainer = evt.to
          const targetProgressColumnId = toContainer?.getAttribute('data-progress-column-id')
          if (!targetProgressColumnId) {
            console.error('未找到目标进度列 data-progress-column-id')
            return
          }

          const updatePromises = []
          Array.from(fromContainer.querySelectorAll('.task-card')).forEach((card, index) => {
            updatePromises.push(updateTask(card.dataset.taskId, { order: index }))
          })
          Array.from(toContainer.querySelectorAll('.task-card')).forEach((card, index) => {
            const currentTaskId = card.dataset.taskId
            // 纵轴=进度列：跨纵轴拖拽只写 progress_column_id + order
            if (String(currentTaskId) === String(taskId)) {
              updatePromises.push(
                updateTask(currentTaskId, {
                  order: index,
                  progress_column_id: String(targetProgressColumnId),
                }),
              )
            } else {
              updatePromises.push(updateTask(currentTaskId, { order: index }))
            }
          })

          try {
            await Promise.all(updatePromises)
            emit('tasks-updated')
          } catch (error) {
            console.error('更新任务失败:', error)
          }
        },
      })
      sortableInstances.value.push(sortable)
    } catch (error) {
      console.error('初始化拖拽功能时发生错误:', error)
    }
  })
}

const handleProgressColumnChange = async ({ taskId, progressColumnId }) => {
  if (taskId == null || progressColumnId == null) return
  const inTarget = filterTodosByKanbanColumn(
    props.todos,
    progressColumnId,
    props.taskStatuses,
  ).filter((t) => String(t.id) !== String(taskId)).length
  try {
    await updateTask(taskId, { progress_column_id: progressColumnId, order: inTarget })
    emit('tasks-updated')
  } catch (error) {
    console.error('通过下拉更新进度列失败:', error)
  }
}

const handleCreateComment = async (taskId, content) => {
  try {
    await createTaskComment(props.tenantId, taskId, content)
    emit('tasks-updated')
  } catch (error) {
    console.error('创建评论出错:', error)
  }
}

const handleUpdateComment = async (...args) => {
  const { taskId, commentId, content } = resolveCommentUpdateArgs(props.todos, args)
  try {
    await updateTaskComment(props.tenantId, taskId, commentId, content)
    emit('tasks-updated')
  } catch (error) {
    console.error('更新评论出错:', error)
  }
}

/**
 * OPT-20260808-021: 看板结构签名 —— 进度列/类别/过滤栏 + 每个分区每列的任务 id 集合。
 * 仅当结构真正变化（增删任务、跨列移动、跨分区移动、列/类别/过滤栏变更）时才重建 Sortable；
 * 内容更新（评论、标题、运行态、轮询刷新）不再触发全量销毁重建，
 * 避免 15s 机器摘要轮询 / SSE 补丁引发的 CPU 与 GC 抖动。
 */
const boardStructureSignature = computed(() => {
  const statuses = Array.isArray(props.taskStatuses) ? props.taskStatuses : []
  const types = Array.isArray(props.taskTypes) ? props.taskTypes : []
  const bars = Array.isArray(props.deliverableFilterBars) ? props.deliverableFilterBars : []
  const parts = []
  parts.push(`s:${statuses.map((s) => s?.id ?? '').join(',')}`)
  parts.push(`t:${types.map((t) => t?.id ?? '').join(',')}`)
  parts.push(
    `b:${bars
      .map(
        (bar) =>
          `${bar?.id ?? ''}:${(bar?.path ?? [])
            .map((seg) => `${seg?.type ?? ''}:${seg?.id ?? ''}`)
            .join('|')}`,
      )
      .join('~')}`,
  )
  const sectionLists = [
    ...(Array.isArray(partitioned.value.sections) ? partitioned.value.sections : []).map(
      (s) => ({ key: String(s?.barId ?? ''), todos: Array.isArray(s?.todos) ? s.todos : [] }),
    ),
    { key: OTHER_SECTION_KEY, todos: Array.isArray(partitioned.value.other) ? partitioned.value.other : [] },
  ]
  const bySectionColumn = new Map()
  for (const section of sectionLists) {
    for (const todo of section.todos) {
      const col = resolveTodoKanbanColumnId(todo, statuses)
      const key = `${section.key}/${col == null ? '__none__' : String(col)}`
      if (!bySectionColumn.has(key)) bySectionColumn.set(key, [])
      bySectionColumn.get(key).push(String(todo?.id ?? ''))
    }
  }
  const colParts = []
  for (const [key, ids] of bySectionColumn) {
    colParts.push(`${key}:${ids.join(',')}`)
  }
  colParts.sort()
  parts.push(`c:${colParts.join(';')}`)
  return parts.join('|')
})

let dragInitTimeout = null

function scheduleDragInit() {
  if (dragInitTimeout) clearTimeout(dragInitTimeout)
  dragInitTimeout = setTimeout(() => {
    dragInitTimeout = null
    if (typeof document !== 'undefined' && document.visibilityState === 'hidden') return
    initDragDrop()
  }, 100)
}

watch(boardStructureSignature, () => {
  scheduleDragInit()
})

onMounted(() => scheduleDragInit())
</script>

<style scoped src="./TaskPanel.css"></style>
