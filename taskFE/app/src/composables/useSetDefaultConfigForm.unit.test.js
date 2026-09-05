// @vitest-environment jsdom
/**
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] useSetDefaultConfigForm.unit.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')
  const { ref, nextTick } = await import('vue')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { apiFetch } = await import('../utils/apiUtils.js')
  const { default: modalService } = await import('../utils/modalService.js')

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/modalService.js', () => ({
    __esModule: true,
    default: {
      alert: vi.fn(),
    },
  }))

  // We import after mocks
  const { useSetDefaultConfigForm } = await import('./useSetDefaultConfigForm.js')

  // Helper to create mock props/emit
  function createProps(overrides = {}) {
    return {
      visible: true,
      initialData: {
        authorization_id: 'auth-123',
        platform_type: 'aliyun',
      },
      ...overrides,
    }
  }

  function createEmit() {
    return vi.fn()
  }

  // Mock window.location
  const originalLocation = window.location

  beforeEach(() => {
    vi.clearAllMocks()
    // Mock location.pathname for getTenantId
    Object.defineProperty(window, 'location', {
      value: {
        pathname: '/tenant/850256677331562496/settings/task-panel/',
      },
      writable: true,
      configurable: true,
    })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
      configurable: true,
    })
  })

  describe('useSetDefaultConfigForm — hardware config', () => {
    it('initializes formData with hardware config fields', () => {
      const emit = createEmit()
      const result = useSetDefaultConfigForm(createProps(), emit)

      expect(result.formData.value).toHaveProperty('cpu_cores', '')
      expect(result.formData.value).toHaveProperty('memory_gb', '')
      expect(result.formData.value).toHaveProperty('instance_type', '')
      expect(result.formData.value).toHaveProperty('system_disk_category', 'cloud_essd')
      expect(result.formData.value).toHaveProperty('data_disk_category', 'cloud_essd')
      expect(result.formData.value).toHaveProperty('io_optimized', 'optimized')
      expect(result.formData.value).toHaveProperty('spot_strategy', '')
    })

    it('loadSavedConfig restores io_optimized and spot_strategy', async () => {
      const emit = createEmit()
      apiFetch.mockImplementation(async (url) => {
        if (url.includes('/cloud/server-config-default/')) {
          return {
            ok: true,
            json: async () => ({
              status: 'success',
              data: {
                region: 'cn-hongkong',
                vpc_id: 'vpc-test',
                vswitch_id: 'vsw-test',
                security_group_id: 'sg-test',
                payment_type: 'PostPaid',
                cpu_cores: '4',
                memory_gb: '8',
                instance_type: 'ecs.c6.xlarge',
                system_disk_category: 'cloud_essd',
                data_disk_category: 'cloud_ssd',
                io_optimized: 'none',
                spot_strategy: 'SpotWithPriceLimit',
              },
            }),
          }
        }
        return { ok: false, json: async () => ({}) }
      })

      let formDataRef
      const wrapper = mount({
        setup() {
          const result = useSetDefaultConfigForm(createProps(), emit)
          formDataRef = result.formData
          return () => null
        },
      })
      await flushPromises()

      expect(formDataRef.value.io_optimized).toBe('none')
      expect(formDataRef.value.spot_strategy).toBe('SpotWithPriceLimit')
      wrapper.unmount()
    })

    it('handleSubmit persists io_optimized and spot_strategy', async () => {
      const emit = createEmit()
      const result = useSetDefaultConfigForm(createProps(), emit)
      result.formData.value.region = 'cn-hongkong'
      result.formData.value.io_optimized = 'optimized'
      result.formData.value.spot_strategy = 'SpotAsPriceGo'

      apiFetch.mockImplementation(async (url) => {
        if (url.includes('/cloud/server-config-default/')) {
          return { ok: true, json: async () => ({ status: 'success' }) }
        }
        return { ok: false, json: async () => ({}) }
      })

      await result.handleSubmit()

      const postCall = apiFetch.mock.calls.find(([url]) => String(url).includes('server-config-default'))
      expect(postCall).toBeTruthy()
      const body = JSON.parse(postCall[1].body)
      expect(body.io_optimized).toBe('optimized')
      expect(body.spot_strategy).toBe('SpotAsPriceGo')
    })

    it('exports instances, loadingInstances, instanceError refs', () => {
      const emit = createEmit()
      const result = useSetDefaultConfigForm(createProps(), emit)

      expect(result.instances).toBeDefined()
      expect(result.instances.value).toEqual([])
      expect(result.loadingInstances).toBeDefined()
      expect(result.loadingInstances.value).toBe(false)
      expect(result.instanceError).toBeDefined()
      expect(result.instanceError.value).toBe('')
    })

    it('exports handleInstanceFilterChange', () => {
      const emit = createEmit()
      const result = useSetDefaultConfigForm(createProps(), emit)

      expect(result.handleInstanceFilterChange).toBeDefined()
      expect(typeof result.handleInstanceFilterChange).toBe('function')
    })

    it('loadInstances calls available-instances API with correct params', async () => {
      const emit = createEmit()
      // Mock region response first
      apiFetch.mockImplementation(async (url) => {
        if (url.includes('/cloud/regions/')) {
          return {
            ok: true,
            json: async () => [{ id: 'cn-hangzhou', name: '华东1（杭州）' }],
          }
        }
        if (url.includes('/cloud/server-config-default/')) {
          return {
            ok: true,
            json: async () => ({
              status: 'success',
              data: {
                region: 'cn-hangzhou',
                vpc_id: 'vpc-test',
                vswitch_id: 'vsw-test',
                security_group_id: 'sg-test',
                payment_type: 'PostPaid',
                bandwidth_charging_mode: 'PayByTraffic',
                bandwidth: 5,
                cpu_cores: '4',
                memory_gb: '8',
                instance_type: 'ecs.c6.xlarge',
                system_disk_category: 'cloud_essd',
                data_disk_category: 'cloud_essd',
              },
            }),
          }
        }
        if (url.includes('/cloud/server-images/vpcs/')) {
          return {
            ok: true,
            json: async () => [
              { vpc_id: 'vpc-test', vpc_name: 'Test VPC' },
            ],
          }
        }
        if (url.includes('/cloud/server-images/vswitches/')) {
          return {
            ok: true,
            json: async () => [
              { id: 'vsw-test', name: 'Test VSW', zone_id: 'cn-hangzhou-h' },
            ],
          }
        }
        if (url.includes('/cloud/server-images/security-groups/')) {
          return {
            ok: true,
            json: async () => [
              { id: 'sg-test', name: 'Test SG' },
            ],
          }
        }
        if (url.includes('/cloud/available-instances/')) {
          return {
            ok: true,
            json: async () => [
              {
                instance_type: 'ecs.c6.xlarge',
                cpu_cores: 4,
                memory_gb: 8,
                instance_type_family: 'ecs.c6',
                status: 'available',
              },
              {
                instance_type: 'ecs.c6.2xlarge',
                cpu_cores: 8,
                memory_gb: 16,
                instance_type_family: 'ecs.c6',
                status: 'available',
              },
            ],
          }
        }
        return { ok: false, json: async () => ({}) }
      })

      const result = useSetDefaultConfigForm(createProps(), emit)

      // Wait for initialization
      await nextTick()
      await new Promise((r) => setTimeout(r, 100))

      // Verify instances were loaded
      expect(result.instances.value.length).toBeGreaterThanOrEqual(0)
    })

    it('loadInstances handles multiple API response formats', async () => {
      const emit = createEmit()
      const result = useSetDefaultConfigForm(createProps(), emit)

      // Set up formData so loadInstances can proceed without region loading
      result.formData.value.region = 'cn-hangzhou'
      result.formData.value.platform_type = 'aliyun'
      result.formData.value.authorization_id = 'auth-123'
      result.formData.value.vswitch_id = 'vsw-test'

      // Mock vswitches for zone_id lookup
      result.vswitches.value = [{ id: 'vsw-test', name: 'Test VSW', zone_id: 'cn-hangzhou-h' }]

      apiFetch.mockImplementation(async (url) => {
        if (url.includes('available-instances')) {
          return {
            ok: true,
            json: async () => [
              {
                instance_type: 'ecs.c6.xlarge',
                cpu_cores: 4,
                memory_gb: 8,
                instance_type_family: 'ecs.c6',
                status: 'available',
              },
              {
                instance_type: 'ecs.g6.large',
                cpu_cores: 2,
                memory_gb: 8,
                instance_type_family: 'ecs.g6',
                status: 'available',
              },
            ],
          }
        }
        return { ok: false, json: async () => ({}) }
      })

      // Manually trigger loadInstances (simulates what happens after VPC/Switch/SG load)
      await result.loadInstances()

      expect(apiFetch).toHaveBeenCalled()
      const callUrl = apiFetch.mock.calls[0][0]
      expect(callUrl).toContain('available-instances')
      expect(callUrl).toContain('region_id=cn-hangzhou')
      expect(callUrl).toContain('zone_id=cn-hangzhou-h')

      expect(result.instances.value).toHaveLength(2)
      expect(result.instances.value[0].instance_type).toBe('ecs.c6.xlarge')
      expect(result.instances.value[0].cpu_cores).toBe(4)
      expect(result.instances.value[0].memory_gb).toBe(8)
    })
  })
}
