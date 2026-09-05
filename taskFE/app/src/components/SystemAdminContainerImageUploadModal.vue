<template>
  <div v-if="visible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">上传容器镜像</h3>
        <button class="text-gray-500 hover:text-gray-700" @click="visible = false">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </button>
      </div>

      <div id="manual-upload-form">
        <form @submit.prevent="$emit('submit')">
          <div class="space-y-4">
            <div>
              <label for="image-name" class="block text-sm font-medium text-gray-700 mb-2">镜像名称</label>
              <input
                id="image-name"
                v-model="form.name"
                type="text"
                class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                placeholder="输入镜像名称"
                required
              >
            </div>

            <div>
              <label for="image-description" class="block text-sm font-medium text-gray-700 mb-2">镜像描述</label>
              <textarea
                id="image-description"
                v-model="form.description"
                class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                placeholder="输入镜像描述"
                rows="3"
              ></textarea>
            </div>

            <div>
              <label for="image-url" class="block text-sm font-medium text-gray-700 mb-2">镜像地址</label>
              <input
                id="image-url"
                v-model="form.image_url"
                type="text"
                class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                placeholder="如：docker.io/library/nginx"
                required
              >
            </div>

            <div>
              <label for="image-size" class="block text-sm font-medium text-gray-700 mb-2">镜像大小 (字节)</label>
              <input
                id="image-size"
                v-model="form.size"
                type="number"
                class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                placeholder="输入镜像大小，可选"
              >
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
              {{ submitting ? '上传中...' : '上传镜像' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  form: { type: Object, required: true },
  submitting: { type: Boolean, default: false }
})

const visible = defineModel('visible', { type: Boolean, default: false })
defineEmits(['submit'])
</script>
