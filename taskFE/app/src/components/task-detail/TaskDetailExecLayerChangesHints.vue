<template>
  <div>
    <p
      v-if="detail && !hasChanges"
      class="text-xs text-gray-500 mt-1"
    >
      {{ detail }}
    </p>
    <p
      v-if="hasChanges && (hasMore || truncated)"
      class="text-xs text-amber-700 mt-1"
      data-testid="task-detail-layer-changes-truncated-hint"
      v-bind="truncatedTraceId ? { 'data-traceId': truncatedTraceId } : {}"
    >
      <template v-if="hasMore">变动文件较多，向下滚动可加载更多。</template>
      <template v-else-if="truncated">目录扫描已达上限，部分文件可能未列入；可刷新重试。</template>
    </p>
    <p
      v-if="loadMoreBusy"
      class="text-xs text-gray-500 mt-1"
      data-testid="task-detail-layer-changes-load-more-busy"
    >
      正在加载更多…
    </p>
  </div>
</template>

<script setup>
defineProps({
  detail: { type: String, default: '' },
  truncated: { type: Boolean, default: false },
  hasChanges: { type: Boolean, default: false },
  hasMore: { type: Boolean, default: false },
  loadMoreBusy: { type: Boolean, default: false },
  /** 产生 truncated/hasMore 提示的最近一次成功请求 traceId；空则不挂属性 */
  truncatedTraceId: { type: String, default: '' },
})
</script>
