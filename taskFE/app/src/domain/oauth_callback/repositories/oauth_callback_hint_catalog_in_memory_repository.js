import { OAuthCallbackHintCatalog } from '../entities/oauth_callback_hint_catalog_entity.js'
import { OAuthCallbackHintCatalogRepository } from './oauth_callback_hint_catalog_repository.js'

export class InMemoryOAuthCallbackHintCatalogRepository extends OAuthCallbackHintCatalogRepository {
  constructor(catalog = new OAuthCallbackHintCatalog()) {
    super()
    this._catalog = catalog
  }

  get catalog() {
    return this._catalog
  }

  resolveRawMessage(providerValue, codeValue) {
    return this._catalog.resolveRawMessage(providerValue, codeValue)
  }
}
