/**
 * GitHub account binding status / auto-save / OAuth-return refresh for linked projects panel.
 */
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { createGithubRepoBindingApi } from '../useGithubRepoBinding.js'
import {
  githubRepoSlugFromUrl,
  resolveSelectedGithubUserIdForRepo,
} from '../../utils/taskDetailBranchAndRepoUtils.js'

/**
 * @param {object} deps
 * @param {object} deps.props
 * @param {() => Promise<void>} deps.fetchRepoOAuthConnectionStatusByRepoUrl
 */
export function useLinkedProjectsGithubBindings({ props, fetchRepoOAuthConnectionStatusByRepoUrl }) {
  const route = useRoute()
  const router = useRouter()

  const githubRepoBindingLoading = ref(false)
  const githubRepoBindingSaving = ref(false)
  const githubBindingSavingRepoUrl = ref('')
  const githubRepoBindingError = ref('')
  const githubRepoBindingErrorTraceId = ref('')
  const githubConnectedOptions = ref([])
  const repoBindingBySlug = ref({})
  const selectedGithubUserIdBySlug = ref({})

  const isGithubRepo = (repoUrl) => Boolean(githubRepoSlugFromUrl(repoUrl))

  const selectedGithubUserIdForRepo = (repoUrl) => {
    const slug = githubRepoSlugFromUrl(repoUrl)
    if (!slug) return ''
    return resolveSelectedGithubUserIdForRepo({
      draft: selectedGithubUserIdBySlug.value[slug],
      boundUserId: repoBindingBySlug.value[slug]?.selected_github_user_id,
      connectedOptions: githubConnectedOptions.value,
    })
  }

  const sleep = (ms) =>
    new Promise((resolve) => {
      setTimeout(resolve, ms)
    })

  const clearGithubQueryFlag = async () => {
    if (!Object.prototype.hasOwnProperty.call(route.query || {}, 'github')) return
    const nextQuery = { ...route.query }
    delete nextQuery.github
    try {
      await router.replace({
        path: route.path,
        query: nextQuery,
        hash: route.hash || undefined,
      })
    } catch {
      // ignore duplicated navigation
    }
  }

  const { fetchGithubRepoBindingStatus, saveGithubRepoBinding } = createGithubRepoBindingApi({
    getIds: () => ({
      tenantId: String(props.tenantId || '').trim(),
      workspaceId: String(props.workspaceId || '').trim(),
      taskId: String(props.taskId || '').trim(),
    }),
    refs: {
      githubRepoBindingLoading,
      githubRepoBindingSaving,
      githubBindingSavingRepoUrl,
      githubRepoBindingError,
      githubRepoBindingErrorTraceId,
      githubConnectedOptions,
      repoBindingBySlug,
      selectedGithubUserIdBySlug,
    },
    selectedGithubUserIdForRepo,
    fetchRepoOAuthConnectionStatusByRepoUrl,
  })

  const fetchGithubRepoBindingStatusAfterOauth = async () => {
    const attempts = 6
    const delayMs = 350
    for (let i = 0; i < attempts; i += 1) {
      await fetchGithubRepoBindingStatus()
      if (i < attempts - 1) {
        await sleep(delayMs)
      }
    }
  }

  const onGithubRepoAccountChange = (repoUrl, value) => {
    const slug = githubRepoSlugFromUrl(repoUrl)
    if (!slug) return
    selectedGithubUserIdBySlug.value = {
      ...selectedGithubUserIdBySlug.value,
      [slug]: String(value || '').trim(),
    }
  }

  /** 自动保存守卫：仅在账户状态回填完毕后执行一次，避免重复触发 */
  let autoSaveGuard = false

  watch(
    () => githubConnectedOptions.value.length,
    (count) => {
      if (autoSaveGuard) return
      if (count !== 1) return
      if (githubRepoBindingLoading.value) return
      if (githubRepoBindingSaving.value) return
      if (githubRepoBindingError.value) return
      if (props.isEditing) return
      const taskProjects = Array.isArray(props.taskProjectsWithDetails) ? props.taskProjectsWithDetails : []
      for (const tp of taskProjects) {
        const repos = Array.isArray(tp.project?.git_repos) ? tp.project.git_repos : []
        for (const repoUrl of repos) {
          if (!isGithubRepo(repoUrl)) continue
          const slug = githubRepoSlugFromUrl(repoUrl)
          if (!slug) continue
          if (repoBindingBySlug.value[slug]?.selected_github_user_id) continue
          const ghUserId = selectedGithubUserIdForRepo(repoUrl)
          if (!ghUserId) continue
          autoSaveGuard = true
          void saveGithubRepoBinding(repoUrl).finally(() => {
            autoSaveGuard = false
          })
          return
        }
      }
    },
  )

  watch(
    () => [props.tenantId, props.workspaceId, props.taskId, props.isEditing],
    () => {
      if (props.isEditing) return
      void fetchGithubRepoBindingStatus()
    },
    { immediate: true },
  )

  watch(
    () => props.taskProjectsWithDetails,
    () => {
      if (props.isEditing) return
      void fetchGithubRepoBindingStatus()
    },
    { deep: true },
  )

  watch(
    () => props.gitOAuthCatalogVersion,
    (version, prev) => {
      if (props.isEditing) return
      if (version === prev) return
      void fetchGithubRepoBindingStatus()
    },
  )

  watch(
    () => route.query.github,
    async (value) => {
      if (String(value || '') !== 'ok' || props.isEditing) return
      await fetchGithubRepoBindingStatusAfterOauth()
      await clearGithubQueryFlag()
    },
    { immediate: true },
  )

  return {
    githubRepoBindingLoading,
    githubRepoBindingSaving,
    githubBindingSavingRepoUrl,
    githubRepoBindingError,
    githubRepoBindingErrorTraceId,
    githubConnectedOptions,
    isGithubRepo,
    selectedGithubUserIdForRepo,
    onGithubRepoAccountChange,
    saveGithubRepoBinding,
    fetchGithubRepoBindingStatus,
  }
}
