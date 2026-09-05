/**
 * Build page button items with ellipsis for large page counts.
 * @param {number} currentPage
 * @param {number} totalPages
 * @param {{ siblingCount?: number }} [options]
 * @returns {(number|'ellipsis')[]}
 */
export function buildPaginationItems(currentPage, totalPages, { siblingCount = 1 } = {}) {
  const total = Math.max(0, Math.floor(Number(totalPages) || 0))
  if (total <= 0) return []
  const current = Math.min(Math.max(1, Math.floor(Number(currentPage) || 1)), total)
  const siblings = Math.max(0, Math.floor(Number(siblingCount) || 0))

  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1)
  }

  const pages = new Set([1, total])
  for (let i = current - siblings; i <= current + siblings; i += 1) {
    if (i >= 1 && i <= total) pages.add(i)
  }
  // Near start/end: fill a compact window so we do not show "1 … 3".
  if (current <= 3 + siblings) {
    for (let i = 2; i <= Math.min(4 + siblings, total - 1); i += 1) pages.add(i)
  }
  if (current >= total - (2 + siblings)) {
    for (let i = Math.max(2, total - (3 + siblings)); i <= total - 1; i += 1) pages.add(i)
  }

  const sorted = [...pages].sort((a, b) => a - b)
  const items = []
  for (let i = 0; i < sorted.length; i += 1) {
    if (i > 0 && sorted[i] - sorted[i - 1] > 1) {
      items.push('ellipsis')
    }
    items.push(sorted[i])
  }
  return items
}
