export const PROJECT_TAG_MAX_LENGTH = 64
export const PROJECT_TAGS_MAX_COUNT = 20

/** 规范化项目标签：trim、去重、过滤空串与超长项 */
export function normalizeProjectTags(tags, { maxCount = PROJECT_TAGS_MAX_COUNT } = {}) {
  if (!Array.isArray(tags)) return []
  const out = []
  const seen = new Set()
  for (const raw of tags) {
    const tag = String(raw ?? '').trim()
    if (!tag || tag.length > PROJECT_TAG_MAX_LENGTH) continue
    const key = tag.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    out.push(tag)
    if (out.length >= maxCount) break
  }
  return out
}

/** 从输入文本解析单个待添加标签（Enter/逗号提交前） */
export function parseProjectTagInput(text) {
  const tag = String(text ?? '').trim()
  if (!tag || tag.length > PROJECT_TAG_MAX_LENGTH) return null
  return tag
}



