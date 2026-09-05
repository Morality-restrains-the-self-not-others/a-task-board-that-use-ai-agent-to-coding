<template>
  <div
    v-if="repoUrls.length"
    class="mt-2 space-y-2"
    data-testid="comment-composer-repo-identity"
  >
    <div
      v-if="gitOauthHint.kind !== 'none'"
      class="rounded border px-2 py-1.5 text-[11px] leading-snug"
      :class="gitOauthHintBannerClass"
      data-testid="comment-composer-git-oauth-hint"
      :data-kind="gitOauthHint.kind"
    >
      <p>{{ gitOauthHint.text }}</p>
      <template v-if="gitOauthHint.kind === 'unbound'">
        <a
          v-for="url in unboundOauthRepoUrls"
          :key="'oauth-link-' + url"
          class="mt-1 mr-2 inline-block font-medium underline underline-offset-2"
          data-testid="comment-composer-git-oauth-settings-link"
          :href="oauthStartHref(url)"
        >去绑定 {{ oauthBindHostLabel(url) }}</a>
      </template>
    </div>
    <p class="text-[11px] text-gray-600 leading-snug">
      本次运行身份：为每个仓库选择 Git 提交署名；GitHub 仓还需选择授权账号。含 HTTPS 远端的提交并运行会先校验 Git OAuth；未绑定将禁用发送。
    </p>
    <div
      v-for="repoUrl in repoUrls"
      :key="repoUrl"
      class="rounded border border-gray-200 bg-gray-50 px-2 py-1.5 space-y-1"
      data-testid="comment-composer-repo-identity-row"
    >
      <a
        v-if="isHttpRepoUrl(repoUrl)"
        :href="String(repoUrl).trim()"
        target="_blank"
        rel="noopener noreferrer"
        class="text-[11px] text-blue-600 truncate hover:underline"
        :title="repoUrl"
        data-testid="comment-composer-repo-url"
      >{{ repoUrl }}</a>
      <p v-else class="text-[11px] text-gray-700 truncate" :title="repoUrl">{{ repoUrl }}</p>
      <a
        v-if="shouldShowRepoOAuthBindButton(repoUrl) && oauthStartHref(repoUrl)"
        class="text-[11px] px-1.5 py-0 border border-primary text-primary rounded bg-white hover:bg-primary/5 inline-block"
        data-testid="comment-repo-oauth-bind-btn"
        :href="oauthStartHref(repoUrl)"
      >
        {{ repoOAuthBindButtonLabel(repoUrl) }}
      </a>
      <label class="block text-[11px] text-slate-700">Git 提交身份</label>
      <select
        class="w-full max-w-md px-1.5 py-1 text-xs border border-gray-300 rounded bg-white"
        :value="gitIdFor(repoUrl)"
        data-testid="comment-repo-git-identity-select"
        @change="onGitIdChange(repoUrl, $event.target.value)"
      >
        <option value="">请选择用于该仓库克隆的身份</option>
        <option v-for="identity in gitIdentities" :key="identity.id" :value="identity.id">
          {{ identityLabel(identity) }}
        </option>
      </select>
      <template v-if="isGithubRepoUrl(repoUrl)">
        <label class="block text-[11px] text-indigo-900">GitHub App 授权账号</label>
        <select
          class="w-full max-w-md px-1.5 py-1 text-xs border border-gray-300 rounded bg-white"
          :value="githubIdFor(repoUrl)"
          data-testid="comment-repo-github-account-select"
          @change="onGithubChange(repoUrl, $event.target.value)"
        >
          <option value="">请选择该仓库要使用的 GitHub 账号</option>
          <option
            v-for="account in githubConnectedOptions"
            :key="'gh-' + account.github_user_id"
            :value="String(account.github_user_id)"
          >
            @{{ account.github_login || ('用户#' + account.github_user_id) }}
          </option>
        </select>
      </template>
      <template v-if="projectFor(repoUrl)">
        <div
          class="flex items-center justify-start gap-2 pt-0.5"
          data-testid="task-nested-repos-auto-clone-control"
        >
          <span class="text-[11px] text-gray-600 leading-tight">自动克隆子仓库</span>
          <label
            class="relative inline-flex items-center cursor-pointer shrink-0"
            data-testid="task-nested-repos-auto-clone-toggle-label"
          >
            <input
              type="checkbox"
              class="sr-only peer"
              role="switch"
              data-testid="task-nested-repos-auto-clone-toggle"
              :checked="autoCloneEnabledFor(repoUrl)"
              :disabled="isAutoCloneSaving(repoUrl)"
              :aria-checked="autoCloneEnabledFor(repoUrl)"
              @change="onAutoCloneToggle(repoUrl, $event)"
            >
            <div
              class="w-8 h-4 bg-gray-200 rounded-full peer peer-focus:outline-none peer-checked:bg-primary peer-disabled:opacity-50 after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-3 after:w-3 after:transition-all peer-checked:after:translate-x-4"
            />
          </label>
        </div>
        <p
          v-if="!autoCloneEnabledFor(repoUrl)"
          class="text-[10px] text-amber-700 leading-tight m-0"
          data-testid="task-nested-repos-auto-clone-off-hint"
        >
          关闭后容器将仅克隆父仓库。
        </p>
        <p
          v-if="autoCloneSaveErrorFor(repoUrl)"
          class="text-[10px] text-red-600 leading-tight m-0"
          role="alert"
          data-testid="task-nested-repos-auto-clone-save-error"
          :data-traceId="autoCloneSaveErrorTraceIdFor(repoUrl) || undefined"
        >
          {{ autoCloneSaveErrorFor(repoUrl) }}
        </p>
      </template>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { resolveAuthenticatedUserId } from '../../utils/sessionUserIdUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { isHttpRepoUrl } from '../../utils/repoUrl.js'
