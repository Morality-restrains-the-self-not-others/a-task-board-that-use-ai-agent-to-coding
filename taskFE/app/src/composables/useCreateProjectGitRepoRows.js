import { ref, computed, watch } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { resolveRepoOAuthAuthorizeUrl, resolveRepoOAuthProviderInfo } from '../utils/repoOAuthAuthorizeUtils.js'
import { useProjectGitOAuthCatalog } from './useProjectGitOAuthCatalog.js'
import { createGithubAppReturnKey, setGithubAppReturnTarget } from '../utils/githubAppReturnStorage.js'
import { buildOauthReturnPath } from '../utils/githubAppContinueNav.js'
import { applyOAuthCallbackFromRoute } from '../utils/gitSiteOAuthCallbackUtils.js'
import { rememberGrantTicket, rememberGrantTicketFromSearch } from '../utils/grantTicketSession.js'
import {
  INVALID_REPO_URL_MSG,
  GIT_REPO_URL_EXAMPLES,
  GIT_REPO_URL_FORMAT_HINT,
  isValidGitRepoUrl,
} from '../utils/gitRepoUrlUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import {
  REPO_INACCESSIBLE_MSG,
  VALIDATE_GIT_REPOS_TIMEOUT_MS,
  indexValidateResultsByUrl,
  resolveValidateGitReposRequestFailure,
} from '../utils/gitRepoValidateError.js'

export {
  INVALID_REPO_URL_MSG,
  GIT_REPO_URL_EXAMPLES,
  GIT_REPO_URL_FORMAT_HINT,
  REPO_INACCESSIBLE_MSG,
}

