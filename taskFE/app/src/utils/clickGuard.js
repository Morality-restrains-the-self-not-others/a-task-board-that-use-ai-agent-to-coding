/** Shared click anti-replay: in-flight lock + debounce + Idempotency-Key. */

import { ref } from 'vue'

export const IDEMPOTENCY_HEADER = 'Idempotency-Key'

export function newIdempotencyKey() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `ik-${Date.now()}-${Math.random().toString(16).slice(2, 10)}`
}

export function mergeIdempotencyHeaders(headers, key) {
  const next = { ...(headers || {}) }
  if (key) {
    next[IDEMPOTENCY_HEADER] = key
  }
  return next
}

export function createClickGuard(options = {}) {
  const debounceMs = options.debounceMs ?? 300
  const now = options.now || (() => Date.now())
  const newKey = options.newKey || newIdempotencyKey
  const busy = ref(false)
  let lastAcceptedAt = 0
  let currentKey = ''

  return {
    busy,
    isBusy: () => busy.value,
    currentKey: () => currentKey,
    async run(fn) {
      const t = now()
      if (busy.value) {
        return { skipped: true, reason: 'in-flight' }
      }
      if (lastAcceptedAt && t - lastAcceptedAt < debounceMs) {
        return { skipped: true, reason: 'debounce' }
      }
      busy.value = true
      lastAcceptedAt = t
      currentKey = newKey()
      try {
        const result = await fn({
          idempotencyKey: currentKey,
          headers: { [IDEMPOTENCY_HEADER]: currentKey },
        })
        return { skipped: false, result }
      } finally {
        busy.value = false
      }
    },
  }
}
