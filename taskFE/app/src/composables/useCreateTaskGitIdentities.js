import { computed, onMounted, ref, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { resolveAuthenticatedUserId } from '../utils/sessionUserIdUtils.js'
import {
  collectCreateTaskRepoUrls,
  formatGitIdentityOptionLabel,
  normalizeCreateTaskRepoIdentities,
} from '../utils/createTaskGitIdentityGate.js'

/**
 * @param {{
 *   editingTask: () => object|null,
 *   projects: () => unknown[],
 *   tenantId: () => string|number|null,
 *   enabled: () => boolean,
 *   repoUrls?: () => string[],
 * }} opts
 */
export function useCreateTaskGitIdentities({ editingTask, projects, tenantId, enabled, repoUrls }) {
  const gitIdentities = ref([])
  const loadError = ref('')
  const loadErrorTraceId = ref('')
  const identitiesLoading = ref(false)

  const settingsHref = computed(() => {
    const tid = String(tenantId() || '').trim()
    if (!tid) return '/profile/git-identities/'
    return `/tenant/${encodeURIComponent(tid)}/profile/git-identities/`
  })

  const requiredRepoUrls = () => {
    if (typeof repoUrls === 'function') {
      const urls = repoUrls()
      return Array.isArray(urls) ? urls : []
    }
    return collectCreateTaskRepoUrls(editingTask(), projects())
  }

  const applyDefaultIdentities = () => {
    if (!enabled()) return
    const task = editingTask()
    if (!task) return
    const urls = requiredRepoUrls()
    if (!urls.length) return
    const defaultId = gitIdentities.value.find((row) => row && row.is_default)
    const defaultIdentityId = String(defaultId?.id || '').trim()
    if (!defaultIdentityId) return
    const current = normalizeCreateTaskRepoIdentities(task.repo_identities)
    const byUrl = new Map(current.map((row) => [row.repo_url, row.git_identity_id]))
    let changed = false
    for (const url of urls) {
      if (byUrl.get(url)) continue
      byUrl.set(url, defaultIdentityId)
      changed = true
    }
    if (!changed) return
    task.repo_identities = urls.map((repo_url) => ({
      repo_url,
      git_identity_id: String(byUrl.get(repo_url) || ''),
    }))
  }

  const loadGitIdentities = async () => {
    identitiesLoading.value = true
    loadError.value = ''
    loadErrorTraceId.value = ''
    try {
      if (!enabled()) {
        gitIdentities.value = []
        return
      }
      const userId = await resolveAuthenticatedUserId()
      const companyId = String(tenantId() || '').trim()
      if (!userId) {
        gitIdentities.value = []
        return
      }
      const qs = companyId ? `?company_id=${encodeURIComponent(companyId)}` : ''
      const response = await apiFetch(
        `/api/git-identities/user/${encodeURIComponent(userId)}/${qs}`,
        { credentials: 'include', headers: { Accept: 'application/json' } },
      )
      const data = await response.json().catch(() => ({}))
      gitIdentities.value = response.ok && Array.isArray(data.identities) ? data.identities : []
      if (!response.ok) {
        loadError.value = String(data.detail || data.error || `HTTP ${response.status}`).trim()
        loadErrorTraceId.value = String(response.traceId || data.trace_id || '').trim()
      }
      applyDefaultIdentities()
    } finally {
      identitiesLoading.value = false
    }
  }

  const gitIdentityIdForUrl = (repoUrl) => {
    const url = String(repoUrl || '').trim()
    const rows = normalizeCreateTaskRepoIdentities(editingTask()?.repo_identities)
    return rows.find((row) => row.repo_url === url)?.git_identity_id || ''
  }

  const setGitIdentityIdForUrl = (repoUrl, identityId) => {
    const task = editingTask()
    if (!task) return
    const url = String(repoUrl || '').trim()
    if (!url) return
    const nextId = String(identityId || '').trim()
    const rows = normalizeCreateTaskRepoIdentities(task.repo_identities).filter((row) => row.repo_url !== url)
    if (nextId) rows.push({ repo_url: url, git_identity_id: nextId })
    task.repo_identities = rows
  }

  watch(
    () => [enabled(), requiredRepoUrls().join('\0')],
    () => {
      if (enabled()) {
        applyDefaultIdentities()
      }
    },
  )

  onMounted(() => {
    void loadGitIdentities()
  })

  watch(enabled, (on) => {
    if (on) void loadGitIdentities()
  })

  return {
    gitIdentities,
    identitiesLoading,
    loadError,
    loadErrorTraceId,
    settingsHref,
    gitIdentityIdForUrl,
    setGitIdentityIdForUrl,
    formatGitIdentityOptionLabel,
    loadGitIdentities,
  }
}
