import { computed, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { warnNetworkFailure, warnOptionalApiFailure } from '../utils/workPanelApiUtils.js'
import {
  normalizeFeatureParamsSourceForSelect,
  readEnvVarSourcesAvailableFlag,
} from '../utils/envParamsSourceSelection.js'

/**
 * @param {{ editingTask: () => object|null, currentWorkspace: () => object|null, tenantId: () => string|number|null, show: () => boolean }} opts
 */
export function useCreateTaskFeatureParams({ editingTask, currentWorkspace, tenantId, show }) {
  const personalConfigs = ref([])
  const resolvedEnvPreview = ref({})
  const isEnvPreviewLoading = ref(false)
  const envPreviewExpanded = ref(false)
  /**
   * 公司/工作空间/个人三类环境变量是否至少存在一个可选项；
   * 未拉取/拉取失败时保持 true（fail-open，避免误禁导致无法创建任务）。
   */
  const featureParamsSourcesAvailable = ref(true)

  const featureParamsSourceModel = computed(() =>
    normalizeFeatureParamsSourceForSelect(editingTask()?.feature_params_source),
  )
  const selectedPersonalConfigIdModel = computed(() =>
    String(editingTask()?.personal_feature_params_config_id || '').trim(),
  )

  const onFeatureParamsSourceUpdate = (value) => {
    const task = editingTask()
    if (!task) return
    task.feature_params_source = normalizeFeatureParamsSourceForSelect(value)
    if (task.feature_params_source !== 'personal') {
      task.personal_feature_params_config_id = ''
      resolvedEnvPreview.value = {}
      envPreviewExpanded.value = false
    }
  }

  const onPersonalConfigIdUpdate = (value) => {
    const task = editingTask()
    if (!task) return
    task.personal_feature_params_config_id = String(value || '').trim()
    resolvedEnvPreview.value = {}
    envPreviewExpanded.value = false
  }

  const fetchPersonalConfigs = async () => {
    try {
      const resp = await apiFetch('/api/personal/feature-params-configs/', {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (resp.ok) {
        const data = await resp.json()
        personalConfigs.value = Array.isArray(data.configs) ? data.configs : []
      } else {
        warnOptionalApiFailure('personal feature-params-configs', resp)
        personalConfigs.value = []
      }
    } catch (error) {
      warnNetworkFailure('personal feature-params-configs', error)
      personalConfigs.value = []
    }
  }

  const onFeatureParamsSourceChange = () => {
    if (featureParamsSourceModel.value === 'personal') {
      void fetchPersonalConfigs()
    }
  }

  /**
   * 拉取公司/工作空间自定义环境变量可用性标志（OPT-20260809-019）与个人配置列表，
   * 计算三类来源是否至少一个可用。
   * 用 view=summary 拉取（data 脱敏、不含 provider API key），来源可用性由后端
   * env_var_sources_available 标志给出，无需再拉 full payload 计算 key 结构。
   * 缺 tenant/workspace id 或请求失败时保持可用（fail-open）。
   * 个人配置须先于可用性计算完成（await fetchPersonalConfigs），避免竞态误判。
   */
  const fetchFeatureParamsSourcesAvailability = async () => {
    const workspaceId = currentWorkspace()?.id != null ? String(currentWorkspace().id).trim() : ''
    const tenant = tenantId() != null ? String(tenantId()).trim() : ''
    if (!tenant || !workspaceId) return
    try {
      const resp = await apiFetch(
        `/api/cloud/feature-params/tenant_id/${tenant}/workspace_id/${workspaceId}?view=summary`,
        { credentials: 'include', headers: { Accept: 'application/json' } },
      )
      await fetchPersonalConfigs()
      if (resp.ok) {
        const body = await resp.json()
        // 响应缺少预期结构（异常/兼容场景）时保持可用（fail-open），避免误禁导致无法创建任务
        const flag = readEnvVarSourcesAvailableFlag(body)
        if (!flag) return
        featureParamsSourcesAvailable.value = Boolean(flag.company)
          || Boolean(flag.workspace)
          || personalConfigs.value.length > 0
      } else {
        warnOptionalApiFailure('feature-params sources availability', resp)
      }
    } catch (error) {
      warnNetworkFailure('feature-params sources availability', error)
    }
  }

  const fetchCreateTaskEnvPreview = async () => {
    const task = editingTask()
    const taskId = task?.id != null ? String(task.id).trim() : ''
    const workspaceId = currentWorkspace()?.id != null ? String(currentWorkspace().id).trim() : ''
    const tenant = tenantId() != null ? String(tenantId()).trim() : ''
    const configId = selectedPersonalConfigIdModel.value
    if (!taskId || !workspaceId || !tenant || !configId) return
    try {
      isEnvPreviewLoading.value = true
      const params = new URLSearchParams({
        source: 'personal',
        personal_config_id: configId,
      })
      const previewUrl = `/api/cloud/compute/feature-params-env-preview/tenant_id/${tenant}/workspace_id/${workspaceId}/task_id/${taskId}/?${params.toString()}`
      const resp = await apiFetch(previewUrl, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (resp.ok) {
        const data = await resp.json()
        resolvedEnvPreview.value = data.env && typeof data.env === 'object' ? data.env : {}
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

  const syncFeatureParamsFromEditingTask = () => {
    const task = editingTask()
    if (!task) return
    task.feature_params_source = normalizeFeatureParamsSourceForSelect(task.feature_params_source)
    if (typeof task.personal_feature_params_config_id !== 'string') {
      task.personal_feature_params_config_id = String(task.personal_feature_params_config_id || '').trim()
    }
    // 可用性拉取内部含个人配置列表加载，避免同一打开会话内重复请求
    void fetchFeatureParamsSourcesAvailability()
  }

  return {
    personalConfigs,
    resolvedEnvPreview,
    isEnvPreviewLoading,
    envPreviewExpanded,
    featureParamsSourcesAvailable,
    featureParamsSourceModel,
    selectedPersonalConfigIdModel,
    onFeatureParamsSourceUpdate,
    onPersonalConfigIdUpdate,
    onFeatureParamsSourceChange,
    fetchCreateTaskEnvPreview,
    syncFeatureParamsFromEditingTask,
  }
}
