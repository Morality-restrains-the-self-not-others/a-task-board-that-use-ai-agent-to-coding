<template>
  <!-- 添加 / 编辑面板模态：从 WorkspaceSettingsTaskPanel.vue 抽出，保持原 DOM 与行为 -->
  <div v-if="show" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white rounded-xl p-6 w-full max-w-md">
      <h4 class="text-lg font-bold mb-4">{{ mode === 'edit' ? '编辑面板' : '添加新面板' }}</h4>
      <div class="space-y-4">
        <div>
          <label :for="nameInputId" class="block text-sm font-medium text-gray-700 mb-1">面板名称</label>
          <input
            :id="nameInputId"
            type="text"
            v-model="draft.name"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            placeholder="例如：待处理任务"
          >
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">面板颜色</label>
          <div class="grid grid-cols-6 gap-2">
            <button
              v-for="color in availableColors"
              :key="color"
              class="w-8 h-8 rounded-full border-2"
              :class="draft.color === color ? 'border-black' : 'border-transparent'"
              :style="{ backgroundColor: color }"
              @click="draft.color = color"
            ></button>
          </div>
        </div>
        <div class="flex justify-end space-x-3 pt-4">
          <button
            class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
            @click="$emit('cancel')"
          >
            取消
          </button>
          <button
            class="btn-primary"
            @click="$emit('save', { ...draft })"
            :disabled="!draft.name || !draft.color"
          >
            保存
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  show: { type: Boolean, default: false },
  mode: { type: String, default: 'add' }, // 'add' | 'edit'
  panel: { type: Object, default: () => ({ id: '', name: '', color: '' }) },
  availableColors: { type: Array, default: () => [] },
})

defineEmits(['save', 'cancel'])

const draft = ref({ id: '', name: '', color: '' })

watch(
  () => [props.show, props.mode, props.panel],
  () => {
    draft.value = {
      id: String(props.panel?.id || ''),
      name: String(props.panel?.name || ''),
      color: String(props.panel?.color || ''),
    }
  },
  { immediate: true },
)

const nameInputId = props.mode === 'edit' ? 'edit-panel-name' : 'panel-name'
</script>
