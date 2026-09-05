/**
 * Tenant GitLab OmniAuth gitlab.rb snippet helpers.
 * Secret is never stored; it is only merged into the on-screen snippet once.
 */

/**
 * Insert one-time client_secret after the identifier line.
 * When the server-rendered snippet carries the placeholder
 * (secret: '<PASTE_CLIENT_SECRET>'), replace it with the real secret.
 * Empty secret or missing identifier leaves snippet unchanged.
 * @param {string} snippet
 * @param {string} secret
 * @returns {string}
 */
export function insertOmniAuthSecret(snippet, secret) {
  const text = String(snippet || '')
  const sec = String(secret || '').trim()
  if (!text || !sec) {
    return text
  }
  // OPT-20260826-013: server snippet now ships a secret placeholder line —
  // replace it in place so the on-screen fragment is paste-ready as-is.
  const placeholder = /^([ \t]*secret:\s*)'<PASTE_CLIENT_SECRET>',?$/m
  if (placeholder.test(text)) {
    const quoted = JSON.stringify(sec)
    return text.replace(placeholder, `$1${quoted},`)
  }
  // Legacy fallback: snippet without placeholder — insert after identifier.
  if (/(^|\n)[ \t]*secret:\s*['"]/.test(text)) {
    return text
  }
  const identMatch = text.match(/^([ \t]*)identifier:\s*.+$/m)
  if (!identMatch) {
    return text
  }
  const indent = identMatch[1]
  const quoted = JSON.stringify(sec)
  return text.replace(identMatch[0], `${identMatch[0]}\n${indent}secret: ${quoted},`)
}