/** 创建项目页：多 Git 仓库行校验与 OAuth 授权流程 */
export function useCreateProjectGitRepoRows(route, router) {
  let gitRepoRowIdSeq = 1
  const gitRepoRows = ref([{ id: gitRepoRowIdSeq, url: '', cloneAlias: '' }])
  const gitRepoRowValidating = ref({})
  const gitRepoRowLastValidated = ref({})
  const gitRepoRowErrors = ref({})
  const gitRepoRowErrorTraceId = ref({})
  const validateGitRepoDebounceTimers = {} // legacy per-row cleared on remove
  let batchValidateTimer = null
  const pendingValidateByRowId = {}
  const repoOAuthActionLoadingByUrl = ref({})
  const repoOAuthActionErrorByUrl = ref({})
  const repoOAuthActionErrorTraceIdByUrl = ref({})
  const gitRepoRowTokenStatus = ref({})
  const retryValidateGuard = createClickGuard({ debounceMs: 300 })

  const setRowError = (rowId, message, traceId = '') => {
    gitRepoRowErrors.value = { ...gitRepoRowErrors.value, [rowId]: message }
    const nextT = { ...gitRepoRowErrorTraceId.value }
    const tid = String(traceId || '').trim()
    if (tid) nextT[rowId] = tid
    else delete nextT[rowId]
    gitRepoRowErrorTraceId.value = nextT
  }

  const clearRowError = (rowId) => {
    if (!(rowId in gitRepoRowErrors.value) && !(rowId in gitRepoRowErrorTraceId.value)) return
    const nextE = { ...gitRepoRowErrors.value }
    delete nextE[rowId]
    gitRepoRowErrors.value = nextE
    const nextT = { ...gitRepoRowErrorTraceId.value }
    delete nextT[rowId]
    gitRepoRowErrorTraceId.value = nextT
  }

  const setGitRepoRowFormatError = (rowId, hasError) => {
    if (hasError) {
      setRowError(rowId, INVALID_REPO_URL_MSG)
      return
    }
    if (gitRepoRowErrors.value[rowId] === INVALID_REPO_URL_MSG) {
      clearRowError(rowId)
    }
  }

  const { touchRepoOAuthButtons, bootstrapGitOAuthCatalog } = useProjectGitOAuthCatalog(
    apiFetch,
    () => String(route.params.tenant || ''),
  )

  const setRepoOAuthActionLoading = (repoUrl, value) => {
    const key = String(repoUrl || '').trim()
    repoOAuthActionLoadingByUrl.value = { ...repoOAuthActionLoadingByUrl.value, [key]: value }
  }
  const setRepoOAuthActionError = (repoUrl, errorMessage = '', traceId = '') => {
    const key = String(repoUrl || '').trim()
    const message = String(errorMessage || '').trim()
    const tid = message ? String(traceId || '').trim() : ''
    repoOAuthActionErrorByUrl.value = { ...repoOAuthActionErrorByUrl.value, [key]: message }
    repoOAuthActionErrorTraceIdByUrl.value = {
      ...repoOAuthActionErrorTraceIdByUrl.value,
      [key]: tid,
    }
  }
  const isRepoOAuthActionDisabled = (repoUrl) => {
    const key = String(repoUrl || '').trim()
    return Boolean(repoOAuthActionLoadingByUrl.value[key])
  }
  const repoOAuthButtonLabel = (repoUrl) => {
    const key = String(repoUrl || '').trim()
    if (repoOAuthActionLoadingByUrl.value[key]) return '跳转中...'
    if (gitRepoRowTokenStatus.value[key] === 'token_error') return '重试'
    return 'OAuth 授权'
  }
  const shouldShowRepoOAuthButton = (repoUrl) => {
    touchRepoOAuthButtons()
    const info = resolveRepoOAuthProviderInfo(repoUrl)
    if (!info) return false
    const key = String(repoUrl || '').trim()
    const ts = gitRepoRowTokenStatus.value[key]
    if (ts === 'token_available') return false
    return Boolean(resolveRepoOAuthAuthorizeUrl(info.provider))
  }
  const repoOAuthErrorByUrl = (repoUrl) => {
    const key = String(repoUrl || '').trim()
    return String(repoOAuthActionErrorByUrl.value[key] || '').trim()
  }
  const repoOAuthErrorTraceIdByUrl = (repoUrl) => {
    const key = String(repoUrl || '').trim()
    return String(repoOAuthActionErrorTraceIdByUrl.value[key] || '').trim()
  }

  const trimmedGitRepos = () =>
    gitRepoRows.value.map((r) => String(r.url || '').trim()).filter(Boolean)

  /** 提交用：有别名时发对象，否则发字符串（向后兼容）。 */
  const trimmedGitRepoPayload = () => {
    const out = []
    for (const r of gitRepoRows.value) {
      const url = String(r.url || '').trim()
      if (!url) continue
      const cloneAlias = String(r.cloneAlias || '').trim()
      if (cloneAlias) {
        out.push({ url, clone_alias: cloneAlias })
      } else {
        out.push(url)
      }
    }
    return out
  }

  const duplicateCloneAliasError = () => {
    const seen = new Map()
    for (const r of gitRepoRows.value) {
      const alias = String(r.cloneAlias || '').trim().toLowerCase()
      if (!alias) continue
      if (seen.has(alias)) return '同一项目内仓库别名不能重复'
      seen.set(alias, r.id)
    }
    return ''
  }

  const addGitRepoRow = () => {
    gitRepoRowIdSeq += 1
    gitRepoRows.value = [...gitRepoRows.value, { id: gitRepoRowIdSeq, url: '', cloneAlias: '' }]
  }

  const removeGitRepoRow = (rowId) => {
    if (gitRepoRows.value.length <= 1) return
    if (validateGitRepoDebounceTimers[rowId]) {
      clearTimeout(validateGitRepoDebounceTimers[rowId])
      delete validateGitRepoDebounceTimers[rowId]
    }
    delete pendingValidateByRowId[rowId]
    gitRepoRows.value = gitRepoRows.value.filter((r) => r.id !== rowId)
    const { [rowId]: _e, ...restErr } = gitRepoRowErrors.value
    const { [rowId]: _tid, ...restTid } = gitRepoRowErrorTraceId.value
    const { [rowId]: _v, ...restVal } = gitRepoRowLastValidated.value
    const { [rowId]: _l, ...restLoading } = gitRepoRowValidating.value
    const { [rowId]: _ts, ...restTokenStatus } = gitRepoRowTokenStatus.value
    gitRepoRowErrors.value = restErr
    gitRepoRowErrorTraceId.value = restTid
    gitRepoRowLastValidated.value = restVal
    gitRepoRowValidating.value = restLoading
    gitRepoRowTokenStatus.value = restTokenStatus
  }

  const gitRepoRowErrorClass = (rowId) => Boolean(gitRepoRowErrors.value[rowId])

  const resolveCurrentPagePathForOauth = () =>
    buildOauthReturnPath({
      pathname: String(window.location?.pathname || ''),
      search: String(window.location?.search || ''),
    })

  const startRepoOAuthConnect = async (repoUrl) => {
    const info = resolveRepoOAuthProviderInfo(repoUrl)
    if (!info) return
    const startApiUrl = resolveRepoOAuthAuthorizeUrl(info.provider)
    if (!startApiUrl) return
    setRepoOAuthActionError(repoUrl, '')
    setRepoOAuthActionLoading(repoUrl, true)
    try {
      const nextPath = resolveCurrentPagePathForOauth()
      const returnKey = createGithubAppReturnKey()
      setGithubAppReturnTarget(returnKey, nextPath)
      const targetUrl =
        `${startApiUrl}?next=${encodeURIComponent(nextPath)}` +
        `&return_key=${encodeURIComponent(returnKey)}` +
        `&repo_url=${encodeURIComponent(repoUrl)}` +
        `&service_provider=${encodeURIComponent(info.service_provider)}`
      const response = await apiFetch(targetUrl, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
        timeout: 10000,
      })
      const data = await response.json().catch(() => ({}))
      const requestTraceId = extractTraceId(response) || extractTraceId(data) || ''
      if (!response.ok) {
        setRepoOAuthActionError(
          repoUrl,
          typeof data?.detail === 'string' ? data.detail : '无法启动 OAuth 授权',
          requestTraceId,
        )
        return
      }
      if (data.authorize_url) {
        window.location.href = data.authorize_url
        return
      }
      setRepoOAuthActionError(repoUrl, '无法启动 OAuth 授权', requestTraceId)
    } catch (e) {
      const isTimeout = e?.name === 'TimeoutError'
      setRepoOAuthActionError(
        repoUrl,
        isTimeout ? 'OAuth 授权服务响应超时，请检查网络后重试' : (e?.message || '启动 OAuth 授权失败'),
        extractTraceId(e),
      )
    } finally {
      setRepoOAuthActionLoading(repoUrl, false)
    }
  }

  const applyValidateResultToRow = (rowId, url, data) => {
    const trimmed = String(url || '').trim()
    gitRepoRowLastValidated.value = { ...gitRepoRowLastValidated.value, [rowId]: trimmed }
    if (data?.token_status) {
      gitRepoRowTokenStatus.value = {
        ...gitRepoRowTokenStatus.value,
        [trimmed]: String(data.token_status),
      }
    }
    if (data && data.is_accessible === false) {
      setRowError(rowId, REPO_INACCESSIBLE_MSG)
    } else {
      clearRowError(rowId)
    }
  }

  const markRowValidated = (rowId, url) => {
    gitRepoRowLastValidated.value = { ...gitRepoRowLastValidated.value, [rowId]: url }
  }

  const applyRequestFailureToTargets = (targets, failure) => {
    targets.forEach(({ rowId, url }) => {
      markRowValidated(rowId, url)
      setRowError(rowId, failure.message, failure.traceId)
    })
  }

  const flushValidateGitRepos = async (opts = {}) => {
    const pending = { ...pendingValidateByRowId }
    Object.keys(pendingValidateByRowId).forEach((k) => {
      delete pendingValidateByRowId[k]
    })
    const targets = []
    for (const [rowIdRaw, repoUrl] of Object.entries(pending)) {
      const rowId = Number.isNaN(Number(rowIdRaw)) ? rowIdRaw : Number(rowIdRaw)
      const trimmed = String(repoUrl || '').trim()
      if (!trimmed) {
        clearRowError(rowId)
        const nextL = { ...gitRepoRowLastValidated.value }
        delete nextL[rowId]
        gitRepoRowLastValidated.value = nextL
        continue
      }
      if (!isValidGitRepoUrl(trimmed)) {
        setGitRepoRowFormatError(rowId, true)
        const nextL = { ...gitRepoRowLastValidated.value }
        delete nextL[rowId]
        gitRepoRowLastValidated.value = nextL
        continue
      }
      setGitRepoRowFormatError(rowId, false)
      targets.push({ rowId, url: trimmed })
    }
    if (!targets.length) return

    const validatingPatch = { ...gitRepoRowValidating.value }
    targets.forEach(({ rowId }) => {
      validatingPatch[rowId] = true
    })
    gitRepoRowValidating.value = validatingPatch

    const urls = [...new Set(targets.map((t) => t.url))]
    try {
      // taskProjectService 分发器要求 tenant_id kv 键值对（无 kv 会 400 tenant_id required）；
      // 约定：action 位置段在前、kv 键值对在后
      const tid = String(route?.params?.tenant || '').trim()
      const response = await apiFetch(
        `/api/projects/validate-git-repos/tenant_id/${encodeURIComponent(tid)}/`,
        {
          method: 'POST',
          credentials: 'include',
          headers: mergeIdempotencyHeaders(
            { 'Content-Type': 'application/json', Accept: 'application/json' },
            opts.idempotencyKey,
          ),
          body: JSON.stringify({ urls, probe_access: true }),
          timeout: VALIDATE_GIT_REPOS_TIMEOUT_MS,
        },
      )
      const data = await response.json().catch(() => ({}))
      const byUrl = indexValidateResultsByUrl(response.ok ? data?.results : null)
      if (!response.ok || !Array.isArray(data?.results)) {
        console.error('Batch validate-git-repos failed', response.status, data)
      }
      const requestFail = resolveValidateGitReposRequestFailure({ response, data })
      targets.forEach(({ rowId, url }) => {
        const entry = byUrl[url]
        if (entry) {
          applyValidateResultToRow(rowId, url, entry)
          return
        }
        markRowValidated(rowId, url)
        setRowError(rowId, requestFail.message, requestFail.traceId)
      })
    } catch (err) {
      console.error('批量校验 Git 仓库失败:', err)
      applyRequestFailureToTargets(targets, resolveValidateGitReposRequestFailure({ err }))
    } finally {
      const nextV = { ...gitRepoRowValidating.value }
      targets.forEach(({ rowId }) => {
        delete nextV[rowId]
      })
      gitRepoRowValidating.value = nextV
    }
  }

  const scheduleValidateGitRepoRow = (rowId, repoUrl) => {
    pendingValidateByRowId[rowId] = String(repoUrl || '').trim()
    if (validateGitRepoDebounceTimers[rowId]) {
      clearTimeout(validateGitRepoDebounceTimers[rowId])
      delete validateGitRepoDebounceTimers[rowId]
    }
    if (batchValidateTimer) clearTimeout(batchValidateTimer)
    batchValidateTimer = setTimeout(() => {
      batchValidateTimer = null
      void flushValidateGitRepos()
    }, 500)
  }

  const onGitRepoRowBlur = (rowId) => {
    const row = gitRepoRows.value.find((r) => r.id === rowId)
    if (!row) return
    pendingValidateByRowId[rowId] = String(row.url || '').trim()
    if (batchValidateTimer) {
      clearTimeout(batchValidateTimer)
      batchValidateTimer = null
    }
    void flushValidateGitRepos()
  }

  const retryValidateGitRepoRow = async (rowId) => {
    const row = gitRepoRows.value.find((r) => r.id === rowId)
    if (!row) return { skipped: true, reason: 'missing-row' }
    pendingValidateByRowId[rowId] = String(row.url || '').trim()
    if (batchValidateTimer) {
      clearTimeout(batchValidateTimer)
      batchValidateTimer = null
    }
    return retryValidateGuard.run(async ({ idempotencyKey }) => {
      await flushValidateGitRepos({ idempotencyKey })
    })
  }

  const gitReposHaveValidUrls = () => {
    for (const url of trimmedGitRepos()) {
      if (!isValidGitRepoUrl(url)) return false
    }
    return true
  }

  const gitReposPendingValidation = computed(() => {
    const repos = trimmedGitRepos()
    if (repos.length === 0) return false
    for (const row of gitRepoRows.value) {
      const t = String(row.url || '').trim()
      if (!t) continue
      if (gitRepoRowValidating.value[row.id]) return true
      if (t !== (gitRepoRowLastValidated.value[row.id] || '')) return true
    }
    return false
  })

  const appendGitRepoFormErrors = (newErrors) => {
    for (const row of gitRepoRows.value) {
      const url = String(row.url || '').trim()
      if (!url) continue
      if (!isValidGitRepoUrl(url)) {
        setGitRepoRowFormatError(row.id, true)
        newErrors.git_repos = INVALID_REPO_URL_MSG
        return
      }
    }
  }

  watch(
    gitRepoRows,
    (rows) => {
      for (const row of rows) {
        const trimmed = String(row.url || '').trim()
        if (!trimmed) {
          clearRowError(row.id)
          const nextL = { ...gitRepoRowLastValidated.value }
          delete nextL[row.id]
          gitRepoRowLastValidated.value = nextL
          continue
        }
        if (!isValidGitRepoUrl(trimmed)) {
          setGitRepoRowFormatError(row.id, true)
          const nextL = { ...gitRepoRowLastValidated.value }
          delete nextL[row.id]
          gitRepoRowLastValidated.value = nextL
          continue
        }
        setGitRepoRowFormatError(row.id, false)
        scheduleValidateGitRepoRow(row.id, trimmed)
      }
    },
    { deep: true }
  )

  const bootstrapGitReposOnMount = async () => {
    await bootstrapGitOAuthCatalog()
    const search = typeof window !== 'undefined' ? String(window.location?.search || '') : ''
    const queryRepo = String(route?.query?.repo_url || '')
    rememberGrantTicketFromSearch(search, queryRepo)
    const queryTicket = String(route?.query?.grant_ticket || '').trim()
    if (queryTicket) {
      rememberGrantTicket(queryTicket, queryRepo)
    }
    // OPT-20260902-006：OAuth 回流后 repo_url 写回第一行 Git 仓库输入框（仅首行空白时），
    // 整页跳转后表单仓库 URL 不丢，用户无需再粘贴。
    const firstRow = gitRepoRows.value[0]
    if (queryRepo && firstRow && !String(firstRow.url || '').trim()) {
      const filled = { ...firstRow, url: queryRepo }
      gitRepoRows.value = [filled, ...gitRepoRows.value.slice(1)]
      scheduleValidateGitRepoRow(filled.id, queryRepo)
    }
    const oauthResult = await applyOAuthCallbackFromRoute(route, router, {
      onError: () => {},
      onSuccess: () => {},
    })
    if (oauthResult) {
      gitRepoRows.value.forEach((row) => {
        const url = String(row.url || '').trim()
        if (url) scheduleValidateGitRepoRow(row.id, url)
      })
    }
  }

  return {
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
    trimmedGitRepos,
    trimmedGitRepoPayload,
    duplicateCloneAliasError,
    gitReposHaveValidUrls,
    gitReposPendingValidation,
    appendGitRepoFormErrors,
    bootstrapGitReposOnMount,
  }
}
