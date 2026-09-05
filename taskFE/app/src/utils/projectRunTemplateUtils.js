const PLATFORM_LABELS = {
  aliyun: '阿里云',
  tencentcloud: '腾讯云',
  huaweicloud: '华为云',
  ctyun: '天翼云',
  cmcc: '移动云',
  cucloud: '联通云',
  baiducloud: '百度智能云',
  aws: 'AWS',
  jdcloud: '京东云',
}

const getPlatformLabel = (type) => type ? (PLATFORM_LABELS[type] || type) : ''

/** 将租户 server-config-default 条目转为项目 server_run_template 存储结构 */
export function defaultConfigToRunTemplate(defaultConfig, cloudPlatformId = '') {
  if (!defaultConfig || typeof defaultConfig !== 'object') return {}
  const zoneId = String(defaultConfig.zone_id || '').trim()
  const vswitchId = String(defaultConfig.vswitch_id || '').trim()
  const labelParts = [
    String(defaultConfig.platform || '').trim(),
    String(defaultConfig.region || '').trim(),
  ].filter(Boolean)
  const cpuCoresRaw = defaultConfig.cpu_cores ?? defaultConfig.cores
  const memoryGbRaw = defaultConfig.memory_gb ?? defaultConfig.memory
  const cpuCores = cpuCoresRaw != null && String(cpuCoresRaw).trim() !== ''
    ? String(cpuCoresRaw).trim()
    : ''
  const memoryGb = memoryGbRaw != null && String(memoryGbRaw).trim() !== ''
    ? String(memoryGbRaw).trim()
    : ''
  const instanceType = String(
    defaultConfig.instance_type || defaultConfig.selected_instance || '',
  ).trim()
  const systemDisk = String(defaultConfig.system_disk_category || '').trim() || 'cloud_essd'
  // 空字符串表示「不要数据盘」，须保留；仅缺省时回落默认盘型
  const dataDisk = Object.prototype.hasOwnProperty.call(defaultConfig, 'data_disk_category')
    ? String(defaultConfig.data_disk_category ?? '')
    : 'cloud_essd'
  const paymentType = String(defaultConfig.payment_type || '').trim()

  const hardware_config = {}
  if (cpuCores) hardware_config.cpu_cores = cpuCores
  if (memoryGb) hardware_config.memory_gb = memoryGb
  if (instanceType) hardware_config.instance_type = instanceType

  const filter_options = {
    system_disk_category: systemDisk,
    data_disk_category: dataDisk,
  }
  if (cpuCores) filter_options.cores = cpuCores
  if (memoryGb) filter_options.memory = memoryGb
  if (paymentType) filter_options.instance_charge_type = paymentType
  // IoOptimized/竞价策略：与实例筛选区对齐（OPT-20260812-020）；空值不写入，
  // 由 available-instances 查询方按其默认处理
  if (defaultConfig.io_optimized) {
    filter_options.io_optimized = defaultConfig.io_optimized === 'optimized'
  }
  if (defaultConfig.spot_strategy) {
    filter_options.spot_strategy = String(defaultConfig.spot_strategy).trim()
  }

  return {
    template_id: defaultConfig.id != null ? String(defaultConfig.id) : '',
    label: labelParts.join(' · ') || '服务器模版',
    platform: String(defaultConfig.platform || '').trim(),
    authorization_id: String(defaultConfig.authorization_id || '').trim(),
    cloud_platform_id: cloudPlatformId ? String(cloudPlatformId) : '',
    region: String(defaultConfig.region || '').trim(),
    zone_id: zoneId,
    vpc_id: String(defaultConfig.vpc_id || '').trim(),
    vswitch_id: vswitchId,
    security_group_id: String(defaultConfig.security_group_id || '').trim(),
    payment_type: paymentType,
    bandwidth_charging_mode: String(defaultConfig.bandwidth_charging_mode || '').trim(),
    bandwidth: defaultConfig.bandwidth ?? null,
    hardware_config,
    filter_options,
    ...(instanceType ? { selected_instance: instanceType } : {}),
  }
}

