import { ref, computed } from 'vue'
import { extractTraceId } from '../utils/traceId.js'
import { apiFetch } from '../utils/apiUtils.js'
import { fetchWorkspaceCloudPlatforms } from '../utils/workspaceCloudPlatformsApi.js'
import { defaultConfigToRunTemplate, summarizeRunTemplate } from '../utils/projectRunTemplateUtils.js'

export function useProjectRunTemplate({ tenantId, projectId, workspaceId, initialTemplate = null }) {
  const resolveTenantId = () => {
    if (typeof tenantId === 'function') return String(tenantId() || '').trim()
    return String(tenantId?.value ?? tenantId ?? '').trim()
  }
  const resolveProjectId = () => {
    if (typeof projectId === 'function') return String(projectId() || '').trim()
    return String(projectId?.value ?? projectId ?? '').trim()
  }
  const resolveWorkspaceId = () => {
    if (typeof workspaceId === 'function') return String(workspaceId() || '').trim()
    return String(workspaceId?.value ?? workspaceId ?? '').trim()
  }
  const loading = ref(false)
  const saving = ref(false)
  const error = ref('')
  const errorTraceId = ref('')
  const templateOptions = ref([])
  const selectedTemplateId = ref('')
  const draftTemplate = ref(initialTemplate && typeof initialTemplate === 'object' ? { ...initialTemplate } : {})
  const hardwareDraft = ref({
    cpu_cores: String(initialTemplate?.hardware_config?.cpu_cores ?? '1'),
    memory_gb: String(initialTemplate?.hardware_config?.memory_gb ?? '1'),
    storage_gb: String(initialTemplate?.hardware_config?.storage_gb ?? '40'),
  })

  const currentSummary = computed(() => summarizeRunTemplate(draftTemplate.value))

  const syncHardwareDraftFromTemplate = (tpl) => {
    const hw = tpl?.hardware_config || {}
    hardwareDraft.value = {
      cpu_cores: String(hw.cpu_cores ?? '1'),
      memory_gb: String(hw.memory_gb ?? '1'),
      storage_gb: String(hw.storage_gb ?? '40'),
    }
  }

  const loadTemplateOptions = async () => {
    const tid = resolveTenantId()
    if (!tid) return
    loading.value = true
    error.value = ''
    errorTraceId.value = ''
    try {
      const defaultsRes = await apiFetch(`/api/cloud/server-config-default/tenant_id/${tid}/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      const defaultsPayload = defaultsRes.ok ? await defaultsRes.json().catch(() => ({})) : {}
      const defaults = Array.isArray(defaultsPayload?.data) ? defaultsPayload.data : []

      const wid = resolveWorkspaceId()
      const platforms = wid ? await fetchWorkspaceCloudPlatforms(tid, wid) : []

      const platformIdByAuth = {}
      const platformIdByType = {}
      for (const p of platforms) {
        const authId = String(p?.authorization_id ?? p?.iam_id ?? '').trim()
        if (authId && p?.id != null) {
          platformIdByAuth[authId] = String(p.id)
        }
        const platformType = String(p?.platform_type || '').trim()
        if (platformType && p?.id != null && !platformIdByType[platformType]) {
          platformIdByType[platformType] = String(p.id)
        }
      }

      templateOptions.value = defaults.map((item) => {
        const authId = String(item.authorization_id || '').trim()
        const platformType = String(item.platform || '').trim()
        const cloudPlatformId =
          platformIdByAuth[authId]
          || platformIdByType[platformType]
          || (platforms.length === 1 ? String(platforms[0].id) : '')
        const tpl = defaultConfigToRunTemplate(item, cloudPlatformId)
        return {
          id: String(item.id ?? authId ?? tpl.label),
          label: tpl.label,
          template: tpl,
        }
      })

      const currentId = String(draftTemplate.value?.template_id || '').trim()
      if (currentId && templateOptions.value.some((o) => o.id === currentId)) {
        selectedTemplateId.value = currentId
      }
    } catch (e) {
      error.value = e?.message || '加载服务器模版失败'
      errorTraceId.value = extractTraceId(e)
    } finally {
      loading.value = false
    }
  }

  const applySelectedTemplate = () => {
    const opt = templateOptions.value.find((o) => o.id === selectedTemplateId.value)
    if (!opt) return
    draftTemplate.value = {
      ...opt.template,
      hardware_config: {
        cpu_cores: hardwareDraft.value.cpu_cores,
        memory_gb: hardwareDraft.value.memory_gb,
        storage_gb: hardwareDraft.value.storage_gb,
      },
    }
  }

  const clearTemplate = () => {
    selectedTemplateId.value = ''
    draftTemplate.value = {}
    hardwareDraft.value = { cpu_cores: '1', memory_gb: '1', storage_gb: '40' }
  }

  const buildPayload = () => {
    if (!selectedTemplateId.value && !Object.keys(draftTemplate.value || {}).length) {
      return {}
    }
    const base = selectedTemplateId.value
      ? (templateOptions.value.find((o) => o.id === selectedTemplateId.value)?.template || draftTemplate.value)
      : draftTemplate.value
    return {
      ...base,
      hardware_config: {
        cpu_cores: hardwareDraft.value.cpu_cores,
        memory_gb: hardwareDraft.value.memory_gb,
        storage_gb: hardwareDraft.value.storage_gb,
      },
    }
  }

  const saveTemplate = async () => {
    const tid = resolveTenantId()
    const pid = resolveProjectId()
    if (!tid || !pid) {
      throw new Error('缺少租户或项目 ID')
    }
    saving.value = true
    error.value = ''
    errorTraceId.value = ''
    try {
      const response = await apiFetch(`/api/projects/${pid}/tenant_id/${tid}/`, {
        method: 'PATCH',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify({
          server_run_template: buildPayload(),
        }),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        errorTraceId.value = extractTraceId(response) || extractTraceId(data)
        throw new Error(typeof data?.detail === 'string' ? data.detail : '保存项目运行模版失败')
      }
      draftTemplate.value = data.server_run_template || buildPayload()
      syncHardwareDraftFromTemplate(draftTemplate.value)
      return data
    } catch (e) {
      errorTraceId.value = errorTraceId.value || extractTraceId(e)
      throw e
    } finally {
      saving.value = false
    }
  }

  const setFromProject = (project) => {
    const tpl = project?.server_run_template
    draftTemplate.value = tpl && typeof tpl === 'object' ? { ...tpl } : {}
    selectedTemplateId.value = String(draftTemplate.value.template_id || '').trim()
    syncHardwareDraftFromTemplate(draftTemplate.value)
  }

  return {
    loading,
    saving,
    error,
    errorTraceId,
    templateOptions,
    selectedTemplateId,
    draftTemplate,
    hardwareDraft,
    currentSummary,
    loadTemplateOptions,
    applySelectedTemplate,
    clearTemplate,
    buildPayload,
    saveTemplate,
    setFromProject,
    syncHardwareDraftFromTemplate,
  }
}
