// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] ServerConfigHardwarePanel.mount.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { createRouter, createMemoryHistory } = await import('vue-router')
  const { default: HardwarePanel } = await import('./ServerConfigHardwarePanel.vue')

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(async () => ({
    ok: true,
    json: async () => ({ status: 'success', platforms: [], records: [], total: 0 }),
    headers: { get: () => null },
  })),
}))
vi.mock('../utils/workspaceCloudPlatformsApi.js', () => ({
  fetchWorkspaceCloudPlatforms: vi.fn(async () => []),
}))
vi.mock('../utils/availableInstancesFetchCoordinator.js', () => ({
  createAvailableInstancesFetchScheduler: () => ({
    schedule: vi.fn(),
    cancel: vi.fn(),
    run: vi.fn(),
  }),
}))
vi.mock('../utils/publicClientIp.js', () => ({
  fetchPublicClientIp: vi.fn(async () => '1.2.3.4'),
}))


describe('ServerConfigHardwarePanel mount', () => {
  it('mounts and shows auto release settings', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/tenant/:tenant/workspace/:workspaceId/task-detail/:taskId/', component: { template: '<div/>' } }],
    })
    await router.push('/tenant/t1/workspace/w1/task-detail/task_1/')
    await router.isReady()
    const errs = []
    const wrapper = mount(HardwarePanel, {
      props: {
        task: { id: 'task_1' },
        workspaceId: 'w1',
        installedImages: [{ id: 'img1', name: 'trae' }],
        selectedImageId: 'img1',
      },
      global: {
        plugins: [router],
        config: {
          errorHandler: (e, _i, info) => errs.push(`${e} :: ${info}`),
        },
      },
    })
    await flushPromises()
    await nextTick()
    expect(errs, JSON.stringify(errs)).toEqual([])
    expect(wrapper.find('[data-testid=server-hardware-config-panel]').exists()).toBe(true)
    expect(wrapper.text()).toMatch(/自动释放/)
  })
})
}
