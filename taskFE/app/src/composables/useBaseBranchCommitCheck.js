import { reactive } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { isCommitHashLike } from '../utils/commitHashLike.js'
import { traceIdFromBody } from '../utils/traceId.js'

/**
 * 创建任务：基准分支输入为 commit hash 时，校验远端是否存在该 commit。
 * @param {{ tenantId: import('vue').Ref<string>|(() => string), getRepoUrl: (projectId: string, repoIndex: number) => string }} opts
 */
export function useBaseBranchCommitCheck({ tenantId, getRepoUrl }) {
  /** @type {Record<string, { exists: boolean, checking: boolean, sha: string, error: string|null, checkedRef: string, traceId: string }>} */
  const commitCheckMap = reactive({})
  /** @type {Record<string, ReturnType<typeof setTimeout>>} */
  const timers = {}
  /** @type {Record<string, number>} */
  const requestSeq = {}

  const resolveTenantId = () => {
    const raw = typeof tenantId === 'function' ? tenantId() : tenantId?.value ?? tenantId
    return String(raw ?? '').trim()
  }

  const mapKey = (projectId, repoIndex) => `${String(projectId)}::${Number(repoIndex)}`

  const clearKey = (key) => {
    if (timers[key]) {
      clearTimeout(timers[key])
      delete timers[key]
    }
    delete commitCheckMap[key]
  }

  const getCommitCheckState = (projectId, repoIndex) => {
    const key = mapKey(projectId, repoIndex)
    return commitCheckMap[key] || null
  }

  const matchesCurrentHash = (state, baseBranch) => {
    if (!state) return false
    const current = String(baseBranch ?? '').trim().toLowerCase()
    return state.checkedRef === current && isCommitHashLike(current)
  }

  const isBaseBranchCommitChecking = (projectId, repoIndex, baseBranch) => {
    const state = getCommitCheckState(projectId, repoIndex)
    return Boolean(state?.checking && matchesCurrentHash(state, baseBranch))
  }

  const isBaseBranchCommitVerified = (projectId, repoIndex, baseBranch) => {
    const state = getCommitCheckState(projectId, repoIndex)
    return Boolean(state?.exists && !state.checking && matchesCurrentHash(state, baseBranch))
  }

  /** hash 已校验且远端不存在（或校验失败），用于弱提示 */
  const isBaseBranchCommitMissing = (projectId, repoIndex, baseBranch) => {
    const state = getCommitCheckState(projectId, repoIndex)
    if (!state || state.checking || state.exists) return false
    return matchesCurrentHash(state, baseBranch)
  }

  const getBaseBranchCommitCheckTraceId = (projectId, repoIndex, baseBranch) => {
    const state = getCommitCheckState(projectId, repoIndex)
    if (!state || !matchesCurrentHash(state, baseBranch)) return ''
    return typeof state.traceId === 'string' ? state.traceId : ''
  }

  const getBaseBranchCommitMissingHint = (projectId, repoIndex, baseBranch) => {
    if (!isBaseBranchCommitMissing(projectId, repoIndex, baseBranch)) return ''
    const state = getCommitCheckState(projectId, repoIndex)
    if (state?.error) return state.error
    return '未找到该 commit'
  }

  const scheduleBaseBranchCommitCheck = (projectId, repoIndex, baseBranch) => {
    const key = mapKey(projectId, repoIndex)
    const ref = String(baseBranch ?? '').trim()

    if (timers[key]) {
      clearTimeout(timers[key])
      delete timers[key]
    }

    if (!projectId || !isCommitHashLike(ref)) {
      clearKey(key)
      return
    }

    commitCheckMap[key] = {
      exists: false,
      checking: true,
      sha: '',
      error: null,
      checkedRef: ref.toLowerCase(),
      traceId: '',
    }

    timers[key] = setTimeout(() => {
      delete timers[key]
      void runCheck(projectId, repoIndex, ref)
    }, 400)
  }

  const resolveTraceId = (response, data, error) => {
    if (response?.traceId) return String(response.traceId)
    const fromBody = traceIdFromBody(data)
    if (fromBody) return fromBody
    if (error?.traceId) return String(error.traceId)
    return ''
  }

  const runCheck = async (projectId, repoIndex, ref) => {
    const key = mapKey(projectId, repoIndex)
    const tid = resolveTenantId()
    const repoUrl = getRepoUrl(projectId, repoIndex)
    const seq = (requestSeq[key] || 0) + 1
    requestSeq[key] = seq

    if (!tid || !repoUrl) {
      commitCheckMap[key] = {
        exists: false,
        checking: false,
        sha: '',
        error: !repoUrl ? '未找到对应仓库地址' : null,
        checkedRef: ref.toLowerCase(),
        traceId: '',
      }
      return
    }

    try {
      const params = new URLSearchParams({ repo_url: repoUrl, ref })
      const response = await apiFetch(
        `/api/projects/tenant_id/${tid}/${encodeURIComponent(String(projectId))}/resolve-ref/?${params.toString()}`,
        {
          credentials: 'include',
          headers: { Accept: 'application/json' },
        }
      )
      if (requestSeq[key] !== seq) return

      if (!response.ok) {
        let errData = null
        try {
          errData = await response.json()
        } catch {
          errData = null
        }
        commitCheckMap[key] = {
          exists: false,
          checking: false,
          sha: '',
          error: `校验失败（HTTP ${response.status}）`,
          checkedRef: ref.toLowerCase(),
          traceId: resolveTraceId(response, errData, null),
        }
        return
      }

      let data = null
      try {
        data = await response.json()
      } catch {
        data = null
      }
      if (requestSeq[key] !== seq) return

      commitCheckMap[key] = {
        exists: Boolean(data?.exists),
        checking: false,
        sha: typeof data?.sha === 'string' ? data.sha : '',
        error: typeof data?.error === 'string' && data.error.trim() ? data.error.trim() : null,
        checkedRef: ref.toLowerCase(),
        traceId: resolveTraceId(response, data, null),
      }
    } catch (e) {
      if (requestSeq[key] !== seq) return
      commitCheckMap[key] = {
        exists: false,
        checking: false,
        sha: '',
        error: e?.message || '网络错误',
        checkedRef: ref.toLowerCase(),
        traceId: resolveTraceId(null, null, e),
      }
    }
  }

  const resetAllCommitChecks = () => {
    Object.keys(timers).forEach((k) => {
      clearTimeout(timers[k])
      delete timers[k]
    })
    Object.keys(commitCheckMap).forEach((k) => {
      delete commitCheckMap[k]
    })
  }

  return {
    commitCheckMap,
    getCommitCheckState,
    isBaseBranchCommitChecking,
    isBaseBranchCommitVerified,
    isBaseBranchCommitMissing,
    getBaseBranchCommitCheckTraceId,
    getBaseBranchCommitMissingHint,
    scheduleBaseBranchCommitCheck,
    resetAllCommitChecks,
  }
}
