<template>
  <button
    type="button"
    class="text-gray-500 hover:text-primary"
    data-testid="workspace-container-image-at-mode-toggle"
    :disabled="saving"
    :title="enabled ? '容器镜像 $ 模式：已开启' : '容器镜像 $ 模式：已关闭'"
    @click="toggle"
  >
    <span class="mr-1">{{ enabled ? '$镜像开' : '$镜像关' }}</span>
  </button>
</template>

<script setup>
import { ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard'

const props = defineProps({
  tenantId: { type: String, required: true },
  workspaceId: { type: String, required: true },
  initialEnabled: { type: Boolean, default: true },
})

const emit = defineEmits(['updated'])

const enabled = ref(!!props.initialEnabled)
const saving = ref(false)

// OPT-20260819-038: 工作空间配置切换为写路径，createClickGuard 防连点双发 PATCH。
const toggleGuard = createClickGuard()

watch(
  () => props.initialEnabled,
  (v) => {
    enabled.value = !!v
  },
)

async function toggle() {
  if (!props.tenantId || !props.workspaceId || saving.value || toggleGuard.isBusy()) return
  const next = !enabled.value
  await toggleGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    try {
      const response = await apiFetch(`/api/projects/workspaces/tenant_id/${props.tenantId}/${props.workspaceId}/`, {
        method: 'PATCH',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({ container_image_at_mode_enabled: next }),
      })
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`)
      }
      enabled.value = next
      emit('updated', next)
    } catch (e) {
      console.error('toggle container_image_at_mode_enabled failed', e)
      alert('更新「容器镜像 @ 模式」失败，请重试')
    } finally {
      saving.value = false
    }
  })
}
</script>
