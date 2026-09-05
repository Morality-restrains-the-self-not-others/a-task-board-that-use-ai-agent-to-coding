<template>
  <TenantPageAccessEmpty v-if="!accessAllowed" page-key="settings.feature_params" />
  <div v-else class="p-8 flex-1 min-w-0 max-h-[calc(100dvh-var(--app-navbar-h))] overflow-y-auto">
    <!-- 内容区作为独立滚动容器：max-h 视口高度减去 sticky 导航栏（高度见 styles.css :root --app-navbar-h，
         单一事实源，OPT-20260809-023），内容超出视口时内部滚动。此前 overflow:visible 仅文档级可滚动，
         操作此元素本身无法滚动。 -->
    <div class="flex flex-col space-y-6">
      <div>
        <h2 class="text-2xl font-bold text-text">环境变量设置</h2>
        <p class="text-sm text-text-light mt-1">这些变量将在容器启动时注入。LLM 相关变量由系统根据下方配置自动生成。</p>
        <p class="text-sm text-text-light mt-1">
          备注：任务执行人可以访问相关信息——任意能够看到任务的人都可以获知相关环境变量。若希望每位任务执行人使用的资源相互隔离，请让其在各自的环境变量设置中单独配置。
        </p>
      </div>

      <!-- Env Vars Table Editor (main content) -->
      <div class="bg-white p-6 rounded-xl shadow">
        <EnvVarTableEditor v-model="form.extra_env_vars" />
      </div>

      <!-- Merged Env Preview -->
      <MergedEnvPreview
        :system-env="systemEnv"
        :user-env-vars="form.extra_env_vars"
      />

      <!-- Collapsible LLM Config Panel -->
      <CollapsibleLLMConfigPanel
        :providers="form.providers"
        :sub-token-providers="subTokenProviders"
        :recommended-providers="recommendedProviders"
        v-model:agent-model="form.agent_model"
        v-model:agent-model-provider="form.agent_model_provider"
        v-model:agent-max-steps="form.agent_max_steps"
        v-model:summary-model="form.summary_model"
        v-model:summary-model-provider="form.summary_model_provider"
        :tenant-id="tenantId"
        :has-any-budget-enabled="hasAnyBudgetEnabled"
      />

      <!-- Save / Reset -->
      <div class="bg-white p-6 rounded-xl shadow">
        <div class="flex items-center gap-3">
          <button class="btn-primary" :disabled="isSaving || saveFormGuard.isBusy()" @click="saveForm">
            {{ isSaving ? '保存中...' : '保存' }}
          </button>
          <span v-if="saveMessage" class="text-sm" :class="saveMessageType === 'success' ? 'text-green-600' : 'text-red-600'">
            {{ saveMessage }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { parseSupportedModels, validateModelsAgainstProviders } from '../utils/featureParamsModelValidation'
import { resolveUseSubToken } from '../utils/featureParamsSubTokenUi.js'
import EnvVarTableEditor from '../components/EnvVarTableEditor.vue'
import MergedEnvPreview from '../components/MergedEnvPreview.vue'
import CollapsibleLLMConfigPanel from '../components/CollapsibleLLMConfigPanel.vue'
import { safeJson } from '@/utils/safeResponseJson.js'
import { humanizeRequestErrorMessage } from '../utils/requestErrorDisplay.js'
import { useTenantPageAccess } from '../composables/useTenantPageAccess.js'
import TenantPageAccessEmpty from '../components/TenantPageAccessEmpty.vue'

const route = useRoute()
const { accessAllowed } = useTenantPageAccess('settings.feature_params')
const tenantId = computed(() => String(route.params.tenant || ''))
const isSaving = ref(false)
const saveMessage = ref('')
const saveMessageType = ref('success')
const recommendedProviders = ref([])
const subTokenProviders = ref([])

const form = reactive({
  providers: [{
    provider: '',
    api_key: '',
    base_url: '',
    supported_models_text: '',
    use_sub_token: false,
    budget_enabled: false,
  }],
  agent_model: '',
  agent_model_provider: '',
  agent_max_steps: '200',
  summary_model: '',
  summary_model_provider: '',
  extra_env_vars: [],
})
const configId = ref('')

const toProviderPayload = (provider) => ({
  provider: String(provider.provider || '').trim(),
  api_key: String(provider.api_key || '').trim(),
  base_url: String(provider.base_url || '').trim(),
  supported_models: parseSupportedModels(provider.supported_models_text || provider.supported_models),
  use_sub_token: resolveUseSubToken(provider.use_sub_token),
  budget_enabled: Boolean(resolveUseSubToken(provider.use_sub_token) && provider.budget_enabled),
})

const getTenantId = () => route.params.tenant || ''

const hasAnyBudgetEnabled = computed(() =>
  form.providers.some((p) => p.use_sub_token && p.budget_enabled),
)

// System env built from LLM config — the "automatic" part
const systemEnv = computed(() => {
  const providers = form.providers
    .filter((item) => item.provider)
    .map(toProviderPayload)
  return {
    TASK_LLM_PROVIDERS_JSON: JSON.stringify(providers),
    TASK_AGENT_MODEL: form.agent_model || '',
    TASK_AGENT_MODEL_PROVIDER: form.agent_model_provider || '',
    TASK_AGENT_MAX_STEPS: form.agent_max_steps || '200',
    TASK_SUMMARY_MODEL: form.summary_model || '',
    TASK_SUMMARY_MODEL_PROVIDER: form.summary_model_provider || '',
    TASK_FEATURE_PARAMS_SCOPE: 'company',
    TASK_FEATURE_PARAMS_CONFIG_ID: configId.value || '',
    TASK_FEATURE_PARAMS_CONFIG_NAME: '公司默认',
  }
})

const saveFormGuard = createClickGuard()

const saveForm = async () => {
  // OPT-20260819-038: 保存环境变量是写操作，防连点双发 POST
  await saveFormGuard.run(async ({ idempotencyKey }) => {
    isSaving.value = true
    saveMessage.value = ''
    try {
      const tenantId = getTenantId()
      if (!tenantId) {
        throw new Error('缺少租户ID')
      }

      const modelErr = validateModelsAgainstProviders(form.providers, {
        agentModel: form.agent_model,
        agentModelProvider: form.agent_model_provider,
        summaryModel: form.summary_model,
        summaryModelProvider: form.summary_model_provider,
      })
      if (modelErr) {
        throw new Error(modelErr)
      }

      const response = await apiFetch(`/api/cloud/feature-params/tenant_id/${tenantId}`, {
        method: 'POST',
        headers: mergeIdempotencyHeaders(
          {
            'X-Feature-Params-Access-Context': 'company_settings',
          },
          idempotencyKey,
        ),
        body: JSON.stringify({
          providers: form.providers.map(toProviderPayload),
          agent_model: form.agent_model,
          agent_model_provider: form.agent_model_provider,
          agent_max_steps: form.agent_max_steps,
          summary_model: form.summary_model,
          summary_model_provider: form.summary_model_provider,
          extra_env_vars: form.extra_env_vars || [],
        })
      })
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.message || '保存失败')
      }

      saveMessageType.value = 'success'
      saveMessage.value = '已保存'
    } catch (error) {
      console.error('保存环境变量失败:', error)
      saveMessageType.value = 'error'
      saveMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '保存失败，请重试'
    } finally {
      isSaving.value = false
    }
  })
}

