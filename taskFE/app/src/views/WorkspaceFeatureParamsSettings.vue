<template>
  <div class="p-8 flex-1 min-w-0 max-h-[calc(100dvh-var(--app-navbar-h))] overflow-y-auto">
    <!-- 内容区作为独立滚动容器：max-h 视口高度减去 sticky 导航栏（高度见 styles.css :root --app-navbar-h，
         单一事实源，OPT-20260809-023），内容超出视口时内部滚动（与公司级 WorkspaceSettingsFeatureParams.vue
         同构，OPT-20260809-022）。此前 overflow:visible 仅文档级可滚动，操作此元素本身无法滚动。 -->
    <div class="flex flex-col space-y-6">
      <div>
        <h2 class="text-2xl font-bold text-text">工作空间环境变量</h2>
        <p class="text-sm text-text-light mt-1">工作空间: {{ workspace?.name || workspaceId }}</p>
        <p class="text-sm text-text-light mt-1">
          备注：任务执行人可以访问相关信息——任意能够看到任务的人都可以获知相关环境变量。若希望每位任务执行人使用的资源相互隔离，请让其在各自的环境变量设置中单独配置。
        </p>
      </div>

      <!-- Governance Toggle -->
      <div class="bg-white p-6 rounded-xl shadow">
        <h3 class="text-lg font-bold text-text mb-3">个人配置权限</h3>
        <label class="flex items-center space-x-3 cursor-pointer">
          <input v-model="allowPersonal" type="checkbox" class="w-5 h-5 text-primary rounded" @change="saveGovernance" />
          <span class="text-sm">允许成员在此工作空间中使用个人智能体资源配置</span>
        </label>
      </div>

      <!-- Unified Tab Card -->
      <div class="bg-white rounded-xl shadow overflow-hidden">
        <!-- Tab header -->
        <div class="px-6 pt-5 pb-0">
          <h3 class="text-lg font-bold text-text mb-3">配置来源</h3>
          <div class="inline-flex rounded-lg border border-gray-200 bg-gray-100 p-0.5" role="tablist">
            <button
              type="button"
              role="tab"
              :aria-selected="useCompanyDefault"
              class="px-5 py-2 rounded-md text-sm font-medium transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-primary/30"
              :class="useCompanyDefault
                ? 'bg-white text-primary shadow-sm'
                : 'text-text-light hover:text-text'"
              @click="useCompanyDefault = true"
            >
              🏢 公司默认
            </button>
            <button
              type="button"
              role="tab"
              :aria-selected="!useCompanyDefault"
              class="px-5 py-2 rounded-md text-sm font-medium transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-primary/30"
              :class="!useCompanyDefault
                ? 'bg-white text-primary shadow-sm'
                : 'text-text-light hover:text-text'"
              @click="useCompanyDefault = false"
            >
              ⚙️ 自定义配置
            </button>
          </div>
        </div>

        <!-- Divider -->
        <div class="border-t border-gray-100 mt-4"></div>

        <!-- Tab content with transition -->
        <Transition name="tab-fade" mode="out-in">
          <div :key="useCompanyDefault ? 'company' : 'custom'" class="px-6 py-5">
            <!-- Company Default Tab -->
            <div v-if="useCompanyDefault">
              <div class="space-y-4">
                <div class="flex items-center gap-2 text-sm text-text-light mb-3">
                  <span>🔒</span>
                  <span v-if="companyConfig">以下配置继承自<strong>公司默认配置</strong>，仅可查看不可编辑</span>
                  <span v-else>公司尚未配置智能体资源配置，以下为<strong>系统默认值</strong></span>
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  <div class="bg-gray-50 rounded-lg p-3 border border-gray-100">
                    <p class="text-xs text-text-light mb-1">智能体模型</p>
                    <p class="text-sm font-semibold text-text">{{ (companyConfig && companyConfig.agent_model) || '未设置' }}</p>
                  </div>
                  <div class="bg-gray-50 rounded-lg p-3 border border-gray-100">
                    <p class="text-xs text-text-light mb-1">摘要模型</p>
                    <p class="text-sm font-semibold text-text">{{ (companyConfig && companyConfig.summary_model) || '未设置' }}</p>
                  </div>
                  <div class="bg-gray-50 rounded-lg p-3 border border-gray-100">
                    <p class="text-xs text-text-light mb-1">最大迭代步数</p>
                    <p class="text-sm font-semibold text-text">{{ (companyConfig && companyConfig.agent_max_steps) || 200 }}</p>
                  </div>
                </div>
                <div class="text-xs text-text-light space-x-4">
                  <span>供应商: {{ (companyConfig && companyConfig.providers || []).length }} 个</span>
                  <span>自定义变量: {{ (companyConfig && companyConfig.extra_env_vars || []).filter(e => e.key).length }} 个</span>
                </div>
              </div>
            </div>

            <!-- Custom Config Tab -->
            <div v-else>
              <EnvVarTableEditor v-model="form.extra_env_vars" />
              <div v-if="companyConfig && companyConfig.extra_env_vars && companyConfig.extra_env_vars.filter(e => e.key).length > 0" class="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-md text-sm text-blue-700">
                公司默认有 {{ companyConfig.extra_env_vars.filter(e => e.key).length }} 个自定义变量。此处定义的变量将覆盖同名的公司默认值。
              </div>
            </div>
          </div>
        </Transition>

        <!-- Merged Env Preview -->
        <div class="px-6 pb-4">
          <MergedEnvPreview
            :system-env="systemEnv"
            :user-env-vars="effectiveUserEnvVars"
          />
        </div>

        <!-- Collapsible LLM Config Panel -->
        <div class="px-6 pb-4">
          <CollapsibleLLMConfigPanel
            :providers="effectiveProviders"
            :sub-token-providers="[]"
            :recommended-providers="[]"
            v-model:agent-model="form.agent_model"
            v-model:agent-model-provider="form.agent_model_provider"
            v-model:agent-max-steps="form.agent_max_steps"
            v-model:summary-model="form.summary_model"
            v-model:summary-model-provider="form.summary_model_provider"
            :tenant-id="tenantId"
            :has-any-budget-enabled="hasAnyBudgetEnabled"
            :readonly="useCompanyDefault"
          />
        </div>

        <!-- Error / Success messages -->
        <div v-if="error" class="px-6 pb-3">
          <div class="bg-red-50 border border-red-200 text-red-700 p-3 rounded-md" :data-traceId="errorTraceId || undefined">{{ error }}</div>
        </div>
        <div v-if="success" class="px-6 pb-3">
          <div class="bg-green-50 border border-green-200 text-green-700 p-3 rounded-md">{{ success }}</div>
        </div>

        <!-- Actions footer -->
        <div class="px-6 py-4 bg-gray-50/50 border-t border-gray-100 flex items-center justify-between">
          <div class="flex space-x-3">
            <button @click="save" :disabled="saving || saveGuard.isBusy()" class="px-6 py-2 bg-primary text-white rounded-md hover:bg-primary-dark disabled:opacity-50 text-sm font-medium transition-colors">
              {{ saving ? '保存中...' : '保存' }}
            </button>
            <button v-if="!useCompanyDefault" @click="resetToCompany" class="px-6 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50 text-sm font-medium transition-colors">
              重置为公司默认
            </button>
          </div>
          <span class="text-xs text-text-light">
            {{ useCompanyDefault ? '当前使用公司默认配置' : '当前使用工作空间自定义配置' }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { extractTraceId } from '../utils/traceId.js'
import { parseSupportedModels, validateModelsAgainstProviders } from '../utils/featureParamsModelValidation'
import { resolveUseSubToken, applySubTokenUiPolicy } from '../utils/featureParamsSubTokenUi.js'
import EnvVarTableEditor from '../components/EnvVarTableEditor.vue'
import MergedEnvPreview from '../components/MergedEnvPreview.vue'
import CollapsibleLLMConfigPanel from '../components/CollapsibleLLMConfigPanel.vue'

const props = defineProps({
  workspace: { type: Object, default: null },
})

const route = useRoute()
const tenantId = computed(() => String(route.params.tenant || ''))
const workspaceId = computed(() => String(route.params.workspace || ''))

const useCompanyDefault = ref(true)
const allowPersonal = ref(false)
const companyConfig = ref(null)
const workspaceConfigId = ref('')
const savedWorkspaceConfig = ref(null)  // stored workspace values for restore when switching back to custom
const saving = ref(false)
// OPT-20260819-038: 保存工作空间配置/治理开关均为写操作，防连点双发
const saveGuard = createClickGuard()
const saveGovernanceGuard = createClickGuard()
const error = ref('')
const errorTraceId = ref('')
const success = ref('')

const form = reactive({
  providers: [],
  agent_model: '',
  agent_model_provider: '',
  agent_max_steps: 200,
  summary_model: '',
  summary_model_provider: '',
  extra_env_vars: [],
})

// Effective company config — always defined, falls back to system defaults
const effectiveCompanyConfig = computed(() => {
  if (companyConfig.value) return companyConfig.value
  return {
    providers: [],
    agent_model: '',
    agent_model_provider: '',
    agent_max_steps: 200,
    summary_model: '',
    summary_model_provider: '',
    extra_env_vars: [],
  }
})

// Which providers to show in CollapsibleLLMConfigPanel
const effectiveProviders = computed(() => {
  if (useCompanyDefault.value) {
    return effectiveCompanyConfig.value.providers || []
  }
  return form.providers
})

// Which user env vars to show in the preview
const effectiveUserEnvVars = computed(() => {
  if (useCompanyDefault.value) {
    return effectiveCompanyConfig.value.extra_env_vars || []
  }
  return form.extra_env_vars || []
})

// System env derived from LLM config
const systemEnv = computed(() => {
  const cfg = useCompanyDefault.value ? effectiveCompanyConfig.value : form
  const providers = (cfg.providers || [])
    .filter((item) => item.provider)
    .map(toProviderPayload)
  return {
    TASK_LLM_PROVIDERS_JSON: JSON.stringify(providers),
    TASK_AGENT_MODEL: cfg.agent_model || '',
    TASK_AGENT_MODEL_PROVIDER: cfg.agent_model_provider || '',
    TASK_AGENT_MAX_STEPS: String(cfg.agent_max_steps || '200'),
    TASK_SUMMARY_MODEL: cfg.summary_model || '',
    TASK_SUMMARY_MODEL_PROVIDER: cfg.summary_model_provider || '',
    TASK_FEATURE_PARAMS_SCOPE: useCompanyDefault.value ? 'company' : 'workspace',
    TASK_FEATURE_PARAMS_CONFIG_ID: useCompanyDefault.value
      ? String((companyConfig.value && companyConfig.value.id) || '')
      : (workspaceConfigId.value || ''),
    TASK_FEATURE_PARAMS_CONFIG_NAME: useCompanyDefault.value
      ? '公司默认'
      : `工作空间配置:${workspaceId.value || ''}`,
  }
})

const hasAnyBudgetEnabled = computed(() =>
  effectiveProviders.value.some((p) => p.use_sub_token && p.budget_enabled),
)

const toProviderPayload = (provider) => ({
  provider: String(provider.provider || '').trim(),
  api_key: String(provider.api_key || '').trim(),
  base_url: String(provider.base_url || '').trim(),
  supported_models: parseSupportedModels(
    Array.isArray(provider.supported_models)
      ? provider.supported_models
      : (provider.supported_models_text || provider.supported_models),
  ),
  use_sub_token: resolveUseSubToken(provider.use_sub_token),
  budget_enabled: Boolean(resolveUseSubToken(provider.use_sub_token) && provider.budget_enabled),
})

const load = async () => {
  errorTraceId.value = ''
  try {
    const resp = await apiFetch(`/api/cloud/feature-params/tenant_id/${tenantId.value}/workspace_id/${workspaceId.value}`, {
      headers: { 'X-Feature-Params-Access-Context': 'workspace_settings' },
    })
    if (!resp.ok) {
      errorTraceId.value = extractTraceId(resp) || ''
      throw new Error('加载失败')
    }
    const json = await resp.json()
    const d = json.data
    useCompanyDefault.value = d.use_company_default
    companyConfig.value = d.company_config || null
    workspaceConfigId.value = d.id ? String(d.id) : ''
    // Capture saved workspace config for restore when switching back to custom
    if (d.workspace_config) {
      savedWorkspaceConfig.value = d.workspace_config
    } else if (!d.use_company_default) {
      // Custom mode: save from top-level fields so we can restore later
      savedWorkspaceConfig.value = {
        providers: d.providers || [],
        agent_model: d.agent_model || '',
        agent_model_provider: d.agent_model_provider || '',
        agent_max_steps: String(d.agent_max_steps || '200'),
        summary_model: d.summary_model || '',
        summary_model_provider: d.summary_model_provider || '',
        extra_env_vars: Array.isArray(d.extra_env_vars) ? [...d.extra_env_vars] : [],
      }
    } else {
      savedWorkspaceConfig.value = null
    }
    allowPersonal.value = json.workspace?.allow_personal_feature_params || false

    // Always populate form: from workspace data (custom) or from effective company config (inherited)
    const source = d.use_company_default
      ? (d.company_config || { providers: [], agent_model: '', agent_model_provider: '', agent_max_steps: 200, summary_model: '', summary_model_provider: '', extra_env_vars: [] })
      : d
    form.providers = source.providers || []
    form.agent_model = source.agent_model || ''
    form.agent_model_provider = source.agent_model_provider || ''
    form.agent_max_steps = parseInt(source.agent_max_steps) || 200
    form.summary_model = source.summary_model || ''
    form.summary_model_provider = source.summary_model_provider || ''
    form.extra_env_vars = Array.isArray(source.extra_env_vars) ? [...source.extra_env_vars] : []
  } catch (e) {
    error.value = '加载工作空间配置失败: ' + e.message
    if (!errorTraceId.value) errorTraceId.value = extractTraceId(e) || ''
  }
}

const saveGovernance = async () => {
  // OPT-20260819-038: 治理开关保存是写操作，防连点/超时重试双发 POST
  await saveGovernanceGuard.run(async ({ idempotencyKey }) => {
    try {
      await apiFetch(`/api/cloud/feature-params/tenant_id/${tenantId.value}/workspace_id/${workspaceId.value}`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'X-Feature-Params-Access-Context': 'workspace_settings',
          },
          idempotencyKey,
        ),
        body: JSON.stringify({
          use_company_default: useCompanyDefault.value,
          allow_personal_feature_params: allowPersonal.value,
        }),
      })
    } catch (e) { /* silent */ }
  })
}

