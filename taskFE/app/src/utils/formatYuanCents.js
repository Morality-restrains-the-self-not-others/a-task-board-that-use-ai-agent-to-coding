/**
 * Billing amounts are stored as integer cents (yuan_cents / points).
 * Format for UI display in yuan.
 *
 * @param {unknown} points
 * @param {{ suffix?: string, empty?: string }} [opts]
 * @returns {string}
 */
export function formatYuanFromCents(points, opts = {}) {
  const empty = opts.empty ?? '—'
  const suffix = opts.suffix ?? ''
  if (points === null || points === undefined || points === '') return empty
  const n = Number(points)
  if (!Number.isFinite(n)) return empty
  const yuan = (n / 100).toFixed(2)
  return suffix ? `${yuan}${suffix}` : yuan
}
