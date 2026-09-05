import { ref, onMounted } from 'vue'
import { createClickGuard } from '../utils/clickGuard.js'
import { persistLoginSuccessCredentials } from '../domain/auth/services/activate_session_service.js'
import { storeUserId } from '../utils/sessionUserIdUtils.js'
import { apiFetch as defaultApiFetch } from '../utils/apiUtils.js'

function defaultAssignHref(href) {
  if (typeof window !== 'undefined' && href) {
    window.location.href = href
  }
}

export function useImpersonationStatus(deps = {}) {
  const doFetch = deps.apiFetch || defaultApiFetch
  const persist = deps.persist || persistLoginSuccessCredentials
  const assignHref = deps.assignHref || defaultAssignHref
  const impersonating = ref(false)
  const targetUsername = ref('')
  const stopping = ref(false)
  const error = ref('')
  const errorTraceId = ref('')
  const guard = createClickGuard()

  async function refresh() {
    try {
      const response = await doFetch('/api/auth/impersonation/status/', {
        method: 'GET',
        headers: { Accept: 'application/json' },
        credentials: 'include',
      })
      if (!response || !response.ok) {
        impersonating.value = false
        return
      }
      const data = await response.json().catch(() => ({}))
      impersonating.value = Boolean(data.impersonating)
      targetUsername.value = String(data.target_username || '')
      // 自愈（OPT-20260824-021 根因修复的存量会话兜底）：确认处于模拟会话时，
      // 以服务端权威 target_user_id 覆盖本地 currentUserId —— 模拟登录早于修复
      // 上线的会话，localStorage 仍残留模拟者 ID，页面以其构造 /api/git-identities/
      // user/{id}/ 等路径会被后端 403（「获取身份列表失败」）。退出模拟时
      // stopImpersonation 的 persist 会把 currentUserId 恢复为模拟者。
      if (data.impersonating && data.target_user_id) {
        storeUserId(String(data.target_user_id))
      }
    } catch {
      impersonating.value = false
    }
  }

  async function stopImpersonation() {
    error.value = ''
    errorTraceId.value = ''
    const run = await guard.run(async () => {
      stopping.value = true
      try {
        const response = await doFetch('/api/auth/impersonation/stop/', {
          method: 'POST',
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
          },
          credentials: 'include',
        })
        const traceId = response.headers?.get?.('X-Trace-Id') || ''
        const data = await response.json().catch(() => ({}))
        if (!response.ok) {
          error.value = data.detail || data.error || `退出模拟失败 (${response.status})`
          errorTraceId.value = data.trace_id || traceId || ''
          return { ok: false }
        }
        await persist(data)
        assignHref(data.redirect_url || '/system-admin/users/')
        return { ok: true }
      } finally {
        stopping.value = false
      }
    })
    return run.result || { ok: false, skipped: run.skipped }
  }

  function bindMounted() {
    onMounted(() => { refresh() })
  }

  return {
    impersonating,
    targetUsername,
    stopping,
    error,
    errorTraceId,
    refresh,
    stopImpersonation,
    bindMounted,
  }
}
