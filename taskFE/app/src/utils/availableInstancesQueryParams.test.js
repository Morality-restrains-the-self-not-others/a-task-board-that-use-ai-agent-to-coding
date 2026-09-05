// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  buildAvailableInstancesFilterParams,
  hasRequiredAvailableInstancesContext,
  isDataDiskCategorySpecified,
  coalesceDataDiskCategory,
} from './availableInstancesQueryParams.js'

const baseFilters = {
  cores: '2',
  memory: '4',
  io_optimized: true,
  system_disk_category: 'cloud_essd',
  data_disk_category: 'cloud_essd',
  spot_strategy: 'SpotAsPriceGo',
  network_category: 'vpc',
}

describe('buildAvailableInstancesFilterParams', () => {
  it('无搜索时附带硬件筛选参数', () => {
    const params = buildAvailableInstancesFilterParams({ filterOptions: baseFilters })
    expect(params.get('Cores')).toBe('2')
    expect(params.get('Memory')).toBe('4')
    expect(params.get('SystemDiskCategory')).toBe('cloud_essd')
    expect(params.get('DataDiskCategory')).toBe('cloud_essd')
    expect(params.get('InstanceType')).toBeNull()
  })

  it('数据盘类型为空（不要数据盘）时不传 DataDiskCategory', () => {
    const params = buildAvailableInstancesFilterParams({
      filterOptions: { ...baseFilters, data_disk_category: '' },
    })
    expect(params.get('SystemDiskCategory')).toBe('cloud_essd')
    expect(params.get('DataDiskCategory')).toBeNull()
  })

  it('有实例编号搜索时只传 InstanceType，忽略硬件筛选', () => {
    const params = buildAvailableInstancesFilterParams({
      filterOptions: baseFilters,
      instanceTypeSearchQuery: ' ecs.g6.large ',
    })
    expect(params.get('InstanceType')).toBe('ecs.g6.large')
    expect(params.get('Cores')).toBeNull()
    expect(params.get('Memory')).toBeNull()
    expect(params.get('SystemDiskCategory')).toBeNull()
    expect(params.get('spot_strategy')).toBeNull()
    expect(params.get('DestinationResource')).toBe('InstanceType')
  })
})

describe('hasRequiredAvailableInstancesContext', () => {
  const ctx = {
    selectedCloudPlatform: 'p1',
    selectedRegion: 'cn-hongkong',
    selectedZone: 'cn-hongkong-b',
    defaultConfig: { authorization_id: 'a1' },
    filterOptions: baseFilters,
  }

  it('搜索模式下可不填 CPU/内存', () => {
    expect(hasRequiredAvailableInstancesContext({
      ...ctx,
      filterOptions: { ...baseFilters, cores: '', memory: '' },
      instanceTypeSearchQuery: 'ecs.g6.large',
    })).toBe(true)
  })

  it('非搜索模式仍要求硬件筛选齐全', () => {
    expect(hasRequiredAvailableInstancesContext({
      ...ctx,
      filterOptions: { ...baseFilters, cores: '' },
    })).toBe(false)
    expect(hasRequiredAvailableInstancesContext(ctx)).toBe(true)
  })

  it('数据盘类型为空（不要数据盘）时仍视为筛选齐全', () => {
    expect(hasRequiredAvailableInstancesContext({
      ...ctx,
      filterOptions: { ...baseFilters, data_disk_category: '' },
    })).toBe(true)
  })

  it('数据盘类型缺失（undefined）时不视为齐全', () => {
    const { data_disk_category: _omit, ...rest } = baseFilters
    expect(hasRequiredAvailableInstancesContext({
      ...ctx,
      filterOptions: rest,
    })).toBe(false)
  })
})

describe('isDataDiskCategorySpecified', () => {
  it('空字符串（不要数据盘）视为已指定', () => {
    expect(isDataDiskCategorySpecified('')).toBe(true)
  })

  it('具体盘型视为已指定', () => {
    expect(isDataDiskCategorySpecified('cloud_essd')).toBe(true)
  })

  it('undefined/null 视为未指定', () => {
    expect(isDataDiskCategorySpecified(undefined)).toBe(false)
    expect(isDataDiskCategorySpecified(null)).toBe(false)
  })
})

describe('coalesceDataDiskCategory', () => {
  it('保留空字符串（不要数据盘）', () => {
    expect(coalesceDataDiskCategory('')).toBe('')
  })

  it('null/undefined 回落默认', () => {
    expect(coalesceDataDiskCategory(undefined)).toBe('cloud_essd')
    expect(coalesceDataDiskCategory(null)).toBe('cloud_essd')
  })

  it('保留已保存盘型', () => {
    expect(coalesceDataDiskCategory('cloud_ssd')).toBe('cloud_ssd')
  })
})
