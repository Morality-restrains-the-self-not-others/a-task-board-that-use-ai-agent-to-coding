import { ref, computed } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { errorFromFailedResponse } from '../../utils/httpError.js'
import { retryTransientHttp } from '../../utils/httpTransientRetry.js'
import { normalizeFeatureParamsSourceForSelect } from '../../utils/envParamsSourceSelection.js'
import { buildAgentModelOptionsFromFeatureParams } from '../../utils/agentModelOptions.js'

async function fetchFeatureParamsOk(url, fallbackMessage) {
  const response = await retryTransientHttp(
    () => apiFetch(url, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    }),
    { url },
  )
  if (!response.ok) {
    throw errorFromFailedResponse(response, fallbackMessage)
  }
  const result = await response.json().catch(() => ({}))
  const traceId = extractTraceId(response) || extractTraceId(result) || String(response.traceId || '').trim()
  return { result, traceId }
}

export function normalizeSelectedAgentModels(models) {
  return Array.from(
    new Set(
      (Array.isArray(models) ? models : [])
        .map((item) => String(item || '').trim())
        .filter((item) => item),
    ),
  )
}

/**
 * 从 workspace feature-params 响应中取出「实际生效」的模型配置。
 * @param {Record<string, unknown>} data
 */
export function resolveEffectiveFeatureParamsData(data) {
  if (!data || typeof data !== 'object') return {}
  if (data.use_company_default) {
    const company = data.company_config && typeof data.company_config === 'object'
      ? data.company_config
      : {}
    return {
      providers: company.providers || [],
      agent_model: company.agent_model || '',
      agent_model_provider: company.agent_model_provider || '',
    }
  }
  return {
    providers: data.providers || [],
    agent_model: data.agent_model || '',
    agent_model_provider: data.agent_model_provider || '',
  }
}

/**
 * 按所选智能体资源拉取模型配置（打开弹窗一次，禁止轮询）。
 * @param {{
 *   tenantId: string,
 *   workspaceId?: string,
 *   source: string,
 *   personalConfigId?: string,
 * }} opts
 * @returns {Promise<{ data: Record<string, unknown>, traceId: string }>}
 */
export async function loadFeatureParamsPayloadForSource(opts) {
  const tenantId = String(opts?.tenantId || '').trim()
  const workspaceId = String(opts?.workspaceId || '').trim()
  const source = normalizeFeatureParamsSourceForSelect(opts?.source) || 'company'
  const personalConfigId = String(opts?.personalConfigId || '').trim()

  if (!tenantId) {
    throw new Error('缺少租户')
  }

  if (source === 'personal') {
    if (!personalConfigId) {
      throw new Error('任务未绑定个人智能体资源配置')
    }
    const { result, traceId } = await fetchFeatureParamsOk(
      '/api/personal/feature-params-configs/',
      '获取个人模型配置失败',
    )
    const configs = Array.isArray(result?.configs) ? result.configs : []
    const matched = configs.find((item) => String(item?.id || '').trim() === personalConfigId)
    if (!matched) {
      const err = new Error('未找到任务绑定的个人智能体资源配置')
      err.traceId = traceId
      throw err
    }
    return { data: matched, traceId }
  }

  if (source === 'workspace') {
    if (!workspaceId) {
      throw new Error('缺少工作空间')
    }
    const { result, traceId } = await fetchFeatureParamsOk(
      `/api/cloud/feature-params/tenant_id/${tenantId}/workspace_id/${workspaceId}?view=summary`,
      '获取工作空间模型配置失败',
    )
    return {
      data: resolveEffectiveFeatureParamsData(result?.data || {}),
      traceId,
    }
  }

  const { result, traceId } = await fetchFeatureParamsOk(
    `/api/cloud/feature-params/tenant_id/${tenantId}?view=summary`,
    '获取模型配置失败',
  )
  return { data: result?.data || {}, traceId }
}

/**
 * @param {{
 *   effectiveTenantId: import('vue').Ref,
 *   effectiveWorkspaceId?: import('vue').Ref,
 *   localTask?: import('vue').Ref,
 *   layerGraphCmdSending: import('vue').Ref,
 *   layerGraphCommandKind: import('vue').Ref,
 * }} deps
 */
