// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  filterInstancesByTypeQuery,
  paginateFilteredInstances,
} from './filterInstancesByTypeQuery.js'

describe('filterInstancesByTypeQuery', () => {
  const sample = [
    { instance_type: 'ecs.g6.large' },
    { instance_type: 'ecs.g7.xlarge' },
    { instance_type: 'ecs.t6.large' },
  ]

  it('空查询返回原列表', () => {
    expect(filterInstancesByTypeQuery(sample, '')).toEqual(sample)
    expect(filterInstancesByTypeQuery(sample, '  ')).toEqual(sample)
    expect(filterInstancesByTypeQuery(sample, null)).toEqual(sample)
  })

  it('按实例编号子串过滤（忽略大小写）', () => {
    expect(filterInstancesByTypeQuery(sample, 'G6')).toEqual([
      { instance_type: 'ecs.g6.large' },
    ])
    expect(filterInstancesByTypeQuery(sample, 'ecs.g')).toHaveLength(2)
    expect(filterInstancesByTypeQuery(sample, 'no-such')).toEqual([])
  })

  it('非数组输入安全降级为空数组', () => {
    expect(filterInstancesByTypeQuery(null, 'g6')).toEqual([])
  })
})

describe('paginateFilteredInstances', () => {
  const items = Array.from({ length: 23 }, (_, i) => ({ instance_type: `ecs.t${i}` }))

  it('按页切片并纠正越界页码', () => {
    const page1 = paginateFilteredInstances(items, 1, 10)
    expect(page1.pageItems).toHaveLength(10)
    expect(page1.totalPages).toBe(3)
    expect(page1.safePage).toBe(1)

    const page3 = paginateFilteredInstances(items, 3, 10)
    expect(page3.pageItems).toHaveLength(3)
    expect(page3.safePage).toBe(3)

    const overflow = paginateFilteredInstances(items, 99, 10)
    expect(overflow.safePage).toBe(3)
    expect(overflow.pageItems).toHaveLength(3)
  })

  it('空列表仍至少 1 页', () => {
    const empty = paginateFilteredInstances([], 1, 10)
    expect(empty.totalPages).toBe(1)
    expect(empty.pageItems).toEqual([])
  })
})
