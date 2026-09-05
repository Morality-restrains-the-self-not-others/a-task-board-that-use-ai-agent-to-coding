/**
 * 镜像组图标 URL：目录/已安装镜像共用。
 * 已安装记录可能未快照 icon_url（存量行），此时用同页目录按 external_image_id / name 回退。
 */

export function imageGroupIconSrc(image) {
  if (!image || typeof image !== 'object') {
    return ''
  }
  const url = image.icon_url || image.image_group?.icon_url || ''
  return typeof url === 'string' ? url.trim() : ''
}

export function resolveInstalledImageIconSrc(image, catalogImages = [], devImages = []) {
  const own = imageGroupIconSrc(image)
  if (own) {
    return own
  }
  const lists = [
    ...(Array.isArray(catalogImages) ? catalogImages : []),
    ...(Array.isArray(devImages) ? devImages : []),
  ]
  const ext = String(image?.external_image_id || '').trim()
  if (ext) {
    const byId = lists.find((c) => String(c?.id || '') === ext)
    const src = imageGroupIconSrc(byId)
    if (src) {
      return src
    }
  }
  const name = String(image?.name || '').trim()
  if (!name) {
    return ''
  }
  const byName = lists.find((c) => String(c?.name || '') === name)
  return imageGroupIconSrc(byName)
}
