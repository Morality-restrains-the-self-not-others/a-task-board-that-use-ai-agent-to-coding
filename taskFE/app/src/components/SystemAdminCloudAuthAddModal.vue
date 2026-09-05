<template>
  <div v-if="visible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">添加云平台服务器镜像</h3>
        <button type="button" class="text-gray-500 hover:text-gray-700 transition-colors" @click="visible = false">
          &times;
        </button>
      </div>

      <form id="add-image-form" @submit.prevent="$emit('submit')">
        <div class="space-y-4">
          <div>
            <label for="platform-type" class="block text-sm font-medium text-gray-700 mb-2">云平台类型</label>
            <select
              id="platform-type"
              v-model="form.platform_type"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              required
              @change="$emit('platform-type-change')"
            >
              <option value="">请选择云平台类型</option>
              <option v-for="platform in cloudPlatforms" :key="platform.value" :value="platform.value">{{ platform.label }}</option>
            </select>
          </div>

          <div>
            <label for="region" class="block text-sm font-medium text-gray-700 mb-2">地域</label>
            <select
              id="region"
              v-model="form.region"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              required
              @change="$emit('region-change')"
            >
              <option value="">请先选择云平台</option>
              <option v-for="region in regions" :key="region.id" :value="region.id">{{ region.name }}</option>
            </select>
          </div>

          <div>
            <label for="cloud-image" class="block text-sm font-medium text-gray-700 mb-2">云平台镜像</label>
            <select
              id="cloud-image"
              v-model="selectedImage"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              required
              @change="$emit('cloud-image-change')"
            >
              <option value="">请先选择地域</option>
              <option v-for="image in images" :key="image.id" :value="image">{{ image.name }} ({{ image.id }})</option>
            </select>
          </div>

          <div>
            <label for="image-name" class="block text-sm font-medium text-gray-700 mb-2">镜像名称</label>
            <input
              id="image-name"
              v-model="form.image_name"
              type="text"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              placeholder="镜像名称"
              required
            >
          </div>

          <div>
            <label for="image-id" class="block text-sm font-medium text-gray-700 mb-2">镜像ID</label>
            <input
              id="image-id"
              v-model="form.image_id"
              type="text"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              placeholder="镜像ID"
              required
            >
          </div>

          <div>
            <label for="os-type" class="block text-sm font-medium text-gray-700 mb-2">操作系统类型</label>
            <input
              id="os-type"
              v-model="form.os_type"
              type="text"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              placeholder="操作系统类型"
            >
          </div>

          <div>
            <label for="os-version" class="block text-sm font-medium text-gray-700 mb-2">操作系统版本</label>
            <input
              id="os-version"
              v-model="form.os_version"
              type="text"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              placeholder="操作系统版本"
            >
          </div>

          <div>
            <label for="image-type" class="block text-sm font-medium text-gray-700 mb-2">镜像类型</label>
            <input
              id="image-type"
              v-model="form.image_type"
              type="text"
              class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
              placeholder="镜像类型"
            >
          </div>

          <div>
            <label for="is-active" class="block text-sm font-medium text-gray-700 mb-2">是否启用</label>
            <input
              id="is-active"
              v-model="form.is_active"
              type="checkbox"
              class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded transition-all duration-300"
              checked
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
            {{ submitting ? '添加中...' : '添加服务器镜像' }}
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
  submitting: { type: Boolean, default: false }
})

const visible = defineModel('visible', { type: Boolean, default: false })
const selectedImage = defineModel('selectedImage', { default: null })

defineEmits(['submit', 'platform-type-change', 'region-change', 'cloud-image-change'])
</script>
