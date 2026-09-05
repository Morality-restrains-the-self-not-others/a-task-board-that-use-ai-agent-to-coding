import { describe, expect, it, vi } from 'vitest'
import {
  buildWorkPanelFilterPutBody,
  fetchWorkPanelFilters,
  normalizeWorkPanelFilterPayload,
  saveWorkPanelFilters,
  workPanelFiltersApiUrl,
} from './workPanelFilterPersistence.js'

describe('workPanelFiltersApiUrl', () => {
  it('builds workspaces sub-resource path (wid positional, kv pairs last)', () => {
    expect(workPanelFiltersApiUrl('850', '901')).toBe(
      '/api/projects/workspaces/901/work-panel-filters/tenant_id/850',
    )
  })
})

describe('normalizeWorkPanelFilterPayload', () => {
  it('returns default for null/invalid', () => {
    const d = normalizeWorkPanelFilterPayload(null)
    expect(d.deliverable_filter_bars).toHaveLength(1)
    expect(d.deliverable_filter_bars[0].path[0].type).toBe('root')
    expect(d.access_filter).toBeNull()
  })

  it('keeps two bars', () => {
    const p = normalizeWorkPanelFilterPayload({
      version: 1,
      deliverable_filter_bars: [
        { id: 'bar-0', path: [{ type: 'root' }] },
        {
          id: 'bar-1',
          path: [
            { type: 'root' },
            { type: 'category', id: 'c1', label: 'Cat' },
          ],
        },
      ],
    })
    expect(p.deliverable_filter_bars).toHaveLength(2)
    expect(p.deliverable_filter_bars[1].path[1].id).toBe('c1')
    expect(p.access_filter).toBeNull()
  })

  it('drops bars over max', () => {
    const bars = Array.from({ length: 21 }, (_, i) => ({
      id: `b${i}`,
      path: [{ type: 'root' }],
    }))
    const p = normalizeWorkPanelFilterPayload({ deliverable_filter_bars: bars })
    expect(p.deliverable_filter_bars).toHaveLength(1)
  })

  it('T1: keeps person/group access_filter', () => {
    const person = normalizeWorkPanelFilterPayload({
      deliverable_filter_bars: [{ id: 'bar-0', path: [{ type: 'root' }] }],
      access_filter: { kind: 'person', id: 'cm1', label: 'Alice' },
    })
    expect(person.access_filter).toEqual({ kind: 'person', id: 'cm1', label: 'Alice' })
    const group = normalizeWorkPanelFilterPayload({
      deliverable_filter_bars: [{ id: 'bar-0', path: [{ type: 'root' }] }],
      access_filter: { kind: 'group', id: 'g1', label: 'Core' },
    })
    expect(group.access_filter).toEqual({ kind: 'group', id: 'g1', label: 'Core' })
  })

  it('T2: invalid access_filter becomes null', () => {
    const p = normalizeWorkPanelFilterPayload({
      deliverable_filter_bars: [{ id: 'bar-0', path: [{ type: 'root' }] }],
      access_filter: { kind: 'nope', id: 'x' },
    })
    expect(p.access_filter).toBeNull()
  })
})

describe('buildWorkPanelFilterPutBody', () => {
  it('T3: serializes bars and access_filter', () => {
    const body = buildWorkPanelFilterPutBody(
      [{ id: 'x', path: [{ type: 'root' }] }],
      { kind: 'person', id: 'cm1', label: 'Alice', memberIds: ['cm1'] },
    )
    expect(body.version).toBe(2)
    expect(body.deliverable_filter_bars[0].id).toBe('x')
    expect(body.access_filter).toEqual({ kind: 'person', id: 'cm1', label: 'Alice' })
  })
})

describe('fetchWorkPanelFilters / saveWorkPanelFilters', () => {
  it('GET applies payload', async () => {
    const apiFetch = vi.fn(async () => ({
      ok: true,
      json: async () => ({
        status: 'success',
        deliverable_filter_bars: [
          { id: 'a', path: [{ type: 'root' }] },
          { id: 'b', path: [{ type: 'root' }] },
        ],
        access_filter: { kind: 'group', id: 'g1', label: 'Ops' },
      }),
    }))
    const p = await fetchWorkPanelFilters({
      apiFetch,
      tenantId: '850',
      workspaceId: '901',
    })
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/projects/workspaces/901/work-panel-filters/tenant_id/850',
      expect.objectContaining({ credentials: 'include' }),
    )
    expect(p.deliverable_filter_bars).toHaveLength(2)
    expect(p.access_filter).toEqual({ kind: 'group', id: 'g1', label: 'Ops' })
  })

  it('PUT sends body with access_filter', async () => {
    const apiFetch = vi.fn(async () => ({
      ok: true,
      json: async () => ({
        status: 'success',
        deliverable_filter_bars: [{ id: 'a', path: [{ type: 'root' }] }],
        access_filter: null,
      }),
    }))
    await saveWorkPanelFilters({
      apiFetch,
      tenantId: '850',
      workspaceId: '901',
      bars: [{ id: 'a', path: [{ type: 'root' }] }],
      accessFilter: { kind: 'person', id: 'cm9', label: 'Bob' },
    })
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/projects/workspaces/901/work-panel-filters/tenant_id/850',
      expect.objectContaining({ method: 'PUT' }),
    )
    const opts = apiFetch.mock.calls[0][1]
    const body = JSON.parse(opts.body)
    expect(body.deliverable_filter_bars).toHaveLength(1)
    expect(body.access_filter).toEqual({ kind: 'person', id: 'cm9', label: 'Bob' })
  })
})
