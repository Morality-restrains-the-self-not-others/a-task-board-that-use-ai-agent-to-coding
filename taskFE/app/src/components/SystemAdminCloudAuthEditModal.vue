<template>
  <div v-if="visible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">编辑云平台服务器镜像</h3>
        <button type="button" class="text-gray-500 hover:text-gray-700 transition-colors" @click="visible = false">
          &times;
        </button>
      </div>

      <form id="edit-image-form" @submit.prevent="$emit('submit')">
        <div v-if="loading" class="flex justify-center items-center py-8">
          <div class="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary"></div>
          <span class="ml-2 text-gray-600">加载地域与镜像列表...</span>
        </div>
        <div v-else class="space-y-4">
          <input id="edit-image-id" v-model="form.id" type="hidden">

          <div>
            <label for="edit-platform-type" class="block text-sm font-medium text-gray-700 mb-2">云平台类型</label>
            <select
              id="edit-platform-type"
              v-model="form.platform_type"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              required
              @change="$emit('platform-type-change')"
            >
              <option v-for="platform in cloudPlatforms" :key="platform.value" :value="platform.value">{{ platform.label }}</option>
            </select>
          </div>

          <div>
            <label for="edit-region" class="block text-sm font-medium text-gray-700 mb-2">地域</label>
            <select
              id="edit-region"
              v-model="form.region"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              required
              @change="$emit('region-change')"
            >
              <option value="">请先选择云平台</option>
              <option v-for="region in regions" :key="region.id" :value="region.id">{{ region.name }}</option>
              <option v-if="form.region && !regions.some(r => r.id === form.region)" :value="form.region">
                当前: {{ form.region }}
              </option>
            </select>
          </div>

          <div>
            <label for="edit-cloud-image" class="block text-sm font-medium text-gray-700 mb-2">云平台镜像</label>
            <select
              id="edit-cloud-image"
              v-model="selectedImage"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              @change="$emit('cloud-image-change')"
            >
              <option :value="null">请先选择地域（可选，不选则手动填写下方字段）</option>
              <option
                v-for="(img, idx) in images"
                :key="(img.id || img.image_id) || `img-${idx}`"
                :value="img"
              >
                {{ (img.name || img.image_name) }} ({{ img.id || img.image_id }})
              </option>
            </select>
          </div>

          <div>
            <label for="edit-image-name" class="block text-sm font-medium text-gray-700 mb-2">镜像名称</label>
            <input
              id="edit-image-name"
              v-model="form.image_name"
              type="text"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              required
            >
          </div>

          <div>
            <label for="edit-image-id-input" class="block text-sm font-medium text-gray-700 mb-2">镜像ID</label>
            <input
              id="edit-image-id-input"
              v-model="form.image_id"
              type="text"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              required
            >
          </div>

          <div>
            <label for="edit-os-type" class="block text-sm font-medium text-gray-700 mb-2">操作系统类型</label>
            <input
              id="edit-os-type"
              v-model="form.os_type"
              type="text"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
            >
          </div>

          <div>
            <label for="edit-os-version" class="block text-sm font-medium text-gray-700 mb-2">操作系统版本</label>
            <input
              id="edit-os-version"
              v-model="form.os_version"
              type="text"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
            >
          </div>

          <div>
            <label for="edit-image-type" class="block text-sm font-medium text-gray-700 mb-2">镜像类型</label>
            <input
              id="edit-image-type"
              v-model="form.image_type"
              type="text"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
            >
          </div>

          <div>
            <label for="edit-is-active" class="block text-sm font-medium text-gray-700 mb-2">是否启用</label>
            <input
              id="edit-is-active"
              v-model="form.is_active"
              type="checkbox"
              class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded transition-all duration-300"
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
            :disabled="submitting || loading"
            class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300"
          >
            {{ submitting ? '保存中...' : '保存修改' }}
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
  regions: { type: Array, default: () => [] },
  images: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  submitting: { type: Boolean, default: false }
})

const visible = defineModel('visible', { type: Boolean, default: false })
const selectedImage = defineModel('selectedImage', { default: null })

defineEmits(['submit', 'platform-type-change', 'region-change', 'cloud-image-change'])
</script>
