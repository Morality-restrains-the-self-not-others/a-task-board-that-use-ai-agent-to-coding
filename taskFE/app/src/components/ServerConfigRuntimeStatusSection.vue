<template>
  <div
    class="p-6 bg-white border border-gray-200 rounded-lg"
    data-testid="server-runtime-redirect-hint"
    :data-comment-id="commentId"
  >
    <div class="flex justify-between items-center mb-2 flex-wrap gap-2">
      <h3 class="text-sm font-medium text-gray-500">服务器运行状态</h3>
      <div class="flex items-center gap-2 flex-wrap">
        <button
          v-if="showRuntimeActionButtons"
          type="button"
          class="px-3 py-1 text-xs border border-gray-300 rounded-md hover:bg-gray-50 text-gray-700"
          @click="openWorkbenchLink"
          :disabled="isWorkbenchLinkLoading"
        >
          {{ isWorkbenchLinkLoading ? '打开中...' : 'Workbench 访问实例' }}
        </button>
        <a
          v-if="effectiveContainerVscodeUrl"
          id="open-server-runtime-vscode-btn"
          :href="effectiveContainerVscodeUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="px-3 py-1 text-xs border border-gray-300 rounded-md hover:bg-gray-50 text-gray-700 inline-flex items-center gap-1"
        >
          <svg class="w-3 h-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
          </svg>
          打开服务器 VS Code
        </a>
        <button
          type="button"
          class="px-3 py-1 text-xs border border-gray-300 rounded-md hover:bg-gray-50"
          @click="fetchServerRuntimeStatus"
          :disabled="isServerRuntimeStatusLoading"
        >
          {{ isServerRuntimeStatusLoading ? '刷新中...' : '刷新状态' }}
        </button>
      </div>
    </div>
    <div class="text-sm text-gray-700">
      <span v-if="isServerRuntimeStatusLoading">正在查询实例状态...</span>
      <span v-else>{{ serverRuntimeStatusDisplayText }}</span>
    </div>
    <div v-if="showRuntimeActionButtons" class="mt-3 flex items-center space-x-2 flex-wrap">
      <button
        type="button"
        class="px-3 py-1.5 text-xs border border-transparent rounded-md text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
        data-testid="comment-runtime-stop-server-btn"
        @click="stopServer"
      >
        停止服务器
      </button>
      <template v-if="serverJumpUrl">
        <a
          :href="serverJumpUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="px-3 py-1.5 text-xs border border-transparent rounded-md text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500"
        >
          跳转到服务器
        </a>
        <span class="text-xs text-gray-500 leading-snug max-w-md">
          默认端口 {{ serverJumpDefaultPort }}；请根据需要自行调整域名或 IP 后的端口号。<span class="text-red-600 font-medium">若无法访问，请检查安全组规则是否已对相应端口放通。</span>
        </span>
      </template>
    </div>
    <div
      v-if="serverRuntimeStatusMessage"
      class="mt-1 text-xs"
      :class="serverRuntimeStatusTraceId ? 'text-amber-700' : 'text-gray-500'"
      data-testid="server-runtime-status-message"
      :data-traceId="serverRuntimeStatusTraceId || undefined"
    >
      {{ serverRuntimeStatusMessage }}
    </div>
    <div v-if="serverRuntimeStatusDetails.length > 0" class="mt-3 p-3 border border-gray-200 rounded-md bg-gray-50">
      <div class="text-xs text-gray-500 mb-2">实例详情</div>
      <div
        class="grid gap-x-6 gap-y-2 text-xs text-gray-700 [grid-template-columns:repeat(auto-fit,minmax(15rem,1fr))]"
      >
        <div v-for="item in serverRuntimeStatusDetails" :key="item.label" class="min-w-0 break-all">
          <span class="text-gray-500">{{ item.label }}：</span>
          <span>{{ item.value }}</span>
        </div>
      </div>
    </div>
    <details v-if="serverRuntimeStatusRawJson" class="mt-3 text-xs">
      <summary class="cursor-pointer text-gray-500 hover:text-gray-700">查看原始响应 JSON</summary>
      <pre class="mt-2 p-3 bg-gray-900 text-gray-100 rounded-md overflow-x-auto whitespace-pre-wrap break-words">{{ serverRuntimeStatusRawJson }}</pre>
    </details>
  </div>
</template>

<script setup>
defineProps({
  showRuntimeActionButtons: { type: Boolean, default: false },
  isWorkbenchLinkLoading: { type: Boolean, default: false },
  effectiveContainerVscodeUrl: { type: String, default: '' },
  isServerRuntimeStatusLoading: { type: Boolean, default: false },
  serverRuntimeStatusDisplayText: { type: String, default: '' },
  serverJumpUrl: { type: String, default: '' },
  serverJumpDefaultPort: { type: [String, Number], default: '' },
  serverRuntimeStatusMessage: { type: String, default: '' },
  /** 本次 runtime-status 请求 traceId；有文案时挂 data-traceId 便于 Loki 排障 */
  serverRuntimeStatusTraceId: { type: String, default: '' },
  serverRuntimeStatusDetails: { type: Array, default: () => [] },
  serverRuntimeStatusRawJson: { type: String, default: '' },
  /** 评论卡片绑定的 comment_id；用于按钮刷新，挂载不再自动查云 */
  commentId: { type: String, default: '' },
  openWorkbenchLink: { type: Function, default: null },
  fetchServerRuntimeStatus: { type: Function, default: null },
  stopServer: { type: Function, default: null },
})
</script>
