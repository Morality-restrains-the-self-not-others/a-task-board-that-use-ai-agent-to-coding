<template>
  <div class="bg-white rounded-lg shadow-md p-6 mb-8" data-testid="project-run-template-panel">
    <div class="flex flex-wrap items-start justify-between gap-3 mb-4">
      <div>
        <h2 class="text-xl font-semibold">项目运行模版</h2>
        <p class="text-sm text-gray-500 mt-1">
          配置与任务详情页一致的服务器硬件选项；创建任务勾选自动运行时将按此模版启动云服务器。
        </p>
      </div>
      <div class="text-sm text-gray-700 space-y-1 min-w-[16rem]">
        <p>
          模版：<span class="font-medium">{{ currentSummary }}</span>
        </p>
        <p
          v-if="hardwareSpecSummary"
          data-testid="run-template-hardware-spec-summary"
          class="text-gray-600 tabular-nums"
        >
          规格：<span class="font-medium">{{ hardwareSpecSummary }}</span>
        </p>
      </div>
    </div>

    <div
      v-if="error"
      class="mb-3 text-sm text-red-600"
      :data-traceId="errorTraceId || undefined"
    >{{ error }}</div>

    <div v-if="!resolvedWorkspaceId" class="mb-4 p-3 bg-amber-50 border border-amber-100 rounded text-sm text-amber-800">
      请先为项目关联工作空间，才能加载云平台与实例列表。
    </div>

    <div v-if="!readonly" class="mb-4">
      <label for="project-run-template-select" class="block text-sm font-medium text-gray-700 mb-1">
        快速应用默认模版
        <span v-if="loading" class="inline-block ml-1 animate-spin h-3 w-3 border-2 border-gray-300 border-t-primary rounded-full"></span>
      </label>
      <select
        id="project-run-template-select"
        v-model="selectedTemplateId"
        class="w-full md:max-w-md border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        :disabled="loading || !resolvedWorkspaceId"
        @change="onTemplatePresetChange"
      >
        <option value="">手动配置（不使用预设）</option>
        <option v-for="opt in templateOptions" :key="opt.id" :value="opt.id">
          {{ opt.label }}
        </option>
      </select>
      <p v-if="!loading && resolvedWorkspaceId && !templateOptions.length" class="mt-2 text-xs text-amber-700">
        当前租户尚未配置服务器默认模版，请先在「工作空间管理 → 机器节点」中设置默认服务器启动配置。
      </p>
    </div>

    <ServerConfigHardwarePanel
      v-if="resolvedWorkspaceId"
      :key="resolvedWorkspaceId"
      ref="hardwarePanelRef"
      :tenant-id="String(tenantId || '')"
      :workspace-id="resolvedWorkspaceId"
      :project-server-run-template="project?.server_run_template"
      :installed-images="installedImages"
      :selected-image-id="projectContainerImageId"
      :run-template-mode="true"
      :readonly="readonly"
      @spec-summary-change="onHardwareSpecSummaryChange"
    />

    <div v-if="!readonly && !hideActions" class="mt-4 flex flex-wrap gap-2">
      <button
        type="button"
        class="bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded-md text-sm disabled:opacity-50"
        :disabled="saving || loading || !resolvedWorkspaceId"
        :aria-busy="saving ? 'true' : 'false'"
        @click="handleSave"
      >
        {{ saving ? '保存中...' : '保存运行模版' }}
      </button>
      <button
        type="button"
        class="border border-gray-300 text-gray-700 px-4 py-2 rounded-md text-sm hover:bg-gray-50 disabled:opacity-50"
        :disabled="saving || loading || !hasDraftConfig"
        @click="handleClear"
      >
        清除模版
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch, nextTick } from 'vue'
import ServerConfigHardwarePanel from './ServerConfigHardwarePanel.vue'
import { useProjectRunTemplate } from '../composables/useProjectRunTemplate.js'
import { summarizeRunTemplate, summarizeRunTemplateHardwareSpecs } from '../utils/projectRunTemplateUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { extractTraceId } from '../utils/traceId.js'

const props = defineProps({
  tenantId: { type: String, required: true },
  projectId: { type: String, required: true },
  project: { type: Object, default: null },
  installedImages: { type: Array, default: () => [] },
  readonly: { type: Boolean, default: false },
  hideActions: { type: Boolean, default: false },
  draftImageId: { type: String, default: '' },
})

