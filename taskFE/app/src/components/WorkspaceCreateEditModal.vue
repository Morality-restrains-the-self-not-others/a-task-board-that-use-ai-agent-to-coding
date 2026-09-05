<template>
  <div v-if="show" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white rounded-xl p-6 w-full max-w-md">
      <h4 class="text-lg font-bold mb-4">{{ editing ? '编辑工作空间' : '添加新工作空间' }}</h4>
      <div class="space-y-4">
        <div>
          <label for="workspace-name" class="block text-sm font-medium text-gray-700 mb-1">工作空间名称</label>
          <input
            id="workspace-name"
            v-model="form.name"
            type="text"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            placeholder="例如：产品开发"
          >
        </div>
        <div>
          <label for="workspace-description" class="block text-sm font-medium text-gray-700 mb-1">工作空间描述</label>
          <input
            id="workspace-description"
            v-model="form.description"
            type="text"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            placeholder="例如：产品开发团队的工作空间"
          >
        </div>
        <div class="flex items-center">
          <input
            id="workspace-is-default"
            v-model="form.is_default"
            type="checkbox"
            class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded"
            data-testid="workspace-is-default"
          >
          <label for="workspace-is-default" class="ml-2 block text-sm text-gray-700">是否设为默认</label>
        </div>
        <div class="flex justify-end space-x-3 pt-4">
          <button
            class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
            type="button"
            @click="$emit('cancel')"
          >
            取消
          </button>
          <button
            class="btn-primary"
            type="button"
            :disabled="!form.name || saving"
            :aria-busy="saving ? 'true' : undefined"
            @click="$emit('save')"
          >
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  show: { type: Boolean, default: false },
  editing: { type: Boolean, default: false },
  form: { type: Object, required: true },
  saving: { type: Boolean, default: false },
})
defineEmits(['save', 'cancel'])
</script>
