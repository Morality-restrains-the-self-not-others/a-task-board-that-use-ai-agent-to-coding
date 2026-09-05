<template>
  <div v-if="enabled" class="mt-8 pt-6 border-t border-gray-200">
    <h4 class="text-base font-semibold text-text mb-3">工作空间默认预算</h4>
    <p class="text-sm text-text-light mb-4">
      金额单位为人民币（元）。
    </p>
    <p v-if="loadingWorkspaces" class="text-sm text-gray-500">加载工作空间…</p>
    <div v-else class="space-y-3">
      <div
        v-for="ws in workspaces"
        :key="ws.value"
        class="flex items-center justify-between border border-gray-200 rounded-lg p-3"
      >
        <span class="font-medium text-text">{{ ws.text }}</span>
        <button class="text-primary hover:underline text-sm" type="button" @click="openBudgetDefaults(ws)">
          LLM 预算默认
        </button>
      </div>
      <p v-if="workspaces.length === 0" class="text-sm text-gray-500">暂无工作空间</p>
    </div>

    <WorkspaceSettingsLlmBudgetModal
      :visible="showBudgetModal"
      :tenant-id="tenantId"
      :workspace-id="modalWorkspaceId"
      :workspace-label="modalWorkspaceLabel"
      @close="showBudgetModal = false"
      @saved="showBudgetModal = false"
    />
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import WorkspaceSettingsLlmBudgetModal from './WorkspaceSettingsLlmBudgetModal.vue'

const props = defineProps({
  tenantId: { type: String, required: true },
  enabled: { type: Boolean, default: false },
})

const workspaces = ref([])
const loadingWorkspaces = ref(false)
const showBudgetModal = ref(false)
const modalWorkspaceId = ref('')
const modalWorkspaceLabel = ref('')

const loadWorkspaces = async () => {
  if (!props.tenantId) return
  loadingWorkspaces.value = true
  try {
    const resp = await apiFetch(`/api/projects/workspaces/tenant_id/${props.tenantId}`, { credentials: 'include' })
    if (!resp.ok) return
    const data = await resp.json()
    const list = Array.isArray(data) ? data : data.results || []
    workspaces.value = list.map((ws) => ({
      value: String(ws.id),
      text: ws.name || String(ws.id),
    }))
  } finally {
    loadingWorkspaces.value = false
  }
}

const openBudgetDefaults = (ws) => {
  modalWorkspaceId.value = ws.value
  modalWorkspaceLabel.value = ws.text
  showBudgetModal.value = true
}

watch(
  () => [props.enabled, props.tenantId],
  ([enabled]) => {
    if (enabled && workspaces.value.length === 0) {
      loadWorkspaces()
    }
  },
  { immediate: true },
)
</script>