const emit = defineEmits(['saved', 'error', 'spec-summary-change'])

const hardwarePanelRef = ref(null)
const liveHardwareSpecSummary = ref('')
const saveGuard = createClickGuard()

const resolvedWorkspaceId = computed(() => {
  const ws = props.project?.workspaces
  if (Array.isArray(ws) && ws.length) return String(ws[0])
  return ''
})

const installedImages = computed(() =>
  Array.isArray(props.installedImages) ? props.installedImages : [],
)

const projectContainerImageId = computed(() =>
  String(props.draftImageId || props.project?.container_image_id || '').trim(),
)

const {
  loading,
  saving,
  error,
  errorTraceId,
  templateOptions,
  selectedTemplateId,
  loadTemplateOptions,
  setFromProject,
} = useProjectRunTemplate({
  tenantId: () => props.tenantId,
  projectId: () => props.projectId,
  workspaceId: () => resolvedWorkspaceId.value,
  initialTemplate: props.project?.server_run_template,
})

const currentSummary = computed(() =>
  summarizeRunTemplate(props.project?.server_run_template),
)

const hardwareSpecSummary = computed(() => {
  if (liveHardwareSpecSummary.value) return liveHardwareSpecSummary.value
  return summarizeRunTemplateHardwareSpecs(props.project?.server_run_template)
})

const onHardwareSpecSummaryChange = (summary) => {
  const next = String(summary || '').trim()
  liveHardwareSpecSummary.value = next
  emit('spec-summary-change', next)
}

let workspaceRefreshToken = 0
let workspaceRefreshPromise = null

const refreshWorkspaceContext = async ({ resetPreset = true } = {}) => {
  if (!resolvedWorkspaceId.value) return
  const token = ++workspaceRefreshToken
  if (workspaceRefreshPromise) {
    await workspaceRefreshPromise
    if (token !== workspaceRefreshToken) return
  }
  workspaceRefreshPromise = (async () => {
    if (resetPreset) {
      selectedTemplateId.value = ''
    }
    await loadTemplateOptions()
    if (token !== workspaceRefreshToken) return
    await nextTick()
    await syncHardwareWithPreset()
  })()
  try {
    await workspaceRefreshPromise
  } finally {
    if (workspaceRefreshPromise) workspaceRefreshPromise = null
  }
}

const hasDraftConfig = computed(() => {
  const payload = hardwarePanelRef.value?.buildRunTemplatePayload?.()
  return Boolean(payload && Object.keys(payload).length)
})

const findTemplateOptionById = (id) => {
  const sel = String(id ?? '').trim()
  if (!sel) return null
  return templateOptions.value.find((o) => String(o.id).trim() === sel) || null
}

const APPLY_PRESET_ERROR_MESSAGES = {
  no_platforms: '应用默认模版失败：当前工作空间未加载到云平台，请先在「设置 → 云平台绑定」中绑定云平台',
  no_match: '应用默认模版失败：默认模版与已绑定云平台不匹配，请在「工作空间管理 → 机器节点」中重新保存默认配置',
  invalid_template: '应用默认模版失败：模版数据无效',
  invalid_shape: '应用默认模版失败：无法解析硬件配置',
}

const applySelectedPresetToHardware = async () => {
  const panel = hardwarePanelRef.value
  if (!panel || !selectedTemplateId.value) return
  const opt = findTemplateOptionById(selectedTemplateId.value)
  if (!opt?.template) {
    error.value = '未找到所选默认模版，请刷新后重试'
    return
  }
  const ok = await panel.applyRunTemplate?.(opt.template)
  if (ok === false) {
    const reason = panel.getLastApplyRunTemplateError?.() || ''
    error.value = APPLY_PRESET_ERROR_MESSAGES[reason]
      || '应用默认模版失败，请确认工作空间已绑定对应云平台'
  } else {
    error.value = ''
  }
}

let syncingHardware = false
const syncHardwareWithPreset = async () => {
  if (syncingHardware) return
  syncingHardware = true
  try {
    const panel = hardwarePanelRef.value
    if (!panel || !resolvedWorkspaceId.value) return
    await panel.initHardwareFromParent?.()
    if (selectedTemplateId.value) {
      await applySelectedPresetToHardware()
    }
  } finally {
    syncingHardware = false
  }
}

