<template>
  <div
    data-testid="comment-composer-env-hardware-slot"
    class="mt-4"
  >
    <div
      v-if="hardwareRunHint"
      class="mb-3 rounded border border-amber-200 bg-amber-50 px-2 py-1.5 text-[11px] leading-snug text-amber-950"
      data-testid="comment-composer-hardware-run-hint"
    >
      {{ hardwareRunHint }}
    </div>
    <div
      data-testid="server-image-config-card"
      class="p-6 bg-white border border-gray-200 rounded-lg"
    >
      <h3 class="text-sm font-medium text-gray-500 mb-3">环境与硬件</h3>
      <ServerConfigHardwarePanel
        ref="panelRef"
        embedded
        v-model:selected-image-id="selectedImageId"
        :installed-images="installedImages"
        :task="task"
        :workspace-id="workspaceId"
        :project-server-run-template="projectServerRunTemplate"
        :require-env-params-source="false"
      />
    </div>
  </div>
</template>

<script setup>
/**
 * 评论区 $镜像 后的硬件真源：直接挂在 composer 内，不经 Teleport。
 * 临时配置由面板内 HardwarePanelHeader 按钮展开，不再另挂摘要条以免与卡体重叠。
 */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import ServerConfigHardwarePanel from '../ServerConfigHardwarePanel.vue'
import {
  registerCommentHardwarePanelReader,
  resolveCommentHardwareRunHint,
} from '../../composables/taskDetail/commentRunHardwareTemplate.js'

const props = defineProps({
  projectServerRunTemplate: { type: Object, default: null },
  installedImages: { type: Array, default: () => [] },
  task: { type: Object, default: null },
  workspaceId: { type: String, default: '' },
})

const selectedImageId = defineModel('selectedImageId', { type: String, default: '' })

const panelRef = ref(null)

const hardwareRunHint = computed(() => {
  const panel = panelRef.value
  if (!panel) return ''
  void panel.hardwareConfigSource
  return resolveCommentHardwareRunHint(panel)
})

onMounted(() => {
  registerCommentHardwarePanelReader(() => panelRef.value)
  void panelRef.value?.initHardwareFromParent?.()
})

onUnmounted(() => {
  registerCommentHardwarePanelReader(() => null)
})

watch(
  () => props.task && (props.task.id || props.task.pk),
  () => {
    void panelRef.value?.initHardwareFromParent?.()
  },
)

defineExpose({
  panelRef,
})
</script>
