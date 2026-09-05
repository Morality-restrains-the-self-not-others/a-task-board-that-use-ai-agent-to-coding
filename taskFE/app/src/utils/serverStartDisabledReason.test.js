import { describe, expect, it } from 'vitest'
import {
  resolveServerStartDisabledReason,
  resolveTemporaryHardwareStartDisabledReason,
} from './serverStartDisabledReason.js'
import { resolveProjectTemplateStartDisabledReason } from './taskStartVmRequest.js'

const readyTemporary = {
  hasCloudContext: true,
  selectedImageId: 'img-1',
  selectedCloudPlatform: '1',
  selectedRegion: 'cn-hangzhou',
  selectedVpc: 'vpc-1',
  selectedZone: 'cn-hangzhou-h:vsw-1',
  selectedInstance: { instance_type: 'ecs.small' },
}

const readyTemplate = {
  hasCloudContext: true,
  selectedImageId: 'img-1',
  taskId: 'task-1',
  runTemplate: {
    cloud_platform_id: '1',
    region: 'cn-hangzhou',
    zone_id: 'cn-hangzhou-h',
    vpc_id: '',
    security_group_id: '',
    selected_instance: 'ecs.c6.large',
    hardware_config: { cpu_cores: 2, memory_gb: 4 },
    filter_options: { spot_strategy: 'SpotAsPriceGo' },
  },
}

describe('resolveTemporaryHardwareStartDisabledReason', () => {
  it('returns empty when prerequisites are satisfied', () => {
    expect(resolveTemporaryHardwareStartDisabledReason(readyTemporary)).toBe('')
  })

  it('explains missing instance selection', () => {
    expect(resolveTemporaryHardwareStartDisabledReason({ ...readyTemporary, selectedInstance: null }))
      .toBe('请从可用实例列表中选择一台实例')
  })

  it('blocks when server is running', () => {
    expect(resolveTemporaryHardwareStartDisabledReason({
      ...readyTemporary,
      isServerRunning: true,
    })).toBe('服务器已在运行，如需重新启动请先停止')
  })

  it('does not treat stale isServerRunning as running when cloud is Stopped', () => {
    expect(resolveTemporaryHardwareStartDisabledReason({
      ...readyTemporary,
      isServerRunning: true,
      serverRuntimeStatus: 'Stopped',
    })).toBe('')
  })
})

describe('resolveProjectTemplateStartDisabledReason', () => {
  it('returns empty when template is complete', () => {
    expect(resolveProjectTemplateStartDisabledReason(readyTemplate)).toBe('')
  })

  it('explains missing instance type in template', () => {
    expect(resolveProjectTemplateStartDisabledReason({
      ...readyTemplate,
      runTemplate: { region: 'cn-hangzhou', cloud_platform_id: '1' },
    })).toBe('项目运行硬件模版未配置实例规格')
  })

  it('does not treat stale isServerRunning as running when cloud is Released', () => {
    expect(resolveProjectTemplateStartDisabledReason({
      ...readyTemplate,
      isServerRunning: true,
      serverRuntimeStatus: 'Released',
    })).toBe('')
  })
})

describe('resolveServerStartDisabledReason', () => {
  it('routes to project template resolver', () => {
    expect(resolveServerStartDisabledReason({
      hardwareConfigSource: 'project_template',
      ...readyTemplate,
    })).toBe('')
  })

  it('routes to temporary resolver', () => {
    expect(resolveServerStartDisabledReason({
      hardwareConfigSource: 'temporary',
      ...readyTemporary,
    })).toBe('')
  })

  it('requireEnvParamsSource 且来源未选时优先提示', () => {
    expect(resolveServerStartDisabledReason({
      hardwareConfigSource: 'temporary',
      ...readyTemporary,
      requireEnvParamsSource: true,
      featureParamsSource: '',
    })).toBe('请先选择智能体资源配置')
  })

  it('requireEnvParamsSource 已选来源时不阻断', () => {
    expect(resolveServerStartDisabledReason({
      hardwareConfigSource: 'temporary',
      ...readyTemporary,
      requireEnvParamsSource: true,
      featureParamsSource: 'company',
    })).toBe('')
  })
})