import {
  collectLinkedRepoUrls,
  commentComposerGitOauthHint,
  isGithubRepoUrl,
  resolveLinkedRepoProject,
} from '../../utils/commentRepoIdentity.js'
import {
  buildRepoOAuthStartHref,
  resolveRepoOAuthAuthorizeLabel,
} from '../../utils/repoOAuthAuthorizeUtils.js'
import { setCommentRepoIdentityDraft } from '../../composables/taskDetail/commentRepoIdentityDraft.js'
import { useLinkedProjectsRepoOAuth } from '../../composables/taskDetail/useLinkedProjectsRepoOAuth.js'
import { mergeSessionGrantIntoReadiness } from '../../utils/commentOAuthGrantCheck.js'
import { rememberGrantTicketFromSearch } from '../../utils/grantTicketSession.js'

if (typeof window !== 'undefined') {
  rememberGrantTicketFromSearch(window.location?.search || '')
}

const props = defineProps({
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
  taskProjectsWithDetails: { type: Array, default: () => [] },
  /** 当前任务评论列表（按时间升序）；最近一条非空 repo_identities 用于预填。 */
  comments: { type: Array, default: () => [] },
  /** 任务级 repo_identities 回退预填（无评论身份时使用）。 */
  taskRepoIdentities: { type: Array, default: () => [] },
})

const emit = defineEmits(['repo-oauth-readiness'])

const gitIdentities = ref([])
const githubConnectedOptions = ref([])
const gitIdByUrl = ref({})
const githubIdByUrl = ref({})
const localAutoCloneByProjectId = ref({})
const autoCloneSavingByProjectId = ref({})
const autoCloneSaveErrorByProjectId = ref({})
const autoCloneSaveErrorTraceIdByProjectId = ref({})

const repoUrls = computed(() => collectLinkedRepoUrls(props.taskProjectsWithDetails))

const repoOAuth = useLinkedProjectsRepoOAuth({ props, emit })
const {
  repoOAuthBindButtonLabel,
  shouldShowRepoOAuthBindButton,
} = repoOAuth

const gitOauthHint = computed(() => commentComposerGitOauthHint(
  repoOAuth.repoOAuthStartReadiness.value,
  repoUrls.value,
))
const unboundOauthRepoUrls = computed(() => {
  const merged = mergeSessionGrantIntoReadiness(
    repoOAuth.repoOAuthStartReadiness.value,
    repoUrls.value,
  )
  return Array.isArray(merged.unboundRepoUrls) ? merged.unboundRepoUrls : []
})
const gitOauthHintBannerClass = computed(() => {
  if (gitOauthHint.value.kind === 'bound') return 'border-emerald-200 bg-emerald-50 text-emerald-900'
  if (gitOauthHint.value.kind === 'unbound') return 'border-amber-200 bg-amber-50 text-amber-950'
  return 'border-gray-200 bg-gray-50 text-gray-700'
})

