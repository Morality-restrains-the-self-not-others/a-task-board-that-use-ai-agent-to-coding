<template>
  <div class="max-w-full px-4 sm:px-6 lg:px-8 py-8">
    <div class="flex flex-row gap-5 items-start">
      <UserCenterSidebar :tenant-id="tenantId" active-menu="feature-params" class="shrink-0" />
      <main class="flex-1 max-w-3xl">
        <div class="flex flex-col space-y-6">
          <div class="flex items-center justify-between">
            <h2 class="text-2xl font-bold text-text">我的智能体资源配置</h2>
            <button @click="openCreate" class="px-4 py-2 bg-primary text-white rounded-md hover:bg-primary-dark">
              + 新建配置
            </button>
          </div>

          <div v-if="error" class="bg-red-50 border border-red-200 text-red-700 p-3 rounded-md" :data-traceId="errorTraceId || undefined">{{ error }}</div>

          <!-- Config Cards -->
          <div v-if="configs.length === 0 && !loading" class="text-center text-text-light py-12">
            暂无个人配置，点击"新建配置"开始
          </div>

          <div v-for="config in configs" :key="config.id" class="bg-white rounded-xl shadow p-6 flex items-center justify-between">
            <div>
              <h3 class="font-bold text-lg text-text">{{ config.name }}</h3>
              <div class="text-sm text-text-light mt-1 space-x-4">
                <span>智能体: {{ config.agent_model || '未设置' }}</span>
                <span>供应商: {{ config.agent_model_provider || '未设置' }}</span>
                <span>最大步数: {{ config.agent_max_steps }}</span>
                <span>供应商数: {{ (config.providers || []).length }}</span>
                <span>自定义变量: {{ (config.extra_env_vars || []).filter(e => e.key).length }} 个</span>
              </div>
              <div class="text-xs text-text-light mt-1">更新于 {{ formatDate(config.updated_at) }}</div>
            </div>
            <div class="flex space-x-2">
              <button @click="editConfig(config)" class="px-3 py-1 border border-gray-300 rounded-md text-sm hover:bg-gray-50">编辑</button>
              <button @click="duplicateConfig(config)" class="px-3 py-1 border border-gray-300 rounded-md text-sm hover:bg-gray-50">复制</button>
              <button @click="deleteConfig(config)" class="px-3 py-1 border border-red-300 rounded-md text-sm text-red-600 hover:bg-red-50">删除</button>
            </div>
          </div>
        </div>

        <!-- Create/Edit Modal -->
        <div v-if="showEditor" class="app-modal-overlay bg-black/50 flex items-center justify-center z-50" @click.self="showEditor = false">
          <div class="bg-white rounded-xl shadow-2xl w-full max-w-3xl max-h-[90vh] overflow-y-auto p-6">
            <h3 class="text-xl font-bold mb-4">{{ editingId ? '编辑配置' : '新建配置' }}</h3>
            <div class="space-y-4">
              <!-- Name -->
              <div>
                <label class="block text-sm font-medium text-text-light mb-1">配置名称 *</label>
                <input v-model.trim="editForm.name" type="text" placeholder="例如：低成本日常任务"
                  class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary" />
              </div>
              <!-- Company -->
              <div>
                <label class="block text-sm font-medium text-text-light mb-1">所属公司 *</label>
                <select v-model="editForm.company_id"
                  class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  <option value="" disabled>请选择公司</option>
                  <option v-for="c in userCompanies" :key="c.id" :value="c.id">{{ c.name }}</option>
                </select>
              </div>

              <!-- Env vars (main content) -->
              <div class="border border-gray-200 rounded-lg p-4">
                <EnvVarTableEditor v-model="editForm.extra_env_vars" />
              </div>

              <!-- Preview -->
              <MergedEnvPreview
                :system-env="systemEnv"
                :user-env-vars="editForm.extra_env_vars || []"
              />

              <!-- Collapsible LLM Config Panel -->
              <CollapsibleLLMConfigPanel
                :providers="editForm.providers"
                :sub-token-providers="[]"
                :recommended-providers="[]"
                v-model:agent-model="editForm.agent_model"
                v-model:agent-model-provider="editForm.agent_model_provider"
                v-model:agent-max-steps="editForm.agent_max_steps"
                v-model:summary-model="editForm.summary_model"
                v-model:summary-model-provider="editForm.summary_model_provider"
                :tenant-id="tenantId"
                :has-any-budget-enabled="hasAnyBudgetEnabled"
              />
            </div>

            <div v-if="editError" class="bg-red-50 text-red-700 p-2 rounded-md mt-3 text-sm" :data-traceId="editErrorTraceId || undefined">{{ editError }}</div>
            <div class="flex justify-end space-x-3 mt-6">
              <button @click="showEditor = false" class="px-4 py-2 border border-gray-300 rounded-md">取消</button>
              <button @click="saveConfig" :disabled="saving || saveConfigGuard.isBusy()" class="px-4 py-2 bg-primary text-white rounded-md disabled:opacity-50">
                {{ saving ? '保存中...' : '保存' }}
              </button>
            </div>
          </div>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { extractTraceId } from '../utils/traceId.js'