const loadForm = async () => {
  const tenantId = getTenantId()
  if (!tenantId) {
    return
  }
  const response = await apiFetch(`/api/cloud/feature-params/tenant_id/${tenantId}`, {
    headers: {
      'X-Feature-Params-Access-Context': 'company_settings',
    },
  })
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}))
    throw new Error(errorData.message || '加载失败')
  }
  const result = await safeJson(response, {})
  const data = result.data || {}
  configId.value = data.id ? String(data.id) : ''
  form.providers = Array.isArray(data.providers) && data.providers.length > 0
    ? data.providers.map((item) => ({
      provider: String(item.provider || '').trim(),
      api_key: String(item.api_key || '').trim(),
      base_url: String(item.base_url || '').trim(),
      supported_models_text: parseSupportedModels(item.supported_models).join('\n'),
      use_sub_token: resolveUseSubToken(item.use_sub_token || isSubTokenProvider(item.provider)),
      budget_enabled: Boolean(item.budget_enabled) && resolveUseSubToken(item.use_sub_token || isSubTokenProvider(item.provider)),
    }))
    : [{
      provider: '',
      api_key: '',
      base_url: '',
      supported_models_text: '',
      use_sub_token: false,
      budget_enabled: false,
    }]
  form.agent_model = data.agent_model || ''
  form.agent_model_provider = data.agent_model_provider || ''
  form.agent_max_steps = data.agent_max_steps || '200'
  form.summary_model = data.summary_model || ''
  form.summary_model_provider = data.summary_model_provider || ''
  form.extra_env_vars = Array.isArray(data.extra_env_vars) ? [...data.extra_env_vars] : []
}

const isSubTokenProvider = (providerName) => {
  const name = String(providerName || '').trim().toLowerCase()
  return subTokenProviders.value.some(p => p.provider_name.toLowerCase() === name)
}

const loadRecommendedProviders = async () => {
  const response = await apiFetch('/api/recommended-llm-providers/')
  if (!response.ok) {
    const error = new Error('加载推荐供应商失败')
    error._errorData = response._errorData
    if (response.traceId) error.traceId = response.traceId
    throw error
  }
  const result = await safeJson(response, {})
  recommendedProviders.value = result.items || []
}

const loadSubTokenProviders = async () => {
  const response = await apiFetch('/api/sub-token-providers/')
  if (!response.ok) {
    console.error('加载派生Token供应商失败')
    return
  }
  const result = await safeJson(response, {})
  subTokenProviders.value = result.items || []
}

onMounted(() => {
  loadRecommendedProviders().catch((error) => {
    console.error('加载推荐供应商失败:', error)
  })
  loadSubTokenProviders().catch((error) => {
    console.error('加载派生Token供应商失败:', error)
  })
  loadForm()
    .catch((error) => {
      console.error('加载环境变量失败:', error)
      saveMessageType.value = 'error'
      saveMessage.value = error.message ? humanizeRequestErrorMessage(error.message) : '加载失败，请刷新重试'
    })
})
</script>
