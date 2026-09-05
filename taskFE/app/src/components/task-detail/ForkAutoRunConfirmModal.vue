<template>
  <div
    v-if="show"
    id="fork-auto-run-confirm-modal"
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-9999 p-4"
    data-testid="fork-auto-run-confirm-modal"
    @click="onOverlayClick"
  >
    <div
      class="bg-white rounded-xl shadow-xl w-full max-w-lg flex flex-col z-10000"
      role="dialog"
      aria-modal="true"
      aria-labelledby="fork-auto-run-confirm-title"
      @click.stop
    >
      <div class="p-6 border-b border-gray-100">
        <h3 id="fork-auto-run-confirm-title" class="text-xl font-bold text-gray-900">
          确认派生任务
        </h3>
        <p class="mt-2 text-sm text-gray-600 leading-relaxed">
          将复制当前任务属性创建新任务（记录 fork_from）；新任务进度从第一列开始。请选择新任务是否自动运行；自动运行可能启动云资源。
        </p>
      </div>

      <div class="px-6 py-4 border-b border-gray-100 space-y-2" data-testid="fork-derive-mode-section">
        <p class="text-sm font-medium text-gray-800">派生方式</p>
        <label class="flex items-center gap-2 text-sm text-gray-800">
          <input
            v-model="deriveMode"
            type="radio"
            name="fork-derive-mode"
            value="fork-only"
            data-testid="fork-mode-fork-only"
            :disabled="forking"
          />
          不自动运行，仅派生
        </label>
        <label class="flex items-center gap-2 text-sm text-gray-800">
          <input
            v-model="deriveMode"
            type="radio"
            name="fork-derive-mode"
            value="auto-run"
            data-testid="fork-mode-auto-run"
            :disabled="forking"
          />
          自动运行并派生
        </label>
      </div>

      <div
        v-if="deriveMode === 'auto-run' && gitIdentityRepoUrls.length > 0"
        class="px-6 py-4 border-b border-gray-100 space-y-3"
        data-testid="fork-auto-run-git-identity-section"
      >
        <p class="text-sm font-medium text-gray-800">本次运行的 Git 提交身份</p>
        <p class="text-xs text-gray-500">
          选择「自动运行并派生」时，须为每个关联仓库选择当前账号的提交身份；该身份会写入自动运行评论。
        </p>
        <div
          v-for="url in gitIdentityRepoUrls"
          :key="url"
          class="rounded-md border border-gray-200 p-2.5"
          data-testid="fork-auto-run-git-identity-row"
        >
          <p class="text-xs text-gray-700 break-all">{{ url }}</p>
          <CreateTaskRepoGitIdentitySelect
            :visible="true"
            :model-value="gitIdentityIdForUrl(url)"
            :identities="gitIdentities"
            :settings-href="gitIdentitySettingsHref"
            @update:model-value="(value) => emit('update-git-identity', { repoUrl: url, gitIdentityId: value })"
          />
        </div>
        <p
          v-if="gitIdentityBlockedReason"
          class="text-sm text-amber-700"
          data-testid="fork-auto-run-git-identity-blocked-reason"
          role="status"
          :data-traceId="gitIdentityBlockedTraceId || undefined"
        >
          {{ gitIdentityBlockedReason }}
        </p>
      </div>

      <div
        v-if="deriveMode === 'auto-run' && oauthRepoRows.length > 0"
        class="px-6 py-4 border-b border-gray-100 space-y-3"
        data-testid="fork-auto-run-oauth-section"
      >
        <p class="text-sm font-medium text-gray-800">仓库 Git OAuth</p>
        <p class="text-xs text-gray-500">
          选择「自动运行并派生」时，容器会立即引导克隆，须事先完成下列仓库授权。
        </p>
        <div
          v-for="row in oauthRepoRows"
          :key="row.repoUrl"
          class="rounded-md border border-gray-200 p-2.5 space-y-1"
          data-testid="fork-auto-run-oauth-row"
        >
          <p class="text-xs text-gray-700 break-all">{{ row.repoUrl }}</p>
          <p
            v-if="row.loading"
            class="text-xs text-gray-500"
            data-testid="fork-auto-run-oauth-loading"
          >
            正在检查 Git OAuth…
          </p>
          <p
            v-else-if="row.bound"
            class="text-xs text-emerald-700"
            data-testid="fork-auto-run-oauth-bound"
          >
            已绑定 Git OAuth
          </p>
          <div v-else class="flex items-center gap-2 flex-wrap">
            <a
              class="text-xs px-2 py-1 border border-blue-200 text-blue-700 rounded hover:bg-blue-50"
              data-testid="fork-auto-run-oauth-bind"
              :href="row.bindHref"
            >
              OAuth 绑定
            </a>
            <span class="text-xs text-gray-500">须完成 {{ row.siteLabel }} 授权</span>
          </div>
          <p
            v-if="row.error"
            class="text-xs text-red-600"
            role="alert"
            data-testid="fork-auto-run-oauth-error"
            :data-traceId="row.errorTraceId || undefined"
          >
            {{ row.error }}
          </p>
        </div>
        <p
          v-if="autoRunBlockedReason"
          class="text-sm text-amber-700"
          data-testid="fork-auto-run-oauth-blocked-reason"
          role="status"
          :data-traceId="autoRunBlockedTraceId || undefined"
        >
          {{ autoRunBlockedReason }}
        </p>
      </div>

      <div
        v-if="deriveMode === 'auto-run'"
        class="px-6 py-4 border-b border-gray-100"
      >
        <ForkAutoRunAgentCopySection
          :disabled="forking"
          :feature-params-source="featureParamsSource"
          :personal-config-id="personalConfigId"
          :personal-configs="personalConfigs"
          :model-options="modelOptions"
          :selected-models="selectedModels"
          :loading="modelsLoading"
          :load-error="modelsLoadError"
          :load-error-trace-id="modelsLoadErrorTraceId"
          @update:feature-params-source="onFeatureParamsSourceChange"
          @update:personal-config-id="onPersonalConfigChange"
          @toggle-model="toggleModel"
          @retry-models="refreshAgentModels"
        />
      </div>

      <div class="flex flex-col sm:flex-row sm:justify-end gap-2 p-6">
        <button
          id="fork-confirm-cancel-btn"
          type="button"
          class="px-4 py-2 border border-gray-300 bg-white text-gray-700 rounded-md text-sm font-medium hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          data-testid="fork-confirm-cancel"
          :disabled="forking"
          @click="emit('close')"
        >
          <!-- Anti-Replay-OK: ui-only 关闭模态，不发写请求 -->
          取消
        </button>
        <button
          id="fork-confirm-submit-btn"
          type="button"
          class="px-4 py-2 bg-primary hover:bg-primary-dark text-white rounded-md text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
          data-testid="fork-confirm-submit"
          :disabled="confirmDisabled"
          :aria-busy="forking ? 'true' : undefined"
          :title="confirmDisabledReason || undefined"
          @click="onConfirm"
        >
          {{ forkingLabel('确认派生') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import CreateTaskRepoGitIdentitySelect from '../CreateTaskRepoGitIdentitySelect.vue'
import ForkAutoRunAgentCopySection from './ForkAutoRunAgentCopySection.vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { normalizeFeatureParamsSourceForSelect } from '../../utils/envParamsSourceSelection.js'
import {
  buildAgentModelOptionsFromFeatureParams,
  preselectDefaultAgentModel,
} from '../../utils/agentModelOptions.js'
import { loadFeatureParamsPayloadForSource } from '../../composables/taskDetail/taskDetailLayerGraphModelOptions.js'
import { clampForkCopyCount } from '../../utils/forkCopyCount.js'

const props = defineProps({
  show: { type: Boolean, default: false },
  forking: { type: Boolean, default: false },
  forkProgressCurrent: { type: Number, default: 0 },
  forkProgressTotal: { type: Number, default: 0 },
  autoRunBlockedReason: { type: String, default: '' },
  autoRunBlockedTraceId: { type: String, default: '' },
  oauthRepoRows: {
    type: Array,
    default: () => [],
  },
  gitIdentityRepoUrls: {
    type: Array,
    default: () => [],
  },
  gitIdentities: {
    type: Array,
    default: () => [],
  },
  gitIdentitySettingsHref: { type: String, default: '' },
  gitIdentityIdForUrl: {
    type: Function,
    default: () => '',
  },
  gitIdentityBlockedReason: { type: String, default: '' },
  gitIdentityBlockedTraceId: { type: String, default: '' },
  tenantId: { type: [String, Number], default: '' },
  workspaceId: { type: [String, Number], default: '' },
  sourceTask: { type: Object, default: null },
})

const emit = defineEmits(['close', 'confirm', 'update-git-identity'])

const deriveMode = ref('')
const featureParamsSource = ref('')
const personalConfigId = ref('')
const personalConfigs = ref([])
const modelOptions = ref([])
const selectedModels = ref([])
const agentModelProvider = ref('')
const defaultAgentModel = ref('')
const modelsLoading = ref(false)
const modelsLoadError = ref('')
const modelsLoadErrorTraceId = ref('')

const isAutoRun = computed(() => deriveMode.value === 'auto-run')

const sourceReady = computed(() => {
  const source = featureParamsSource.value
  if (!source) return false
  if (source === 'personal' && !String(personalConfigId.value || '').trim()) return false
  return true
})

const confirmDisabledReason = computed(() => {
  if (!deriveMode.value) return '请选择派生方式'
  if (!isAutoRun.value) return ''
  if (props.autoRunBlockedReason) return props.autoRunBlockedReason
  if (props.gitIdentityBlockedReason) return props.gitIdentityBlockedReason
  if (!sourceReady.value) return '请选择智能体资源配置'
  if (selectedModels.value.length === 0) return '请至少选择一个智能体'
  return ''
})

const confirmDisabled = computed(() => {
  if (props.forking) return true
  return Boolean(confirmDisabledReason.value)
})

watch(() => props.show, (open) => {
  if (!open) return
  deriveMode.value = ''
  resetAgentCopyFromSourceTask()
}, { immediate: true })

watch(deriveMode, (mode) => {
  if (mode === 'auto-run') {
    void refreshAgentModels()
  }
})

watch([featureParamsSource, personalConfigId], () => {
  if (deriveMode.value === 'auto-run') {
    void refreshAgentModels()
  }
})

function resetAgentCopyFromSourceTask() {
  const task = props.sourceTask || {}
  featureParamsSource.value = normalizeFeatureParamsSourceForSelect(task.feature_params_source)
  personalConfigId.value = featureParamsSource.value === 'personal'
    ? String(task.personal_feature_params_config_id || '').trim()
    : ''
  selectedModels.value = []
  modelOptions.value = []
  agentModelProvider.value = ''
  defaultAgentModel.value = ''
  modelsLoadError.value = ''
  modelsLoadErrorTraceId.value = ''
  if (featureParamsSource.value === 'personal') {
    void fetchPersonalConfigs()
  } else {
    personalConfigs.value = []
  }
}

async function fetchPersonalConfigs() {
  try {
    const resp = await apiFetch('/api/personal/feature-params-configs/', {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    if (!resp.ok) return
    const data = await resp.json().catch(() => ({}))
    personalConfigs.value = Array.isArray(data?.configs) ? data.configs : []
  } catch (error) {
    console.error('[ForkAutoRunConfirmModal] event=personal_configs_failed', error)
  }
}

function onFeatureParamsSourceChange(value) {
  featureParamsSource.value = value
  if (value !== 'personal') {
    personalConfigId.value = ''
    personalConfigs.value = []
  } else {
    void fetchPersonalConfigs()
  }
  selectedModels.value = []
}

function onPersonalConfigChange(value) {
  personalConfigId.value = value
  selectedModels.value = []
}

function toggleModel(model) {
  const name = String(model || '').trim()
  if (!name) return
  const set = new Set(selectedModels.value)
  if (set.has(name)) set.delete(name)
  else {
    if (set.size >= 99) return
    set.add(name)
  }
  selectedModels.value = Array.from(set)
}

async function refreshAgentModels() {
  const source = featureParamsSource.value
  if (!source) {
    modelOptions.value = []
    return
  }
  if (source === 'personal' && !String(personalConfigId.value || '').trim()) {
    modelOptions.value = []
    selectedModels.value = []
    return
  }
  modelsLoading.value = true
  modelsLoadError.value = ''
  modelsLoadErrorTraceId.value = ''
  try {
    const { data } = await loadFeatureParamsPayloadForSource({
      tenantId: String(props.tenantId || '').trim(),
      workspaceId: String(props.workspaceId || '').trim(),
      source,
      personalConfigId: personalConfigId.value,
    })
    const built = buildAgentModelOptionsFromFeatureParams(data)
    agentModelProvider.value = built.provider
    defaultAgentModel.value = built.defaultModel
    modelOptions.value = built.options
    selectedModels.value = preselectDefaultAgentModel(built.options, built.defaultModel)
    if (!built.provider || built.options.length === 0) {
      modelsLoadError.value = '未配置可用模型，请先到智能体资源配置页面设置智能体模型提供商与支持模型列表。'
    }
  } catch (error) {
    console.error('[ForkAutoRunConfirmModal] event=agent_models_load_failed', error)
    modelOptions.value = []
    selectedModels.value = []
    modelsLoadError.value = error?.message || '获取模型配置失败'
    modelsLoadErrorTraceId.value = extractTraceId(error) || String(error?.traceId || '').trim()
  } finally {
    modelsLoading.value = false
  }
}

function emitConfirm(autoRun) {
  if (autoRun) {
    const agents = selectedModels.value.slice(0, 99).map((model) => ({
      provider: agentModelProvider.value,
      model,
    }))
    emit('confirm', {
      autoRun: true,
      copyCount: clampForkCopyCount(agents.length),
      featureParamsSource: featureParamsSource.value,
      personalFeatureParamsConfigId: personalConfigId.value,
      agentModelProvider: agentModelProvider.value,
      agents,
    })
    return
  }
  emit('confirm', { autoRun: false, copyCount: 1 })
}

function forkingLabel(idleText) {
  if (!props.forking) return idleText
  const total = Number(props.forkProgressTotal) || 0
  const current = Number(props.forkProgressCurrent) || 0
  if (total > 1 && current > 0) return `派生中 ${current}/${total}…`
  return '派生中…'
}

function onOverlayClick() {
  if (props.forking) return
  emit('close')
}

function onConfirm() {
  if (confirmDisabled.value) return
  emitConfirm(isAutoRun.value)
}
</script>
