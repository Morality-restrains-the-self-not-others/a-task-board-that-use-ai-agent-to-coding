<template>
  <div data-alias="view-create-project" class="p-8">
    <div class="max-w-3xl mx-auto">
      <h1 class="text-[clamp(1.5rem,3vw,2.5rem)] font-bold text-text mb-8">创建项目</h1>
      
      <!-- 通用错误提示 -->
      <div
        v-if="generalError"
        class="mb-6 p-4 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
        role="alert"
        :data-traceId="generalErrorTraceId || undefined"
      >
        {{ generalError }}
      </div>
      
      <div class="bg-white rounded-xl shadow-md p-8">
        <form @submit.prevent="submitForm">
          <!-- Git 仓库置顶：用户常需先完成仓库授权再填写项目信息 -->
          <div class="mb-6" data-testid="create-project-git-repos-section">
            <div class="flex items-center justify-between mb-2">
              <span class="block text-sm font-medium text-text">Git 仓库（可选，可添加多个）</span>
              <button
                type="button"
                class="text-sm text-primary hover:underline"
                @click="addGitRepoRow"
              >
                添加仓库
              </button>
            </div>
            <div class="space-y-3">
              <div
                v-for="(row, index) in gitRepoRows"
                :key="row.id"
                class="flex gap-2 items-start"
              >
                <div class="flex-1 min-w-0">
                  <div class="flex gap-2">
                    <input
                      :id="index === 0 ? 'gitRepo0' : undefined"
                      type="text"
                      v-model="row.url"
                      placeholder="Git 仓库 URL（https://、git@主机:路径 或 ssh://）"
                      :class="['flex-1 min-w-0 px-4 py-3 border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary', gitRepoRowErrorClass(row.id) ? 'border-red-500' : 'border-gray-300']"
                      @blur="onGitRepoRowBlur(row.id)"
                    >
                    <input
                      v-model="row.cloneAlias"
                      type="text"
                      placeholder="别名（可选）"
                      data-testid="git-repo-clone-alias-input"
                      class="w-28 shrink-0 px-3 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                    >
                  </div>
                  <p v-if="gitRepoRowValidating[row.id]" class="mt-1 text-sm text-gray-500">校验中...</p>
                  <template v-else-if="gitRepoRowErrors[row.id] === INVALID_REPO_URL_MSG">
                    <p
                      class="mt-1 text-sm text-red-600"
                      role="alert"
                      data-testid="git-repo-url-format-hint"
                    >
                      {{ gitRepoRowErrors[row.id] }}
                    </p>
                    <p class="mt-1 text-xs text-gray-500">
                      正确格式示例：
                      <span v-for="(example, exampleIndex) in GIT_REPO_URL_EXAMPLES" :key="example">
                        <code class="text-xs bg-gray-100 px-1 py-0.5 rounded">{{ example }}</code><span v-if="exampleIndex < GIT_REPO_URL_EXAMPLES.length - 1">、</span>
                      </span>
                    </p>
                  </template>
                  <div
                    v-else-if="gitRepoRowErrors[row.id]"
                    class="flex flex-wrap items-center gap-x-2 gap-y-1"
                  >
                    <p
                      class="mt-1 text-sm text-red-600"
                      role="alert"
                      data-testid="git-repo-validate-error"
                      :data-traceId="gitRepoRowErrorTraceId[row.id] || undefined"
                    >{{ gitRepoRowErrors[row.id] }}</p>
                    <button
                      v-if="isGitRepoValidateRetryable(gitRepoRowErrors[row.id])"
                      type="button"
                      data-testid="git-repo-validate-retry"
                      class="text-xs px-2 py-0.5 border border-primary text-primary rounded bg-white hover:bg-primary/5 disabled:opacity-50 disabled:cursor-not-allowed"
                      :disabled="Boolean(gitRepoRowValidating[row.id])"
                      :aria-busy="Boolean(gitRepoRowValidating[row.id]) || undefined"
                      @click="retryValidateGitRepoRow(row.id)"
                    >重试</button>
                  </div>
                  <div
                    v-if="gitRepoRowErrors[row.id] === REPO_INACCESSIBLE_MSG && shouldShowRepoOAuthButton(row.url)"
                    class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1"
                    data-testid="repo-oauth-authorize-row"
                  >
                    <span
                      class="text-sm text-gray-700"
                      data-testid="repo-oauth-authorize-prompt"
                    >是否现在去授权</span>
                    <button
                      type="button"
                      data-testid="repo-oauth-authorize-button"
                      class="text-xs px-2 py-0.5 border border-primary text-primary rounded bg-white hover:bg-primary/5 disabled:opacity-50 disabled:cursor-not-allowed"
                      :disabled="isRepoOAuthActionDisabled(row.url)"
                      @click="startRepoOAuthConnect(row.url)"
                    >
                      {{ repoOAuthButtonLabel(row.url) }}
                    </button>
                    <p
                      v-if="repoOAuthErrorByUrl(row.url)"
                      class="text-xs text-red-600"
                      :data-traceId="repoOAuthErrorTraceIdByUrl(row.url) || undefined"
                    >
                      {{ repoOAuthErrorByUrl(row.url) }}
                    </p>
                  </div>
                </div>
                <button
                  v-if="gitRepoRows.length > 1"
                  type="button"
                  class="mt-3 shrink-0 text-sm text-red-600 hover:underline"
                  @click="removeGitRepoRow(row.id)"
                >
                  移除
                </button>
              </div>
            </div>
            <p class="mt-2 text-xs text-gray-500">{{ GIT_REPO_URL_FORMAT_HINT }}</p>
            <p v-if="errors.git_repos" class="mt-2 text-sm text-red-600">{{ errors.git_repos }}</p>
            <label
              v-if="hasAnyGitRepoUrl"
              class="mt-3 flex items-start gap-2 text-sm text-text cursor-pointer"
              data-testid="create-project-auto-clone-nested-repos-label"
            >
              <input
                v-model="formData.auto_clone_nested_repos"
                type="checkbox"
                class="mt-0.5"
                data-testid="create-project-auto-clone-nested-repos"
              >
              <span>
                自动克隆子仓库
                <span class="block text-xs text-gray-500 font-normal">
                  开启后，容器启动时会根据父仓 `.gitmodules` 发现并克隆子仓库；关闭则仅克隆父仓库。
                </span>
              </span>
            </label>
          </div>

          <div class="mb-6">
            <label for="projectName" class="block text-sm font-medium text-text mb-2">项目名称 <span class="text-red-500">*</span></label>
            <input 
              type="text" 
              id="projectName" 
              v-model="formData.name" 
              placeholder="请输入项目名称" 
              :class="['w-full px-4 py-3 border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary', errors.name ? 'border-red-500' : 'border-gray-300']"
            >
            <p v-if="errors.name" class="mt-1 text-sm text-red-600">{{ errors.name }}</p>
          </div>
          
          <div class="mb-6">
            <label for="projectDescription" class="block text-sm font-medium text-text mb-2">项目描述 <span class="text-red-500">*</span></label>
            <textarea 
              id="projectDescription" 
              v-model="formData.description" 
              rows="4" 
              placeholder="请输入项目描述" 
              :class="['w-full px-4 py-3 border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary', errors.description ? 'border-red-500' : 'border-gray-300']"
            ></textarea>
            <p v-if="errors.description" class="mt-1 text-sm text-red-600">{{ errors.description }}</p>
          </div>

          <div class="mb-6">
            <ProjectTagsInput v-model="formData.tags" />
            <button
              type="button"
              class="mt-2 text-sm px-3 py-1.5 rounded-md border border-gray-300 text-gray-700 hover:bg-gray-50"
              data-testid="create-project-tags-sync-daydaymoney"
              @click="syncTagsFromAidevYaml"
            >
              从 daydaymoney.yaml 同步
            </button>
            <p v-if="tagsSyncError" class="mt-1 text-sm text-red-600" role="alert">{{ tagsSyncError }}</p>
          </div>
          
          <div class="mb-6">
            <label for="containerImage" class="block text-sm font-medium text-text mb-2">已安装镜像</label>
            <select 
              id="containerImage" 
              v-model="formData.container_image" 
              class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
            >
              <option value="">无</option>
              <option 
                v-for="image in installedImages" 
                :key="image.id" 
                :value="image.id"
              >
                {{ image.name }}{{ image.version ? ':' + image.version : '' }}
              </option>
            </select>
          </div>
          
          <div class="mb-6">
            <label for="workspace" class="block text-sm font-medium text-text mb-2">工作空间 <span class="text-red-500">*</span></label>
            <select 
              id="workspace" 
              v-model="formData.workspace" 
              :class="['w-full px-4 py-3 border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary', errors.workspace ? 'border-red-500' : 'border-gray-300']"
            >
              <option value="">请选择工作空间</option>
              <option 
                v-for="workspace in workspaces" 
                :key="workspace.id" 
                :value="workspace.id"
              >
                {{ workspace.name }}
              </option>
            </select>
            <p v-if="errors.workspace" class="mt-1 text-sm text-red-600">{{ errors.workspace }}</p>
          </div>

          <ProjectRunTemplatePanel
            v-if="formData.workspace"
            :key="formData.workspace"
            ref="runTemplatePanelRef"
            :tenant-id="String(route.params.tenant || '')"
            project-id="draft"
            :project="runTemplateProjectSnapshot"
            hide-actions
          />
          <p v-if="formData.workspace" class="text-xs text-gray-500 mb-6">
            配置运行模版后，在工作面板创建任务时可勾选「是否自动运行」，系统将按此模版自动启动服务器。
          </p>
          
          <div class="flex flex-col items-end gap-2">
            <p
              v-if="createButtonDisabled && createButtonDisabledReason"
              id="create-project-disabled-hint"
              class="text-sm text-amber-600"
              role="status"
              data-testid="create-project-disabled-hint"
            >
              {{ createButtonDisabledReason }}
            </p>
            <div class="flex justify-end space-x-4">
              <router-link :to="`/tenant/${route.params.tenant}/projects/`" class="btn-secondary px-6 py-3 rounded-lg">
                取消
              </router-link>
              <button
                type="submit"
                class="btn-primary px-6 py-3 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
                :disabled="createButtonDisabled"
                :title="createButtonDisabledReason"
                :aria-describedby="createButtonDisabled && createButtonDisabledReason ? 'create-project-disabled-hint' : undefined"
              >
                <span v-if="loading">创建中...</span>
                <span v-else>创建项目</span>
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
/* @alias:view-create-project */
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useCreateProjectGitRepoRows } from '../composables/useCreateProjectGitRepoRows.js'
import { isGitRepoValidateRetryable } from '../utils/gitRepoValidateError.js'
import { useCreateProjectForm } from '../composables/useCreateProjectForm.js'
import ProjectTagsInput from '../components/ProjectTagsInput.vue'
import ProjectRunTemplatePanel from '../components/ProjectRunTemplatePanel.vue'

