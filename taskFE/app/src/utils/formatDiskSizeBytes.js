/**
 * Format repository disk usage for display (1024-based).
 * @param {number} bytes
 * @returns {string}
 */
export function formatDiskSizeBytes(bytes) {
  if (typeof bytes !== 'number' || !Number.isFinite(bytes) || bytes < 0) {
    throw new Error(`formatDiskSizeBytes: invalid bytes ${String(bytes)}`)
  }
  if (bytes < 1024) return `${bytes} B`
  const kb = bytes / 1024
  if (kb < 1024) return `${trimOneDecimal(kb)} KB`
  const mb = kb / 1024
  if (mb < 1024) return `${trimOneDecimal(mb)} MB`
  const gb = mb / 1024
  return `${trimOneDecimal(gb)} GB`
}

function trimOneDecimal(n) {
  const rounded = Math.round(n * 10) / 10
  return Number.isInteger(rounded) ? String(rounded) : rounded.toFixed(1)
}
