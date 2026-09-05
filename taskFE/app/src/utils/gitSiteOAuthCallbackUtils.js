import {
  GITHUB_CALLBACK_HINTS,
  GITLAB_CALLBACK_HINTS,
} from '../domain/oauth_callback/entities/oauth_callback_hint_catalog_entity.js'
import { InMemoryOAuthCallbackHintCatalogRepository } from '../domain/oauth_callback/repositories/oauth_callback_hint_catalog_in_memory_repository.js'
import { InMemoryOAuthCallbackToastSkipPolicyRepository } from '../domain/oauth_callback/repositories/oauth_callback_toast_skip_policy_in_memory_repository.js'
import { OAuthCallbackMessageResolutionService } from '../domain/oauth_callback/services/oauth_callback_message_resolution_service.js'
import { OAuthCallbackRouteApplicationService } from '../domain/oauth_callback/services/oauth_callback_route_application_service.js'

export { GITHUB_CALLBACK_HINTS, GITLAB_CALLBACK_HINTS }

const defaultHintRepository = new InMemoryOAuthCallbackHintCatalogRepository()
const defaultSkipPolicyRepository = new InMemoryOAuthCallbackToastSkipPolicyRepository()
const defaultResolutionService = new OAuthCallbackMessageResolutionService(defaultHintRepository)
const defaultRouteApplicationService = new OAuthCallbackRouteApplicationService(
  defaultResolutionService,
)

export const resolveOAuthCallbackMessage = (provider, code) => {
  const outcome = defaultResolutionService.resolve(provider, code)
  return { severity: outcome.severityValue, message: outcome.messageValue }
}

export const shouldSkipOAuthCallbackToast = (path) =>
  defaultSkipPolicyRepository.shouldSkipToast(path)

export const applyOAuthCallbackFromRoute = async (route, router, options = {}) => {
  const consumed = defaultRouteApplicationService.consumeFromRoute(route)
  if (!consumed) return null

  const { outcome, detected } = consumed
  const { severity, message } = {
    severity: outcome.severityValue,
    message: outcome.messageValue,
  }
  const traceId = severity === 'error' ? String(consumed.traceId || '').trim() : ''

  if (severity === 'error' && typeof options.onError === 'function') {
    options.onError(message, { traceId })
  }
  if (severity === 'success' && typeof options.onSuccess === 'function') {
    options.onSuccess(message)
  }

  await router.replace({
    path: route.path,
    query: consumed.clearedQuery,
  })

  return {
    provider: outcome.providerValue,
    code: outcome.codeValue,
    message,
    severity,
    queryKey: detected.queryKey,
    traceId,
  }
}

export const setupOAuthCallbackToastGuard = (router, toastService) => {
  router.afterEach((to) => {
    if (shouldSkipOAuthCallbackToast(to.path)) return
    void applyOAuthCallbackFromRoute(to, router, {
      onError: (msg, meta = {}) => toastService.error(msg, 5000, { traceId: meta.traceId }),
    })
  })
}
