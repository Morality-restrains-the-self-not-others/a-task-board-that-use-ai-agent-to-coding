import { ref } from 'vue'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'
import { persistLoginSuccessCredentials } from '../domain/auth/services/activate_session_service.js'
import { getActiveToken } from '../domain/auth/services/saved_accounts_store.js'
import { getStoredUserId, storeUserId } from '../utils/sessionUserIdUtils.js'
import { apiFetch as defaultApiFetch } from '../utils/apiUtils.js'
import { FORBIDDEN_IMPERSONATION_REASONS } from '../components/ImpersonateReasonModal.vue'

export const IMPERSONATOR_BACKUP_KEY = 'impersonatorAccountBackup'

// Keep in sync with taskAuth/domain NestedImpersonationClientMessage.
export const NESTED_IMPERSONATION_DETAIL = '已在模拟登录中，请先退出'

function defaultAssignHref(href) {
  if (typeof window !== 'undefined' && href) {
    window.location.href = href
  }
}

function storage() {
  try {
    return typeof sessionStorage !== 'undefined' ? sessionStorage : null
  } catch {
    return null
  }
}

export function backupActorSlot({ userId, token, store } = {}) {
  const s = store || storage()
  if (!s) return
  s.setItem(IMPERSONATOR_BACKUP_KEY, JSON.stringify({
    userId: String(userId || ''),
    token: String(token || ''),
  }))
}

export function useImpersonateUser(deps = {}) {
  const doFetch = deps.apiFetch || defaultApiFetch
  const persist = deps.persist || persistLoginSuccessCredentials
  const assignHref = deps.assignHref || defaultAssignHref
  const store = deps.sessionStore || null
  const impersonating = ref(false)
  const impersonateError = ref('')
  const impersonateErrorTraceId = ref('')
  const guard = createClickGuard()

  function isAlreadyImpersonatingDetail(detail) {
    const text = String(detail || '')
    return text.toLowerCase().includes('already impersonating')
      || text.includes(NESTED_IMPERSONATION_DETAIL)
      || text.includes('已在模拟登录中')
  }

  async function redirectIfAlreadyImpersonatingSameTarget(uid) {
    const response = await doFetch('/api/auth/impersonation/status/', {
      method: 'GET',
      headers: { Accept: 'application/json' },
      credentials: 'include',
    })
    if (!response || !response.ok) return false
    const data = await response.json().catch(() => ({}))
    if (!data.impersonating) return false
    if (String(data.target_user_id || '') !== uid) return false
    // 与 useImpersonationStatus.refresh 的自愈同构：409 复用会话场景下
    // localStorage currentUserId 可能仍是模拟者 ID，先切到目标用户再跳转，
    // 避免落地页以旧 ID 构造的 API 路径被后端 403。
    storeUserId(String(data.target_user_id))
    assignHref(data.redirect_url || '/')
    return true
  }

  async function impersonateUser(targetUserId, reason) {
    const uid = String(targetUserId || '').trim()
    const why = String(reason || '').trim()
    impersonateError.value = ''
    impersonateErrorTraceId.value = ''
    if (!uid) {
      impersonateError.value = '缺少用户 ID'
      return { skipped: false, ok: false }
    }
    if ([...why].length < 8) {
      impersonateError.value = '请填写至少 8 个字的理由'
      return { skipped: false, ok: false }
    }
    if (FORBIDDEN_IMPERSONATION_REASONS.has(why)) {
      impersonateError.value = '请填写具体的模拟登录理由，不要使用弹窗说明文案'
      return { skipped: false, ok: false }
    }
    const run = await guard.run(async ({ headers, idempotencyKey }) => {
      impersonating.value = true
      try {
        let actorToken = ''
        let actorUserId = ''
        try { actorToken = await getActiveToken() } catch { /* ignore */ }
        try { actorUserId = getStoredUserId() || '' } catch { /* ignore */ }
        backupActorSlot({ userId: actorUserId, token: actorToken, store: store || storage() })

        const response = await doFetch(`/api/system-admin/users/${uid}/impersonate/`, {
          method: 'POST',
          headers: mergeIdempotencyHeaders({
            'Content-Type': 'application/json',
            Accept: 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
            ...headers,
          }, idempotencyKey),
          credentials: 'include',
          body: JSON.stringify({ reason: why }),
        })
        const traceId = response.headers?.get?.('X-Trace-Id') || response.headers?.get?.('x-trace-id') || ''
        const data = await response.json().catch(() => ({}))
        if (!response.ok) {
          const detail = data.detail || data.error || `模拟登录失败 (${response.status})`
          impersonateErrorTraceId.value = data.trace_id || traceId || ''
          if (response.status === 409 && isAlreadyImpersonatingDetail(detail)) {
            const resumed = await redirectIfAlreadyImpersonatingSameTarget(uid)
            if (resumed) {
              return { ok: true }
            }
          }
          impersonateError.value = detail
          return { ok: false }
        }
        await persist(data)
        assignHref(data.redirect_url || '/')
        return { ok: true, data }
      } finally {
        impersonating.value = false
      }
    })
    if (run.skipped) return { skipped: true, ok: false }
    return { skipped: false, ...(run.result || { ok: false }) }
  }

  return {
    impersonating,
    impersonateError,
    impersonateErrorTraceId,
    impersonateUser,
  }
}
