/**
 * 按实例编号（instance_type）子串过滤可用实例列表（大小写不敏感）。
 * @param {Array<{ instance_type?: string }>} instances
 * @param {string} query
 * @returns {Array}
 */
export function filterInstancesByTypeQuery(instances, query) {
  const list = Array.isArray(instances) ? instances : []
  const q = String(query || '').trim().toLowerCase()
  if (!q) return list
  return list.filter((item) => String(item?.instance_type || '').toLowerCase().includes(q))
}

/**
 * 在过滤后的列表上计算分页切片。
 * @param {Array} filtered
 * @param {number} currentPage 从 1 开始
 * @param {number} pageSize
 * @returns {{ pageItems: Array, totalPages: number, safePage: number }}
 */
export function paginateFilteredInstances(filtered, currentPage, pageSize) {
  const list = Array.isArray(filtered) ? filtered : []
  const size = Math.max(1, Number(pageSize) || 10)
  const totalPages = Math.max(1, Math.ceil(list.length / size) || 1)
  let safePage = Number(currentPage) || 1
  if (safePage < 1) safePage = 1
  if (safePage > totalPages) safePage = totalPages
  const startIndex = (safePage - 1) * size
  return {
    pageItems: list.slice(startIndex, startIndex + size),
    totalPages,
    safePage,
  }
}
