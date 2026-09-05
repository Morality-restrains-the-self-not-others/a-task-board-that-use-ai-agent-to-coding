<template>
  <div class="mt-4">
    <div class="flex items-center space-x-2 flex-wrap">
      <button type="button" id="start-server-btn" @click="panel.startServer" class="px-3 py-1.5 border border-transparent rounded-md text-xs font-medium text-white bg-primary hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary disabled:bg-gray-400 disabled:cursor-not-allowed disabled:hover:bg-gray-400" :disabled="panel.isStartServerDisabled" :title="panel.startServerDisabledReason">
        {{ panel.startServerButtonLabel }}
      </button>
      <button type="button" id="stop-server-btn" @click="emitStop" class="px-3 py-1.5 border border-transparent rounded-md text-xs font-medium text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500" :class="{ hidden: !(panel.isServerRunning || panel.isServerRuntimeRunning) }">
        停止服务器
      </button>
      <template v-if="panel.serverJumpUrl">
        <a
          id="jump-to-server-btn"
          :href="panel.serverJumpUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="px-3 py-1.5 border border-transparent rounded-md text-xs font-medium text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500"
        >
          跳转到服务器
        </a>
        <span class="text-xs text-gray-500 leading-snug max-w-md">
          默认端口 {{ panel.serverJumpDefaultPort }}；请根据需要自行调整域名或 IP 后的端口号。
        </span>
      </template>
    </div>
    <p
      v-if="panel.isStartServerDisabled && panel.startServerDisabledReason"
      class="mt-2 text-xs text-amber-700"
      data-testid="start-server-disabled-reason"
      role="status"
    >
      {{ panel.startServerDisabledReason }}
    </p>
    <p
      v-if="!panel.isStartServerDisabled && panel.activeHardwareConfigSourceLabel"
      class="mt-2 text-xs text-gray-500"
      data-testid="start-server-config-source"
    >
      {{ panel.activeHardwareConfigSourceLabel }}
    </p>
  </div>
</template>

<script setup>
defineProps({
  panel: { type: Object, required: true },
})
const emit = defineEmits(['stop-server'])
function emitStop() {
  emit('stop-server')
}
</script>