const save = async () => {
  // OPT-20260819-038: 保存工作空间配置是写操作，防连点/超时重试双发 POST
  await saveGuard.run(async ({ idempotencyKey }) => {
    saving.value = true
    error.value = ''
    errorTraceId.value = ''
    success.value = ''
    try {
      const body = {
        use_company_default: useCompanyDefault.value,
        allow_personal_feature_params: allowPersonal.value,
      }
      if (!useCompanyDefault.value) {
        const modelErr = validateModelsAgainstProviders(form.providers, {
          agentModel: form.agent_model,
          agentModelProvider: form.agent_model_provider,
          summaryModel: form.summary_model,
          summaryModelProvider: form.summary_model_provider,
        })
        if (modelErr) {
          throw new Error(modelErr)
        }
        Object.assign(body, {
          providers: (form.providers || []).map((item) => applySubTokenUiPolicy({ ...item })),
          agent_model: form.agent_model,
          agent_model_provider: form.agent_model_provider,
          agent_max_steps: String(form.agent_max_steps),
          summary_model: form.summary_model,
          summary_model_provider: form.summary_model_provider,
          extra_env_vars: form.extra_env_vars || [],
        })
      }
      const resp = await apiFetch(`/api/cloud/feature-params/tenant_id/${tenantId.value}/workspace_id/${workspaceId.value}`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          {
            'Content-Type': 'application/json',
            'X-Feature-Params-Access-Context': 'workspace_settings',
          },
          idempotencyKey,
        ),
        body: JSON.stringify(body),
      })
      if (!resp.ok) {
        const err = await resp.json()
        errorTraceId.value = extractTraceId(resp) || extractTraceId(err) || ''
        throw new Error(err.message || '保存失败')
      }
      success.value = '保存成功'
      await load()
    } catch (e) {
      error.value = e.message
    } finally {
      saving.value = false
    }
  })
}

