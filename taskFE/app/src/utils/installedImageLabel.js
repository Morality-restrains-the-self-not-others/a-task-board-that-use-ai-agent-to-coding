/**
 * 已安装容器镜像展示标签（创建任务下拉 / 评论 $镜像 / 关联项目默认镜像共用）。
 */

/**
 * 镜像更新时间展示串：优先目录快照 updated_at（镜像更新时间），缺失回退 installed_at
 * （安装时间，038 迁移已回填存量行）。解析失败或两者皆空返回 ''（调用方决定是否隐藏）。
 */
export function formatInstalledImageUpdateTime(img) {
  const raw = img?.updated_at ?? img?.installed_at
  if (raw == null || raw === '') return ''
  const d = new Date(raw)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function formatInstalledImageRunLabel(name, version) {
  const n = String(name || '').trim()
  const v = version == null ? '' : String(version).trim()
  if (!v) return n
  if (!n) return v
  if (n === v || n.endsWith(`:${v}`)) return n
  return `${n}:${v}`
}

function imageRecordLabel(img) {
  if (!img || typeof img !== 'object') return ''
  return formatInstalledImageRunLabel(img.name || img.image_name, img.version ?? img.tag)
}

function findImageById(images, imageId) {
  const id = String(imageId || '').trim()
  if (!id || !Array.isArray(images)) return null
  return images.find((item) => String(item?.id) === id) || null
}

/**
 * 解析任务默认镜像展示名：目录命中 → 嵌套对象 → 评论 mention 标签。
 * 禁止用目录里「同名不同 id」的镜像冒充已绑定镜像。
 */
export function resolveTaskDefaultImageLabel({
  imageId = '',
  installedImages = [],
  nestedImage = null,
  comments = [],
} = {}) {
  const id = String(imageId || nestedImage?.id || '').trim()
  const fromCatalog = findImageById(installedImages, id)
  if (fromCatalog) return imageRecordLabel(fromCatalog)

  if (nestedImage && String(nestedImage.id || '').trim() === id) {
    const nestedLabel = imageRecordLabel(nestedImage)
    if (nestedLabel) return nestedLabel
  }

  if (!id) return ''

  const list = Array.isArray(comments) ? comments : []
  for (const comment of list) {
    if (String(comment?.container_image_id || '').trim() === id) {
      const label = String(comment.container_image_label || comment.containerImageLabel || '').trim()
      if (label) return formatInstalledImageRunLabel(label, comment.container_image_version)
    }
    const mentions = Array.isArray(comment?.mentions) ? comment.mentions : []
    const mention = mentions.find(
      (m) => m?.type === 'installed_image' && String(m?.id || '').trim() === id,
    )
    if (mention) {
      const label = formatInstalledImageRunLabel(mention.name || mention.image_name, mention.version ?? mention.tag)
      if (label) return label
    }
  }
  return ''
}

export function formatTaskDefaultImageDisplay(label, imageId, removed = false) {
  const text = String(label || '').trim()
  if (text) return text
  // OPT-20260820-045: 后端标记 container_image_removed（镜像已不在租户已安装目录）时
  // 区分「已卸载」而非笼统的「已绑定」，避免用户以为默认镜像仍可用。
  if (removed) return '已卸载'
  if (String(imageId || '').trim()) return '已绑定'
  return '未设置'
}
