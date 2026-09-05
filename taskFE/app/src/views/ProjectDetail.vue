<template>
  <div class="p-8">
    <!-- 加载状态 -->
    <div v-if="loading" class="flex justify-center items-center h-64">
      <div class="text-center">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto mb-4"></div>
        <p class="text-gray-600">加载中...</p>
      </div>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="bg-red-50 border-l-4 border-red-500 p-4 mb-6" data-testid="project-detail-error" :data-traceId="errorTraceId || undefined">
      <div class="flex">
        <div class="flex-shrink-0">
          <svg class="h-5 w-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-2.694-.833-3.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z"></path>
          </svg>
        </div>
        <div class="ml-3">
          <p class="text-sm text-red-700">{{ error }}</p>
        </div>
      </div>
    </div>

    <!-- 项目详情内容 -->
    <div v-else>
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-3xl font-bold">项目详情</h1>
        <div class="space-x-3 flex flex-wrap items-center gap-2">
          <button @click="editProject" class="bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded-md transition-colors">
            编辑项目
          </button>
          <button @click="deleteProject" class="bg-red-500 hover:bg-red-600 text-white px-4 py-2 rounded-md transition-colors" :disabled="deleteProjectGuard.isBusy()">
            删除项目
          </button>
          <EntityRevisionPanel
            v-if="revisionListUrl"
            :list-url="revisionListUrl"
            title-field="name"
          />
        </div>
      </div>

      <!-- 项目详情卡片 -->
      <div class="bg-white rounded-lg shadow-md p-6 mb-8">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <div class="mb-4">
              <h2 class="text-xl font-semibold mb-2">基本信息</h2>
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="block text-sm font-medium text-gray-700 mb-1">项目ID</label>
                  <p class="text-gray-900">{{ project.id }}</p>
                </div>
                <div>
                  <label class="block text-sm font-medium text-gray-700 mb-1">项目名称</label>
                  <p class="text-gray-900">{{ project.name }}</p>
                </div>
                <div class="col-span-2">
                  <label class="block text-sm font-medium text-gray-700 mb-1">项目描述</label>
                  <p class="text-gray-900 whitespace-pre-wrap">{{ project.description || '未设置' }}</p>
                </div>
                <ProjectDetailInlineEditableFields
                  field="tags"
                  class="col-span-2"
                  :project="project"
                  :tenant-id="String(route.params.tenant || '')"
                  :project-id="String(project.id || route.params.id || '')"
                  @saved="onInlineFieldSaved"
                />
                <div class="col-span-2" data-testid="project-run-template-summary">
                  <label class="block text-sm font-medium text-gray-700 mb-1">运行模版摘要</label>
                  <p class="text-gray-900" data-testid="project-run-template-summary-text">{{ runTemplateSummary }}</p>
                </div>
                <ProjectDetailInlineEditableFields
                  field="auto_run"
                  :project="project"
                  :tenant-id="String(route.params.tenant || '')"
                  :project-id="String(project.id || route.params.id || '')"
                  :git-auth-gate="autoRunGitGate"
                  @saved="onInlineFieldSaved"
                />
                <ProjectDetailGitReposSection
                  :project="project"
                  @oauth-callback-success="onOAuthCallbackSuccess"
                  @auto-run-git-gate="onAutoRunGitGate"
                />
              </div>
            </div>

            <ProjectDetailInlineEditableFields
              field="image"
              :project="project"
              :installed-images="installedImages"
              :referenced-container-image="referencedContainerImage"
              :tenant-id="String(route.params.tenant || '')"
              :project-id="String(project.id || route.params.id || '')"
              :live-run-template-getter="readLiveRunTemplate"
              :installed-images-loaded="installedImagesLoaded"
              @saved="onInlineFieldSaved"
              @request-load-images="loadInstalledImages"
              @draft-image-id="onDraftImageId"
            />
          </div>

          <div>
            <div class="mb-4">
              <h2 class="text-xl font-semibold mb-2">关联信息</h2>
              <div class="mb-3">
                <label class="block text-sm font-medium text-gray-700 mb-1">所属公司</label>
                <p class="text-gray-900" data-testid="project-company-name">{{ companyDisplayName }}</p>
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">关联工作空间</label>
                <div class="space-y-1">
                  <p v-for="workspace in workspaces" :key="workspace.id" class="text-gray-900">{{ workspace.name }}</p>
                  <p v-if="!workspaces || workspaces.length === 0" class="text-gray-500">未关联工作空间</p>
                </div>
              </div>
            </div>

            <div>
              <h2 class="text-xl font-semibold mb-2">时间信息</h2>
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="block text-sm font-medium text-gray-700 mb-1">创建时间</label>
                  <p class="text-gray-900">{{ formatDate(project.created_at) }}</p>
                </div>
                <div>
                  <label class="block text-sm font-medium text-gray-700 mb-1">更新时间</label>
                  <p class="text-gray-900">{{ formatDate(project.updated_at) }}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <ProjectRunTemplatePanel
        ref="runTemplatePanelRef"
        :tenant-id="String(route.params.tenant || '')"
        :project-id="String(project.id || route.params.id || '')"
        :project="project"
        :installed-images="installedImages"
        :draft-image-id="draftImageId"
        @saved="(data) => { project = data }"
        @spec-summary-change="onRunTemplateSpecSummaryChange"
      />

      <WorkspaceAssociation
        :project="project"
        :workspaces="workspaces"
        :available-workspaces="availableWorkspaces"
        :loading="loading"
        :tenant-id="route.params.tenant"
        @update:project="(updatedProject) => project = updatedProject"
        @update:workspaces="async (updatedWorkspaceIds) => {
          await fetchWorkspaces(updatedWorkspaceIds);
          await fetchAvailableWorkspaces();
        }"
        @update:available-workspaces="(updatedWorkspaces) => availableWorkspaces = updatedWorkspaces"
      />
    </div>
  </div>