export function createLayerGraphModelOptionsState(deps) {
  const {
    effectiveTenantId,
    effectiveWorkspaceId,
    localTask,
    layerGraphCmdSending,
    layerGraphCommandKind,
  } = deps

  const layerGraphModelProvider = ref('')
  const layerGraphDefaultModel = ref('')
  const layerGraphModelOptions = ref([])
  const layerGraphSelectedModel = ref('')
  const layerGraphModelLoading = ref(false)
  const layerGraphModelLoadError = ref('')
  const layerGraphModelLoadErrorTraceId = ref('')

  const layerGraphModelSelectDisabled = computed(
    () => layerGraphCmdSending.value || layerGraphModelLoading.value || layerGraphCommandKind.value !== 'trae',
  )

  function applyLayerGraphModelSettings(data, taskAgentModel = '') {
    const built = buildAgentModelOptionsFromFeatureParams(data)
    layerGraphModelProvider.value = built.provider
    layerGraphDefaultModel.value = built.defaultModel
    layerGraphModelOptions.value = built.options
    const optionSet = new Set(built.options)
    const cur = String(layerGraphSelectedModel.value || '').trim()
    const taskModel = String(taskAgentModel || '').trim()
    if (cur && optionSet.has(cur)) {
      layerGraphSelectedModel.value = cur
    } else if (taskModel && optionSet.has(taskModel)) {
      // OPT-20260825-014：派生副本持久化的 agent_model 优先作为默认展示模型，
      // 运维无需依赖任务级 feature-params 即可确认这一份实际 overlay 的模型。
      layerGraphSelectedModel.value = taskModel
    } else if (built.defaultModel && optionSet.has(built.defaultModel)) {
      layerGraphSelectedModel.value = built.defaultModel
    } else {
      layerGraphSelectedModel.value = ''
    }
    if (!built.provider || built.options.length === 0) {
      layerGraphModelLoadError.value = '未配置可用模型，请先到智能体资源配置页面设置智能体模型提供商与支持模型列表。'
      layerGraphModelLoadErrorTraceId.value = ''
    } else {
      layerGraphModelLoadError.value = ''
      layerGraphModelLoadErrorTraceId.value = ''
    }
  }

  function clearLayerGraphModelSettings() {
    layerGraphModelProvider.value = ''
    layerGraphDefaultModel.value = ''
    layerGraphModelOptions.value = []
    layerGraphSelectedModel.value = ''
    layerGraphModelLoadError.value = ''
    layerGraphModelLoadErrorTraceId.value = ''
  }

  async function loadFeatureParamsPayload() {
    const task = localTask?.value || {}
    return loadFeatureParamsPayloadForSource({
      tenantId: String(effectiveTenantId?.value || '').trim(),
      workspaceId: String(effectiveWorkspaceId?.value || '').trim(),
      source: task.feature_params_source,
      personalConfigId: task.personal_feature_params_config_id,
    })
  }

  const fetchLayerGraphModelOptions = async () => {
    const tenantId = effectiveTenantId?.value
    if (!tenantId) {
      clearLayerGraphModelSettings()
      return
    }
    layerGraphModelLoading.value = true
    layerGraphModelLoadErrorTraceId.value = ''
    try {
      const { data } = await loadFeatureParamsPayload()
      const task = localTask?.value || {}
      applyLayerGraphModelSettings(data, task.agent_model)
      layerGraphModelLoadErrorTraceId.value = ''
    } catch (error) {
      console.error('获取智能体模型配置失败:', error)
      clearLayerGraphModelSettings()
      layerGraphModelLoadError.value = error?.message || '获取模型配置失败'
      layerGraphModelLoadErrorTraceId.value = extractTraceId(error) || String(error?.traceId || '').trim()
    } finally {
      layerGraphModelLoading.value = false
    }
  }

  return {
    layerGraphModelProvider,
    layerGraphDefaultModel,
    layerGraphModelOptions,
    layerGraphSelectedModel,
    layerGraphModelLoading,
    layerGraphModelLoadError,
    layerGraphModelLoadErrorTraceId,
    layerGraphModelSelectDisabled,
    fetchLayerGraphModelOptions,
    normalizeSelectedAgentModels,
  }
}
