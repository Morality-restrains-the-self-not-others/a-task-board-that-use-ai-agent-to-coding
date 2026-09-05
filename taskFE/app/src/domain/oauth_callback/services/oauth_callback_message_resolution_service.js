import { OAuthCallbackOutcome } from '../value_objects/oauth_callback_outcome_value_object.js'

export class OAuthCallbackMessageResolutionService {
  constructor(hintCatalogRepository) {
    this._hintCatalogRepository = hintCatalogRepository
  }

  resolve(provider, code) {
    const providerValue = typeof provider === 'string' ? provider : provider?.value
    const codeValue = typeof code === 'string' ? code : code?.value
    const raw = this._hintCatalogRepository.resolveRawMessage(providerValue, codeValue)
    return new OAuthCallbackOutcome({
      provider: providerValue,
      code: codeValue,
      severity: raw.severity,
      message: raw.message,
    })
  }
}
