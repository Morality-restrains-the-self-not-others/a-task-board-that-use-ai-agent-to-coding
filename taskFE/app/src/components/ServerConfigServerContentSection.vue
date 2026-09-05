<template>
  <div
    class="p-6 bg-white border border-gray-200 rounded-lg"
    data-testid="server-content-section"
    :data-comment-id="commentId || undefined"
  >
    <div class="flex justify-between items-center mb-2 flex-wrap gap-2">
      <h3 class="text-sm font-medium text-gray-500">服务器内容</h3>
      <button
        type="button"
        class="px-3 py-1 text-xs border border-gray-300 rounded-md hover:bg-gray-50"
        @click="onRefresh"
        :disabled="isServerContentLoading"
      >
        {{ isServerContentLoading ? '拉取中...' : '刷新内容' }}
      </button>
    </div>
    <div
      v-if="serverContentMessage"
      class="text-xs mb-3"
      :class="serverContentMessageTraceId ? 'text-amber-700' : 'text-gray-500'"
      data-testid="server-content-message"
      :data-traceId="serverContentMessageTraceId || undefined"
    >{{ serverContentMessage }}</div>
    <div class="text-xs text-gray-500 mb-2" v-if="serverContentTargetUrl">
      来源：{{ serverContentTargetUrl }}
    </div>
    <pre class="p-3 bg-gray-900 text-gray-100 rounded-md overflow-x-auto whitespace-pre-wrap break-words text-xs min-h-40">{{ serverContentText || '暂无内容' }}</pre>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { trimCommentId } from '../utils/cloudComputeCommentQuery.js'

const props = defineProps({
  commentId: { type: String, default: '' },
  isServerContentLoading: { type: Boolean, default: false },
  serverContentMessage: { type: String, default: '' },
  serverContentMessageTraceId: { type: String, default: '' },
  serverContentTargetUrl: { type: String, default: '' },
  serverContentText: { type: String, default: '' },
  fetchServerContent: { type: Function, default: null },
})

function onRefresh() {
  if (typeof props.fetchServerContent === 'function') props.fetchServerContent()
}

onMounted(() => {
  if (!trimCommentId(props.commentId)) return
  onRefresh()
})
</script>
