import { ref, computed } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { extractTraceId } from '../utils/traceId.js'
import { normalizeProjectTags } from '../utils/projectTagsUtils.js'
import { mergeDaydaymoneyTags, parseDaydaymoneyYaml } from '../utils/daydaymoneyMeta.js'
import {
  collectSessionGrantTickets,
  consumeSessionGrantTickets,
  gitRepoUrlsFromPayload,
} from '../utils/grantTicketSession.js'

/**
 * CreateProject form state + submit/validate helpers (line-limit split from CreateProject.vue).
 */
export function useCreateProjectForm({
  route,
  router,
  gitRepoRows,
  trimmedGitRepoPayload,
  duplicateCloneAliasError,
  gitReposHaveValidUrls,
  gitReposPendingValidation,
  appendGitRepoFormErrors,
  runTemplatePanelRef,
}) {
  const formData = ref({
    name: '',
    description: '',
    container_image: '',
    workspace: '',
    tags: [],
    auto_clone_nested_repos: true,
  })

  const loading = ref(false)
  const errors = ref({})
  const generalError = ref('')
  const generalErrorTraceId = ref('')
  const tagsSyncError = ref('')
  const installedImages = ref([])
  const workspaces = ref([])

  const hasAnyGitRepoUrl = computed(() =>
    (gitRepoRows.value || []).some((row) => String(row?.url || '').trim()),
  )

  const runTemplateProjectSnapshot = computed(() => ({
    workspaces: formData.value.workspace ? [formData.value.workspace] : [],
    server_run_template: {},
  }))

  const isFormValid = () => {
    if (!formData.value.name.trim()) return false
    if (!formData.value.description?.trim()) return false
    if (!formData.value.workspace) return false
    return gitReposHaveValidUrls()
  }

  const createButtonDisabled = computed(() => {
    if (loading.value) return true
    if (!isFormValid()) return true
    return gitReposPendingValidation.value
  })

  const createButtonDisabledReason = computed(() => {
    if (loading.value) return '创建中，请稍候'
    if (!formData.value.name.trim()) return '请填写项目名称'
    if (!formData.value.description?.trim()) return '请填写项目描述'
    if (!formData.value.workspace) return '请选择工作空间'
    if (!gitReposHaveValidUrls()) return '请修正 Git 仓库地址'
    if (gitReposPendingValidation.value) return '正在校验 Git 仓库，请稍候'
    return ''
  })

  const syncTagsFromAidevYaml = () => {
    tagsSyncError.value = ''
    const raw = window.prompt('请粘贴 daydaymoney.yaml 内容：')
    if (raw == null) return
    try {
      const meta = parseDaydaymoneyYaml(raw)
      formData.value.tags = mergeDaydaymoneyTags(formData.value.tags, meta.tags, meta.service_id)
    } catch (e) {
      tagsSyncError.value = e?.message || 'daydaymoney.yaml 解析失败'
    }
  }

  const validateForm = () => {
    const newErrors = {}
    generalError.value = ''
    generalErrorTraceId.value = ''

    if (!formData.value.name.trim()) {
      newErrors.name = '项目名称不能为空'
    } else if (formData.value.name.length > 255) {
      newErrors.name = '项目名称不能超过255个字符'
    }

    if (!formData.value.description || !formData.value.description.trim()) {
      newErrors.description = '项目描述不能为空'
    } else if (formData.value.description.length > 2000) {
      newErrors.description = '项目描述不能超过2000个字符'
    }

    appendGitRepoFormErrors(newErrors)
    const aliasDup = duplicateCloneAliasError()
    if (aliasDup) {
      newErrors.git_repos = aliasDup
    }

    if (!formData.value.workspace) {
      newErrors.workspace = '请选择工作空间'
    }

    errors.value = newErrors
    return Object.keys(newErrors).length === 0
  }

  const loadInstalledImages = async () => {
    try {
      const tenantId = route.params.tenant
      if (!tenantId) return
      const response = await apiFetch(`/api/cloud/installed-images/tenant_id/${tenantId}`, {
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
      })
      if (response.ok) {
        const data = await response.json()
        installedImages.value = Array.isArray(data) ? data : data.results || []
      }
    } catch (err) {
      console.error('加载已安装镜像失败:', err)
    }
  }

  const loadWorkspaces = async () => {
    try {
      const tenantId = route.params.tenant
      if (!tenantId) {
        console.error('缺少租户ID')
        return
      }
      const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenantId}`)
      if (response.ok) {
        workspaces.value = await response.json()
      }
    } catch (err) {
      console.error('加载工作空间失败:', err)
    }
  }

  const parseApiErrors = (errorData) => {
    const newErrors = {}
    if (typeof errorData !== 'object' || errorData === null) return newErrors

    const fieldMap = {
      name: 'name',
      description: 'description',
      git_repos: 'git_repos',
      workspaces_ids: 'workspace',
      container_image_id: 'container_image',
      tags: 'tags',
    }

    for (const [apiField, formField] of Object.entries(fieldMap)) {
      const msg = errorData[apiField]
      if (Array.isArray(msg) && msg.length > 0) {
        newErrors[formField] = msg[0]
      } else if (typeof msg === 'string') {
        newErrors[formField] = msg
      }
    }
    return newErrors
  }

  const submitForm = async () => {
    if (!validateForm()) {
      generalError.value = '请完善必填项后再提交'
      return
    }

    errors.value = {}
    generalError.value = ''
    generalErrorTraceId.value = ''
    loading.value = true

    try {
      const repos = trimmedGitRepoPayload()
      const runTemplate = runTemplatePanelRef.value?.buildPayload?.() ?? {}
      const requestData = {
        name: formData.value.name.trim(),
        description: formData.value.description?.trim() || '',
        git_repos: repos,
        workspace: formData.value.workspace,
        tags: normalizeProjectTags(formData.value.tags),
        server_run_template: runTemplate,
        auto_clone_nested_repos: !!formData.value.auto_clone_nested_repos,
      }

      if (formData.value.container_image) {
        requestData.container_image_id = formData.value.container_image
        const matched = installedImages.value.find(
          (image) => String(image.id) === String(formData.value.container_image),
        )
        if (matched?.name) {
          requestData.container_image = matched.name
        }
      }

      const body = { ...requestData }
      if (body.workspace) {
        body.workspaces_ids = [body.workspace]
        delete body.workspace
      }
      if (!body.container_image_id) {
        delete body.container_image_id
      }

      const tickets = collectSessionGrantTickets(
        gitRepoUrlsFromPayload(repos),
        [route?.query?.grant_ticket],
      )
      if (tickets.length) {
        body.grant_ticket = tickets[0]
        body.grant_tickets = tickets
      }

      const response = await apiFetch(`/api/projects/tenant_id/${route.params.tenant}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(body),
      })

      if (response.ok) {
        // OPT-20260902-025：随 create-project 提交的 grant_ticket 已消费，从 session 移除
        if (tickets.length) consumeSessionGrantTickets(tickets)
        const created = await response.json().catch(() => ({}))
        if (created?.id) {
          router.push(`/tenant/${route.params.tenant}/projects/`)
          return
        }
        generalErrorTraceId.value = extractTraceId(response) || ''
        const detail = created.detail || created.message || created.error
        generalError.value = detail || '创建项目失败：响应缺少项目 id'
        return
      }

      const errorData = response._errorData ?? (await response.json().catch(() => ({})))
      const fieldErrors = parseApiErrors(errorData)
      errors.value = fieldErrors
      generalErrorTraceId.value = extractTraceId(response) || ''

      if (Object.keys(fieldErrors).length > 0) {
        generalError.value = '请根据提示修正表单后再提交'
      } else {
        const detail = errorData.detail || errorData.message || errorData.error
        const message = Array.isArray(detail) ? detail.join('; ') : detail || '创建项目失败'
        generalError.value = message
      }
    } catch (err) {
      console.error('创建项目失败:', err)
      generalError.value = '网络错误，请检查网络连接后重试'
      generalErrorTraceId.value = extractTraceId(err) || ''
      showRequestError('网络错误，请检查网络连接后重试', err)
    } finally {
      loading.value = false
    }
  }

  return {
    formData,
    loading,
    errors,
    generalError,
    generalErrorTraceId,
    tagsSyncError,
    installedImages,
    workspaces,
    hasAnyGitRepoUrl,
    runTemplateProjectSnapshot,
    createButtonDisabled,
    createButtonDisabledReason,
    syncTagsFromAidevYaml,
    loadInstalledImages,
    loadWorkspaces,
    submitForm,
  }
}
