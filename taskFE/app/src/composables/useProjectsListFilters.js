import { computed, ref } from 'vue'
import { normalizeProjectTags } from '../utils/projectTagsUtils.js'
import {
  collectAvailableProjectTags,
  filterProjectsBySearchAndTags,
  hasActiveProjectFilters,
  parseTagsQueryParam,
  serializeTagsQueryParam,
} from '../utils/projectListFilterUtils.js'

/**
 * Projects 列表搜索/标签过滤。
 * @param {{
 *   route: import('vue-router').RouteLocationNormalizedLoaded,
 *   router: import('vue-router').Router,
 *   projects: import('vue').Ref<unknown[]>,
 * }} deps
 */
export function useProjectsListFilters({ route, router, projects }) {
  const searchQuery = ref('')
  const selectedTagFilters = ref(parseTagsQueryParam(route.query.tags))

  const filteredProjects = computed(() =>
    filterProjectsBySearchAndTags(projects.value, {
      searchQuery: searchQuery.value,
      selectedTags: selectedTagFilters.value,
    }),
  )

  const availableTags = computed(() => collectAvailableProjectTags(projects.value))

  const hasActiveFilters = computed(() =>
    hasActiveProjectFilters({
      searchQuery: searchQuery.value,
      selectedTags: selectedTagFilters.value,
    }),
  )

  const emptyFilterMessage = computed(() => {
    const q = searchQuery.value.trim()
    const tags = normalizeProjectTags(selectedTagFilters.value)
    if (q && tags.length) {
      return `未找到名称/描述匹配「${q}」且包含标签「${tags.join('、')}」中任一的项目`
    }
    if (q) return `未找到匹配「${q}」的项目`
    if (tags.length) return `未找到包含标签「${tags.join('、')}」的项目`
    return '未找到匹配的项目'
  })

  const isTagSelected = (tag) => {
    const key = String(tag || '').toLowerCase()
    return selectedTagFilters.value.some((t) => String(t).toLowerCase() === key)
  }

  const syncTagsToQuery = () => {
    const query = { ...route.query }
    const serialized = serializeTagsQueryParam(selectedTagFilters.value)
    if (serialized) query.tags = serialized
    else delete query.tags
    router.replace({ path: route.path, query })
  }

  const toggleTagFilter = (tag) => {
    const key = String(tag || '').toLowerCase()
    const current = [...selectedTagFilters.value]
    const idx = current.findIndex((t) => String(t).toLowerCase() === key)
    if (idx >= 0) current.splice(idx, 1)
    else current.push(tag)
    selectedTagFilters.value = normalizeProjectTags(current)
    syncTagsToQuery()
  }

  const clearAllFilters = () => {
    searchQuery.value = ''
    selectedTagFilters.value = []
    syncTagsToQuery()
  }

  const applyTagsQueryFromRoute = (raw) => {
    const parsed = parseTagsQueryParam(raw)
    const current = serializeTagsQueryParam(selectedTagFilters.value)
    const incoming = serializeTagsQueryParam(parsed)
    if (current !== incoming) {
      selectedTagFilters.value = parsed
    }
  }

  return {
    searchQuery,
    selectedTagFilters,
    filteredProjects,
    availableTags,
    hasActiveFilters,
    emptyFilterMessage,
    isTagSelected,
    toggleTagFilter,
    clearAllFilters,
    applyTagsQueryFromRoute,
  }
}
