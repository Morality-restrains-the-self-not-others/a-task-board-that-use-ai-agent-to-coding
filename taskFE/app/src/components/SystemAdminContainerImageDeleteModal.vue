<template>
  <div v-if="visible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">删除容器镜像</h3>
        <button class="text-gray-500 hover:text-gray-700" @click="visible = false">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </button>
      </div>

      <p class="text-gray-700 mb-6">您确定要删除这个容器镜像吗？此操作不可恢复。</p>

      <form id="delete-image-form" @submit.prevent="$emit('submit')">
        <div class="flex justify-end space-x-3">
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
            class="px-4 py-3 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-all duration-300"
          >
            {{ submitting ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
defineProps({
  submitting: { type: Boolean, default: false }
})

const visible = defineModel('visible', { type: Boolean, default: false })
defineEmits(['submit'])
</script>