/** 将项目 server_run_template 转为 ServerConfigHardwarePanel 可应用的 server_config 形状 */
export function runTemplateToServerConfigShape(template) {
  if (!template || typeof template !== 'object') return null
  const hw = template.hardware_config && typeof template.hardware_config === 'object'
    ? template.hardware_config
    : {}
  const zoneId = String(template.zone_id || '').trim()
  const vswitchId = String(template.vswitch_id || '').trim()
  const instanceType = resolveRunTemplateInstanceType(template)
  return {
    platform: template.platform || '',
    platform_id: template.cloud_platform_id || template.platform_id || '',
    region: template.region || '',
    zone_id: zoneId,
    vpc_id: String(template.vpc_id || '').trim(),
    vswitch_id: vswitchId,
    security_group_id: template.security_group_id || '',
    authorization_id: template.authorization_id || '',
    selected_instance: instanceType,
    hardware_config: {
      cpu_cores: hw.cpu_cores ?? '1',
      memory_gb: hw.memory_gb ?? '1',
      storage_gb: hw.storage_gb ?? '40',
      ...(instanceType ? { instance_type: instanceType } : {}),
    },
    filter_options: template.filter_options && typeof template.filter_options === 'object'
      ? template.filter_options
      : {},
  }
}

/** 判断 hardware_config 是否为未同步实例规格前的占位默认值（1核1GB） */
export function isPlaceholderHardwareCoresMemory(hw = {}) {
  return String(hw.cpu_cores ?? '') === '1' && String(hw.memory_gb ?? '') === '1'
}

/**
 * 从面板状态解析应写入 hardware_config 的 CPU/内存。
 * 优先级：已选实例规格 > 过滤器（当 hardware 仍为占位值时）> hardware 草稿 > 过滤器兜底。
 */
export function resolveHardwareCoresMemoryForRunTemplate({
  hardwareConfig = {},
  filterOptions = {},
  selectedInstance = null,
} = {}) {
  const instanceCpu = selectedInstance?.cpu_cores ?? selectedInstance?.CpuCoreCount
  const instanceMem = selectedInstance?.memory_gb ?? selectedInstance?.MemorySize
  if (selectedInstance && Number(instanceCpu) > 0) {
    return {
      cpu_cores: String(instanceCpu),
      memory_gb: String(Number(instanceMem) > 0 ? instanceMem : (filterOptions?.memory ?? hardwareConfig?.memory_gb ?? '1')),
    }
  }

  const hwCpu = hardwareConfig?.cpu_cores
  const hwMem = hardwareConfig?.memory_gb
  if (isPlaceholderHardwareCoresMemory(hardwareConfig)) {
    const filterCpu = filterOptions?.cores
    const filterMem = filterOptions?.memory
    if ((filterCpu != null && filterCpu !== '') || (filterMem != null && filterMem !== '')) {
      return {
        cpu_cores: String(filterCpu ?? hwCpu ?? '1'),
        memory_gb: String(filterMem ?? hwMem ?? '1'),
      }
    }
  }

  return {
    cpu_cores: String(hwCpu ?? filterOptions?.cores ?? '1'),
    memory_gb: String(hwMem ?? filterOptions?.memory ?? '1'),
  }
}

