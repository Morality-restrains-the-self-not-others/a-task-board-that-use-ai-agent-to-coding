import { describe, expect, it } from 'vitest'
import {
  normalizeProjectTags,
  parseProjectTagInput,
  PROJECT_TAG_MAX_LENGTH,
  PROJECT_TAGS_MAX_COUNT,
} from './projectTagsUtils.js'

describe('projectTagsUtils', () => {
  it('normalizeProjectTags trims dedupes and caps count', () => {
    const tags = [' Alpha ', 'beta', 'ALPHA', '  ', 'gamma']
    expect(normalizeProjectTags(tags)).toEqual(['Alpha', 'beta', 'gamma'])
    const many = Array.from({ length: 25 }, (_, i) => `tag-${i}`)
    expect(normalizeProjectTags(many).length).toBe(PROJECT_TAGS_MAX_COUNT)
  })

  it('normalizeProjectTags drops overlong tags', () => {
    const long = 'x'.repeat(PROJECT_TAG_MAX_LENGTH + 1)
    expect(normalizeProjectTags(['ok', long])).toEqual(['ok'])
  })

  it('parseProjectTagInput rejects empty and overlong', () => {
    expect(parseProjectTagInput('  hello ')).toBe('hello')
    expect(parseProjectTagInput('')).toBeNull()
    expect(parseProjectTagInput('x'.repeat(PROJECT_TAG_MAX_LENGTH + 1))).toBeNull()
  })
})