import { parseSupportedModels, validateModelsAgainstProviders } from '../utils/featureParamsModelValidation'
import { resolveUseSubToken, applySubTokenUiPolicy } from '../utils/featureParamsSubTokenUi.js'
import UserCenterSidebar from '../components/UserCenterSidebar.vue'
import EnvVarTableEditor from '../components/EnvVarTableEditor.vue'
import MergedEnvPreview from '../components/MergedEnvPreview.vue'
import CollapsibleLLMConfigPanel from '../components/CollapsibleLLMConfigPanel.vue'

const route = useRoute()
const tenantId = computed(() => String(route.params.id || ''))

const userCompanies = ref([])
const configs = ref([])
const loading = ref(false)
const error = ref('')
const errorTraceId = ref('')
const showEditor = ref(false)
const editingId = ref(null)
const editError = ref('')
const editErrorTraceId = ref('')
const saving = ref(false)

const defaultForm = () => ({
  name: '',
  company_id: '',
  agent_model: '',
  agent_model_provider: '',
  agent_max_steps: 200,
  summary_model: '',
  summary_model_provider: '',
  providers: [],
  extra_env_vars: [],
})

const editForm = reactive(defaultForm())

const hasAnyBudgetEnabled = computed(() =>
  (editForm.providers || []).some((p) => p.use_sub_token && p.budget_enabled),
)

const systemEnv = computed(() => {
  const providers = (editForm.providers || [])
    .filter((item) => item.provider)
    .map(toProviderPayload)
  return {
    TASK_LLM_PROVIDERS_JSON: JSON.stringify(providers),
    TASK_AGENT_MODEL: editForm.agent_model || '',
    TASK_AGENT_MODEL_PROVIDER: editForm.agent_model_provider || '',
    TASK_AGENT_MAX_STEPS: String(editForm.agent_max_steps || '200'),
    TASK_SUMMARY_MODEL: editForm.summary_model || '',
    TASK_SUMMARY_MODEL_PROVIDER: editForm.summary_model_provider || '',
    TASK_FEATURE_PARAMS_SCOPE: 'personal',
    TASK_FEATURE_PARAMS_CONFIG_ID: editingId.value ? String(editingId.value) : '',
    TASK_FEATURE_PARAMS_CONFIG_NAME: editForm.name || '',
  }
})

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

async function loadCompanies() {
  try {
    const resp = await apiFetch('/api/accounts/users/profile/')
    if (resp.ok) {
      const data = await resp.json()
      userCompanies.value = (data.company_nicknames || []).map(c => ({
        id: c.company_id,
        name: c.company_name || c.company_id,
      }))
    }
  } catch (e) { /* keep empty list */ }
}

