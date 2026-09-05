// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] hardwarePanelStartVm.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  describe('hardwarePanelStartVm', () => {
    beforeEach(() => {
      vi.resetModules()
      vi.clearAllMocks()
    })

    it('auto SG 时创建查询并写入 client_public_ip', async () => {
      vi.doMock('./publicClientIp.js', () => ({
        queryClientPublicIpForAutoSg: vi.fn(async () => '203.0.113.88'),
      }))
      vi.doMock('./apiUtils.js', () => ({ apiFetch: vi.fn() }))
      vi.doMock('./startVmHttpResult.js', () => ({
        applyStartVmHttpResult: vi.fn(() => 'accepted'),
        buildStartVmErrorStatusUpdate: vi.fn(),
      }))
      vi.doMock('./taskStartVmRequest.js', () => ({
        buildTaskStartVmRequestFromRunTemplate: vi.fn(),
      }))
      const { attachClientPublicIpForAutoSg } = await import('./hardwarePanelStartVm.js')
      const body = await attachClientPublicIpForAutoSg({ auto_create_security_group: true, foo: 1 })
      expect(body.client_public_ip).toBe('203.0.113.88')
      expect(body.foo).toBe(1)
    })

    it('非 auto SG 不查询', async () => {
      const query = vi.fn(async () => '203.0.113.88')
      vi.doMock('./publicClientIp.js', () => ({ queryClientPublicIpForAutoSg: query }))
      vi.doMock('./apiUtils.js', () => ({ apiFetch: vi.fn() }))
      vi.doMock('./startVmHttpResult.js', () => ({
        applyStartVmHttpResult: vi.fn(),
        buildStartVmErrorStatusUpdate: vi.fn(),
      }))
      vi.doMock('./taskStartVmRequest.js', () => ({
        buildTaskStartVmRequestFromRunTemplate: vi.fn(),
      }))
      const { attachClientPublicIpForAutoSg } = await import('./hardwarePanelStartVm.js')
      const body = await attachClientPublicIpForAutoSg({ auto_create_security_group: false })
      expect(body.client_public_ip).toBeUndefined()
      expect(query).not.toHaveBeenCalled()
    })

    it('手动启动体以已选实例规格覆盖占位 CPU/内存', async () => {
      vi.doMock('./publicClientIp.js', () => ({
        queryClientPublicIpForAutoSg: vi.fn(async () => ''),
      }))
      vi.doMock('./apiUtils.js', () => ({ apiFetch: vi.fn() }))
      vi.doMock('./startVmHttpResult.js', () => ({
        applyStartVmHttpResult: vi.fn(),
        buildStartVmErrorStatusUpdate: vi.fn(),
      }))
      vi.doMock('./taskStartVmRequest.js', () => ({
        buildTaskStartVmRequestFromRunTemplate: vi.fn(),
      }))
      const { buildManualHardwareStartVmBody } = await import('./hardwarePanelStartVm.js')
      const body = await buildManualHardwareStartVmBody({
        taskId: 'task_1',
        containerImageId: 'img_1',
        hardwareConfig: { cpu_cores: 1, memory_gb: 1, storage_gb: 40 },
        filterOptions: { spot_strategy: 'SpotAsPriceGo' },
        selectedInstance: {
          instance_type: 'ecs.e-c1m4.xlarge',
          cpu_cores: 4,
          memory_gb: 16,
        },
        selectedBandwidth: {},
        defaultBandwidth: 5,
        regionId: 'cn-hongkong',
        zoneId: 'cn-hongkong-c',
        vpcId: 'vpc-1',
        vswitchId: 'vsw-1',
        autoCreateVswitch: false,
        cloudPlatformId: '1',
        authorizationId: 'auth-1',
        securityGroupId: 'sg-1',
        autoCreateSecurityGroup: false,
        bandwidth: 5,
        bandwidthChargingMode: 'PayByTraffic',
        autoReleaseEnabled: false,
        autoReleaseMinutes: 30,
      })
      expect(body.hardware_config.instance_type).toBe('ecs.e-c1m4.xlarge')
      expect(body.hardware_config.cpu_cores).toBe(4)
      expect(body.hardware_config.memory_gb).toBe(16)
      expect(body.hardware_config.storage_gb).toBe(40)
      expect(body.selected_instance).toBe('ecs.e-c1m4.xlarge')
      expect(body.runtime_source).toBe('cloud_vm_manual')
    })
  })
}
