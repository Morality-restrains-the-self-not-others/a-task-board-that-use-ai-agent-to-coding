import { describe, expect, it } from 'vitest'
import {
  ACCESS_TOKEN_PAGE_SIZE,
  buildAccessTokenListView,
  countAccessTokensByStatus,
  filterAccessTokensByStatus,
  normalizeAccessTokenStatusFilter,
  paginateItems,
  sortAccessTokens,
} from './accessTokenListView.js'

const sample = [
  { id: '1', name: 'chrome', is_revoked: false, created_at: '2026-07-17T07:49:16Z', last_used_at: '2026-07-17T07:49:34Z' },
  { id: '2', name: 'chrome', is_revoked: true, created_at: '2026-07-17T04:34:13Z', last_used_at: '2026-07-17T04:40:00Z' },
  { id: '3', name: 'cli', is_revoked: false, created_at: '2026-07-16T01:00:00Z', last_used_at: null },
  { id: '4', name: 'old', is_revoked: true, created_at: '2026-07-01T00:00:00Z', last_used_at: '2026-07-02T00:00:00Z' },
]

describe('normalizeAccessTokenStatusFilter', () => {
  it('defaults to active', () => {
    expect(normalizeAccessTokenStatusFilter()).toBe('active')
    expect(normalizeAccessTokenStatusFilter('')).toBe('active')
    expect(normalizeAccessTokenStatusFilter('unknown')).toBe('active')
  })

  it('accepts revoked and all', () => {
    expect(normalizeAccessTokenStatusFilter('revoked')).toBe('revoked')
    expect(normalizeAccessTokenStatusFilter('ALL')).toBe('all')
  })
})

describe('filterAccessTokensByStatus', () => {
  it('filters active and revoked', () => {
    expect(filterAccessTokensByStatus(sample, 'active').map((t) => t.id)).toEqual(['1', '3'])
    expect(filterAccessTokensByStatus(sample, 'revoked').map((t) => t.id)).toEqual(['2', '4'])
    expect(filterAccessTokensByStatus(sample, 'all')).toHaveLength(4)
  })
})

describe('sortAccessTokens', () => {
  it('puts active before revoked, then by recency', () => {
    expect(sortAccessTokens(sample).map((t) => t.id)).toEqual(['1', '3', '2', '4'])
  })
})

describe('paginateItems', () => {
  it('pages with default size', () => {
    const items = Array.from({ length: 23 }, (_, i) => ({ id: String(i + 1) }))
    const page1 = paginateItems(items, 1)
    expect(page1.pageSize).toBe(ACCESS_TOKEN_PAGE_SIZE)
    expect(page1.items).toHaveLength(10)
    expect(page1.totalPages).toBe(3)
    expect(page1.currentPage).toBe(1)

    const page3 = paginateItems(items, 3)
    expect(page3.items).toHaveLength(3)
    expect(page3.currentPage).toBe(3)
  })

  it('clamps out-of-range page', () => {
    const page = paginateItems([{ id: 'a' }], 99, 10)
    expect(page.currentPage).toBe(1)
    expect(page.items).toHaveLength(1)
  })
})

describe('countAccessTokensByStatus / buildAccessTokenListView', () => {
  it('counts statuses', () => {
    expect(countAccessTokensByStatus(sample)).toEqual({ active: 2, revoked: 2, all: 4 })
  })

  it('builds filtered page view', () => {
    const view = buildAccessTokenListView(sample, { statusFilter: 'revoked', page: 1, pageSize: 1 })
    expect(view.statusFilter).toBe('revoked')
    expect(view.total).toBe(2)
    expect(view.totalPages).toBe(2)
    expect(view.items).toHaveLength(1)
    expect(view.items[0].id).toBe('2')
    expect(view.counts.active).toBe(2)
  })
})
