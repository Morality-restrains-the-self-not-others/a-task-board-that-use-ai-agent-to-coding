import { ref, computed } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import {
  normalizeFeatureParamsSourceForSelect,
  persistTaskFeatureParamsBinding,
  readEnvVarSourcesAvailableFlag,
  resolveEnvParamsSourceRequiredHint,
} from '../../utils/envParamsSourceSelection.js'
import { resolveRelayContextIds, resolveServerConfigTaskId } from '../../utils/serverConfigRouteHelpers.js'

/**
 * ServerConfig 智能体资源配置来源选择与持久化。
 */
export function useServerConfigFeatureParams({ props, emit, route, workspaceId }) {
  const featureParamsSource = ref('')
  const personalConfigs = ref([])
  const selectedPersonalConfigId = ref('')
  const resolvedEnvPreview = ref({})
  const isEnvPreviewLoading = ref(false)
  const envPreviewExpanded = ref(false)
  const featureParamsPersistError = ref('')
  /**
   * 公司/工作空间/个人三类环境变量是否至少存在一个可选项；
   * 未拉取/拉取失败时保持 true（fail-open，避免误禁导致无法选择来源）。
   */
  const featureParamsSourcesAvailable = ref(true)

  const envParamsSourceRequiredHint = computed(() =>
    resolveEnvParamsSourceRequiredHint({
      featureParamsSource: featureParamsSource.value,
      selectedPersonalConfigId: selectedPersonalConfigId.value,
    }),
  )

  const initFeatureParamsSource = () => {
    featureParamsSource.value = normalizeFeatureParamsSourceForSelect(
      props.task?.feature_params_source,
    )
    selectedPersonalConfigId.value = featureParamsSource.value === 'personal'
      ? String(props.task?.personal_feature_params_config_id || '').trim()
      : ''
    if (featureParamsSource.value === 'personal') {
      fetchPersonalConfigs()
    }
    // 镜像区选择器可用性：与 create-task-modal 同源（OPT-20260809-019 标志）
    void fetchFeatureParamsSourcesAvailability()
  }

  /**
   * 拉取公司/工作空间自定义环境变量可用性标志（view=summary）与个人配置列表，
   * 计算三类来源是否至少一个可用（OPT-20260809-018）。
   * 用 view=summary 拉取（data 脱敏、不含 provider API key），来源可用性由后端
   * env_var_sources_available 标志给出（OPT-20260809-019）。
   * 任务详情场景缺 workspace id 时降级为公司级 GET（workspace 恒 false）。
   * 缺 tenant 或请求失败/缺标志时保持可用（fail-open）。
   * 个人配置须先于可用性计算完成（await fetchPersonalConfigs），避免竞态误判。
   */
  const fetchFeatureParamsSourcesAvailability = async () => {
    const ctx = resolveRelayContextIds({
      route,
      task: props.task,
      workspaceId,
      taskId: props.taskId,
      tenantId: props.tenantId,
    })
    const tenant = ctx.tenant_id
    if (!tenant) return
    const url = ctx.workspace_id
      ? `/api/cloud/feature-params/tenant_id/${tenant}/workspace_id/${ctx.workspace_id}?view=summary`
      : `/api/cloud/feature-params/tenant_id/${tenant}?view=summary`
    try {
      const resp = await apiFetch(url, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      await fetchPersonalConfigs()
      if (resp.ok) {
        const body = await resp.json()
        const flag = readEnvVarSourcesAvailableFlag(body)
        if (!flag) return
        featureParamsSourcesAvailable.value = Boolean(flag.company)
          || Boolean(flag.workspace)
          || personalConfigs.value.length > 0
      }
    } catch (error) {
      console.error('获取环境变量来源可用性失败:', error)
    }
  }

  const fetchPersonalConfigs = async () => {
    try {
      const resp = await apiFetch('/api/personal/feature-params-configs/', {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (resp.ok) {
        const data = await resp.json()
        personalConfigs.value = data.configs || []
      }
    } catch (error) {
      console.error('获取个人配置列表失败:', error)
    }
  }

  const fetchEnvPreview = async () => {
    if (!selectedPersonalConfigId.value) return
    try {
      isEnvPreviewLoading.value = true
      const ctx = resolveRelayContextIds({
        route,
        task: props.task,
        workspaceId,
        taskId: props.taskId,
        tenantId: props.tenantId,
      })
      const params = new URLSearchParams({
        source: 'personal',
        personal_config_id: selectedPersonalConfigId.value,
      })
      const { tenant_id, workspace_id, task_id } = ctx
      const previewUrl = `/api/cloud/compute/feature-params-env-preview/tenant_id/${tenant_id}/workspace_id/${workspace_id}/task_id/${task_id}/?${params.toString()}`
      const resp = await apiFetch(previewUrl, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (resp.ok) {
        const data = await resp.json()
        resolvedEnvPreview.value = data.env || {}
        envPreviewExpanded.value = true
      } else {
        const errData = await resp.json().catch(() => ({}))
        console.error('预览环境变量失败:', resp.status, errData.message || resp.statusText)
      }
    } catch (error) {
      console.error('预览环境变量失败:', error)
    } finally {
      isEnvPreviewLoading.value = false
    }
  }

  const isFeatureParamsSelectionUnchanged = () => {
    const nextSource = normalizeFeatureParamsSourceForSelect(featureParamsSource.value)
    const taskSource = normalizeFeatureParamsSourceForSelect(props.task?.feature_params_source)
    if (nextSource !== taskSource) return false
    if (nextSource !== 'personal') return true
    return String(selectedPersonalConfigId.value || '').trim()
      === String(props.task?.personal_feature_params_config_id || '').trim()
  }

  let featureParamsPersistGeneration = 0

  const persistFeatureParamsSourceIfReady = async () => {
    if (isFeatureParamsSelectionUnchanged()) return
    featureParamsPersistError.value = ''
    const generation = ++featureParamsPersistGeneration
    const result = await persistTaskFeatureParamsBinding({
      apiFetch,
      taskId: resolveServerConfigTaskId({
        route,
        task: props.task,
        taskId: props.taskId,
        tenantId: props.tenantId,
        workspaceId,
      }),
      featureParamsSource: featureParamsSource.value,
      selectedPersonalConfigId: selectedPersonalConfigId.value,
    })
    if (generation !== featureParamsPersistGeneration) return
    if (result.skipped) return
    if (!result.ok) {
      console.error('持久化智能体资源配置来源失败:', result.message || result.status)
      featureParamsPersistError.value = String(result.message || '').trim()
        || `保存智能体资源配置失败（HTTP ${result.status || '未知'}）`
      return
    }
    featureParamsPersistError.value = ''
    emit('task-updated', {
      ...props.task,
      feature_params_source: result.data.feature_params_source,
      personal_feature_params_config_id: result.data.personal_feature_params_config_id || '',
    })
  }

  const onFeatureParamsSourceUpdate = (value) => {
    featureParamsSource.value = value
    featureParamsPersistError.value = ''
  }

  const onPersonalConfigIdUpdate = (value) => {
    selectedPersonalConfigId.value = value
    featureParamsPersistError.value = ''
    void persistFeatureParamsSourceIfReady()
  }

  const onSourceChange = () => {
    envPreviewExpanded.value = false
    resolvedEnvPreview.value = {}
    featureParamsPersistError.value = ''
    if (featureParamsSource.value === 'personal') {
      fetchPersonalConfigs()
      return
    }
    selectedPersonalConfigId.value = ''
    void persistFeatureParamsSourceIfReady()
  }

  return {
    featureParamsSource,
    personalConfigs,
    selectedPersonalConfigId,
    resolvedEnvPreview,
    isEnvPreviewLoading,
    envPreviewExpanded,
    featureParamsPersistError,
    featureParamsSourcesAvailable,
    envParamsSourceRequiredHint,
    initFeatureParamsSource,
    fetchFeatureParamsSourcesAvailability,
    fetchEnvPreview,
    onFeatureParamsSourceUpdate,
    onPersonalConfigIdUpdate,
    onSourceChange,
  }
}