async function loadConfigs() {
  loading.value = true
  error.value = ''
  errorTraceId.value = ''
  try {
    const resp = await apiFetch('/api/personal/feature-params-configs/')
    if (!resp.ok) {
      const body = await resp.json().catch(() => ({}))
      errorTraceId.value = extractTraceId(resp) || extractTraceId(body) || ''
      // 透传服务端真实错误信息（如「读取个人配置失败」/「未认证」），不再硬编码通用文案
      throw new Error(body.message || '加载失败')
    }
    configs.value = (await resp.json()).configs || []
  } catch (e) {
    error.value = '加载个人配置失败: ' + e.message
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  Object.assign(editForm, defaultForm())
  if (userCompanies.value.length === 1) {
    editForm.company_id = userCompanies.value[0].id
  }
  editError.value = ''
  editErrorTraceId.value = ''
  showEditor.value = true
}

function editConfig(config) {
  editingId.value = config.id
  Object.assign(editForm, JSON.parse(JSON.stringify(config)))
  showEditor.value = true
}

function duplicateConfig(config) {
  editingId.value = null
  const copy = JSON.parse(JSON.stringify(config))
  copy.name = config.name + ' (副本)'
  Object.assign(editForm, copy)
  showEditor.value = true
}

const deleteConfigGuard = createClickGuard()
const saveConfigGuard = createClickGuard()

async function deleteConfig(config) {
  if (!confirm(`确定删除 "${config.name}"？`)) return
  // OPT-20260819-038: 删除配置是写操作，防连点双发 DELETE
  await deleteConfigGuard.run(async ({ idempotencyKey }) => {
    try {
      const resp = await apiFetch(`/api/personal/feature-params-configs/${config.id}/`, {
        method: 'DELETE',
        headers: mergeIdempotencyHeaders({}, idempotencyKey),
      })
      if (!resp.ok) {
        errorTraceId.value = extractTraceId(resp) || ''
      }
      await loadConfigs()
    } catch (e) {
      error.value = '删除失败: ' + e.message
      errorTraceId.value = extractTraceId(e) || ''
    }
  })
}

async function saveConfig() {
  // OPT-20260819-038: 保存配置是写操作，防连点双发 POST/PUT
  await saveConfigGuard.run(async ({ idempotencyKey }) => {
    if (!editForm.name.trim()) { editError.value = '配置名称不能为空'; editErrorTraceId.value = ''; return }
    if (!editForm.company_id) { editError.value = '请选择所属公司'; editErrorTraceId.value = ''; return }
    const modelErr = validateModelsAgainstProviders(editForm.providers, {
      agentModel: editForm.agent_model,
      agentModelProvider: editForm.agent_model_provider,
      summaryModel: editForm.summary_model,
      summaryModelProvider: editForm.summary_model_provider,
    })
    if (modelErr) { editError.value = modelErr; editErrorTraceId.value = ''; return }
    saving.value = true
    editError.value = ''
    editErrorTraceId.value = ''
    try {
      const url = editingId.value
        ? `/api/personal/feature-params-configs/${editingId.value}/`
        : '/api/personal/feature-params-configs/'
      const method = editingId.value ? 'PUT' : 'POST'
      const resp = await apiFetch(url, {
        method,
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify({
          ...editForm,
          providers: (editForm.providers || []).map((item) => applySubTokenUiPolicy({ ...item })),
        }),
      })
      if (!resp.ok) {
        const err = await resp.json()
        editErrorTraceId.value = extractTraceId(resp) || extractTraceId(err) || ''
        throw new Error(err.message || '保存失败')
      }
      showEditor.value = false
      await loadConfigs()
    } catch (e) {
      editError.value = e.message
    } finally {
      saving.value = false
    }
  })
}

function formatDate(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleString()
}

onMounted(async () => {
  await Promise.all([loadConfigs(), loadCompanies()])
})
</script>
