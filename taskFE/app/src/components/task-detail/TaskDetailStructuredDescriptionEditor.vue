<template>
  <div data-testid="task-detail-edit-description-structured">
    <div v-if="mode === 'raw'" class="space-y-2">
      <textarea
        :value="modelValue"
        @input="$emit('update:modelValue', $event.target.value)"
        rows="4"
        class="w-full text-gray-700 border border-gray-300 rounded-md px-3 py-2 resize-y focus:outline-none focus:ring-primary focus:border-primary"
        placeholder="请输入任务描述"
      />
      <button type="button" class="text-xs px-2 py-0.5 rounded border border-gray-300 bg-white text-gray-600 hover:bg-gray-50" @click="$emit('switch-to-structured')">切换到结构化编辑</button>
    </div>
    <div v-else class="space-y-3">
      <div v-for="field in fields" :key="field.key">
        <label :for="'task-structured-' + field.key" class="block text-sm font-medium text-gray-600">{{ field.heading }}</label>
        <textarea :id="'task-structured-' + field.key" :value="fieldValues[field.key]" @input="onFieldChange(field.key, $event.target.value)" rows="2" class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary" :placeholder="'可选：' + field.heading" :data-testid="'task-detail-edit-field-' + field.key" />
      </div>
      <div>
        <label for="task-structured-forbidNewDeps" class="block text-sm font-medium text-gray-600">是否禁止新增依赖</label>
        <select id="task-structured-forbidNewDeps" :value="fieldValues.forbidNewDeps" @change="onFieldChange('forbidNewDeps', $event.target.value)" class="mt-1 block w-full max-w-xs px-3 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-primary focus:border-primary" data-testid="task-detail-edit-field-forbidNewDeps">
          <option value="">未指定</option>
          <option value="yes">是</option>
          <option value="no">否</option>
        </select>
      </div>
      <button type="button" class="text-xs px-2 py-0.5 rounded border border-gray-300 bg-white text-gray-600 hover:bg-gray-50" @click="$emit('switch-to-raw')">切换到纯文本编辑</button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { CREATE_TASK_STRUCTURED_FIELDS } from '../../utils/createTaskDescriptionCompose.js'

const props = defineProps({
  modelValue: { type: String, default: '' },
  mode: { type: String, default: 'structured' },
  editingTask: { type: Object, default: null },
})

defineEmits(['update:modelValue', 'switch-to-structured', 'switch-to-raw'])

const fields = CREATE_TASK_STRUCTURED_FIELDS.filter((f) => f.kind !== 'deps')

const fieldValues = computed(() => {
  const task = props.editingTask
  if (!task || typeof task !== 'object') return {}
  const vals = {}
  for (const f of CREATE_TASK_STRUCTURED_FIELDS) {
    vals[f.key] = task[f.key] ?? ''
  }
  return vals
})

function onFieldChange(key, value) {
  if (props.editingTask && typeof props.editingTask === 'object') {
    props.editingTask[key] = value
  }
}
</script>
