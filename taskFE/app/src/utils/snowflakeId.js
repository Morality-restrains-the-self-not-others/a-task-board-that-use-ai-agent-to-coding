/**
 * Snowflake ID 在前端必须以 string 持有（见 11_id_field_string_transit.md）。
 * 兼容旧版 API 仍以 JSON number 返回大整数时的精度丢失问题。
 */

/** 需在 JSON.parse 前保留为字符串的字段名（镜像市场 catalog / 安装） */
const SNOWFLAKE_JSON_KEYS = new Set([
  'id',
  'vendor_id',
  'container_id',
  'external_image_id',
  'image_group_id',
  'userdata_template_id',
])

/**
 * 将响应体中的大整数 ID 字段预先改为 JSON string，再 parse。
 * @param {string} text
 */
export function parseJsonPreservingSnowflakeIds(text) {
  if (typeof text !== 'string' || text === '') {
    return null
  }
  const patched = text.replace(
    /"((?:id|vendor_id|container_id|external_image_id|image_group_id|userdata_template_id))"\s*:\s*(\d{15,})/g,
    '"$1":"$2"'
  )
  return JSON.parse(patched)
}

/**
 * 将任意值规范为 Snowflake ID 字符串（用于请求体 / 比较）。
 * @param {unknown} value
 * @returns {string}
 */
export function coerceSnowflakeId(value) {
  if (value === null || value === undefined) return ''
  if (typeof value === 'string') return value.trim()
  if (typeof value === 'number' && Number.isFinite(value)) {
    // 已丢失精度的 number 无法恢复；仍转为 string 供调用方显式失败
    return String(value)
  }
  return String(value).trim()
}

/**
 * 规范化镜像市场条目中的 id / vendor.id（防御性，配合后端 string 出站）。
 * @param {Record<string, unknown>} item
 */
export function normalizeMarketplaceImageItem(item) {
  if (!item || typeof item !== 'object') return item
  if ('id' in item) {
    item.id = coerceSnowflakeId(item.id)
  }
  if (item.vendor && typeof item.vendor === 'object' && 'id' in item.vendor) {
    item.vendor.id = coerceSnowflakeId(item.vendor.id)
  }
  return item
}

/**
 * @param {unknown} data
 * @returns {Array<Record<string, unknown>>}
 */
export function normalizeMarketplaceImageList(data) {
  const list = Array.isArray(data) ? data : []
  return list.map((row) => normalizeMarketplaceImageItem({ ...row, vendor: row.vendor ? { ...row.vendor } : row.vendor }))
}

/**
 * @param {Response} response
 */
export async function readJsonPreservingSnowflakeIds(response) {
  const text = await response.text()
  if (!text) return null
  return parseJsonPreservingSnowflakeIds(text)
}

export { SNOWFLAKE_JSON_KEYS }
