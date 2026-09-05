// @vitest-environment jsdom
/**
 * 回归测试：OPT-20260807-031 — e105bad 迁移把 server-images 子路由误写为字面量 $1，
 * 导致 VPC/交换机/安全组列表请求全部 404（{"error":"server-images sub-route not found","path":"$1"}）。
 * 后端约定：/api/cloud/server-images/{sub}/tenant_id/{tid}/，sub ∈ vpcs|vswitches|security-groups|...
 */
if (!process.env.VITEST) {
  console.log('[skip] useServerConfigHardwarePanel.serverImagesUrls.unit.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: {
      params: { tenant: 'TENANT1' },
      query: {},
    },
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: hoisted.apiFetchMock,
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.routeMock,
  }))

  // 必须在 mock 之后导入
  const { useServerConfigHardwarePanel } = await import('./useServerConfigHardwarePanel.js')

  function setupPanel() {
    const panel = useServerConfigHardwarePanel({}, vi.fn(), ref(''))
    panel.cloudPlatforms.value = [
      { id: 'plat-1', authorization_id: 'auth-123', platform_type: 'aliyun' },
    ]
    panel.selectedCloudPlatform.value = 'plat-1'
    panel.ensureDefaultConfigForPlatform('plat-1')
    panel.selectedRegion.value = 'cn-chengdu'
    return panel
  }

  function serverImagesUrls() {
    return hoisted.apiFetchMock.mock.calls
      .map((call) => String(call[0]))
      .filter((url) => url.includes('/api/cloud/server-images/'))
  }

  beforeEach(() => {
    vi.clearAllMocks()
    // 默认拒绝所有请求；仅 server-images 返回空数组，隔离无关 watcher 触发的请求
    hoisted.apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/api/cloud/server-images/')) {
        return Promise.resolve({ ok: true, json: async () => [] })
      }
      return Promise.resolve({ ok: false, status: 500, json: async () => ({}) })
    })
  })

  it('VPC 列表请求使用 vpcs 子路由而非 $1', async () => {
    const panel = setupPanel()
    await panel.loadVpcs()

    const urls = serverImagesUrls()
    expect(urls.length).toBeGreaterThan(0)
    expect(urls.some((u) => u.includes('/api/cloud/server-images/vpcs/tenant_id/TENANT1/'))).toBe(true)
    expect(urls.some((u) => u.includes('region_id=cn-chengdu') && u.includes('authorization_id=auth-123'))).toBe(true)
  })

  it('交换机与安全组列表请求使用 vswitches / security-groups 子路由而非 $1', async () => {
    const panel = setupPanel()
    await panel.loadVpcs() // 会连带拉一次安全组（自动创建 VPC 分支）
    panel.selectedVpc.value = 'vpc-999'
    await panel.loadVswitches()

    const urls = serverImagesUrls()
    expect(urls.some((u) => u.includes('/api/cloud/server-images/vswitches/tenant_id/TENANT1/'))).toBe(true)
    expect(urls.some((u) => u.includes('/api/cloud/server-images/security-groups/tenant_id/TENANT1/'))).toBe(true)
    expect(urls.some((u) => u.includes('vpc_id=vpc-999'))).toBe(true)
  })

  it('任何 server-images 请求都不再携带字面量 $1（根因回归门禁）', async () => {
    const panel = setupPanel()
    await panel.loadVpcs()
    panel.selectedVpc.value = 'vpc-999'
    await panel.loadVswitches()

    const urls = serverImagesUrls()
    expect(urls.length).toBeGreaterThan(0)
    for (const url of urls) {
      expect(url).not.toContain('$1')
      expect(url).toMatch(/^\/api\/cloud\/server-images\/(vpcs|vswitches|security-groups)\/tenant_id\//)
    }
  })

  function availableInstancesUrls() {
    return hoisted.apiFetchMock.mock.calls
      .map((call) => String(call[0]))
      .filter((url) => url.includes('/available-instances/'))
  }

  it('fetchAvailableInstances 空 data_disk_category（不要数据盘）仍拉取且不传 DataDiskCategory（OPT-20260812-001）', async () => {
    const panel = setupPanel()
    panel.selectedZone.value = 'cn-chengdu-b'
    panel.filterOptions.value.data_disk_category = ''
    hoisted.apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/available-instances/')) {
        return Promise.resolve({ ok: true, json: async () => ({ status: 'success', instance_types: [] }) })
      }
      return Promise.resolve({ ok: false, status: 500, json: async () => ({}) })
    })
    await panel.fetchAvailableInstances()
    const urls = availableInstancesUrls()
    expect(urls.length).toBeGreaterThan(0)
    expect(urls[0]).not.toContain('DataDiskCategory')
    expect(urls[0]).toContain('SystemDiskCategory=cloud_essd')
  })

  it('fetchAvailableInstances data_disk_category 指定时传 DataDiskCategory（OPT-20260812-001）', async () => {
    const panel = setupPanel()
    panel.selectedZone.value = 'cn-chengdu-b'
    panel.filterOptions.value.data_disk_category = 'cloud_ssd'
    hoisted.apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/available-instances/')) {
        return Promise.resolve({ ok: true, json: async () => ({ status: 'success', instance_types: [] }) })
      }
      return Promise.resolve({ ok: false, status: 500, json: async () => ({}) })
    })
    await panel.fetchAvailableInstances()
    const urls = availableInstancesUrls()
    expect(urls.length).toBeGreaterThan(0)
    expect(urls[0]).toContain('DataDiskCategory=cloud_ssd')
  })
}
