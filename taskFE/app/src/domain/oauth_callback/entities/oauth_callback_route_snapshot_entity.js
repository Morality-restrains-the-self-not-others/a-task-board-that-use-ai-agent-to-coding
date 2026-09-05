import { OAuthGitProvider } from '../value_objects/oauth_git_provider_value_object.js'
import { OAuthCallbackCode } from '../value_objects/oauth_callback_code_value_object.js'

export class OAuthCallbackRouteSnapshot {
  constructor({ path = '', query = {} } = {}) {
    this.path = String(path || '')
    this.query = { ...query }
  }

  static fromRoute(route) {
    return new OAuthCallbackRouteSnapshot({
      path: route?.path,
      query: route?.query ?? {},
    })
  }

  /** 单次导航只消费第一个非空 provider query（gitlab 优先于 github） */
  detectCallbackParam() {
    if (this.query.gitlab != null && this.query.gitlab !== '') {
      return {
        provider: new OAuthGitProvider('gitlab'),
        queryKey: 'gitlab',
        code: new OAuthCallbackCode(this.query.gitlab),
      }
    }
    if (this.query.github != null && this.query.github !== '') {
      return {
        provider: new OAuthGitProvider('github'),
        queryKey: 'github',
        code: new OAuthCallbackCode(this.query.github),
      }
    }
    return null
  }

  queryWithoutKey(queryKey) {
    const next = { ...this.query }
    delete next[queryKey]
    delete next.trace_id
    delete next.traceId
    return next
  }

  /** OAuth 失败回跳上的 trace_id（供 data-traceId）；成功或缺失时为空 */
  callbackTraceId() {
    const raw = this.query.trace_id ?? this.query.traceId
    if (raw == null || raw === '') return ''
    return String(raw).trim()
  }
}
