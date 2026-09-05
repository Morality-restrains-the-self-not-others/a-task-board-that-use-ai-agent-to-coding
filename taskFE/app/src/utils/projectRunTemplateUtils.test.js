import { describe, expect, it } from 'vitest'
import {
  defaultConfigToRunTemplate,
  hardwarePanelStateToRunTemplate,
  resolveHardwareCoresMemoryForRunTemplate,
  isPlaceholderHardwareCoresMemory,
  resolveProjectServerRunTemplateFromProjects,
  runTemplateToServerConfigShape,
  summarizeRunTemplate,
  summarizeRunTemplateWithPrice,
  summarizeRunTemplateHardwareSpecs,
  extractPriceLabelFromHardwareSpecSummary,
  appendPriceToRunTemplateSummary,
  formatGpuSpec,
  buildRunTemplateHardwareSpecSummary,
  formatInstanceHourlyPrice,
  projectHasConfiguredRunTemplate,
  runTemplateIsHardwareComplete,
  resolveTemplateForImagePatch,
  buildAutoRunStartVmRequest,
} from '../utils/projectRunTemplateUtils.js'

describe('projectRunTemplateUtils', () => {
  it('defaultConfigToRunTemplate maps server-config-default row', () => {
    const tpl = defaultConfigToRunTemplate({
      id: '100',
      platform: 'aliyun',
      region: 'cn-hangzhou',
      zone_id: 'cn-hangzhou-h',
      vpc_id: 'vpc-1',
      vswitch_id: 'vsw-1',
      security_group_id: 'sg-1',
      authorization_id: 'auth-9',
    }, '42')
    expect(tpl.template_id).toBe('100')
    expect(tpl.cloud_platform_id).toBe('42')
    expect(tpl.region).toBe('cn-hangzhou')
  })

  it('defaultConfigToRunTemplate maps hardware filters into filter_options and selected_instance', () => {
    const tpl = defaultConfigToRunTemplate({
      id: '200',
      platform: 'aliyun',
      region: 'cn-hongkong',
      zone_id: 'cn-hongkong-d',
      vpc_id: 'vpc-hk',
      vswitch_id: 'vsw-hk',
      security_group_id: 'sg-hk',
      authorization_id: 'auth-hk',
      payment_type: 'PostPaid',
      cpu_cores: 2,
      memory_gb: 8,
      instance_type: 'ecs.g6.large',
      system_disk_category: 'cloud_ssd',
      data_disk_category: '',
    }, '7')
    expect(tpl.zone_id).toBe('cn-hongkong-d')
    expect(tpl.selected_instance).toBe('ecs.g6.large')
    expect(tpl.hardware_config).toEqual({
      cpu_cores: '2',
      memory_gb: '8',
      instance_type: 'ecs.g6.large',
    })
    expect(tpl.filter_options).toEqual({
      cores: '2',
      memory: '8',
      system_disk_category: 'cloud_ssd',
      data_disk_category: '',
      instance_charge_type: 'PostPaid',
    })
  })

  it('defaultConfigToRunTemplate maps io_optimized and spot_strategy into filter_options', () => {
    const tpl = defaultConfigToRunTemplate({
      id: '300',
      platform: 'aliyun',
      region: 'cn-hongkong',
      zone_id: 'cn-hongkong-d',
      vpc_id: 'vpc-hk',
      vswitch_id: 'vsw-hk',
      security_group_id: 'sg-hk',
      authorization_id: 'auth-io',
      io_optimized: 'optimized',
      spot_strategy: 'SpotAsPriceGo',
    }, '7')
    expect(tpl.filter_options.io_optimized).toBe(true)
    expect(tpl.filter_options.spot_strategy).toBe('SpotAsPriceGo')

    const tplNone = defaultConfigToRunTemplate({
      id: '301',
      platform: 'aliyun',
      region: 'cn-hongkong',
      authorization_id: 'auth-io2',
      io_optimized: 'none',
      spot_strategy: '',
    }, '7')
    expect(tplNone.filter_options.io_optimized).toBe(false)
    expect(tplNone.filter_options.spot_strategy).toBeUndefined()
  })

  it('runTemplateToServerConfigShape produces hardware panel shape', () => {
    const shape = runTemplateToServerConfigShape({
      cloud_platform_id: '7',
      region: 'cn-hangzhou',
      zone_id: 'cn-hangzhou-h',
      vpc_id: 'vpc-1',
      vswitch_id: 'vsw-1',
      hardware_config: { cpu_cores: 2, memory_gb: 4, storage_gb: 80 },
    })
    expect(shape.platform_id).toBe('7')
    expect(shape.zone_id).toBe('cn-hangzhou-h')
    expect(shape.vpc_id).toBe('vpc-1')
    expect(shape.hardware_config.memory_gb).toBe(4)
  })

  it('runTemplateToServerConfigShape carries selected_instance into hardware_config.instance_type', () => {
    const shape = runTemplateToServerConfigShape({
      cloud_platform_id: '7',
      region: 'cn-hongkong',
      selected_instance: 'ecs.g6.large',
      filter_options: { cores: '2', memory: '8', system_disk_category: 'cloud_essd' },
      hardware_config: { cpu_cores: '2', memory_gb: '8' },
    })
    expect(shape.selected_instance).toBe('ecs.g6.large')
    expect(shape.hardware_config.instance_type).toBe('ecs.g6.large')
    expect(shape.filter_options.cores).toBe('2')
    expect(shape.filter_options.memory).toBe('8')
  })

  it('resolveProjectServerRunTemplateFromProjects prefers linked project', () => {
    const projects = [
      { id: '1', server_run_template: {} },
      { id: '2', server_run_template: { region: 'cn-beijing' } },
    ]
    const linked = [{ project_id: '2' }]
    expect(resolveProjectServerRunTemplateFromProjects(projects, linked)?.region).toBe('cn-beijing')
  })

  it('formatGpuSpec renders GPU fields', () => {
    expect(formatGpuSpec({ gpu_type: 'Tesla T4', gpu_count: 1 })).toBe('Tesla T4')
    expect(formatGpuSpec({ gpu_cores: 2, gpu_memory: 16 })).toBe('2核 显存16GB')
    expect(formatGpuSpec({})).toBeNull()
  })

  it('buildRunTemplateHardwareSpecSummary joins CPU memory GPU price', () => {
    expect(buildRunTemplateHardwareSpecSummary({
      cpuCores: 2,
      memoryGb: 4,
      gpuSpec: 'Tesla T4',
      priceText: '0.68 元/时',
    })).toBe('CPU 2核 · 内存 4GB · GPU Tesla T4 · 价格 0.68 元/时')
    expect(buildRunTemplateHardwareSpecSummary({
      cpuCores: 2,
      memoryGb: 4,
      priceLoading: true,
    })).toBe('CPU 2核 · 内存 4GB · GPU 无 · 价格加载中…')
  })

  it('summarizeRunTemplateHardwareSpecs reads hardware_config', () => {
    expect(summarizeRunTemplateHardwareSpecs({
      hardware_config: { cpu_cores: 4, memory_gb: 8 },
    })).toBe('CPU 4核 · 内存 8GB · GPU 无')
    expect(summarizeRunTemplateHardwareSpecs({
      selected_instance: 'ecs.c9i.large',
      hardware_config: { cpu_cores: 1, memory_gb: 1, instance_type: 'ecs.c9i.large' },
    })).toBe('实例 ecs.c9i.large')
  })

  it('formatInstanceHourlyPrice formats CNY hourly price', () => {
    expect(formatInstanceHourlyPrice('CNY', 0.68)).toBe('0.68 元/时')
    expect(formatInstanceHourlyPrice('CNY', '无法获取价格')).toBeNull()
  })

  it('summarizeRunTemplate uses label when present', () => {
    expect(summarizeRunTemplate({ label: '生产模版' })).toBe('生产模版')
    expect(summarizeRunTemplate({})).toBe('未设置')
  })

  it('summarizeRunTemplate appends selected instance type', () => {
    expect(summarizeRunTemplate({
      label: 'aliyun · cn-hongkong',
      selected_instance: 'ecs.g6.large',
    })).toBe('aliyun · cn-hongkong · ecs.g6.large')
    expect(summarizeRunTemplate({
      platform: 'aliyun',
      region: 'cn-hangzhou',
      hardware_config: { instance_type: 'ecs.c6.xlarge' },
    })).toBe('aliyun · cn-hangzhou · ecs.c6.xlarge')
    expect(summarizeRunTemplate({
      label: '香港模版 · ecs.g6.large',
      selected_instance: 'ecs.g6.large',
    })).toBe('香港模版 · ecs.g6.large')
  })

  it('extractPriceLabelFromHardwareSpecSummary reads trailing price segment', () => {
    expect(extractPriceLabelFromHardwareSpecSummary(
      'CPU 2核 · 内存 8GB · GPU 无 · 价格 0.206128 元/时',
    )).toBe('价格 0.206128 元/时')
    expect(extractPriceLabelFromHardwareSpecSummary(
      'CPU 2核 · 内存 8GB · GPU 无 · 价格加载中…',
    )).toBe('价格加载中…')
    expect(extractPriceLabelFromHardwareSpecSummary('CPU 2核 · 内存 8GB · GPU 无')).toBe('')
  })

  it('appendPriceToRunTemplateSummary appends price without duplicating', () => {
    expect(appendPriceToRunTemplateSummary(
      'ljy0808-阿里云 · cn-hongkong · ecs.u2a-c1m4.large',
      '价格 0.206128 元/时',
    )).toBe('ljy0808-阿里云 · cn-hongkong · ecs.u2a-c1m4.large · 价格 0.206128 元/时')
    expect(appendPriceToRunTemplateSummary(
      'ljy0808-阿里云 · cn-hongkong · ecs.u2a-c1m4.large · 价格 0.206128 元/时',
      '价格 0.206128 元/时',
    )).toBe('ljy0808-阿里云 · cn-hongkong · ecs.u2a-c1m4.large · 价格 0.206128 元/时')
    expect(appendPriceToRunTemplateSummary('未设置', '价格 0.1 元/时')).toBe('未设置')
  })

  it('summarizeRunTemplateWithPrice appends price from hardware spec summary', () => {
    expect(summarizeRunTemplateWithPrice({
      label: 'ljy0808-阿里云 · cn-hongkong',
      selected_instance: 'ecs.u2a-c1m4.large',
    }, 'CPU 2核 · 内存 8GB · GPU 无 · 价格 0.206128 元/时')).toBe(
      'ljy0808-阿里云 · cn-hongkong · ecs.u2a-c1m4.large · 价格 0.206128 元/时',
    )
    expect(summarizeRunTemplateWithPrice({
      label: 'ljy0808-阿里云 · cn-hongkong',
      selected_instance: 'ecs.u2a-c1m4.large',
    }, '0.206128 元/时')).toBe(
      'ljy0808-阿里云 · cn-hongkong · ecs.u2a-c1m4.large · 价格 0.206128 元/时',
    )
  })

  it('projectHasConfiguredRunTemplate detects configured template', () => {
    expect(projectHasConfiguredRunTemplate({ server_run_template: {} })).toBe(false)
    expect(projectHasConfiguredRunTemplate({ server_run_template: { region: 'cn-beijing' } })).toBe(true)
  })

  it('runTemplateIsHardwareComplete requires platform/cloud + region + instance', () => {
    expect(runTemplateIsHardwareComplete({ region: 'cn-beijing' })).toBe(false)
    expect(runTemplateIsHardwareComplete({
      platform: 'aliyun',
      region: 'cn-hangzhou',
    })).toBe(false)
    expect(runTemplateIsHardwareComplete({
      platform: 'aliyun',
      region: 'cn-hangzhou',
      selected_instance: 'ecs.g7.xlarge',
    })).toBe(true)
    expect(runTemplateIsHardwareComplete({
      cloud_platform_id: 'p1',
      region: 'cn-hangzhou',
      hardware_config: { instance_type: 'ecs.g7.xlarge' },
    })).toBe(true)
  })

  it('resolveTemplateForImagePatch prefers complete saved over incomplete live draft', () => {
    const saved = {
      platform: 'aliyun',
      region: 'cn-hangzhou',
      selected_instance: 'ecs.g7.xlarge',
    }
    const incompleteLive = {
      platform: 'aliyun',
      region: 'cn-hangzhou',
      label: '半成品',
    }
    expect(resolveTemplateForImagePatch(incompleteLive, saved)).toEqual(saved)
    expect(resolveTemplateForImagePatch({
      ...saved,
      selected_instance: 'ecs.g8y.xlarge',
    }, saved).selected_instance).toBe('ecs.g8y.xlarge')
  })

  it('buildAutoRunStartVmRequest builds start-vm payload', () => {
    const req = buildAutoRunStartVmRequest({
      taskId: 't1',
      containerImageId: 'img1',
      runTemplate: {
        region: 'cn-hangzhou',
        zone_id: 'cn-hangzhou-b',
        vpc_id: 'vpc-1',
        vswitch_id: 'vsw-1',
        cloud_platform_id: 'cp1',
        authorization_id: 'auth1',
        security_group_id: 'sg-1',
        hardware_config: { cpu_cores: 2, memory_gb: 4, storage_gb: 40 },
      },
    })
    expect(req?.apiPath).toBe('start-vm')
    expect(req?.body.task_id).toBe('t1')
    expect(req?.body.hardware_config.cpu_cores).toBe(2)
  })

  it('hardwarePanelStateToRunTemplate maps panel state', () => {
    const tpl = hardwarePanelStateToRunTemplate({
      selectedCloudPlatform: '1',
      cloudPlatforms: [{ id: '1', platform_name: '阿里云', platform_type: 'aliyun' }],
      cloudPlatformDefaultConfigs: { 1: { platform: 'aliyun', authorization_id: 'auth-1' } },
      selectedRegion: 'cn-hongkong',
      selectedZone: 'cn-hongkong-b:vsw-1',
      selectedVpc: 'vpc-1',
      selectedSecurityGroup: 'sg-1',
      hardwareConfig: { cpu_cores: '2', memory_gb: '4', storage_gb: '80' },
      filterOptions: { cores: '2', memory: '4', instance_charge_type: 'PostPaid' },
      selectedInstance: { instance_type: 'ecs.g6.large' },
      selectedBandwidth: { 'ecs.g6.large': 5 },
      templateMeta: { template_id: '100', label: '香港模版' },
    })
    expect(tpl.region).toBe('cn-hongkong')
    expect(tpl.hardware_config.instance_type).toBe('ecs.g6.large')
    expect(tpl.selected_instance).toBe('ecs.g6.large')
    expect(tpl.label).toBe('香港模版')
  })

  it('resolveHardwareCoresMemoryForRunTemplate prefers selected instance over placeholder hardware', () => {
    expect(resolveHardwareCoresMemoryForRunTemplate({
      hardwareConfig: { cpu_cores: '1', memory_gb: '1' },
      filterOptions: { cores: 2, memory: 8 },
      selectedInstance: { instance_type: 'ecs.g6.large', cpu_cores: 2, memory_gb: 8 },
    })).toEqual({ cpu_cores: '2', memory_gb: '8' })
  })

  it('resolveHardwareCoresMemoryForRunTemplate falls back to filter when hardware is placeholder', () => {
    expect(resolveHardwareCoresMemoryForRunTemplate({
      hardwareConfig: { cpu_cores: '1', memory_gb: '1' },
      filterOptions: { cores: 2, memory: 8 },
    })).toEqual({ cpu_cores: '2', memory_gb: '8' })
  })

  it('isPlaceholderHardwareCoresMemory detects default 1核1GB', () => {
    expect(isPlaceholderHardwareCoresMemory({ cpu_cores: '1', memory_gb: '1' })).toBe(true)
    expect(isPlaceholderHardwareCoresMemory({ cpu_cores: 2, memory_gb: 8 })).toBe(false)
  })

  it('hardwarePanelStateToRunTemplate syncs cpu/memory from selected instance when hardware draft is placeholder', () => {
    const tpl = hardwarePanelStateToRunTemplate({
      selectedCloudPlatform: '1',
      cloudPlatforms: [{ id: '1', platform_name: '阿里云', platform_type: 'aliyun' }],
      cloudPlatformDefaultConfigs: { 1: { platform: 'aliyun', authorization_id: 'auth-1' } },
      selectedRegion: 'cn-hongkong',
      selectedZone: 'cn-hongkong-d:',
      hardwareConfig: { cpu_cores: '1', memory_gb: '1', storage_gb: '40' },
      filterOptions: { cores: 2, memory: 8, instance_charge_type: 'PostPaid' },
      selectedInstance: { instance_type: 'ecs.g6.large', cpu_cores: 2, memory_gb: 8 },
      selectedBandwidth: { 'ecs.g6.large': 5 },
    })
    expect(tpl.hardware_config.cpu_cores).toBe('2')
    expect(tpl.hardware_config.memory_gb).toBe('8')
    expect(tpl.hardware_config.instance_type).toBe('ecs.g6.large')
    expect(tpl.filter_options.cores).toBe(2)
    expect(tpl.filter_options.memory).toBe(8)
  })

  it('buildAutoRunStartVmRequest uses filter_options when saved hardware_config is placeholder', () => {
    const req = buildAutoRunStartVmRequest({
      taskId: 't1',
      containerImageId: 'img1',
      runTemplate: {
        region: 'cn-hongkong',
        zone_id: 'cn-hongkong-d',
        vpc_id: '',
        vswitch_id: '',
        cloud_platform_id: 'cp1',
        authorization_id: 'auth1',
        security_group_id: '',
        hardware_config: { cpu_cores: '1', memory_gb: '1', storage_gb: 40, instance_type: 'ecs.g6.large' },
        selected_instance: 'ecs.g6.large',
        filter_options: { cores: 2, memory: 8 },
      },
    })
    expect(req?.body.hardware_config.cpu_cores).toBe(2)
    expect(req?.body.hardware_config.memory_gb).toBe(8)
    expect(req?.body.hardware_config.instance_type).toBe('ecs.g6.large')
  })

  it('buildAutoRunStartVmRequest includes selected_instance', () => {
    const req = buildAutoRunStartVmRequest({
      taskId: 't1',
      containerImageId: 'img1',
      runTemplate: {
        region: 'cn-hangzhou',
        zone_id: 'cn-hangzhou-b',
        vpc_id: 'vpc-1',
        vswitch_id: 'vsw-1',
        cloud_platform_id: 'cp1',
        authorization_id: 'auth1',
        security_group_id: 'sg-1',
        hardware_config: { cpu_cores: 2, memory_gb: 4, storage_gb: 40, instance_type: 'ecs.g6.large' },
        selected_instance: 'ecs.g6.large',
      },
    })
    expect(req?.body.selected_instance).toBe('ecs.g6.large')
    expect(req?.body.hardware_config.instance_type).toBe('ecs.g6.large')
  })
})
