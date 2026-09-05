/** Fork 一次派生的副本数量：默认 1，最多 99。 */

export const FORK_COPY_COUNT_MIN = 1
export const FORK_COPY_COUNT_MAX = 99

export function clampForkCopyCount(value) {
  const n = Number.parseInt(String(value ?? ''), 10)
  if (!Number.isFinite(n)) return FORK_COPY_COUNT_MIN
  if (n < FORK_COPY_COUNT_MIN) return FORK_COPY_COUNT_MIN
  if (n > FORK_COPY_COUNT_MAX) return FORK_COPY_COUNT_MAX
  return n
}

export function forkCopyIdempotencyKey(batchKey, index) {
  return `${batchKey}:${index}`
}