const oauthReturnPath = () => {
  if (typeof window === 'undefined') return ''
  return `${window.location.pathname || ''}${window.location.search || ''}`
}
const oauthStartHref = (repoUrl) => buildRepoOAuthStartHref(repoUrl, { nextPath: oauthReturnPath() })
const oauthBindHostLabel = (repoUrl) => resolveRepoOAuthAuthorizeLabel(repoUrl)

const identityLabel = (identity) => {
  const name = String(identity?.git_user_name || identity?.name || '').trim()
  const email = String(identity?.git_user_email || identity?.email || '').trim()
  if (name && email) return `${name} <${email}>`
  return name || email || String(identity?.id || '')
}

const gitIdFor = (url) => String(gitIdByUrl.value[url] || '')
const githubIdFor = (url) => String(githubIdByUrl.value[url] || '')

const persistDraft = () => {
  setCommentRepoIdentityDraft(repoUrls.value.map((repo_url) => ({
    repo_url,
    git_identity_id: gitIdFor(repo_url),
    github_user_id: githubIdFor(repo_url),
  })))
}

const onGitIdChange = (url, value) => {
  gitIdByUrl.value = { ...gitIdByUrl.value, [url]: value }
  persistDraft()
}

const onGithubChange = (url, value) => {
  githubIdByUrl.value = { ...githubIdByUrl.value, [url]: value }
  persistDraft()
}

const projectFor = (repoUrl) => resolveLinkedRepoProject(props.taskProjectsWithDetails, repoUrl)

const autoCloneEnabledFor = (repoUrl) => {
  const row = projectFor(repoUrl)
  if (!row) return true
  const override = localAutoCloneByProjectId.value[row.projectId]
  if (override !== undefined) return override
  return row.autoCloneNestedRepos
}

const isAutoCloneSaving = (repoUrl) => {
  const row = projectFor(repoUrl)
  return Boolean(row && autoCloneSavingByProjectId.value[row.projectId])
}

const autoCloneSaveErrorFor = (repoUrl) => {
  const row = projectFor(repoUrl)
  return row ? String(autoCloneSaveErrorByProjectId.value[row.projectId] || '') : ''
}

const autoCloneSaveErrorTraceIdFor = (repoUrl) => {
  const row = projectFor(repoUrl)
  return row ? String(autoCloneSaveErrorTraceIdByProjectId.value[row.projectId] || '') : ''
}

const patchAutoCloneState = (projectId, fieldRef, value) => {
  fieldRef.value = { ...fieldRef.value, [projectId]: value }
}

async function onAutoCloneToggle(repoUrl, event) {
  const row = projectFor(repoUrl)
  const next = !!event?.target?.checked
  const revert = () => {
    if (event?.target) event.target.checked = !next
  }
  if (!row) {
    revert()
    return
  }
  const tenantId = String(props.tenantId || '').trim()
  if (!tenantId) {
    patchAutoCloneState(row.projectId, autoCloneSaveErrorByProjectId, '缺少租户上下文，无法保存')
    revert()
    return
  }
  patchAutoCloneState(row.projectId, autoCloneSaveErrorByProjectId, '')
  patchAutoCloneState(row.projectId, autoCloneSaveErrorTraceIdByProjectId, '')
  patchAutoCloneState(row.projectId, autoCloneSavingByProjectId, true)
  try {
    const response = await apiFetch(
      `/api/projects/tenant_id/${encodeURIComponent(tenantId)}/${encodeURIComponent(row.projectId)}/`,
      {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ auto_clone_nested_repos: next }),
      },
    )
    if (!response.ok) {
      const errData = response._errorData ?? (await response.json().catch(() => ({})))
      const message = typeof errData.error === 'string'
        ? errData.error
        : (errData.detail || errData.message || '保存失败')
      patchAutoCloneState(
        row.projectId,
        autoCloneSaveErrorTraceIdByProjectId,
        extractTraceId(response) || extractTraceId(errData) || '',
      )
      patchAutoCloneState(row.projectId, autoCloneSaveErrorByProjectId, message)
      revert()
      return
    }
    const updated = await response.json().catch(() => ({}))
    const saved =
      updated.auto_clone_nested_repos === undefined
        ? next
        : Boolean(updated.auto_clone_nested_repos)
    patchAutoCloneState(row.projectId, localAutoCloneByProjectId, saved)
  } catch (err) {
    patchAutoCloneState(row.projectId, autoCloneSaveErrorByProjectId, '网络错误，请重试')
    patchAutoCloneState(row.projectId, autoCloneSaveErrorTraceIdByProjectId, extractTraceId(err) || '')
    revert()
  } finally {
    patchAutoCloneState(row.projectId, autoCloneSavingByProjectId, false)
  }
}

