import { formatContainerImageArchitectures } from './containerImageArchitecture.js'
import { coerceSnowflakeId } from './snowflakeId.js'

/**
 * 在已安装镜像列表中按 Snowflake ID 查找条目（字符串比较，避免 number 精度丢失）。
 * @param {unknown[]} installedImages
 * @param {unknown} imageId
 */
export function resolveInstalledImageById(installedImages, imageId) {
  const id = coerceSnowflakeId(imageId)
  if (!id) return null
  const list = Array.isArray(installedImages) ? installedImages : []
  return list.find((img) => coerceSnowflakeId(img?.id) === id) || null
}

/**
 * 将已安装镜像格式化为人类可读的展示文案（名称、版本、架构）。
 * @param {{
 *   image?: Record<string, unknown> | null,
 *   imageId?: unknown,
 *   storedName?: unknown,
 *   includeArchitecture?: boolean,
 *   treatMissingBindingAsUnavailable?: boolean,
 * }} [options]
 */
export function formatInstalledImageDisplayLabel({
  image = null,
  imageId = '',
  storedName = '',
  includeArchitecture = true,
  treatMissingBindingAsUnavailable = false,
} = {}) {
  const matched = image && typeof image === 'object' ? image : null
  const id = coerceSnowflakeId(imageId || matched?.id)
  if (treatMissingBindingAsUnavailable && id && !matched) {
    return `镜像不可用或已卸载（ID：${id}）`
  }
  const name = String(matched?.name || storedName || '').trim()
  const version = matched?.version ? String(matched.version).trim() : ''
  const arch = includeArchitecture ? formatContainerImageArchitectures(matched) : ''
  const archSuffix = arch ? ` · 架构 ${arch}` : ''

  if (name) {
    const versionSuffix = version ? ` (${version})` : ''
    return `${name}${versionSuffix}${archSuffix}`
  }

  const imageUrl = String(matched?.image_url || '').trim()
  if (imageUrl) {
    return `${imageUrl}${archSuffix}`
  }

  if (!id) return '未设置'
  return `镜像不可用或已卸载（ID：${id}）`
}

/**
 * 规范化已安装镜像 API 列表项（id 强制为 string）。
 * @param {unknown} data
 */
export function normalizeInstalledImageList(data) {
  const list = Array.isArray(data) ? data : (data?.results || [])
  if (!Array.isArray(list)) return []
  return list.map((row) => ({
    ...row,
    id: coerceSnowflakeId(row?.id),
  }))
}
