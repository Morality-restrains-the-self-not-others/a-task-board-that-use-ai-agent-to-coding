/**
 * 任务详情关联仓库 GitLab 探活：独立于任务 GET，失败不挡首屏。
 * Anti-Replay-OK: read-only probe POST; duplicates only re-probe.
 */
import { ref, unref, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { resolveRepoOAuthProviderInfo } from '../../utils/repoOAuthAuthorizeUtils.js'
import {
  indexValidateResultsByUrl,
  resolveValidateGitReposRequestFailure,
  VALIDATE_GIT_REPOS_TIMEOUT_MS,
} from '../../utils/gitRepoValidateError.js'

function normalizeRepoUrlKey(repoUrl) {
  return String(repoUrl || '').trim()
}

export function useTaskLinkedRepoGitProbe({ tenantId, repoUrls } = {}) {
  const gitRepoTokenStatus = ref({})
  const gitRepoStatusLoading = ref({})
  const gitRepoProbeErrorByUrl = ref({})
  const gitRepoProbeTraceIdByUrl = ref({})

  const collectTargets = () => {
    const urls = unref(repoUrls)
    const list = Array.isArray(urls) ? urls : []
    return list
      .map((u) => normalizeRepoUrlKey(u))
      .filter((u) => u && resolveRepoOAuthProviderInfo(u))
  }

  const fetchLinkedRepoGitProbe = async () => {
    const tid = String(unref(tenantId) || '').trim()
    const targets = collectTargets()
    if (!tid || targets.length === 0) {
      gitRepoStatusLoading.value = {}
      return
    }
    const loadingPatch = { ...gitRepoStatusLoading.value }
    targets.forEach((url) => {
      loadingPatch[normalizeRepoUrlKey(url)] = true
    })
    gitRepoStatusLoading.value = loadingPatch

    const errorPatch = { ...gitRepoProbeErrorByUrl.value }
    const tracePatch = { ...gitRepoProbeTraceIdByUrl.value }
    const statusPatch = { ...gitRepoTokenStatus.value }
    const seen = new Set()

    try {
      const response = await apiFetch(
        `/api/projects/validate-git-repos/tenant_id/${encodeURIComponent(tid)}/`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
          credentials: 'include',
          body: JSON.stringify({ urls: targets, probe_access: true }),
          timeout: VALIDATE_GIT_REPOS_TIMEOUT_MS,
        },
      )
      const data = await response.json().catch(() => ({}))
      const byUrl = indexValidateResultsByUrl(response.ok ? data?.results : null)
      const requestFail = resolveValidateGitReposRequestFailure({ response, data })
      targets.forEach((url) => {
        const key = normalizeRepoUrlKey(url)
        const entry = byUrl[url] || byUrl[key]
        seen.add(key)
        if (entry) {
          statusPatch[key] = String(entry.token_status || '').trim() || 'not_applicable'
          errorPatch[key] = ''
          tracePatch[key] = ''
          return
        }
        errorPatch[key] = requestFail.message
        tracePatch[key] = requestFail.traceId || ''
      })
    } catch (err) {
      const failure = resolveValidateGitReposRequestFailure({ err })
      targets.forEach((url) => {
        const key = normalizeRepoUrlKey(url)
        seen.add(key)
        errorPatch[key] = failure.message
        tracePatch[key] = failure.traceId || ''
      })
    }

    const loadingDone = { ...gitRepoStatusLoading.value }
    seen.forEach((key) => {
      delete loadingDone[key]
    })
    gitRepoTokenStatus.value = statusPatch
    gitRepoStatusLoading.value = loadingDone
    gitRepoProbeErrorByUrl.value = errorPatch
    gitRepoProbeTraceIdByUrl.value = tracePatch
  }

  watch(
    () => `${String(unref(tenantId) || '')}\n${collectTargets().join('\n')}`,
    () => {
      void fetchLinkedRepoGitProbe()
    },
    { immediate: true },
  )

  return {
    gitRepoTokenStatus,
    gitRepoStatusLoading,
    gitRepoProbeErrorByUrl,
    gitRepoProbeTraceIdByUrl,
    fetchLinkedRepoGitProbe,
  }
}