/** 将 ServerConfigHardwarePanel 当前选择状态转为 server_run_template 存储结构 */
export function hardwarePanelStateToRunTemplate({
  selectedCloudPlatform = '',
  cloudPlatforms = [],
  cloudPlatformDefaultConfigs = {},
  selectedRegion = '',
  selectedZone = '',
  selectedVpc = '',
  selectedSecurityGroup = '',
  hardwareConfig = {},
  filterOptions = {},
  selectedInstance = null,
  selectedBandwidth = {},
  selectedBandwidthChargingMode = 'PayByTraffic',
  templateMeta = {},
} = {}) {
  const platform = cloudPlatforms.find((p) => String(p.id) === String(selectedCloudPlatform))
  const defaultConfig = cloudPlatformDefaultConfigs[selectedCloudPlatform]

  let zoneId = ''
  let vswitchId = ''
  if (selectedZone) {
    const parts = String(selectedZone).split(':')
    zoneId = String(parts[0] || '').trim()
    if (parts.length > 1) {
      vswitchId = String(parts[1] || '').trim()
    }
  }

  const instanceType = selectedInstance?.instance_type
    ? String(selectedInstance.instance_type).trim()
    : ''
  const bandwidth = instanceType
    ? Number(selectedBandwidth[instanceType] ?? 5)
    : 5

  const platformLabel = platform?.remark
    ? `${platform.remark}-${getPlatformLabel(platform.platform_type)}`
    : (getPlatformLabel(platform?.platform_type) || defaultConfig?.platform || '')
  const labelParts = [platformLabel, selectedRegion].filter(Boolean)

  const resolvedCoresMemory = resolveHardwareCoresMemoryForRunTemplate({
    hardwareConfig,
    filterOptions,
    selectedInstance,
  })
  const hw = {
    cpu_cores: resolvedCoresMemory.cpu_cores,
    memory_gb: resolvedCoresMemory.memory_gb,
    storage_gb: hardwareConfig?.storage_gb ?? '40',
  }
  if (instanceType) {
    hw.instance_type = instanceType
  }

  return {
    template_id: String(templateMeta.template_id || '').trim(),
    label: String(templateMeta.label || labelParts.join(' · ') || '服务器模版').trim(),
    platform: String(defaultConfig?.platform || platform?.platform_type || '').trim(),
    authorization_id: String(defaultConfig?.authorization_id || '').trim(),
    cloud_platform_id: selectedCloudPlatform ? String(selectedCloudPlatform) : '',
    region: String(selectedRegion || '').trim(),
    zone_id: zoneId,
    vpc_id: selectedVpc === 'auto_create_vpc' ? '' : String(selectedVpc || '').trim(),
    vswitch_id: vswitchId,
    security_group_id: selectedSecurityGroup === 'auto_create_security_group'
      ? ''
      : String(selectedSecurityGroup || '').trim(),
    payment_type: String(filterOptions?.instance_charge_type || 'PostPaid').trim(),
    bandwidth_charging_mode: selectedBandwidthChargingMode || 'PayByTraffic',
    bandwidth,
    hardware_config: hw,
    filter_options: filterOptions && typeof filterOptions === 'object' ? { ...filterOptions } : {},
    ...(instanceType ? { selected_instance: instanceType } : {}),
  }
}

/** 从 server_run_template 解析已选实例规格 */
export function resolveRunTemplateInstanceType(template) {
  if (!template || typeof template !== 'object') return ''
  const fromSelected = String(template.selected_instance || '').trim()
  if (fromSelected) return fromSelected
  const hw = template.hardware_config
  if (hw && typeof hw === 'object') {
    return String(hw.instance_type || '').trim()
  }
  return ''
}

/** 格式化实例按量付费价格（小时） */
export function formatInstanceHourlyPrice(currency, amount) {
  if (amount == null || amount === '' || amount === '无法获取价格') return null
  const normalizedCurrency = currency || 'CNY'
  const suffix = normalizedCurrency === 'CNY' ? '元/时' : `${normalizedCurrency}/h`
  return `${amount} ${suffix}`
}

/** 从实例或 GPU 字段对象生成可读 GPU 规格文案；无 GPU 时返回 null */
export function formatGpuSpec(spec = {}) {
  if (!spec || typeof spec !== 'object') return null
  const gpuType = String(spec.gpu_type || spec.gpuType || spec.GpuSpec || '').trim()
  const gpuCores = Number(spec.gpu_cores ?? spec.gpuCores ?? spec.GPUAmount ?? spec.GpuCoreCount ?? 0)
  const gpuCount = Number(spec.gpu_count ?? spec.gpuCount ?? spec.GpuCount ?? 0)
  const gpuMemory = Number(spec.gpu_memory ?? spec.gpuMemory ?? spec.GpuMemory ?? 0)

  if (!gpuType && !gpuCores && !gpuCount && !gpuMemory) return null

  const parts = []
  if (gpuType) {
    parts.push(gpuType)
  } else if (gpuCores > 0) {
    parts.push(`${gpuCores}核`)
  }
  if (gpuCount > 1) parts.push(`×${gpuCount}`)
  if (gpuMemory > 0) parts.push(`显存${gpuMemory}GB`)
  return parts.join(' ') || null
}

/** 构建运行模版硬件规格一行摘要（CPU / 内存 / GPU / 价格） */
export function buildRunTemplateHardwareSpecSummary({
  cpuCores,
  memoryGb,
  gpuSpec,
  priceText,
  priceLoading = false,
} = {}) {
  const parts = []
  const cpu = cpuCores != null && cpuCores !== '' ? String(cpuCores).trim() : ''
  const memory = memoryGb != null && memoryGb !== '' ? String(memoryGb).trim() : ''

  if (cpu) parts.push(`CPU ${cpu}核`)
  if (memory) parts.push(`内存 ${memory}GB`)
  if (gpuSpec) {
    parts.push(`GPU ${gpuSpec}`)
  } else if (cpu || memory) {
    parts.push('GPU 无')
  }
  if (priceLoading) {
    parts.push('价格加载中…')
  } else if (priceText) {
    parts.push(`价格 ${priceText}`)
  }
  return parts.length ? parts.join(' · ') : ''
}

