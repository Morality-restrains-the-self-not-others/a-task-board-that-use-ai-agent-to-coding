// @vitest-environment jsdom
/**
 * 回归测试：OPT-20260821-005 — 快速切换镜像时 loadRegions 不得复用旧镜像请求 / 套用旧地域。
 * 根因：loadRegions 用单飞 loadRegionsPromise 合并并发，但未按镜像区分——换镜像时旧请求
 * 的 in-flight promise 被直接返回，旧镜像地域列表落到新镜像选择上，且可用实例不再重取。
 */
if (!process.env.VITEST) {
  console.log('[skip] useServerConfigHardwarePanel.loadRegionsGeneration.unit.test.js requires vitest runtime')
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

  function deferred() {
    let resolve
    const promise = new Promise((res) => { resolve = res })
    return { promise, resolve }
  }

  function setupPanel() {
    const selectedImageId = ref('')
    const panel = useServerConfigHardwarePanel({}, vi.fn(), selectedImageId)
    panel.cloudPlatforms.value = [
      { id: 'plat-1', authorization_id: 'auth-123', platform_type: 'aliyun' },
    ]
    panel.selectedCloudPlatform.value = 'plat-1'
    panel.ensureDefaultConfigForPlatform('plat-1')
    return { panel, selectedImageId }
  }

  function defaultMock() {
    hoisted.apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/api/cloud/server-images/')) {
        return Promise.resolve({ ok: true, json: async () => [] })
      }
      if (String(url).includes('/available-instances/')) {
        return Promise.resolve({ ok: true, json: async () => ({ status: 'success', instance_types: [] }) })
      }
      return Promise.resolve({ ok: false, status: 500, json: async () => ({}) })
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
    defaultMock()
  })

  it('快速切换镜像时旧 loadRegions 结果被丢弃，新镜像地域生效', async () => {
    const deferredA = deferred()
    const deferredB = deferred()
    const installedCalls = []
    hoisted.apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/api/cloud/installed-images/IMG_A/regions/')) {
        installedCalls.push('IMG_A')
        return deferredA.promise
      }
      if (String(url).includes('/api/cloud/installed-images/IMG_B/regions/')) {
        installedCalls.push('IMG_B')
        return deferredB.promise
      }
      if (String(url).includes('/api/cloud/server-images/')) {
        return Promise.resolve({ ok: true, json: async () => [] })
      }
      if (String(url).includes('/available-instances/')) {
        return Promise.resolve({ ok: true, json: async () => ({ status: 'success', instance_types: [] }) })
      }
      return Promise.resolve({ ok: false, status: 500, json: async () => ({}) })
    })

    const { panel, selectedImageId } = setupPanel()
    selectedImageId.value = 'IMG_A'
    const p1 = panel.loadRegions()
    selectedImageId.value = 'IMG_B'
    const p2 = panel.loadRegions()

    // 两个镜像请求都已发出：B 未复用 A 的 in-flight promise
    expect(installedCalls).toEqual(['IMG_A', 'IMG_B'])

    // A 后到：代际已过期，不得污染 regions
    deferredA.resolve({ ok: true, json: async () => [{ id: 'region-a', name: 'Region A' }] })
    await p1
    expect(panel.regions.value.map((r) => r.region_id)).not.toContain('region-a')

    // B 到达：写入新镜像地域
    deferredB.resolve({ ok: true, json: async () => [{ id: 'region-b', name: 'Region B' }] })
    await p2
    const regionIds = panel.regions.value.map((r) => r.region_id)
    expect(regionIds).toContain('region-b')
    expect(regionIds).not.toContain('region-a')
  })

  it('同镜像并发 loadRegions 仍合并为单次请求（单飞不破坏）', async () => {
    const deferredA = deferred()
    const installedCalls = []
    hoisted.apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/api/cloud/installed-images/IMG_A/regions/')) {
        installedCalls.push('IMG_A')
        return deferredA.promise
      }
      if (String(url).includes('/api/cloud/server-images/')) {
        return Promise.resolve({ ok: true, json: async () => [] })
      }
      if (String(url).includes('/available-instances/')) {
        return Promise.resolve({ ok: true, json: async () => ({ status: 'success', instance_types: [] }) })
      }
      return Promise.resolve({ ok: false, status: 500, json: async () => ({}) })
    })

    const { panel, selectedImageId } = setupPanel()
    selectedImageId.value = 'IMG_A'
    const p1 = panel.loadRegions()
    const p2 = panel.loadRegions()
    expect(installedCalls).toEqual(['IMG_A'])
    // async 包装使两个返回值不是同一对象，但内部单飞 promise 只发一次请求即可证明
    deferredA.resolve({ ok: true, json: async () => [{ id: 'region-a', name: 'Region A' }] })
    await Promise.all([p1, p2])
    expect(panel.regions.value.map((r) => r.region_id)).toContain('region-a')
  })

  it('换镜像后可用实例以新镜像重新拉取', async () => {
    const deferredB = deferred()
    let instancesCalls = 0
    hoisted.apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/api/cloud/installed-images/IMG_B/regions/')) {
        return deferredB.promise
      }
      if (String(url).includes('/api/cloud/server-images/')) {
        return Promise.resolve({ ok: true, json: async () => [] })
      }
      if (String(url).includes('/available-instances/')) {
        instancesCalls += 1
        return Promise.resolve({ ok: true, json: async () => ({ status: 'success', instance_types: [] }) })
      }
      return Promise.resolve({ ok: false, status: 500, json: async () => ({}) })
    })

    const { panel, selectedImageId } = setupPanel()
    panel.hardwareBootstrapDone.value = true
    selectedImageId.value = 'IMG_A'
    await panel.loadRegions() // IMG_A 无镜像地域，兜底 500 → regions 空，不触发实例拉取
    expect(instancesCalls).toBe(0)

    panel.selectedZone.value = 'cn-chengdu-b' // 可用实例拉取需要 zone（hasRequiredAvailableInstancesContext）
    selectedImageId.value = 'IMG_B'
    deferredB.resolve({ ok: true, json: async () => [{ id: 'region-b', name: 'Region B' }] })
    await new Promise((resolve) => setTimeout(resolve, 0)) // 让 watch → loadRegions → schedule 微任务链完成

    expect(panel.regions.value.map((r) => r.region_id)).toContain('region-b')
    expect(instancesCalls).toBeGreaterThan(0)
  })
}
