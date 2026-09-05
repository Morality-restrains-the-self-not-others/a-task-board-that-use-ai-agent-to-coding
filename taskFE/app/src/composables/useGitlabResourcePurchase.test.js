if (!process.env.VITEST) {
  console.log('[skip] useGitlabResourcePurchase.test.js requires vitest runtime')
} else {
const { computed, ref } = await import('vue')
const { beforeEach, describe, expect, it, vi } = await import('vitest')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))
const apiFetch = hoisted.apiFetch

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

const { useGitlabResourcePurchase } = await import('./useGitlabResourcePurchase.js')

describe('useGitlabResourcePurchase', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads regions without default region and estimates cost locally', async () => {
    apiFetch
      .mockResolvedValueOnce({
        ok: true,
        traceId: 't1',
        json: async () => ({
          regions: [{ slug: 'tencent-sh-1', name: '腾讯上海一区' }],
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        traceId: 't1',
        json: async () => ({
          provisioning_status: 'not_purchased',
        }),
      })
    const tenantId = computed(() => '850256677331562496')
    const api = useGitlabResourcePurchase(tenantId)
    await api.load()
    expect(api.region.value).toBe('')
    expect(api.availableRegions.value[0].slug).toBe('tencent-sh-1')
    expect(api.provisioningStatus.value).toBe('not_purchased')
    api.diskUnitPrice.value = 5
    api.trafficUnitPrice.value = 4
    api.form.disk_gb = 10
    api.form.disk_months = 3
    api.form.traffic_prepaid_gb = 5
    expect(api.estimatedCost.value).toBe(170)
  })

  it('load discovers admin-granted gitlab resource without a preselected region', async () => {
    apiFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          regions: [{ slug: 'tencent-sh-1', name: '腾讯上海一区' }],
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          tenant_id: '877397588196749312',
          resources: [
            {
              region: 'tencent-sh-1',
              region_name: '腾讯上海一区',
              disk_gb: 1,
              traffic_prepaid_gb: 0,
              provisioning_status: 'active',
            },
          ],
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          region: 'tencent-sh-1',
          region_name: '腾讯上海一区',
          disk_gb: 1,
          traffic_prepaid_gb: 0,
          disk_used_gb: 0.01049,
          traffic_used_gb: 0,
          traffic_download_allowed: false,
          provisioning_status: 'active',
        }),
      })
    const tenantId = computed(() => '877397588196749312')
    const api = useGitlabResourcePurchase(tenantId)
    await api.load()
    expect(apiFetch.mock.calls[1][0]).toBe(
      '/api/billing/gitlab-resources/tenant_id/877397588196749312/'
    )
    expect(String(apiFetch.mock.calls[1][0])).not.toContain('region=')
    expect(String(apiFetch.mock.calls[2][0])).toContain('region=tencent-sh-1')
    expect(api.region.value).toBe('tencent-sh-1')
    expect(api.regionName.value).toBe('腾讯上海一区')
    expect(api.provisioningStatus.value).toBe('active')
    expect(api.currentDiskGb.value).toBe(1)
    expect(api.diskUsedGb.value).toBe(0.01049)
    expect(api.trafficUsedGb.value).toBe(0)
    expect(api.trafficDownloadAllowed.value).toBe(false)
  })

  it('does not expose wallet purchase POST', () => {
    const tenantId = ref('1')
    const api = useGitlabResourcePurchase(tenantId)
    expect(api.purchase).toBeUndefined()
    expect(Object.keys(api)).not.toContain('purchase')
  })

  it('applyView defaults provisioningStatus to not_purchased for empty data', () => {
    const tenantId = ref('1')
    const api = useGitlabResourcePurchase(tenantId)
    expect(api.provisioningStatus.value).toBe('not_purchased')
  })

  it('applyView reads provisioning_status from response data', () => {
    const tenantId = ref('1')
    const api = useGitlabResourcePurchase(tenantId)
    // Simulate what load() does internally — call applyView with not_purchased data
    api.load = vi.fn() // suppress actual network call
    // Directly verify provisioningStatus reflects the three states
    // not_purchased
    api.applyView({ provisioning_status: 'not_purchased' })
    expect(api.provisioningStatus.value).toBe('not_purchased')
    // pending_admin
    api.applyView({ provisioning_status: 'pending_admin' })
    expect(api.provisioningStatus.value).toBe('pending_admin')
    // active
    api.applyView({ provisioning_status: 'active' })
    expect(api.provisioningStatus.value).toBe('active')
  })

  it('applyView maps disk_used_gb and traffic_used_gb independently', () => {
    const tenantId = ref('1')
    const api = useGitlabResourcePurchase(tenantId)
    api.applyView({
      disk_gb: 1,
      traffic_prepaid_gb: 1,
      disk_used_gb: 0.01049,
      traffic_used_gb: 0,
    })
    expect(api.diskUsedGb.value).toBe(0.01049)
    expect(api.trafficUsedGb.value).toBe(0)
  })

  it('applyView handles missing provisioning_status gracefully', () => {
    const tenantId = ref('1')
    const api = useGitlabResourcePurchase(tenantId)
    api.applyView({ disk_gb: 10 })
    expect(api.provisioningStatus.value).toBe('not_purchased')
    expect(api.estimatedCost.value).toBe(0) // no price data
  })
})
}
