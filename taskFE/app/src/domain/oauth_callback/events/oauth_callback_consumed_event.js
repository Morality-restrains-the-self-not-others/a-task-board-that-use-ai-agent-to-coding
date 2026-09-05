export class OAuthCallbackConsumed {
  constructor({ provider, code, severity, occurredAt = new Date() }) {
    this.provider = String(provider || '')
    this.code = String(code || '')
    this.severity = String(severity || '')
    this.occurredAt = occurredAt instanceof Date ? occurredAt : new Date(occurredAt)
  }
}
