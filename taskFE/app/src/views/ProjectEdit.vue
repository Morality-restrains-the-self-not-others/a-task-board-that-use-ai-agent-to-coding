<template>
  <div class="p-8">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-3xl font-bold">编辑项目</h1>
      <div class="flex items-center gap-2">
        <EntityRevisionPanel
          v-if="revisionListUrl"
          :list-url="revisionListUrl"
          title-field="name"
        />
        <button @click="cancelEdit" class="bg-gray-500 hover:bg-gray-600 text-white px-4 py-2 rounded-md transition-colors">
          取消
        </button>
      </div>
    </div>

    <div
      v-if="generalError"
      class="mb-6 p-4 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
      role="alert"
      :data-traceId="generalErrorTraceId || undefined"
    >
      {{ generalError }}
    </div>

    <!-- 编辑表单 -->
    <div class="bg-white rounded-lg shadow-md p-6">
      <form @submit.prevent="updateProject">
        <div class="space-y-6">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div class="md:col-span-2">
              <label for="projectName" class="block text-sm font-medium text-gray-700 mb-1">项目名称 *</label>
              <input
                type="text"
                id="projectName"
                v-model="formData.name"
                class="w-full border border-gray-300 rounded-md px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                placeholder="请输入项目名称"
                required
              />
            </div>

            <div class="md:col-span-2">
              <ProjectTagsInput v-model="formData.tags" />
            </div>

            <div class="md:col-span-2">
              <div class="flex items-center justify-between mb-1">
                <span class="block text-sm font-medium text-gray-700">Git 仓库（可选，可多个）</span>
                <button type="button" class="text-sm text-blue-600 hover:underline" @click="addGitRepoRow">
                  添加仓库
                </button>
              </div>
              <div class="space-y-2">
                <div v-for="(row, index) in gitRepoRows" :key="row.id" class="flex gap-2 items-start">
                  <div class="flex-1 min-w-0">
                    <div class="flex gap-2">
                      <input
                        :id="index === 0 ? 'gitRepo0' : undefined"
                        v-model="row.url"
                        type="text"
                        :class="['flex-1 min-w-0 border rounded-md px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent', gitRepoRowFormatErrors[row.id] ? 'border-red-500' : 'border-gray-300']"
                        placeholder="Git 仓库 URL（https://、git@主机:路径 或 ssh://）"
                        data-testid="git-repo-url-input"
                        @blur="refreshGitRepoFormatErrors"
                      />
                      <input
                        v-model="row.cloneAlias"
                        type="text"
                        class="w-40 shrink-0 border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                        placeholder="别名（可选）"
                        data-testid="git-repo-clone-alias-input"
                        :aria-label="`仓库别名 ${index + 1}`"
                      />
                    </div>
                    <template v-if="gitRepoRowFormatErrors[row.id] === INVALID_REPO_URL_MSG">
                      <p
                        class="mt-1 text-sm text-red-600"
                        role="alert"
                        data-testid="git-repo-url-format-hint"
                      >
                        {{ gitRepoRowFormatErrors[row.id] }}
                      </p>
                      <p class="mt-1 text-xs text-gray-500">
                        正确格式示例：
                        <span v-for="(example, exampleIndex) in GIT_REPO_URL_EXAMPLES" :key="example">
                          <code class="text-xs bg-gray-100 px-1 py-0.5 rounded">{{ example }}</code><span v-if="exampleIndex < GIT_REPO_URL_EXAMPLES.length - 1">、</span>
                        </span>
                      </p>
                    </template>
                  </div>
                  <button
                    v-if="gitRepoRows.length > 1"
                    type="button"
                    class="mt-2 shrink-0 text-sm text-red-600 hover:underline"
                    @click="removeGitRepoRow(row.id)"
                  >
                    移除
                  </button>
                </div>
              </div>
              <p class="mt-2 text-xs text-gray-500">{{ GIT_REPO_URL_FORMAT_HINT }}</p>
            </div>
          </div>

          <div>
            <label for="projectDescription" class="block text-sm font-medium text-gray-700 mb-1">项目描述</label>
            <textarea
              id="projectDescription"
              v-model="formData.description"
              rows="4"
              class="w-full border border-gray-300 rounded-md px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="请输入项目描述"
            ></textarea>
          </div>

          <div>
            <label for="containerImage" class="block text-sm font-medium text-gray-700 mb-1">已安装镜像</label>
            <select
              id="containerImage"
              v-model="formData.container_image_id"
              class="w-full border border-gray-300 rounded-md px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              @click="loadInstalledImages"
            >
              <option value="">未设置</option>
              <option v-if="isLoadingImages" disabled>加载中...</option>
              <option v-else v-for="image in installedImages" :key="String(image.id)" :value="String(image.id)">
                {{ image.name }}
                <template v-if="image.version"> ({{ image.version }})</template>
                <template v-if="formatContainerImageArchitectures(image)"> · 架构 {{ formatContainerImageArchitectures(image) }}</template>
                <template v-if="image.hardware_summary"> - {{ image.hardware_summary }}</template>
              </option>
            </select>
            <p v-if="selectedImageArchitectureLabel" class="mt-1 text-xs text-gray-500">
              支持 CPU 架构：{{ selectedImageArchitectureLabel }}
            </p>
          </div>

          <div>
            <label
              class="flex items-start gap-2 cursor-pointer"
              :class="{ 'opacity-60 cursor-not-allowed': !canEnableAutoRun }"
            >
              <input
                id="project-default-auto-run"
                type="checkbox"
                class="mt-1 h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                v-model="formData.default_auto_run"
                :disabled="!canEnableAutoRun"
                data-testid="project-default-auto-run"
              />
              <span class="text-sm">
                <span class="font-medium text-gray-700">是否允许自动运行</span>
                <span v-if="canEnableAutoRun" class="block text-xs text-gray-500 mt-0.5">
                  开启后，创建任务允许勾选自动运行并按下方运行模版启动机器节点；关闭后创建任务将不允许自动启动机器节点
                </span>
                <span v-else class="block text-xs text-gray-500 mt-0.5">
                  {{ autoRunDisabledHint }}
                </span>
              </span>
            </label>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-2">关联工作空间</label>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div
                v-for="workspace in allWorkspaces"
                :key="workspace.id"
                class="flex items-center p-3 border border-gray-200 rounded-md hover:bg-gray-50 cursor-pointer"
              >
                <input
                  type="checkbox"
                  v-model="formData.workspaces_ids"
                  :value="workspace.id"
                  class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <label class="ml-2 text-sm text-gray-900 cursor-pointer">{{ workspace.name }} ({{ workspace.id }})</label>
              </div>
            </div>
          </div>

          <ProjectRunTemplatePanel
            ref="runTemplatePanelRef"
            :tenant-id="String(route.params.tenant || '')"
            :project-id="String(route.params.id || '')"
            :project="projectSnapshotForPanel"
            :installed-images="installedImages"
            hide-actions
          />
          <p class="text-xs text-gray-500 -mt-4">
            配置运行模版后，在工作面板创建任务时可勾选「是否自动运行」，系统将按此模版自动启动服务器。
          </p>
        </div>

        <div class="mt-6">
          <button
            type="submit"
            class="bg-blue-500 hover:bg-blue-600 text-white px-6 py-3 rounded-md transition-colors font-medium disabled:opacity-50"
            :disabled="isSaving || updateProjectGuard.isBusy()"
          >
            {{ isSaving ? '保存中...' : '保存更改' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { apiFetch } from '../utils/apiUtils.js';
import { extractTraceId } from '../utils/traceId.js';
import { normalizeProjectTags } from '../utils/projectTagsUtils.js';
import { projectHasConfiguredRunTemplate, runTemplateIsHardwareComplete } from '../utils/projectRunTemplateUtils.js';
import { formatContainerImageArchitectures } from '../utils/containerImageArchitecture.js';
import { readJsonPreservingSnowflakeIds } from '../utils/snowflakeId.js';
import { normalizeInstalledImageList } from '../utils/installedImageDisplay.js';
import ProjectTagsInput from '../components/ProjectTagsInput.vue';
import ProjectRunTemplatePanel from '../components/ProjectRunTemplatePanel.vue';
import EntityRevisionPanel from '../components/entity-revision/EntityRevisionPanel.vue';

import { ref, onMounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { useProjectGitRepoRows } from '../composables/useProjectGitRepoRows.js'

const route = useRoute()
const router = useRouter()

const revisionListUrl = computed(() => {
  const tid = String(route.params.tenant || '').trim()
  const pid = String(route.params.id || '').trim()
  if (!tid || !pid) return ''
  return `/api/projects/tenant_id/${tid}/${pid}/revisions/`
})

const formData = ref({
  name: '',
  description: '',
  container_image_id: '',
  workspaces_ids: [],
  tags: [],
  default_auto_run: false,
})

const projectSnapshot = ref(null)
const runTemplatePanelRef = ref(null)

/** 编辑页勾选工作空间会实时反映到运行模版面板（首项决定云平台上下文） */
const projectSnapshotForPanel = computed(() => {
  if (!projectSnapshot.value) return null
  return {
    ...projectSnapshot.value,
    container_image_id: formData.value.container_image_id,
    workspaces: Array.isArray(formData.value.workspaces_ids)
      ? [...formData.value.workspaces_ids]
      : [],
  }
})

const hasConfiguredRunTemplate = computed(() => {
  const panelPayload = runTemplatePanelRef.value?.buildPayload?.()
  if (runTemplateIsHardwareComplete(panelPayload)) {
    return true
  }
  return runTemplateIsHardwareComplete(projectSnapshot.value?.server_run_template)
    || projectHasConfiguredRunTemplate(projectSnapshot.value)
})

const canEnableAutoRun = computed(() => {
  const imageId = String(formData.value.container_image_id || '').trim()
  return Boolean(imageId) && hasConfiguredRunTemplate.value
})

const autoRunDisabledHint = computed(() => {
  if (!String(formData.value.container_image_id || '').trim()) {
    return '请先选择已安装镜像'
  }
  return '请先配置项目运行模版（云平台、地域与实例）后再启用'
})

watch(canEnableAutoRun, (enabled) => {
  if (!enabled) {
    formData.value.default_auto_run = false
  }
})
const generalError = ref('')
const generalErrorTraceId = ref('')
const isSaving = ref(false)
// OPT-20260819-038: 更新项目是写操作，防连点/超时重试双发 PUT
const updateProjectGuard = createClickGuard()

const {
  gitRepoRows,
  gitRepoRowFormatErrors,
  trimmedGitRepoPayload,
  duplicateCloneAliasError,
  refreshGitRepoFormatErrors,
  addGitRepoRow,
  removeGitRepoRow,
  setGitRepoRowsFromApi,
  GIT_REPO_URL_EXAMPLES,
  GIT_REPO_URL_FORMAT_HINT,
  INVALID_REPO_URL_MSG,
} = useProjectGitRepoRows()

const installedImages = ref([])
const isLoadingImages = ref(false)
const hasLoadedImages = ref(false)
const allWorkspaces = ref([])

const selectedImageArchitectureLabel = computed(() => {
  const imageId = String(formData.value.container_image_id || '').trim()
  if (!imageId) return ''
  const matched = installedImages.value.find((image) => String(image.id) === imageId)
  return formatContainerImageArchitectures(matched)
})

const parseApiError = async (response) => {
  const data = await response.json().catch(() => ({}))
  if (typeof data?.detail === 'string') return data.detail
  if (data && typeof data === 'object') {
    const parts = Object.entries(data).flatMap(([key, val]) => {
      if (Array.isArray(val)) return val.map((m) => `${key}: ${m}`)
      if (typeof val === 'string') return [`${key}: ${val}`]
      return []
    })
    if (parts.length) return parts.join('；')
  }
  return '保存失败，请稍后重试'
}

const fetchProjectDetail = async () => {
  try {
    const projectId = route.params.id
    const response = await apiFetch(`/api/projects/tenant_id/${route.params.tenant}/${projectId}/`, {
      headers: {
        'Content-Type': 'application/json'
      },
      credentials: 'include'
    })

    if (response.ok) {
      const data = await response.json()
      projectSnapshot.value = data
      formData.value = {
        name: data.name,
        description: data.description,
        container_image_id: data.container_image_id ? String(data.container_image_id) : '',
        workspaces_ids: [...data.workspaces],
        tags: normalizeProjectTags(Array.isArray(data.tags) ? data.tags : []),
        default_auto_run: Boolean(data.server_run_template?.default_auto_run),
      }
      setGitRepoRowsFromApi(data)
    } else {
      console.error('Failed to fetch project detail')
      router.back()
    }
  } catch (error) {
    console.error('Error fetching project detail:', error)
    router.back()
  }
}

const loadInstalledImages = async () => {
  if (hasLoadedImages.value) {
    return
  }
  
  try {
    isLoadingImages.value = true
    const tenantId = route.params.tenant
    const response = await apiFetch(`/api/cloud/installed-images/tenant_id/${tenantId}`, {
      headers: {
        'Content-Type': 'application/json'
      },
      credentials: 'include'
    })

    if (response.ok) {
      const data = await readJsonPreservingSnowflakeIds(response)
      installedImages.value = normalizeInstalledImageList(data)
      hasLoadedImages.value = true
    }
  } catch (error) {
    console.error('Error fetching installed images:', error)
  } finally {
    isLoadingImages.value = false
  }
}

const fetchAllWorkspaces = async () => {
  try {
    const tenantId = route.params.tenant
    if (!tenantId) {
      console.error('缺少租户ID')
      return
    }
    
    const response = await apiFetch(`/api/projects/workspaces/tenant_id/${tenantId}`, {
      headers: {
        'Content-Type': 'application/json'
      },
      credentials: 'include'
    })

    if (response.ok) {
      const data = await response.json()
      allWorkspaces.value = Array.isArray(data) ? data : data.results || []
    }
  } catch (error) {
    console.error('Error fetching workspaces:', error)
  }
}

const resolveSelectedContainerImageName = () => {
  const imageId = String(formData.value.container_image_id || '').trim()
  if (!imageId) return ''
  const matched = installedImages.value.find((image) => String(image.id) === imageId)
  return String(matched?.name || '').trim()
}

const buildServerRunTemplatePayload = () => {
  const panelPayload = runTemplatePanelRef.value?.buildPayload?.() ?? {}
  const existingTemplate =
    projectSnapshot.value?.server_run_template &&
    typeof projectSnapshot.value.server_run_template === 'object'
      ? { ...projectSnapshot.value.server_run_template }
      : {}
  // 仅当面板草稿硬件完整时才合并，避免「有地域无实例」半成品抹掉已存实例规格
  const panelConfigured = runTemplateIsHardwareComplete(panelPayload)
  const baseTemplate = panelConfigured ? { ...existingTemplate, ...panelPayload } : existingTemplate
  return {
    ...baseTemplate,
    default_auto_run: Boolean(formData.value.default_auto_run),
  }
}

const updateProject = async () => {
  generalError.value = ''
  generalErrorTraceId.value = ''
  const formatErr = refreshGitRepoFormatErrors()
  if (formatErr) {
    generalError.value = formatErr
    return
  }
  const dupAlias = duplicateCloneAliasError()
  if (dupAlias) {
    generalError.value = dupAlias
    return
  }
  // OPT-20260819-038: 更新项目是写操作，防连点/超时重试双发 PUT
  await updateProjectGuard.run(async ({ idempotencyKey }) => {
    isSaving.value = true
    try {
      const projectId = route.params.id
      const body = {
        ...formData.value,
        tags: normalizeProjectTags(formData.value.tags),
        git_repos: trimmedGitRepoPayload(),
        server_run_template: buildServerRunTemplatePayload(),
      }
      delete body.default_auto_run
      if (!String(body.container_image_id || '').trim()) {
        body.container_image_id = ''
        body.container_image = ''
      } else {
        body.container_image = resolveSelectedContainerImageName()
      }
      const response = await apiFetch(`/api/projects/tenant_id/${route.params.tenant}/${projectId}/`, {
        method: 'PUT',
        headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
        body: JSON.stringify(body),
        credentials: 'include'
      })

      if (response.ok) {
        router.push(`/tenant/${route.params.tenant}/projects/${projectId}/`)
      } else {
        generalError.value = await parseApiError(response)
        generalErrorTraceId.value = extractTraceId(response) || ''
      }
    } catch (error) {
      console.error('Error updating project:', error)
      generalError.value = '网络错误，请稍后重试'
      generalErrorTraceId.value = extractTraceId(error) || ''
    } finally {
      isSaving.value = false
    }
  })
}

const cancelEdit = () => {
  router.back()
}

onMounted(() => {
  fetchProjectDetail()
  fetchAllWorkspaces()
  loadInstalledImages()
})
</script>
