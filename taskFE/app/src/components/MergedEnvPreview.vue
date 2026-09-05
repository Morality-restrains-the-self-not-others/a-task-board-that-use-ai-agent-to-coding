<template>
  <div class="bg-white p-6 rounded-xl shadow">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-xl font-bold text-text">环境变量完整预览</h3>
      <span class="text-xs text-text-light">容器启动时将收到以下所有变量</span>
    </div>
    <div class="border border-gray-200 rounded-lg overflow-hidden">
      <!-- System vars section -->
      <div class="bg-gray-50 px-3 py-2 border-b border-gray-200">
        <span class="text-xs font-medium text-gray-500">系统自动生成</span>
      </div>
      <div class="px-3 py-2 font-mono text-xs text-gray-600 max-h-48 overflow-y-auto">
        <div v-for="(value, key) in systemEnv" :key="key" class="py-0.5">{{ key }}={{ value }}</div>
      </div>
      <!-- Divider -->
      <div class="border-t-2 border-dashed border-gray-300"></div>
      <!-- User vars section -->
      <div class="bg-blue-50 px-3 py-2 border-b border-gray-200">
        <span class="text-xs font-medium text-blue-600">用户自定义 ({{ userCount }} 个)</span>
      </div>
      <div class="px-3 py-2 font-mono text-xs text-blue-700 max-h-48 overflow-y-auto">
        <div v-if="userCount === 0" class="text-gray-400 italic">暂无自定义变量</div>
        <div v-for="entry in userEnvVars" :key="entry.key" class="py-0.5">{{ entry.key }}={{ entry.value || '' }}</div>
      </div>
    </div>
    <button class="mt-3 px-4 py-2 border border-gray-300 rounded-md text-sm text-gray-600 hover:bg-gray-50" @click="copyAll">📋 复制全部到剪贴板</button>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  systemEnv: { type: Object, default: () => ({}) },
  userEnvVars: { type: Array, default: () => [] },
})

const userCount = computed(() => (props.userEnvVars || []).filter(e => e.key).length)

function copyAll() {
  const lines = []
  for (const [k, v] of Object.entries(props.systemEnv || {})) {
    lines.push(`${k}=${v}`)
  }
  for (const e of (props.userEnvVars || [])) {
    if (e.key) lines.push(`${e.key}=${e.value || ''}`)
  }
  navigator.clipboard.writeText(lines.join('\n') + '\n')
}
</script>
