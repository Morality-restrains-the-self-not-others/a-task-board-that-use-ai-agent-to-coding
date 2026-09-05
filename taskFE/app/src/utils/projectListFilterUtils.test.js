import { describe, expect, it } from 'vitest'
import {
  collectAvailableProjectTags,
  filterProjectsBySearchAndTags,
  hasActiveProjectFilters,
  parseTagsQueryParam,
  serializeTagsQueryParam,
} from './projectListFilterUtils.js'

describe('projectListFilterUtils', () => {
  const sample = [
    { id: '1', name: 'Alpha', description: 'first', tags: ['frontend', 'vue'] },
    { id: '2', name: 'Beta', description: 'second', tags: ['backend'] },
    { id: '3', name: 'Gamma', description: 'mix', tags: [] },
  ]

  it('collectAvailableProjectTags dedupes and sorts', () => {
    expect(collectAvailableProjectTags(sample)).toEqual(['backend', 'frontend', 'vue'])
  })

  it('filterProjectsBySearchAndTags filters by search', () => {
    const out = filterProjectsBySearchAndTags(sample, { searchQuery: 'beta' })
    expect(out.map((p) => p.id)).toEqual(['2'])
  })

  it('filterProjectsBySearchAndTags filters by single tag', () => {
    const out = filterProjectsBySearchAndTags(sample, { selectedTags: ['frontend'] })
    expect(out.map((p) => p.id)).toEqual(['1'])
  })

  it('filterProjectsBySearchAndTags uses OR for multiple tags', () => {
    const out = filterProjectsBySearchAndTags(sample, { selectedTags: ['frontend', 'backend'] })
    expect(out.map((p) => p.id)).toEqual(['1', '2'])
  })

  it('combines search and tags', () => {
    const out = filterProjectsBySearchAndTags(sample, {
      searchQuery: 'a',
      selectedTags: ['backend'],
    })
    expect(out.map((p) => p.id)).toEqual(['2'])
  })

  it('parseTagsQueryParam and serialize roundtrip', () => {
    expect(parseTagsQueryParam('frontend, backend')).toEqual(['frontend', 'backend'])
    expect(serializeTagsQueryParam(['vue', 'frontend'])).toBe('vue,frontend')
  })

  it('hasActiveProjectFilters detects active state', () => {
    expect(hasActiveProjectFilters({})).toBe(false)
    expect(hasActiveProjectFilters({ searchQuery: 'x' })).toBe(true)
    expect(hasActiveProjectFilters({ selectedTags: ['a'] })).toBe(true)
  })
})
