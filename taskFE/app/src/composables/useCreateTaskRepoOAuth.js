import { computed, ref, watch } from 'vue'
import { apiFetch as defaultApiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { resolveAuthenticatedUserId } from '../utils/sessionUserIdUtils.js'
import { hasSessionGrantForRepo, rememberGrantTicketFromSearch } from '../utils/grantTicketSession.js'
import {
  buildCreateTaskProjectListBoundMap,
  collectCreateTaskOAuthRepoRows,
  collectCreateTaskOAuthRepoUrls,
  fetchCreateTaskProjectL2BoundByUrl,
  resolveCreateTaskOauthBlockedReason,
} from '../utils/createTaskOauthGate.js'

/**
 * @param {{
 *   editingTask?: () => object|null,
 *   projects?: () => unknown[],
 *   enabled: () => boolean,
 *   repoUrls?: () => string[],
 *   repoRows?: () => { url: string, projectId: string }[],
 *   tenantId?: () => string,
 *   apiFetch?: Function,
 *   resolveBlockedReason?: (readiness: object) => string,
 * }} opts
 */
export function useCreateTaskRepoOAuth({
  editingTask = () => null,
  projects = () => [],
  enabled,
  repoUrls,
  repoRows,
  tenantId = () => '',
  apiFetch = defaultApiFetch,
  resolveBlockedReason = resolveCreateTaskOauthBlockedReason,
}) {
  const boundByUrl = ref({})
  const loadingByUrl = ref({})
  const errorByUrl = ref({})
  const errorTraceIdByUrl = ref({})
  let fetchGeneration = 0

  const oauthRepoRows = computed(() => {
    if (typeof repoRows === 'function') {
      return repoRows() || []
    }
    return collectCreateTaskOAuthRepoRows(editingTask(), projects())
  })

  const oauthRepoUrls = computed(() => {
    if (typeof repoUrls === 'function') {
      return repoUrls()
    }
    return collectCreateTaskOAuthRepoUrls(editingTask(), projects())
  })

  const fetchConnections = async () => {
    const generation = ++fetchGeneration
    const applyMaps = (loadingMap, boundMap, errorMap, traceMap) => {
      if (generation !== fetchGeneration) return
      loadingByUrl.value = loadingMap
      boundByUrl.value = boundMap
      errorByUrl.value = errorMap
      errorTraceIdByUrl.value = traceMap
    }
    if (!enabled()) {
      applyMaps({}, {}, {}, {})
      return
    }
    const urls = oauthRepoUrls.value
    const loadingMap = {}
    const boundMap = {}
    const errorMap = {}
    const traceMap = {}
    for (const url of urls) {
      loadingMap[url] = true
      boundMap[url] = false
      errorMap[url] = ''
      traceMap[url] = ''
    }
    applyMaps({ ...loadingMap }, { ...boundMap }, { ...errorMap }, { ...traceMap })
    if (urls.length === 0) return

    if (typeof window !== 'undefined') {
      rememberGrantTicketFromSearch(window.location.search || '')
    }
    // OPT-20260902-020: 列表/详情缓存带真实 token_available 时视为 bound，跳过 validate-git-repos POST
    const listBoundByUrl = buildCreateTaskProjectListBoundMap(oauthRepoRows.value, projects())
    const isListBound = (url) => listBoundByUrl[url] === true
    const userId = await resolveAuthenticatedUserId()
    if (generation !== fetchGeneration) return
    if (!userId) {
      for (const url of urls) {
        if (hasSessionGrantForRepo(url) || isListBound(url)) {
          boundMap[url] = true
          loadingMap[url] = false
          continue
        }
        loadingMap[url] = false
        errorMap[url] = '缺少 userId，无法检查 OAuth 绑定状态'
      }
      applyMaps({ ...loadingMap }, { ...boundMap }, { ...errorMap }, { ...traceMap })
      return
    }

    const unbound = []
    for (const url of urls) {
      try {
        if (hasSessionGrantForRepo(url) || isListBound(url)) {
          boundMap[url] = true
          loadingMap[url] = false
          errorMap[url] = ''
          traceMap[url] = ''
          continue
        }
      } catch (error) {
        errorMap[url] = error?.message || 'OAuth 绑定状态检查失败'
        traceMap[url] = error?.traceId || extractTraceId(error) || ''
      }
      unbound.push(url)
    }

    if (unbound.length > 0) {
      try {
        const projectL2 = await fetchCreateTaskProjectL2BoundByUrl({
          apiFetch,
          tenantId: typeof tenantId === 'function' ? tenantId() : tenantId,
          rows: oauthRepoRows.value,
          unboundUrls: unbound,
        })
        if (generation !== fetchGeneration) return
        for (const url of unbound) {
          boundMap[url] = projectL2.bound[url] === true
          if (projectL2.errorByUrl[url]) {
            errorMap[url] = projectL2.errorByUrl[url]
            traceMap[url] = projectL2.errorTraceIdByUrl[url] || ''
          }
          loadingMap[url] = false
        }
      } catch (error) {
        for (const url of unbound) {
          errorMap[url] = error?.message || 'OAuth 绑定状态检查失败'
          traceMap[url] = error?.traceId || extractTraceId(error) || ''
          boundMap[url] = false
          loadingMap[url] = false
        }
      }
    }

    applyMaps({ ...loadingMap }, { ...boundMap }, { ...errorMap }, { ...traceMap })
  }

  watch(
    () => [
      Boolean(enabled()),
      oauthRepoUrls.value.join('\n'),
      oauthRepoRows.value.map((row) => `${row.projectId}:${row.url}`).join('\n'),
      String(typeof tenantId === 'function' ? tenantId() : tenantId || ''),
    ].join('|'),
    () => {
      fetchConnections()
    },
    { immediate: true },
  )

  const readiness = computed(() => {
    const urls = oauthRepoUrls.value
    const loading = urls.some((url) => Boolean(loadingByUrl.value[url]))
    const unboundRepoUrls = urls.filter((url) => boundByUrl.value[url] !== true)
    const firstErrorUrl = urls.find((url) => String(errorByUrl.value[url] || '').trim())
    const checkError = firstErrorUrl ? String(errorByUrl.value[firstErrorUrl] || '').trim() : ''
    const checkErrorTraceId = firstErrorUrl
      ? String(errorTraceIdByUrl.value[firstErrorUrl] || '').trim()
      : ''
    return {
      hasOAuthRepos: urls.length > 0,
      loading,
      allBound: urls.length === 0 || unboundRepoUrls.length === 0,
      unboundRepoUrls,
      checkError,
      checkErrorTraceId,
    }
  })

  const blockedReason = computed(() => {
    if (!enabled()) return ''
    return resolveBlockedReason(readiness.value)
  })

  const blockedTraceId = computed(() => {
    if (!enabled()) return ''
    if (!readiness.value.checkError) return ''
    return String(readiness.value.checkErrorTraceId || '')
  })

  return {
    boundByUrl,
    loadingByUrl,
    errorByUrl,
    errorTraceIdByUrl,
    oauthRepoUrls,
    blockedReason,
    blockedTraceId,
  }
}
