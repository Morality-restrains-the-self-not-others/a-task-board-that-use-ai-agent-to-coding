/**
 * Format GitLab used-GB for quota cards (same口径 as GitLab settings).
 * @param {unknown} n
 * @returns {string}
 */
export function formatUsedGb(n) {
  const v = Number(n)
  if (!Number.isFinite(v) || v <= 0) return '0'
  if (Number.isInteger(v)) return String(v)
  return String(Math.round(v * 1e6) / 1e6)
}

/**
 * Clamp used/quota to a 0–100 progress percentage.
 * @param {unknown} used
 * @param {unknown} quota
 * @returns {number}
 */
export function usagePercent(used, quota) {
  const q = Number(quota)
  if (!Number.isFinite(q) || q <= 0) return 0
  const u = Number(used)
  if (!Number.isFinite(u) || u <= 0) return 0
  return Math.min(100, (u / q) * 100)
}
