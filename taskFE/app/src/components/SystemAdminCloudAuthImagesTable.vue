<template>
  <div v-if="loading" class="flex justify-center items-center py-20">
    <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary"></div>
    <span class="ml-3 text-gray-600">加载云平台服务器镜像列表中...</span>
  </div>

  <div v-else class="overflow-x-auto">
    <table class="min-w-full divide-y divide-gray-200">
      <thead class="bg-gray-50">
        <tr>
          <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
          <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">云平台</th>
          <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">镜像名称</th>
          <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">镜像ID</th>
          <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">地域</th>
          <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">操作系统</th>
          <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">是否启用</th>
          <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">所属公司</th>
          <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">创建时间</th>
          <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
        </tr>
      </thead>
      <tbody class="bg-white divide-y divide-gray-200">
        <tr v-if="images.length === 0">
          <td colspan="10" class="px-6 py-12 text-center">
            <div class="flex flex-col items-center justify-center">
              <svg class="w-16 h-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"></path>
              </svg>
              <h3 class="text-lg font-medium text-gray-900 mb-1">暂无云平台服务器镜像</h3>
              <p class="text-gray-500 mb-6">系统中还没有任何云平台服务器镜像，请点击"添加云平台服务器镜像"按钮创建第一个镜像。</p>
              <button
                class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
                @click="$emit('add')"
              >
                添加云平台服务器镜像
              </button>
            </div>
          </td>
        </tr>
        <tr v-for="image in images" :key="image.id">
          <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ image.id }}</td>
          <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ getPlatformLabel(image.platform_type) }}</td>
          <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ image.image_name }}</td>
          <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ image.image_id }}</td>
          <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ image.region }}</td>
          <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
            {{ image.os_type ? `${image.os_type} ${image.os_version || ''}` : '-' }}
          </td>
          <td class="px-6 py-4 whitespace-nowrap">
            <span
              class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full"
              :class="image.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'"
            >
              {{ image.is_active ? '是' : '否' }}
            </span>
          </td>
          <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
            {{ image.company ? image.company.name : '-' }}
          </td>
          <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ formatDate(image.created_at) }}</td>
          <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
            <button class="text-primary hover:text-primary/90 mr-3" @click="$emit('edit', image)">编辑</button>
            <button class="text-red-600 hover:text-red-800" @click="$emit('delete', image)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
defineProps({
  loading: { type: Boolean, default: false },
  images: { type: Array, default: () => [] },
  getPlatformLabel: { type: Function, required: true },
  formatDate: { type: Function, required: true }
})

defineEmits(['add', 'edit', 'delete'])
</script>
