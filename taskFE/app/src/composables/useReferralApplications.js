import { ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

const REASON_MIN = 8
const REASON_MAX = 500

/**
 * 系统管理：推荐码申请列表（审批 / 取消资格 / 审计）
 */
export function useReferralApplications() {
  const loadingApps = ref(false)
  const appError = ref('')
  const appErrorTraceId = ref('')
  const applications = ref([])
  const totalApps = ref(0)
  const statusFilter = ref('')
  const limit = ref(50)
  const offset = ref(0)
  const actingId = ref(null)
  const actingType = ref('')
  const actionTarget = ref(null)
  const actionType = ref('')
  const actionReason = ref('')
  const actionReasonError = ref('')
  const auditTarget = ref(null)
  const auditItems = ref([])
  const auditLoading = ref(false)
  const actionGuard = createClickGuard()

  // 推荐码反查：输入推荐码 → 找到持有该码的具体用户（管理员排查用）
  const codeQuery = ref('')
  const codeSearching = ref(false)
  const codeResult = ref(null) // {found:false} | {found:true, code, user_id, username, channel_name, is_default, status}
  const codeError = ref('')
  const codeErrorTraceId = ref('')
  // OPT-20260824-007：反查命中且该用户在当前列表时高亮对应行（滚动定位）
  const highlightUserId = ref('')
  let highlightTimer = null

  const lookupShareCode = async () => {
    const code = String(codeQuery.value || '').trim()
    if (!code) {
      codeError.value = '请输入推荐码'
      codeErrorTraceId.value = ''
      return
    }
    codeSearching.value = true
    codeResult.value = null
    codeError.value = ''
    codeErrorTraceId.value = ''
    try {
      const response = await apiFetch(`/api/system-admin/referral/share-code/lookup/?code=${encodeURIComponent(code)}`, {
        headers: { Accept: 'application/json' },
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        codeError.value = data.detail || data.error || '推荐码查询失败'
        codeErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
        return
      }
      codeResult.value = data
      // OPT-20260824-007：命中用户若在当前列表，高亮并滚动定位对应行
      if (data && data.found && data.user_id) {
        const exists = applications.value.some((a) => a.user_id === data.user_id)
        if (exists) {
          highlightUserId.value = data.user_id
          if (highlightTimer) clearTimeout(highlightTimer)
          highlightTimer = setTimeout(() => {
            highlightUserId.value = ''
          }, 3000)
        }
      }
    } catch (error) {
      codeError.value = error?.message || '推荐码查询失败'
      codeErrorTraceId.value = extractTraceId(error) || ''
    } finally {
      codeSearching.value = false
    }
  }

  const statusClass = (app) => {
    switch (app.status) {
      case 'pending': return 'bg-yellow-100 text-yellow-800'
      case 'approved': return app.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-600'
      case 'rejected': return 'bg-red-100 text-red-800'
      case 'expired': return 'bg-gray-100 text-gray-600'
      case 'revoked': return 'bg-orange-100 text-orange-800'
      default: return 'bg-gray-100 text-gray-600'
    }
  }

  const formatDate = (iso) => {
    if (!iso) return '—'
    const d = new Date(iso)
    return d.toLocaleDateString('zh-CN', {
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit',
    })
  }

  const validateReason = (raw) => {
    const reason = String(raw || '').trim()
    if ([...reason].length < REASON_MIN || [...reason].length > REASON_MAX) {
      return { ok: false, reason: '', error: `理由须 ${REASON_MIN}–${REASON_MAX} 字` }
    }
    return { ok: true, reason, error: '' }
  }

  const loadApplications = async () => {
    loadingApps.value = true
    appError.value = ''
    appErrorTraceId.value = ''
    try {
      const params = new URLSearchParams()
      if (statusFilter.value) params.set('status', statusFilter.value)
      params.set('limit', String(limit.value))
      params.set('offset', String(offset.value))
      const response = await apiFetch(`/api/system-admin/referral/applications/?${params.toString()}`, {
        headers: { Accept: 'application/json' },
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        appError.value = data.error || '加载申请列表失败'
        appErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
        return
      }
      applications.value = data.items || []
      totalApps.value = data.total || 0
    } catch (error) {
      appError.value = error?.message || '加载申请列表失败'
      appErrorTraceId.value = extractTraceId(error) || ''
    } finally {
      loadingApps.value = false
    }
  }

  const openAction = (app, type) => {
    actionTarget.value = app
    actionType.value = type
    actionReason.value = ''
    actionReasonError.value = ''
  }

  const closeAction = () => {
    actionTarget.value = null
    actionType.value = ''
    actionReason.value = ''
    actionReasonError.value = ''
  }

  const submitAction = async () => {
    if (!actionTarget.value || !actionType.value) return
    const checked = validateReason(actionReason.value)
    if (!checked.ok) {
      actionReasonError.value = checked.error
      return
    }
    const app = actionTarget.value
    const type = actionType.value
    await actionGuard.run(async ({ idempotencyKey, headers }) => {
      actingId.value = app.id
      actingType.value = type
      appError.value = ''
      appErrorTraceId.value = ''
      try {
        const response = await apiFetch(`/api/system-admin/referral/applications/${app.id}/${type}/`, {
          method: 'POST',
          headers: mergeIdempotencyHeaders({
            'Content-Type': 'application/json',
            Accept: 'application/json',
          }, idempotencyKey),
          body: JSON.stringify({ reason: checked.reason }),
        })
        const data = await response.json().catch(() => ({}))
        if (!response.ok) {
          appError.value = data.error || '操作失败'
          appErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || extractTraceId(headers) || ''
          return
        }
        closeAction()
        await loadApplications()
      } catch (error) {
        appError.value = error?.message || '操作失败'
        appErrorTraceId.value = extractTraceId(error) || ''
      } finally {
        actingId.value = null
        actingType.value = ''
      }
    })
  }

  const openAudit = async (app) => {
    auditTarget.value = app
    auditItems.value = []
    auditLoading.value = true
    appError.value = ''
    appErrorTraceId.value = ''
    try {
      const response = await apiFetch(`/api/system-admin/referral/applications/${app.id}/audit/`, {
        headers: { Accept: 'application/json' },
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok) {
        appError.value = data.error || '加载审计失败'
        appErrorTraceId.value = extractTraceId(response) || extractTraceId(data) || ''
        return
      }
      auditItems.value = data.items || []
    } catch (error) {
      appError.value = error?.message || '加载审计失败'
      appErrorTraceId.value = extractTraceId(error) || ''
    } finally {
      auditLoading.value = false
    }
  }

  const prevPage = () => {
    offset.value = Math.max(0, offset.value - limit.value)
    loadApplications()
  }

  const nextPage = () => {
    offset.value = offset.value + limit.value
    loadApplications()
  }

  const actionTitle = () => {
    switch (actionType.value) {
      case 'approve': return '通过申请'
      case 'reject': return '拒绝申请'
      case 'revoke': return '取消分账资格'
      default: return '确认操作'
    }
  }

  return {
    loadingApps,
    appError,
    appErrorTraceId,
    applications,
    totalApps,
    statusFilter,
    limit,
    offset,
    actingId,
    actingType,
    actionTarget,
    actionType,
    actionReason,
    actionReasonError,
    auditTarget,
    auditItems,
    auditLoading,
    statusClass,
    formatDate,
    loadApplications,
    openAction,
    closeAction,
    submitAction,
    openAudit,
    prevPage,
    nextPage,
    actionTitle,
    codeQuery,
    codeSearching,
    codeResult,
    codeError,
    codeErrorTraceId,
    highlightUserId,
    lookupShareCode,
  }
}