const router = useRouter()
const route = useRoute()
const runTemplatePanelRef = ref(null)

const {
  REPO_INACCESSIBLE_MSG,
  INVALID_REPO_URL_MSG,
  GIT_REPO_URL_EXAMPLES,
  GIT_REPO_URL_FORMAT_HINT,
  gitRepoRows,
  gitRepoRowValidating,
  gitRepoRowErrors,
  gitRepoRowErrorTraceId,
  addGitRepoRow,
  removeGitRepoRow,
  gitRepoRowErrorClass,
  onGitRepoRowBlur,
  retryValidateGitRepoRow,
  shouldShowRepoOAuthButton,
  repoOAuthButtonLabel,
  isRepoOAuthActionDisabled,
  repoOAuthErrorByUrl,
  repoOAuthErrorTraceIdByUrl,
  startRepoOAuthConnect,
  trimmedGitRepoPayload,
  duplicateCloneAliasError,
  gitReposHaveValidUrls,
  gitReposPendingValidation,
  appendGitRepoFormErrors,
  bootstrapGitReposOnMount,
} = useCreateProjectGitRepoRows(route, router)

const {
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
} = useCreateProjectForm({
  route,
  router,
  gitRepoRows,
  trimmedGitRepoPayload,
  duplicateCloneAliasError,
  gitReposHaveValidUrls,
  gitReposPendingValidation,
  appendGitRepoFormErrors,
  runTemplatePanelRef,
})

onMounted(async () => {
  await loadInstalledImages()
  await loadWorkspaces()
  await bootstrapGitReposOnMount()
})
</script>

<style scoped></style>
