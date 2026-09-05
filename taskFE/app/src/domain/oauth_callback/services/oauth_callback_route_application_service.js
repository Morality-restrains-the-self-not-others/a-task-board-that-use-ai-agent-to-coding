import { OAuthCallbackConsumed } from '../events/oauth_callback_consumed_event.js'
import { OAuthCallbackRouteSnapshot } from '../entities/oauth_callback_route_snapshot_entity.js'

export class OAuthCallbackRouteApplicationService {
  constructor(messageResolutionService) {
    this._messageResolutionService = messageResolutionService
  }

  /**
   * 解析路由上的 OAuth 回调 query，返回领域结果（无副作用）。
   * 路由替换与通知由应用层根据返回值执行。
   */
  consumeFromRoute(route) {
    const snapshot = OAuthCallbackRouteSnapshot.fromRoute(route)
    const detected = snapshot.detectCallbackParam()
    if (!detected) {
      return null
    }

    const outcome = this._messageResolutionService.resolve(
      detected.provider,
      detected.code,
    )

    const event = new OAuthCallbackConsumed({
      provider: outcome.providerValue,
      code: outcome.codeValue,
      severity: outcome.severityValue,
    })

    return {
      snapshot,
      detected,
      outcome,
      event,
      traceId: snapshot.callbackTraceId(),
      clearedQuery: snapshot.queryWithoutKey(detected.queryKey),
    }
  }
}