/** 从 server_run_template 静态字段生成硬件规格摘要（无实时价格） */
export function summarizeRunTemplateHardwareSpecs(template) {
  if (!template || typeof template !== 'object') return ''
  const hw = template.hardware_config && typeof template.hardware_config === 'object'
    ? template.hardware_config
    : {}
  const instanceType = resolveRunTemplateInstanceType(template)
  const looksLikePlaceholder = instanceType
    && String(hw.cpu_cores ?? '') === '1'
    && String(hw.memory_gb ?? '') === '1'
  const summary = looksLikePlaceholder
    ? ''
    : buildRunTemplateHardwareSpecSummary({
      cpuCores: hw.cpu_cores,
      memoryGb: hw.memory_gb,
      gpuSpec: formatGpuSpec(hw),
    })
  if (summary) return summary
  return instanceType ? `实例 ${instanceType}` : ''
}

/** 将基础摘要与实例规格合并（避免重复拼接） */
export function appendInstanceTypeToSummary(baseSummary, instanceType) {
  const base = String(baseSummary || '').trim()
  const instance = String(instanceType || '').trim()
  if (!base || base === '未设置' || !instance) return base
  if (base.includes(instance)) return base
  return `${base} · ${instance}`
}

/**
 * 从硬件规格摘要中提取价格片段（含「价格」前缀）。
 * 例：`CPU 2核 · 内存 8GB · GPU 无 · 价格 0.206128 元/时` → `价格 0.206128 元/时`
 */
export function extractPriceLabelFromHardwareSpecSummary(specSummary) {
  const text = String(specSummary || '').trim()
  if (!text) return ''
  const match = text.match(/(?:^|·\s*)(价格(?:加载中…|\s+.+))$/)
  return match ? String(match[1] || '').trim() : ''
}

/** 将价格片段追加到运行模版身份摘要（避免重复） */
export function appendPriceToRunTemplateSummary(baseSummary, priceLabel) {
  const base = String(baseSummary || '').trim()
  const rawPrice = String(priceLabel || '').trim()
  if (!base || base === '未设置' || !rawPrice) return base
  const normalized = rawPrice.startsWith('价格') ? rawPrice : `价格 ${rawPrice}`
  if (base.includes(normalized) || /价格(?:加载中…|\s)/.test(base)) return base
  return `${base} · ${normalized}`
}

export function summarizeRunTemplate(template) {
  if (!template || typeof template !== 'object') return '未设置'
  const keys = Object.keys(template).filter((k) => {
    const v = template[k]
    if (v == null || v === '') return false
    if (typeof v === 'object' && !Array.isArray(v) && Object.keys(v).length === 0) return false
    return true
  })
  if (!keys.length) return '未设置'
  const instanceType = resolveRunTemplateInstanceType(template)
  const label = String(template.label || '').trim()
  if (label) return appendInstanceTypeToSummary(label, instanceType)
  const parts = [template.platform, template.region].filter(Boolean)
  const base = parts.length ? parts.join(' · ') : '已配置（未命名）'
  return appendInstanceTypeToSummary(base, instanceType)
}

/** 身份摘要 + 可选价格（用于项目详情「运行模版摘要」等一眼可见处） */
export function summarizeRunTemplateWithPrice(template, priceLabelOrSpecSummary = '') {
  const base = summarizeRunTemplate(template)
  const raw = String(priceLabelOrSpecSummary || '').trim()
  if (!raw) return base
  const fromSpec = extractPriceLabelFromHardwareSpecSummary(raw)
  const priceLabel = fromSpec || (raw.startsWith('价格') ? raw : `价格 ${raw}`)
  return appendPriceToRunTemplateSummary(base, priceLabel)
}

/** 项目是否已配置运行硬件模版（与后端 project_has_configured_run_template 语义对齐） */
export function projectHasConfiguredRunTemplate(project) {
  return summarizeRunTemplate(project?.server_run_template) !== '未设置'
}

/**
 * 更换镜像时的「完整运行模版」判定，与 taskProjectService runTemplateIsConfigured 对齐：
 * 需云平台（platform 或 cloud_platform_id）+ region + 实例规格。
 */
