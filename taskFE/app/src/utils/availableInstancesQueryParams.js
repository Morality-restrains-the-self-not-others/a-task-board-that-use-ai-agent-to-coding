/**
 * 构建 available-instances 查询的硬件/实例过滤参数。
 * 当 instanceTypeSearchQuery 非空时：只按实例编号（InstanceType）查，忽略 CPU/内存/磁盘/竞价等硬件筛选。
 *
 * @param {object} opts
 * @param {object} opts.filterOptions
 * @param {string} [opts.instanceTypeSearchQuery]
 * @param {string|null} [opts.nextToken]
 * @returns {URLSearchParams}
 */
export function buildAvailableInstancesFilterParams({
  filterOptions = {},
  instanceTypeSearchQuery = '',
  nextToken = null,
} = {}) {
  const filterParams = new URLSearchParams()
  const search = String(instanceTypeSearchQuery || '').trim()

  filterParams.append('DestinationResource', 'InstanceType')
  filterParams.append('ResourceType', 'instance')

  if (search) {
    filterParams.append('InstanceType', search)
    if (nextToken) {
      filterParams.append('NextToken', String(nextToken))
    }
    return filterParams
  }

  if (filterOptions.cores) {
    filterParams.append('Cores', filterOptions.cores)
  }
  if (filterOptions.memory) {
    filterParams.append('Memory', filterOptions.memory)
  }
  if (filterOptions.io_optimized) {
    filterParams.append('IoOptimized', 'optimized')
  } else {
    filterParams.append('IoOptimized', 'none')
  }
  if (filterOptions.system_disk_category) {
    filterParams.append('SystemDiskCategory', filterOptions.system_disk_category)
  }
  if (filterOptions.data_disk_category) {
    filterParams.append('DataDiskCategory', filterOptions.data_disk_category)
  }
  if (filterOptions.spot_strategy) {
    filterParams.append('spot_strategy', filterOptions.spot_strategy)
  }
  if (filterOptions.network_category) {
    filterParams.append('NetworkCategory', filterOptions.network_category)
  }
  if (nextToken) {
    filterParams.append('NextToken', String(nextToken))
  }
  return filterParams
}

/**
 * 数据盘类型是否已指定。空字符串表示「不要数据盘」，视为有效选择；
 * undefined/null 表示尚未选择。
 */
export function isDataDiskCategorySpecified(category) {
  return category !== undefined && category !== null
}

/**
 * 从已保存配置解析数据盘类型。空字符串表示「不要数据盘」，须保留；
 * 仅 null/undefined 回落默认盘型。
 */
export function coalesceDataDiskCategory(value, fallback = 'cloud_essd') {
  return value ?? fallback
}

/** 实例编号搜索模式下，仅需云平台/地域/可用区；不必填 CPU/内存等硬件筛选。 */
export function hasRequiredAvailableInstancesContext({
  selectedCloudPlatform,
  selectedRegion,
  selectedZone,
  defaultConfig,
  filterOptions = {},
  instanceTypeSearchQuery = '',
} = {}) {
  if (!selectedCloudPlatform || !selectedRegion || !selectedZone || !defaultConfig) {
    return false
  }
  if (String(instanceTypeSearchQuery || '').trim()) {
    return true
  }
  return Boolean(
    filterOptions.cores
    && filterOptions.memory
    && filterOptions.system_disk_category
    && isDataDiskCategorySpecified(filterOptions.data_disk_category)
    && filterOptions.spot_strategy,
  )
}