// OPT-20260816-007：$镜像提交前预填最近一条评论的仓库身份，否则回退任务级。
// 只填充空槽（不覆盖用户已选），数据异步到达后仍可补一次。
let prefillApplied = false

function resolveLatestCommentRepoIdentities() {
  const rows = Array.isArray(props.comments) ? props.comments : []
  for (let i = rows.length - 1; i >= 0; i--) {
    const idents = rows[i]?.repo_identities
    if (Array.isArray(idents) && idents.length) return idents
  }
  return []
}

function applyPrefillSelections() {
  if (prefillApplied) return
  const fromComments = resolveLatestCommentRepoIdentities()
  const sources = fromComments.length
    ? fromComments
    : Array.isArray(props.taskRepoIdentities)
      ? props.taskRepoIdentities
      : []
  if (!sources.length) return
  const git = { ...gitIdByUrl.value }
  const github = { ...githubIdByUrl.value }
  let changed = false
  for (const row of sources) {
    const url = String(row?.repo_url || '').trim()
    if (!url) continue
    const gid = String(row?.git_identity_id || '').trim()
    const ghid = String(row?.github_user_id || '').trim()
    if (gid && !git[url]) {
      git[url] = gid
      changed = true
    }
    if (ghid && !github[url]) {
      github[url] = ghid
      changed = true
    }
  }
  if (changed) {
    gitIdByUrl.value = git
    githubIdByUrl.value = github
    persistDraft()
  }
  prefillApplied = true
}

const loadGitIdentities = async () => {
  const tenantId = String(props.tenantId || '').trim()
  if (!tenantId) {
    gitIdentities.value = []
    return
  }
  const userId = await resolveAuthenticatedUserId()
  if (!userId) {
    gitIdentities.value = []
    return
  }
  const response = await apiFetch(
    `/api/git-identities/user/${encodeURIComponent(userId)}/?company_id=${encodeURIComponent(tenantId)}`,
    { credentials: 'include', headers: { Accept: 'application/json' } },
  )
  const data = await response.json().catch(() => ({}))
  gitIdentities.value = response.ok && Array.isArray(data.identities) ? data.identities : []
}

const loadGithubConnections = async () => {
  const tenantId = String(props.tenantId || '').trim()
  const workspaceId = String(props.workspaceId || '').trim()
  const taskId = String(props.taskId || '').trim()
  if (!tenantId || !workspaceId || !taskId) {
    githubConnectedOptions.value = []
    return
  }
  const response = await apiFetch(
    `/api/cloud/compute/github-credential-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`,
    { credentials: 'include', headers: { Accept: 'application/json' } },
  )
  const data = await response.json().catch(() => ({}))
  githubConnectedOptions.value = response.ok && Array.isArray(data.github_connections)
    ? data.github_connections.filter((x) => x && x.connected && x.github_user_id != null)
    : []
}

watch(repoUrls, () => {
  persistDraft()
}, { immediate: true })

// 评论列表/任务级身份异步到达后补一次预填（幂等，仅填空槽）。
watch(() => [props.comments, props.taskRepoIdentities], applyPrefillSelections, { deep: true })

onMounted(async () => {
  if (typeof window !== 'undefined') {
    rememberGrantTicketFromSearch(window.location?.search || '')
  }
  await loadGitIdentities()
  await loadGithubConnections()
  applyPrefillSelections()
  persistDraft()
})
</script>
