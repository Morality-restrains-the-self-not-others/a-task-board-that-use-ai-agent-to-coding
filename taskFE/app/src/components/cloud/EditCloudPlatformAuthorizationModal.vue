<template>
  <div v-if="visible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">编辑云平台授权</h3>
        <button
          type="button"
          class="text-gray-500 hover:text-gray-700 transition-colors"
          @click="emit('close')"
        >
          &times;
        </button>
      </div>

      <form @submit.prevent="emit('submit')">
        <div class="space-y-4">
          <input v-model="form.id" type="hidden" />

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">云平台类型</label>
            <select
              v-model="form.platform_type"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              required
            >
              <option
                v-for="platform in cloudPlatforms"
                :key="platform.value"
                :value="platform.value"
                :disabled="platform.value !== 'aliyun'"
              >
                {{ platform.label }}
              </option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">Access Key</label>
            <input
              v-model="form.access_key"
              type="text"
              autocomplete="off"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300 font-mono text-sm"
              placeholder="Access Key"
              required
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">Secret Key</label>
            <input
              v-model="form.secret_key"
              type="text"
              autocomplete="off"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300 font-mono text-sm"
              placeholder="Secret Key"
              required
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">备注</label>
            <input
              v-model="form.remark"
              type="text"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              placeholder="备注"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">是否启用</label>
            <input
              v-model="form.is_active"
              type="checkbox"
              class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded transition-all duration-300"
            />
          </div>
        </div>

        <div class="flex justify-end space-x-3 mt-6">
          <button
            type="button"
            class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300"
            @click="emit('close')"
          >
            取消
          </button>
          <button
            type="submit"
            :disabled="editingAuthorization"
            class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300"
          >
            {{ editingAuthorization ? '保存中...' : '保存修改' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
/**
 * 编辑云平台授权弹窗（从 WorkspaceSettingsCloudPlatform 拆分）。
 */
defineProps({
  visible: { type: Boolean, default: false },
  form: { type: Object, required: true },
  cloudPlatforms: { type: Array, required: true },
  editingAuthorization: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'submit'])
</script>
