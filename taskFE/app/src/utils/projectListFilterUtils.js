import { normalizeProjectTags } from './projectTagsUtils.js'

/** 从项目列表收集去重标签（按字母排序，保序可选 count） */
export function collectAvailableProjectTags(projects) {
  const seen = new Map()
  for (const project of projects || []) {
    for (const tag of normalizeProjectTags(project?.tags)) {
      const key = tag.toLowerCase()
      if (!seen.has(key)) seen.set(key, tag)
    }
  }
  return [...seen.values()].sort((a, b) => a.localeCompare(b, 'zh-CN'))
}

/**
 * 筛选项目：名称/描述关键词 + 标签（选中标签之间为 OR：含任一即可；未选标签时不限）
 */
export function filterProjectsBySearchAndTags(projects, { searchQuery = '', selectedTags = [] } = {}) {
  const list = Array.isArray(projects) ? projects : []
  const q = String(searchQuery || '').trim().toLowerCase()
  const tags = normalizeProjectTags(selectedTags)

  return list.filter((project) => {
    if (q) {
      const name = String(project?.name || '').toLowerCase()
      const desc = String(project?.description || '').toLowerCase()
      if (!name.includes(q) && !desc.includes(q)) return false
    }
    if (tags.length) {
      const projectTags = normalizeProjectTags(project?.tags)
      const projectTagKeys = new Set(projectTags.map((t) => t.toLowerCase()))
      const matchesAny = tags.some((t) => projectTagKeys.has(t.toLowerCase()))
      if (!matchesAny) return false
    }
    return true
  })
}

export function parseTagsQueryParam(raw) {
  if (raw == null || raw === '') return []
  return normalizeProjectTags(String(raw).split(','))
}

export function serializeTagsQueryParam(selectedTags) {
  const tags = normalizeProjectTags(selectedTags)
  return tags.length ? tags.join(',') : ''
}

export function hasActiveProjectFilters({ searchQuery = '', selectedTags = [] } = {}) {
  return Boolean(String(searchQuery || '').trim()) || normalizeProjectTags(selectedTags).length > 0
}