const onTemplatePresetChange = async () => {
  if (!selectedTemplateId.value) {
    error.value = ''
    await hardwarePanelRef.value?.bootstrapManualConfig?.()
    return
  }
  await applySelectedPresetToHardware()
}

const buildPayload = () => hardwarePanelRef.value?.buildRunTemplatePayload?.() ?? {}

const handleSave = async () => {
  await saveGuard.run(async ({ idempotencyKey }) => {
    error.value = ''
    errorTraceId.value = ''
    const payload = buildPayload()
    if (!Object.keys(payload).length) {
      error.value = '请至少选择云平台与地域，并配置实例参数'
      emit('error', error.value)
      return
    }
    try {
      const tid = String(props.tenantId || '').trim()
      const pid = String(props.projectId || '').trim()
      saving.value = true
      const { apiFetch } = await import('../utils/apiUtils.js')
      const response = await apiFetch(`/api/projects/tenant_id/${tid}/${pid}/`, {
        method: 'PATCH',
        credentials: 'include',
        headers: mergeIdempotencyHeaders(
          { 'Content-Type': 'application/json', Accept: 'application/json' },
          idempotencyKey,
        ),
        body: JSON.stringify({ server_run_template: payload }),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        errorTraceId.value = extractTraceId(response) || extractTraceId(data)
        throw new Error(typeof data?.detail === 'string' ? data.detail : (typeof data?.error === 'string' ? data.error : '保存项目运行模版失败'))
      }
      emit('saved', data)
    } catch (e) {
      error.value = e?.message || '保存失败'
      errorTraceId.value = errorTraceId.value || extractTraceId(e)
      emit('error', error.value)
    } finally {
      saving.value = false
    }
  })
}

const handleClear = async () => {
  selectedTemplateId.value = ''
  errorTraceId.value = ''
  try {
    const tid = String(props.tenantId || '').trim()
    const pid = String(props.projectId || '').trim()
    saving.value = true
    const { apiFetch } = await import('../utils/apiUtils.js')
    const response = await apiFetch(`/api/projects/tenant_id/${tid}/${pid}/`, {
      method: 'PATCH',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ server_run_template: {} }),
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      errorTraceId.value = extractTraceId(response) || extractTraceId(data)
      throw new Error(typeof data?.detail === 'string' ? data.detail : '清除项目运行模版失败')
    }
    await hardwarePanelRef.value?.initHardwareFromParent?.()
    liveHardwareSpecSummary.value = ''
    emit('saved', data)
  } catch (e) {
    error.value = e?.message || '清除失败'
    errorTraceId.value = errorTraceId.value || extractTraceId(e)
    emit('error', error.value)
  } finally {
    saving.value = false
  }
}

watch(
  () => props.project,
  (p) => {
    if (p) setFromProject(p)
  },
  { deep: true },
)

watch(
  () => props.project?.workspaces?.[0],
  async (wsId, prevWsId) => {
    const wid = wsId != null ? String(wsId).trim() : ''
    const prevWid = prevWsId != null ? String(prevWsId).trim() : ''
    if (!wid || wid === prevWid || prevWsId === undefined) return
    await refreshWorkspaceContext()
  },
)

watch(resolvedWorkspaceId, async (wid, prev) => {
  if (!wid || wid === prev || prev === undefined) return
  await refreshWorkspaceContext()
})

watch(
  [hardwarePanelRef, () => templateOptions.value.length, loading],
  async ([panel, , isLoading]) => {
    if (!panel || !resolvedWorkspaceId.value || isLoading) return
    await syncHardwareWithPreset()
  },
  { flush: 'post' },
)

watch(projectContainerImageId, async (id, prev) => {
  if (prev === undefined) return
  await nextTick()
  const panel = hardwarePanelRef.value
  if (!panel?.scheduleFetchAvailableInstances) return
  await panel.scheduleFetchAvailableInstances({ immediate: true })
})

onMounted(async () => {
  if (props.project) setFromProject(props.project)
  if (resolvedWorkspaceId.value) {
    await refreshWorkspaceContext({ resetPreset: false })
  }
})

defineExpose({
  saveTemplate: handleSave,
  buildPayload,
  refreshWorkspaceContext,
})
</script>
