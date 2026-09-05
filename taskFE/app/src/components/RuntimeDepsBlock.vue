<template>
    <details
      v-if="environments && environments.length > 0"
      class="mt-3 rounded-lg border border-gray-100 bg-gray-50/50 px-3 py-2 open:bg-gray-50"
    >
      <summary class="cursor-pointer list-none text-sm font-medium text-gray-700 marker:hidden [&::-webkit-details-marker]:hidden hover:text-primary">
        服务器镜像依赖 · {{ environments.length }} 项
      </summary>
      <div class="mt-3 space-y-3 border-l-2 border-gray-200 pl-3">
        <div v-for="(env, index) in environments" :key="index" class="space-y-0.5 text-xs text-gray-600">
          <div><span class="font-medium text-gray-700">平台</span> {{ env.platform_type_display || env.platform_type }}</div>
          <div><span class="font-medium text-gray-700">区域</span> {{ env.region || '—' }}</div>
          <div><span class="font-medium text-gray-700">镜像</span> {{ env.image_name || '—' }}</div>
          <div><span class="font-medium text-gray-700">操作系统</span> {{ env.os_type || '—' }}</div>
          <div><span class="font-medium text-gray-700">系统版本</span> {{ env.os_version || '—' }}</div>
          <div><span class="font-medium text-gray-700">硬件要求</span> {{ env.hardware_summary || '—' }}</div>
        </div>
      </div>
    </details>
  
</template>

<script setup>
defineProps({
  environments: {
    type: Array,
    default: () => [],
  },
})
</script>
