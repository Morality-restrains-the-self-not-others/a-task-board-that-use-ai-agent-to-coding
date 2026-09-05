import { describe, expect, it } from 'vitest'
import { buildPaginationItems } from './paginationPages.js'

describe('buildPaginationItems', () => {
  it('returns empty for non-positive total', () => {
    expect(buildPaginationItems(1, 0)).toEqual([])
    expect(buildPaginationItems(1, -2)).toEqual([])
  })

  it('lists all pages when total is small', () => {
    expect(buildPaginationItems(1, 5)).toEqual([1, 2, 3, 4, 5])
    expect(buildPaginationItems(3, 7)).toEqual([1, 2, 3, 4, 5, 6, 7])
  })

  it('inserts ellipsis for large totals around the current page', () => {
    expect(buildPaginationItems(5, 20)).toEqual([1, 'ellipsis', 4, 5, 6, 'ellipsis', 20])
  })

  it('fills near the start without a left ellipsis gap of 1', () => {
    expect(buildPaginationItems(1, 20)).toEqual([1, 2, 3, 4, 5, 'ellipsis', 20])
  })

  it('fills near the end without a right ellipsis gap of 1', () => {
    expect(buildPaginationItems(20, 20)).toEqual([1, 'ellipsis', 16, 17, 18, 19, 20])
  })

  it('clamps current page into [1, total]', () => {
    expect(buildPaginationItems(0, 20)[0]).toBe(1)
    expect(buildPaginationItems(99, 20)).toContain(20)
  })
})
