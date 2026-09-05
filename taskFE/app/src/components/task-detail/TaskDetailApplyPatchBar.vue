<template>
  <div
    v-if="applyEnabled"
    class="mt-3 rounded-lg border border-amber-200 bg-amber-50/60 p-3 text-xs"
    data-testid="task-detail-apply-patch-bar"
  >
    <p class="font-medium text-amber-900 mb-2">Apply patch（实验）</p>
    <p class="text-amber-800/90 mb-2">
      将 unified diff 或补丁正文提交到容器需后端接口；未配置时仅演示 UI。
    </p>
    <textarea
      v-model="patchText"
      rows="4"
      class="w-full text-xs font-mono border border-amber-200 rounded p-2 bg-white"
      placeholder="--- a/foo&#10;+++ b/foo&#10;..."
    />
    <div class="mt-2 flex flex-wrap gap-2">
      <button
        type="button"
        class="px-3 py-1.5 rounded bg-amber-600 text-white text-xs font-medium disabled:opacity-50"
        :disabled="!patchText.trim() || applying"
        @click="onApply"
      >
        {{ applying ? '提交中…' : 'Apply' }}
      </button>
      <span v-if="lastMessage" class="text-gray-700 self-center">{{ lastMessage }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../../utils/clickGuard.js'

const props = defineProps({
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
})

const applyEnabled = computed(() => import.meta.env.VITE_ENABLE_APPLY_PATCH === 'true')

const patchText = ref('')
const applying = ref(false)
const lastMessage = ref('')

const applyPatchGuard = createClickGuard()

async function onApply() {
  const body = patchText.value.trim()
  if (!body) return
  const tenantId = props.tenantId
  const workspaceId = props.workspaceId
  const taskId = props.taskId
  if (!tenantId || !workspaceId || !taskId) {
    lastMessage.value = '缺少租户/工作区/任务'
    return
  }
  // OPT-20260819-038: 提交 patch 是写操作，防连点/超时重试双发 POST
  await applyPatchGuard.run(async ({ idempotencyKey }) => {
    applying.value = true
    lastMessage.value = ''
    try {
      const path = `/api/cloud/container-apply-patch/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}`
      const r = await apiFetch(path, {
        method: 'POST',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ patch: body }),
      })
      if (!r.ok) {
        const j = await r.json().catch(() => ({}))
        lastMessage.value = typeof j.detail === 'string' ? j.detail : `HTTP ${r.status}`
        return
      }
      lastMessage.value = '已提交'
      patchText.value = ''
    } catch (e) {
      lastMessage.value = e?.message || '请求失败'
    } finally {
      applying.value = false
    }
  })
}
</script>