const resetToCompany = () => {
  useCompanyDefault.value = true
  save()
}

onMounted(() => {
  load()
})

// When toggling modes, populate form from the correct source
watch(useCompanyDefault, (val) => {
  if (val) {
    // Switching to company default: populate form from effective company config (display only, readonly)
    const co = effectiveCompanyConfig.value
    form.providers = co.providers || []
    form.agent_model = co.agent_model || ''
    form.agent_model_provider = co.agent_model_provider || ''
    form.agent_max_steps = parseInt(co.agent_max_steps) || 200
    form.summary_model = co.summary_model || ''
    form.summary_model_provider = co.summary_model_provider || ''
    form.extra_env_vars = Array.isArray(co.extra_env_vars) ? [...co.extra_env_vars] : []
  } else if (!val) {
    // Switching to custom mode: restore previously saved workspace config (if any)
    const restore = savedWorkspaceConfig.value
    if (restore) {
      form.providers = restore.providers || []
      form.agent_model = restore.agent_model || ''
      form.agent_model_provider = restore.agent_model_provider || ''
      form.agent_max_steps = parseInt(restore.agent_max_steps) || 200
      form.summary_model = restore.summary_model || ''
      form.summary_model_provider = restore.summary_model_provider || ''
      form.extra_env_vars = Array.isArray(restore.extra_env_vars) ? [...restore.extra_env_vars] : []
    }
    // If no saved workspace config, keep current form values (from company config)
    // so the user can start customizing from the company defaults
  }
})
</script>

<style scoped>
.tab-fade-enter-active,
.tab-fade-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.tab-fade-enter-from {
  opacity: 0;
  transform: translateY(6px);
}
.tab-fade-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
