<template>
  <div>
    <div v-if="isFieldEnabled('owner')">
      <label for="task-owner-input" class="block text-sm font-medium text-gray-700">负责人（单人）</label>
      <p class="mt-0.5 text-xs text-gray-500">当前工作空间可访问成员；可在输入框中输入关键字快速筛选</p>
      <div class="relative mt-1">
        <input
          id="task-owner-input"
          type="text"
          class="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
          :placeholder="ownerInputPlaceholder"
          :value="showOwnerPicker ? ownerFilterQuery : ownerDisplayLabel"
          autocomplete="off"
          @input="onOwnerSearchInput"
          @focus="onOwnerInputFocus"
        >
        <div v-if="isCollaboratorsLoading" class="absolute right-3 top-1/2 transform -translate-y-1/2 pointer-events-none">
          <svg class="animate-spin h-4 w-4 text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
        </div>
        <div
          v-if="showOwnerPicker && !isCollaboratorsLoading"
          class="absolute z-20 mt-1 w-full bg-white shadow-lg rounded-md border border-gray-300 max-h-56 overflow-auto"
          @mousedown.prevent
        >
          <button
            type="button"
            class="w-full text-left px-3 py-2 text-sm hover:bg-gray-100 text-gray-600 border-b border-gray-100"
            @click="selectOwnerMember(null)"
          >
            未指派
          </button>
          <div
            v-if="collaboratorsError"
            class="px-3 py-2 text-sm text-red-600"
            :data-traceId="collaboratorsErrorTraceId || undefined"
          >{{ collaboratorsError }}</div>
          <template v-else-if="filteredOwnersForPicker.length > 0">
            <button
              v-for="person in filteredOwnersForPicker"
              :key="`owner-opt-${person.id}`"
              type="button"
              class="w-full text-left px-3 py-2 text-sm hover:bg-gray-100 text-gray-800"
              @click="selectOwnerMember(person)"
            >
              {{ getCollaboratorDisplayName(person) }}
            </button>
          </template>
          <div v-else class="px-3 py-2 text-sm text-gray-500">
            {{ localCollaborators.length === 0 ? '暂无可选成员，请确认工作空间已加载' : '无匹配成员，请调整关键字' }}
          </div>
        </div>
      </div>
      <div
        v-if="collaboratorsError && !showOwnerPicker"
        class="mt-1 text-sm text-red-600"
        :data-traceId="collaboratorsErrorTraceId || undefined"
      >
        {{ collaboratorsError }}
      </div>
    </div>
    <div v-if="isFieldEnabled('operator')" class="mt-4">
      <label for="task-operator-input" class="block text-sm font-medium text-gray-700">操作员（单人）</label>
      <p class="mt-0.5 text-xs text-gray-500">与负责人相同成员池；仅可指派一人</p>
      <div class="relative mt-1">
        <input
          id="task-operator-input"
          type="text"
          class="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary"
          :placeholder="ownerInputPlaceholder"
          :value="showOperatorPicker ? operatorFilterQuery : operatorDisplayLabel"
          autocomplete="off"
          @input="onOperatorSearchInput"
          @focus="onOperatorInputFocus"
        >
        <div
          v-if="showOperatorPicker && !isCollaboratorsLoading"
          class="absolute z-20 mt-1 w-full bg-white shadow-lg rounded-md border border-gray-300 max-h-56 overflow-auto"
          @mousedown.prevent
        >
          <button
            type="button"
            class="w-full text-left px-3 py-2 text-sm hover:bg-gray-100 text-gray-600 border-b border-gray-100"
            @click="selectOperatorMember(null)"
          >
            未指派
          </button>
          <template v-if="filteredOperatorsForPicker.length > 0">
            <button
              v-for="person in filteredOperatorsForPicker"
              :key="`operator-opt-${person.id}`"
              type="button"
              class="w-full text-left px-3 py-2 text-sm hover:bg-gray-100 text-gray-800"
              @click="selectOperatorMember(person)"
            >
              {{ getCollaboratorDisplayName(person) }}
            </button>
          </template>
          <div v-else class="px-3 py-2 text-sm text-gray-500">
            {{ localCollaborators.length === 0 ? '暂无可选成员，请确认工作空间已加载' : '无匹配成员，请调整关键字' }}
          </div>
        </div>
      </div>
    </div>
    <div v-if="isFieldEnabled('assignees')" class="mt-4">
      <label for="task-assignees-filter" class="block text-sm font-medium text-gray-700">协作者（可多选）</label>
      <p class="mt-0.5 text-xs text-gray-500">与负责人相同成员池；点击「+」后在列表中输入关键字筛选并添加</p>
      <div class="relative">
        <div id="task-assignees" class="mt-1 min-h-[44px] w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm bg-white flex items-start justify-between gap-2">
          <div class="flex-1 flex flex-wrap gap-2">
            <span v-if="selectedAssigneeTags.length === 0" class="text-sm text-gray-400">未选择协作者</span>
            <span
              v-for="tag in selectedAssigneeTags"
              :key="`selected-assignee-${tag.id}`"
              class="inline-flex items-center gap-1 px-2 py-1 rounded bg-blue-50 text-blue-700 text-xs"
            >
              {{ tag.name }}
              <button
                type="button"
                class="text-blue-500 hover:text-blue-700"
                @click.stop="removeAssignee(tag.id)"
                :aria-label="`移除${tag.name}`"
              >
                ×
              </button>
            </span>
          </div>
          <button
            type="button"
            class="w-7 h-7 rounded-full border border-gray-300 text-gray-600 hover:bg-gray-50 flex items-center justify-center flex-shrink-0"
            @click.stop="toggleAssigneePicker"
            aria-label="添加协作者"
          >
            +
          </button>
        </div>
        <div
          v-if="showAssigneePicker"
          class="absolute z-20 mt-1 w-full bg-white shadow-lg rounded-md border border-gray-300 max-h-56 flex flex-col overflow-hidden"
          @mousedown.prevent
        >
          <p class="px-3 pt-2 text-xs text-gray-500">输入关键字筛选可添加的成员</p>
          <input
            id="task-assignees-filter"
            type="text"
            class="mx-3 mt-1 mb-2 px-2 py-1.5 border border-gray-300 rounded text-sm focus:outline-none focus:ring-primary focus:border-primary"
            placeholder="输入关键字筛选…"
            v-model="assigneeFilterQuery"
            autocomplete="off"
            @click.stop
          >
          <div class="overflow-auto max-h-44 border-t border-gray-100">
            <div v-if="isCollaboratorsLoading" class="px-3 py-2 text-gray-500">加载中...</div>
            <div
              v-else-if="collaboratorsError"
              class="px-3 py-2 text-red-500"
              :data-traceId="collaboratorsErrorTraceId || undefined"
            >{{ collaboratorsError }}</div>
            <ul v-else-if="filteredAvailableCollaborators.length > 0" class="py-0.5">
              <li
                v-for="person in filteredAvailableCollaborators"
                :key="`assignee-option-${person.id}`"
                class="px-3 py-2 hover:bg-gray-100 cursor-pointer flex items-center justify-between gap-2"
                @click.stop="addAssignee(person.id)"
              >
                <span class="text-sm text-gray-700">{{ getCollaboratorDisplayName(person) }}</span>
                <span class="text-xs text-blue-600 flex-shrink-0">添加</span>
              </li>
            </ul>
            <div v-else class="px-3 py-2 text-gray-500">
              {{ availableCollaborators.length === 0 ? '无可添加人员' : '无匹配成员，请调整关键字' }}
            </div>
          </div>
        </div>
        <div v-if="isCollaboratorsLoading" class="absolute right-12 top-4">
          <svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
        </div>
      </div>
      <div
        v-if="collaboratorsError"
        class="mt-1 text-sm text-red-600"
        :data-traceId="collaboratorsErrorTraceId || undefined"
      >
        {{ collaboratorsError }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { useCreateTaskCollaborators } from '../composables/useCreateTaskCollaborators.js'
import { defaultCreateTaskFieldSettings, isCreateTaskFieldEnabled } from '../utils/createTaskFieldSettings.js'

const props = defineProps({
  editingTask: {
    type: Object,
    required: true,
  },
  tenantId: {
    type: [String, Number],
    default: null,
  },
  currentWorkspace: {
    type: Object,
    default: null,
  },
  show: {
    type: Boolean,
    default: false,
  },
  fieldSettings: {
    type: Object,
    default: () => defaultCreateTaskFieldSettings(),
  },
})

const isFieldEnabled = (key) => isCreateTaskFieldEnabled(props.fieldSettings, key)

const {
  showAssigneePicker,
  showOwnerPicker,
  showOperatorPicker,
  ownerFilterQuery,
  operatorFilterQuery,
  assigneeFilterQuery,
  localCollaborators,
  isCollaboratorsLoading,
  collaboratorsError,
  collaboratorsErrorTraceId,
  ownerInputPlaceholder,
  ownerDisplayLabel,
  operatorDisplayLabel,
  filteredOwnersForPicker,
  filteredOperatorsForPicker,
  selectedAssigneeTags,
  filteredAvailableCollaborators,
  availableCollaborators,
  getCollaboratorDisplayName,
  onOwnerInputFocus,
  onOwnerSearchInput,
  selectOwnerMember,
  onOperatorInputFocus,
  onOperatorSearchInput,
  selectOperatorMember,
  addAssignee,
  removeAssignee,
  toggleAssigneePicker,
  closePickers,
} = useCreateTaskCollaborators({
  editingTask: () => props.editingTask,
  tenantId: () => props.tenantId,
  currentWorkspace: () => props.currentWorkspace,
  show: () => props.show,
})

defineExpose({ closePickers })
</script>
