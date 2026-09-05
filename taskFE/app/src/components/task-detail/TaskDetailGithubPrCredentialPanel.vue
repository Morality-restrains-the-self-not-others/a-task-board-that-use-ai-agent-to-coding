<template>
  <div
    class="p-4 bg-white border border-gray-200 rounded-lg shadow-sm"
    data-testid="task-github-credential-panel"
  >
    <div class="flex flex-wrap items-start justify-between gap-3 mb-3">
      <div class="min-w-0 flex-1">
        <h3 class="text-sm font-semibold text-gray-900">GitHub 与自动创建 PR</h3>
        <p class="text-xs text-gray-500 mt-1 leading-relaxed">
          代码层级推送成功后，平台会用<strong class="font-medium text-gray-700">您在本站绑定的 GitHub 授权</strong>尝试创建
          Pull Request（含私有仓库访问）。请按顺序完成下方两步。
        </p>
      </div>
      <div class="shrink-0 flex flex-col items-end gap-1">
        <span
          v-if="githubCredentialLoading"
          class="inline-flex items-center text-xs text-gray-400"
        >
          加载中…
        </span>
        <span
          v-else-if="githubPrReady"
          class="inline-flex items-center rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-800"
        >
          可自动创建 PR
        </span>
        <span
          v-else
          class="inline-flex items-center rounded-full border border-amber-200 bg-amber-50 px-2.5 py-0.5 text-xs font-medium text-amber-900"
        >
          尚缺步骤
        </span>
      </div>
    </div>

    <ol class="space-y-3" aria-label="GitHub 自动 PR 前置步骤">
      <li
        class="flex gap-3 items-start rounded-lg border border-gray-100 bg-gray-50/80 p-3"
      >
        <span
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-white text-xs font-semibold text-gray-700 shadow-sm ring-1 ring-gray-200"
          aria-hidden="true"
        >
          1
        </span>
        <div class="min-w-0 flex-1 space-y-2">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-sm font-medium text-gray-900">绑定 GitHub 账号</span>
            <span
              v-if="githubCredential.github_app_connected"
              class="inline-flex items-center rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[11px] font-medium text-emerald-800"
            >
              已连接（{{ connectedGithubOptions.length }}）<span v-if="githubCredential.github_login"> · @{{ githubCredential.github_login }}</span>
            </span>
            <span
              v-else
              class="inline-flex items-center rounded-full border border-gray-200 bg-white px-2 py-0.5 text-[11px] font-medium text-gray-600"
            >
              未连接
            </span>
          </div>
          <p class="text-xs text-gray-500 leading-relaxed">
            通过 GitHub App 授权本站，以便调用 GitHub API（创建 PR、读私有仓库等）。
          </p>
          <div class="flex flex-wrap items-center gap-2">
            <button
              type="button"
              class="px-3 py-1.5 text-sm border border-gray-300 rounded-md bg-white hover:bg-gray-50 disabled:opacity-50"
              :disabled="githubCredentialActionLoading"
              @click="startGithubAppConnect"
            >
              {{ githubCredential.github_app_connected ? '重新连接 GitHub' : '连接 GitHub' }}
            </button>
            <router-link
              v-if="githubCredential.github_app_connected"
              :to="gitSiteOauthHref"
              class="text-xs font-medium text-primary hover:text-primary/80 underline-offset-2 hover:underline"
            >
              在账号中心管理授权
            </router-link>
          </div>
        </div>
      </li>

      <li
        class="flex gap-3 items-start rounded-lg border border-gray-100 bg-gray-50/80 p-3"
      >
        <span
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-white text-xs font-semibold text-gray-700 shadow-sm ring-1 ring-gray-200"
          aria-hidden="true"
        >
          2
        </span>
        <div class="min-w-0 flex-1 space-y-2">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-sm font-medium text-gray-900">按仓库选择 GitHub 授权账号</span>
          </div>
          <p class="text-xs text-gray-500 leading-relaxed">
            在「关联项目」各仓库的
            <strong class="font-medium text-gray-700">「GitHub App 授权」</strong>
            分区选择账号（靛色边框区块）。勿与同页的
            <strong class="font-medium text-gray-700">「Git 提交身份」</strong>
            （commit name/email）混淆。
          </p>
        </div>
      </li>
    </ol>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { showRequestError } from '../../utils/requestErrorDisplay.js'
import {
  createGithubAppReturnKey,
  setGithubAppReturnTarget,
} from '../../utils/githubAppReturnStorage.js'

const props = defineProps({
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
})

const githubCredential = ref({
  approved_for_task: false,
  github_app_connected: false,
  github_login: '',
  github_connections: [],
  repo_bindings: [],
  all_repo_bound: false,
})
const githubCredentialLoading = ref(false)
const githubCredentialActionLoading = ref(false)

const githubPrReady = computed(
  () => githubCredential.value.github_app_connected,
)

const connectedGithubOptions = computed(() =>
  (githubCredential.value.github_connections || []).filter(
    (item) => item && item.connected && item.github_user_id != null,
  ),
)

const gitSiteOauthHref = computed(() => {
  const tenantId = String(props.tenantId || '').trim()
  if (tenantId) {
    return `/tenant/${tenantId}/profile/git-site-oauth/`
  }
  return '/profile/git-site-oauth/'
})

const fetchGithubCredentialStatus = async () => {
  const tenantId = String(props.tenantId || '').trim()
  const workspaceId = String(props.workspaceId || '').trim()
  const taskId = String(props.taskId || '').trim()
  if (!tenantId || !workspaceId || !taskId) return
  githubCredentialLoading.value = true
  try {
    const response = await apiFetch(
      `/api/cloud/compute/github-credential-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`,
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    const data = await response.json().catch(() => ({}))
    if (response.ok) {
      githubCredential.value = {
        approved_for_task: Boolean(data.approved_for_task),
        github_app_connected: Boolean(data.github_app_connected),
        github_login: typeof data.github_login === 'string' ? data.github_login.trim() : '',
        github_connections: Array.isArray(data.github_connections) ? data.github_connections : [],
        repo_bindings: Array.isArray(data.repo_bindings) ? data.repo_bindings : [],
        all_repo_bound: Boolean(data.all_repo_bound),
      }
    }
  } catch (e) {
    console.warn('加载 GitHub 凭据状态失败', e)
  } finally {
    githubCredentialLoading.value = false
  }
}

const startGithubAppConnect = async () => {
  githubCredentialActionLoading.value = true
  try {
    const nextPath = `${window.location.pathname}${window.location.search || ''}`
    const returnKey = createGithubAppReturnKey()
    setGithubAppReturnTarget(returnKey, nextPath)
    const startUrl =
      `/api/git-oauth/github-app-start/?next=${encodeURIComponent(nextPath)}` +
      `&return_key=${encodeURIComponent(returnKey)}`
    const response = await apiFetch(startUrl, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      showRequestError(typeof data.detail === 'string' ? data.detail : '无法启动 GitHub 授权', data)
      return
    }
    if (data.authorize_url) {
      window.location.href = data.authorize_url
    }
  } catch (e) {
    showRequestError(e?.message || '启动 GitHub 授权失败', e)
  } finally {
    githubCredentialActionLoading.value = false
  }
}

watch(
  () => [props.tenantId, props.workspaceId, props.taskId],
  () => {
    void fetchGithubCredentialStatus()
  },
  { immediate: true },
)
</script>
