import { useRoute } from 'vue-router'
import { setCookie } from '../../utils/cookieUtils.js'
import { sanitizeOidcResumeNext, resolveOidcResumeTarget } from '../../utils/oidcResumeUrl.js'
import { resolveAuthenticatedUserId } from '../../utils/sessionUserIdUtils.js'

export function useLoginOidcResume() {
  const route = useRoute()

  function readOidcResumeNextFromRoute() {
    const raw = route.query?.next
    const value = Array.isArray(raw) ? raw[0] : raw
    return sanitizeOidcResumeNext(typeof value === 'string' ? value : '')
  }

  async function resumeOidcFlowIfReady() {
    const resumeNext = readOidcResumeNextFromRoute()
    if (!resumeNext) return false
    const userId = await resolveAuthenticatedUserId()
    if (!userId) return false
    try {
      setCookie('userId', String(userId), 30)
      window.location.href = resolveOidcResumeTarget(resumeNext)
      return true
    } catch (error) {
      console.warn('[Login] OIDC resume skipped:', error)
      return false
    }
  }

  return {
    readOidcResumeNextFromRoute,
    resumeOidcFlowIfReady,
  }
}
