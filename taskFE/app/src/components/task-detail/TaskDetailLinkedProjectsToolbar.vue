<template>
  <div>
    <div class="mb-2 pb-1.5 border-b border-gray-100 flex flex-wrap items-center justify-between gap-2">
      <h3 class="text-sm font-semibold text-gray-800 m-0">关联项目</h3>
      <p
        class="m-0 text-[11px] text-gray-500 leading-snug"
        data-testid="task-default-container-image"
        :title="defaultImageId || undefined"
      >
        默认镜像
        <span
          class="ml-1 font-mono"
          :class="defaultImageDisplay === '未设置' ? 'text-gray-400' : 'text-gray-800'"
        >{{ defaultImageDisplay }}</span>
      </p>
    </div>
    <p v-if="!isEditing" class="mb-2 text-[11px] text-gray-500 leading-snug">
      任务绑定的代码仓库与基准分支（展示）。本次运行的提交身份与授权身份在下方添加评论时选择。
    </p>
    <p
      v-if="!isEditing && staleRepoSyncError"
      class="mb-2 text-[11px] text-red-600 leading-snug"
      v-bind="staleRepoSyncErrorTraceId ? { 'data-traceId': staleRepoSyncErrorTraceId } : {}"
    >
      {{ staleRepoSyncError }}
    </p>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { formatTaskDefaultImageDisplay } from '../../utils/installedImageLabel.js'

const props = defineProps({
  isEditing: { type: Boolean, required: true },
  staleRepoSyncError: { type: String, default: '' },
  staleRepoSyncErrorTraceId: { type: String, default: '' },
  defaultImageLabel: { type: String, default: '' },
  defaultImageId: { type: String, default: '' },
  defaultImageRemoved: { type: Boolean, default: false },
})

const defaultImageDisplay = computed(() =>
  formatTaskDefaultImageDisplay(
    props.defaultImageLabel,
    props.defaultImageId,
    props.defaultImageRemoved,
  ),
)
</script>
