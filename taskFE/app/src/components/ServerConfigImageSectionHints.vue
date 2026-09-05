<template>
  <div>
    <p v-if="architectureLabel" class="mt-2 text-xs text-gray-500">
      支持 CPU 架构：{{ architectureLabel }}
    </p>
    <ServerConfigMachineOwnerHint
      :runtime-status="runtimeStatus"
      :viewer-task-id="viewerTaskId"
      :tenant-id="tenantId"
      :workspace-id="workspaceId"
      :server-url="serverUrl"
      :container-page-url="containerPageUrl"
      :mock-container-running="mockContainerRunning"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { formatContainerImageArchitectures } from '../utils/containerImageArchitecture.js'
import ServerConfigMachineOwnerHint from './ServerConfigMachineOwnerHint.vue'

const props = defineProps({
  installedImages: { type: Array, default: () => [] },
  selectedImageId: { type: [String, Number], default: '' },
  task: { type: Object, default: null },
  runtimeStatus: { type: Object, default: null },
  viewerTaskId: { type: String, default: '' },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  serverUrl: { type: String, default: '' },
  containerPageUrl: { type: String, default: '' },
  mockContainerRunning: { type: Boolean, default: false },
})

const architectureLabel = computed(() =>
  formatContainerImageArchitectures(
    props.installedImages.find((img) => String(img.id) === String(props.selectedImageId)) ||
      props.task?.container_image,
  ),
)
</script>