export function runTemplateIsHardwareComplete(template) {
  if (!template || typeof template !== 'object' || Array.isArray(template)) return false
  const platform = String(template.platform || '').trim()
  const cloudID = String(template.cloud_platform_id || '').trim()
  const region = String(template.region || '').trim()
  if (!platform && !cloudID) return false
  if (!region) return false
  return Boolean(resolveRunTemplateInstanceType(template))
}

/**
 * 镜像 PATCH 附带的模版：优先完整 live 草稿，否则回退已保存完整模版，
 * 避免硬件面板「有地域无实例」的半成品覆盖库内完整模版。
 */
export function resolveTemplateForImagePatch(liveTemplate, savedTemplate) {
  if (runTemplateIsHardwareComplete(liveTemplate)) return liveTemplate
  if (runTemplateIsHardwareComplete(savedTemplate)) return savedTemplate
  if (liveTemplate && typeof liveTemplate === 'object' && !Array.isArray(liveTemplate)
    && Object.keys(liveTemplate).length) {
    return liveTemplate
  }
  if (savedTemplate && typeof savedTemplate === 'object' && !Array.isArray(savedTemplate)) {
    return savedTemplate
  }
  return null
}

/** 根据项目运行模版构建 start-vm / start-vm-auto 请求体；无法构建时返回 null */
export function buildAutoRunStartVmRequest({ taskId, containerImageId, runTemplate }) {
  const shape = runTemplateToServerConfigShape(runTemplate)
  if (!shape || !taskId || !containerImageId) return null
  if (!shape.region || !shape.platform_id) return null

  const autoCreateVpc = !shape.vpc_id
  const autoCreateVswitch = !shape.vswitch_id && Boolean(shape.zone_id)
  const autoCreateSecurityGroup = !shape.security_group_id
  const apiPath = (autoCreateVpc || autoCreateVswitch || autoCreateSecurityGroup)
    ? 'start-vm-auto'
    : 'start-vm'

  const instanceType = runTemplate?.selected_instance
    || shape.hardware_config?.instance_type
    || null

  const resolvedCoresMemory = resolveHardwareCoresMemoryForRunTemplate({
    hardwareConfig: shape.hardware_config,
    filterOptions: runTemplate?.filter_options || shape.filter_options,
  })
  const hardwareConfig = {
    cpu_cores: Number(resolvedCoresMemory.cpu_cores),
    memory_gb: Number(resolvedCoresMemory.memory_gb),
    storage_gb: Number(shape.hardware_config?.storage_gb ?? 40),
  }
  if (instanceType) {
    hardwareConfig.instance_type = String(instanceType)
  }

  return {
    apiPath,
    body: {
      task_id: String(taskId),
      container_image_id: String(containerImageId),
      hardware_config: hardwareConfig,
      region_id: shape.region,
      zone_id: shape.zone_id || null,
      vpc_id: autoCreateVpc ? null : shape.vpc_id,
      vswitch_id: shape.vswitch_id || null,
      auto_create_vswitch: autoCreateVswitch,
      cloud_platform_id: shape.platform_id,
      authorization_id: shape.authorization_id || null,
      filter_options: shape.filter_options || {},
      selected_instance: instanceType,
      security_group_id: shape.security_group_id || null,
      auto_create_security_group: autoCreateSecurityGroup,
      bandwidth: runTemplate?.bandwidth ?? 1,
      bandwidth_charging_mode: runTemplate?.bandwidth_charging_mode || 'PayByTraffic',
      // 前端模版构建默认；任务详情模版启动会在 taskStartVmRequest 覆盖为 cloud_vm_template
      runtime_source: 'cloud_vm_auto_run',
    },
  }
}

export function resolveProjectServerRunTemplateFromProjects(projects, linkedProjectRows = []) {
  if (!Array.isArray(projects) || !projects.length) return null
  const linkedIds = (Array.isArray(linkedProjectRows) ? linkedProjectRows : [])
    .map((row) => String(row?.project_id || '').trim())
    .filter(Boolean)
  const ordered = linkedIds.length
    ? linkedIds.map((id) => projects.find((p) => String(p?.id) === id)).filter(Boolean)
    : projects
  for (const project of ordered) {
    const tpl = project?.server_run_template
    if (tpl && typeof tpl === 'object' && Object.keys(tpl).length > 0) {
      return tpl
    }
  }
  return null
}
