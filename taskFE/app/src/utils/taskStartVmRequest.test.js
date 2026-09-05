import { describe, expect, it } from 'vitest'
import { buildTaskStartVmRequestFromRunTemplate } from './taskStartVmRequest.js'

describe('buildTaskStartVmRequestFromRunTemplate', () => {
  it('builds start-vm-auto when vpc/security group are auto-created', () => {
    const req = buildTaskStartVmRequestFromRunTemplate({
      taskId: 'task-1',
      containerImageId: 'img-1',
      runTemplate: {
        cloud_platform_id: '7',
        region: 'cn-hongkong',
        zone_id: 'cn-hongkong-d',
        vpc_id: '',
        security_group_id: '',
        selected_instance: 'ecs.c6.large',
        hardware_config: { cpu_cores: 2, memory_gb: 4, storage_gb: 40 },
        filter_options: { spot_strategy: 'SpotAsPriceGo', system_disk_category: 'cloud_essd' },
        bandwidth: 5,
      },
      autoReleaseEnabled: true,
      autoReleaseMinutes: 30,
    })
    expect(req?.apiPath).toBe('start-vm-auto')
    expect(req?.body.task_id).toBe('task-1')
    expect(req?.body.hardware_config.instance_type).toBe('ecs.c6.large')
    expect(req?.body.auto_create_security_group).toBe(true)
    expect(req?.body.auto_release_enabled).toBe(true)
    expect(req?.body.auto_release_minutes).toBe(30)
    expect(req?.body.runtime_source).toBe('cloud_vm_template')
  })
})