</template>

<script setup>
import { apiFetch } from '../utils/apiUtils.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { extractTraceId } from '../utils/traceId.js'
import { projectDetailFetchErrorMessage } from '../utils/projectDetailErrors.js'
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import WorkspaceAssociation from '../components/WorkspaceAssociation.vue'
import ProjectRunTemplatePanel from '../components/ProjectRunTemplatePanel.vue'
import ProjectDetailInlineEditableFields from '../components/ProjectDetailInlineEditableFields.vue'
import ProjectDetailGitReposSection from '../components/ProjectDetailGitReposSection.vue'
import EntityRevisionPanel from '../components/entity-revision/EntityRevisionPanel.vue'
import { runTemplateIsHardwareComplete, summarizeRunTemplateWithPrice } from '../utils/projectRunTemplateUtils.js'
import {
  normalizeInstalledImageList,
  resolveInstalledImageById,
} from '../utils/installedImageDisplay.js'
import { coerceSnowflakeId, readJsonPreservingSnowflakeIds } from '../utils/snowflakeId.js'
import { fetchTenantCompanyDisplayName } from '../utils/companyDisplayName.js'
import { rememberCreateTaskPreferredProject } from '../utils/createTaskPreferredProject.js'

const route = useRoute()
const router = useRouter()

const revisionListUrl = computed(() => {
  const tid = String(route.params.tenant || '').trim()
  const pid = String(project.value?.id || route.params.id || '').trim()
  if (!tid || !pid) return ''
  return `/api/projects/tenant_id/${tid}/${pid}/revisions/`
})

const project = ref({
  id: '',
  name: '',
  description: '',
  git_repos: [],
  git_repos_status: [],
  container_image: '',
  container_image_id: '',
  server_run_template: {},
  tags: [],
  workspaces: [],
  company: '',
  created_at: '',
  updated_at: '',
})

/** 硬件面板回传的实时规格摘要（含价格）；用于「运行模版摘要」一眼展示价格 */
const liveRunTemplateSpecSummary = ref('')
const runTemplatePanelRef = ref(null)
const draftImageId = ref('')

const onDraftImageId = (id) => {
  draftImageId.value = String(id || '').trim()
}

