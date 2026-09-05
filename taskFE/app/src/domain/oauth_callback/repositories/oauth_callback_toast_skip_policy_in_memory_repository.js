import { OAuthCallbackToastSkipPolicyRepository } from './oauth_callback_toast_skip_policy_repository.js'

const DEFAULT_SKIP_SUBSTR = '/profile/git-site-oauth'

export class InMemoryOAuthCallbackToastSkipPolicyRepository extends OAuthCallbackToastSkipPolicyRepository {
  constructor(skipSubstrings = [DEFAULT_SKIP_SUBSTR]) {
    super()
    this._skipSubstrings = [...skipSubstrings]
  }

  shouldSkipToast(path) {
    const normalized = String(path || '')
    return this._skipSubstrings.some((s) => normalized.includes(s))
  }
}
