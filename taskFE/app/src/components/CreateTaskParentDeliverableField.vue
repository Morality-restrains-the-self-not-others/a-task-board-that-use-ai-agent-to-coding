<template>
  <div
    v-if="showField"
    data-testid="create-task-parent-deliverable"
  >
    <label for="task-parent-deliverable" class="block text-sm font-medium text-gray-700">上层交付物</label>
    <select
      id="task-parent-deliverable"
      :value="modelValue"
      class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
      required
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option value="">请选择上层交付物</option>
      <option
        v-for="parent in candidates"
        :key="parent.id"
        :value="parent.id"
      >
        {{ parentOptionLabel(parent) }}
      </option>
    </select>
    <p v-if="candidates.length === 0" class="mt-1 text-xs text-amber-700">
      暂无上一层交付物任务，请先创建顶层/上一层交付物
    </p>
  </div>
</template>

<script setup>
import { computed, watch } from 'vue'
import {
  isTopLevelDeliverableCategory,
  listParentDeliverableCandidates,
} from '../utils/workPanelDeliverableAggregation.js'
import { deliverableContentOptionLabel } from '../utils/workPanelDeliverableContent.js'

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  taskTypes: { type: Array, default: () => [] },
  todos: { type: Array, default: () => [] },
  selectedCategoryId: { type: [String, Number], default: '' },
  /** 编辑当前任务时排除自身，避免选为自己的上层 */
  excludeTaskId: { type: [String, Number], default: '' },
})
const emit = defineEmits(['update:modelValue'])

function parentOptionLabel(parent) {
  return deliverableContentOptionLabel(parent) || parent?.title || ''
}

const showField = computed(() => !isTopLevelDeliverableCategory(props.taskTypes, props.selectedCategoryId))
const candidates = computed(() => {
  const list = listParentDeliverableCandidates(props.todos, props.taskTypes, props.selectedCategoryId)
  const exclude = props.excludeTaskId != null && props.excludeTaskId !== '' ? String(props.excludeTaskId) : ''
  if (!exclude) return list
  return list.filter((c) => c.id !== exclude)
})

watch(
  () => String(props.selectedCategoryId ?? ''),
  (nextId, prevId) => {
    if (nextId === String(prevId ?? '')) return
    if (isTopLevelDeliverableCategory(props.taskTypes, nextId)) {
      if (props.modelValue) emit('update:modelValue', '')
      return
    }
    const parent = props.modelValue != null && props.modelValue !== '' ? String(props.modelValue) : ''
    if (parent && !candidates.value.some((c) => c.id === parent)) {
      emit('update:modelValue', '')
    }
  },
)
</script>