const readLiveRunTemplate = () => {
  const saved =
    project.value?.server_run_template && typeof project.value.server_run_template === 'object'
      ? project.value.server_run_template
      : {}
  const live = runTemplatePanelRef.value?.buildPayload?.()
  // 硬件面板常返回「有地域无实例」半成品；更换镜像须优先完整草稿，否则回退已保存模版
  if (live && typeof live === 'object' && runTemplateIsHardwareComplete(live)) return live
  if (runTemplateIsHardwareComplete(saved)) return saved
  if (live && typeof live === 'object' && Object.keys(live).length) return live
  return saved
}

const runTemplateSummary = computed(() =>
  summarizeRunTemplateWithPrice(
    project.value?.server_run_template,
    liveRunTemplateSpecSummary.value,
  ),
)

const onRunTemplateSpecSummaryChange = (summary) => {
  liveRunTemplateSpecSummary.value = String(summary || '').trim()
}

const companyName = ref('')
const companyDisplayName = computed(() => String(companyName.value || '').trim() || '未设置')

const workspaces = ref([])
const availableWorkspaces = ref([])
const installedImages = ref([])
const installedImagesLoaded = ref(false)
const referencedContainerImage = ref(null)
const loading = ref(true)
const error = ref('')
const errorTraceId = ref('')
// OPT-20260819-038: 删除项目是写操作，防连点/超时重试双发 DELETE
const deleteProjectGuard = createClickGuard()
const autoRunGitGate = ref({
  blocked: false,
  code: '',
  message: '',
  displayLabel: '',
  traceId: '',
})

const onAutoRunGitGate = (gate) => {
  autoRunGitGate.value = gate && typeof gate === 'object'
    ? gate
    : { blocked: false, code: '', message: '', displayLabel: '', traceId: '' }
}

const onInlineFieldSaved = (data) => {
  if (!data || typeof data !== 'object') return
  project.value = {
    ...project.value,
    ...data,
    container_image_id: coerceSnowflakeId(data?.container_image_id ?? project.value.container_image_id),
  }
  const tenantId = route.params.tenant
  if (tenantId) {
    void resolveReferencedContainerImage(tenantId, project.value.container_image_id)
  }
}

const onOAuthCallbackSuccess = async () => {
  await fetchProjectDetail()
}

const fetchProjectDetail = async () => {
  try {
    loading.value = true
    error.value = ''
    errorTraceId.value = ''
    const projectId = route.params.id
    const tenantId = route.params.tenant

    if (!tenantId) {
      throw new Error('缺少租户ID')
    }

    companyName.value = ''
    // taskProjectService handleProjectsRoute：projectId 位置段在前、kv 键值对在后
    const [response, fetchedCompanyName] = await Promise.all([
      apiFetch(`/api/projects/${projectId}/tenant_id/${tenantId}/`, {
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      }),
      fetchTenantCompanyDisplayName(apiFetch, tenantId),
    ])
    companyName.value = fetchedCompanyName

    if (response.ok) {
      const data = await response.json()
      project.value = {
        ...data,
        container_image_id: coerceSnowflakeId(data?.container_image_id),
        git_repos_status: Array.isArray(data.git_repos_status) ? data.git_repos_status : [],
      }
      rememberCreateTaskPreferredProject(tenantId, project.value.id)
      await fetchWorkspaces(data.workspaces)
      if (installedImages.value.length > 0) {
        await resolveReferencedContainerImage(tenantId, project.value.container_image_id)
      }
    } else {
      const errorData = await response.json().catch(() => ({}))
      errorTraceId.value = extractTraceId(response) || extractTraceId(errorData) || ''
      throw new Error(projectDetailFetchErrorMessage(response.status, errorData.detail))
    }
  } catch (err) {
    error.value = err.message
    errorTraceId.value = errorTraceId.value || extractTraceId(err) || ''
    console.error('Error fetching project detail:', err)
  } finally {
    loading.value = false
  }
}

