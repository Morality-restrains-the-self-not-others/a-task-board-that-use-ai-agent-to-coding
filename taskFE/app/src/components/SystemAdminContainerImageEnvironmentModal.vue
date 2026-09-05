<template>
  <div v-if="visible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-2xl p-6">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">设置容器镜像运行环境</h3>
        <button class="text-gray-500 hover:text-gray-700" @click="visible = false">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </button>
      </div>

      <form id="set-environment-form" @submit.prevent="$emit('submit')">
        <div v-if="loading" class="flex justify-center items-center py-10">
          <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary"></div>
        </div>
        <div v-else class="space-y-6">
          <div id="environment-settings" class="space-y-4">
            <div v-for="platform in cloudPlatforms" :key="platform.value" class="bg-gray-50 p-4 rounded-lg">
              <h4 class="text-md font-medium mb-3">{{ platform.label }}运行环境</h4>
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label :for="`${platform.value}-server-image`" class="block text-sm font-medium text-gray-700 mb-2">
                    选择服务器镜像
                  </label>
                  <select
                    :id="`${platform.value}-server-image`"
                    v-model="form[platform.value]"
                    class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                  >
                    <option value="">无关联镜像</option>
                    <option
                      v-for="image in serverImages.filter(img => img.platform_type === platform.value)"
                      :key="image.id"
                      :value="image.id"
                    >
                      {{ image.image_name }} ({{ image.region }})
                    </option>
                    <option
                      v-if="serverImages.filter(img => img.platform_type === platform.value).length === 0"
                      value=""
                      disabled
                    >
                      暂无可用镜像
                    </option>
                  </select>
                  <p
                    v-if="serverImages.filter(img => img.platform_type === platform.value).length === 0"
                    class="text-xs text-gray-500 mt-1"
                  >
                    请先在"云平台服务器镜像设置"页面添加{{ platform.label }}服务器镜像
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="flex justify-end space-x-3 mt-6">
          <button
            type="button"
            class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300"
            @click="visible = false"
          >
            取消
          </button>
          <button
            type="submit"
            :disabled="submitting"
            class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300"
          >
            {{ submitting ? '保存中...' : '保存设置' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
defineProps({
  form: { type: Object, required: true },
  cloudPlatforms: { type: Array, default: () => [] },
  serverImages: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  submitting: { type: Boolean, default: false }
})

const visible = defineModel('visible', { type: Boolean, default: false })
defineEmits(['submit'])
</script>