const fetchWorkspaces = async (workspaceIds) => {
  try {
    if (workspaceIds && workspaceIds.length > 0) {
      const tenantId = route.params.tenant
      if (!tenantId) {
        console.error('缺少租户ID')
        return
      }

      const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenantId}?ids=${workspaceIds.join(',')}`, {
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      })

      if (response.ok) {
        const data = await response.json()
        workspaces.value = Array.isArray(data) ? data : data.results || []
      }
    } else {
      workspaces.value = []
    }
  } catch (err) {
    console.error('Error fetching workspaces:', err)
  }
}

const resolveReferencedContainerImage = async (tenantId, imageId) => {
  const normalizedId = coerceSnowflakeId(imageId)
  if (!normalizedId || resolveInstalledImageById(installedImages.value, normalizedId)) {
    referencedContainerImage.value = null
    return
  }
  try {
    const response = await apiFetch(`/api/cloud/installed-images/tenant_id/${tenantId}/${normalizedId}/`, {
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
    })
    if (!response.ok) {
      referencedContainerImage.value = null
      return
    }
    const data = await readJsonPreservingSnowflakeIds(response)
    referencedContainerImage.value =
      data && typeof data === 'object'
        ? { ...data, id: coerceSnowflakeId(data.id) }
        : null
  } catch (err) {
    referencedContainerImage.value = null
    console.error('Error fetching referenced installed image:', err)
  }
}

const loadInstalledImages = async () => {
  try {
    const tenantId = route.params.tenant
    if (!tenantId) return
    const response = await apiFetch(`/api/cloud/installed-images/tenant_id/${tenantId}`, {
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
    })
    if (!response.ok) return
    const data = await readJsonPreservingSnowflakeIds(response)
    installedImages.value = normalizeInstalledImageList(data)
    await resolveReferencedContainerImage(tenantId, project.value?.container_image_id)
  } catch (err) {
    console.error('Error fetching installed images:', err)
  } finally {
    installedImagesLoaded.value = true
  }
}

const fetchAvailableWorkspaces = async () => {
  try {
    const tenantId = route.params.tenant
    if (!tenantId) {
      console.error('缺少租户ID')
      return
    }

    const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenantId}`, {
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
    })

    if (response.ok) {
      const data = await response.json()
      const allWorkspaces = Array.isArray(data) ? data : data.results || []
      availableWorkspaces.value = allWorkspaces.filter((ws) =>
        !project.value.workspaces.includes(ws.id),
      )
    }
  } catch (err) {
    console.error('Error fetching available workspaces:', err)
  }
}

const formatDate = (dateString) => {
  if (!dateString) {
    return '未设置'
  }
  try {
    const date = new Date(dateString)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  } catch (err) {
    console.error('Error formatting date:', err)
    return dateString
  }
}

const editProject = () => {
  router.push(`/tenant/${route.params.tenant}/projects/${project.value.id}/edit/`)
}

const deleteProject = async () => {
  if (confirm('确定要删除这个项目吗？')) {
    // OPT-20260819-038: 删除项目是写操作，防连点/超时重试双发 DELETE
    await deleteProjectGuard.run(async ({ idempotencyKey }) => {
      try {
        loading.value = true
        errorTraceId.value = ''
        const tenantId = route.params.tenant
        const response = await apiFetch(`/api/projects/${project.value.id}/tenant_id/${tenantId}/`, {
          method: 'DELETE',
          credentials: 'include',
          headers: mergeIdempotencyHeaders({}, idempotencyKey),
        })

        if (response.ok) {
          router.push(`/tenant/${tenantId}/projects/`)
        } else {
          const errorData = await response.json().catch(() => ({}))
          errorTraceId.value = extractTraceId(response) || extractTraceId(errorData) || ''
          throw new Error(errorData.detail || '删除项目失败')
        }
      } catch (err) {
        error.value = err.message
        errorTraceId.value = errorTraceId.value || extractTraceId(err) || ''
        console.error('Error deleting project:', err)
      } finally {
        loading.value = false
      }
    })
  }
}

onMounted(async () => {
  await Promise.all([fetchProjectDetail(), loadInstalledImages()])
  await fetchAvailableWorkspaces()
})
</script>

<style scoped></style>
