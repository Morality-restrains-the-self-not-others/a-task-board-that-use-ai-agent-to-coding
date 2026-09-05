// @vitest-environment jsdom
/**
 * 回归：previous-server-config 不得只靠 props.task.id。
 * 任务详情 URL 已有 taskId、task 对象尚未带 id 时仍应请求。
 */
if (!process.env.VITEST) {
  console.log('[skip] useServerConfigHardwarePanel.taskId.unit.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: {
      params: {
        tenant: 'TENANT1',
        workspaceId: 'w1',
        taskId: 'task_881388002226499584',
      },
      query: {},
    },
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: hoisted.apiFetchMock,
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.routeMock,
  }))

  const { useServerConfigHardwarePanel } = await import('./useServerConfigHardwarePanel.js')

  beforeEach(() => {
    vi.clearAllMocks()
    hoisted.routeMock.params = {
      tenant: 'TENANT1',
      workspaceId: 'w1',
      taskId: 'task_881388002226499584',
    }
    hoisted.apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ status: 'success', server_config: null }),
    })
  })

  function previousConfigUrls() {
    return hoisted.apiFetchMock.mock.calls
      .map((call) => String(call[0]))
      .filter((url) => url.includes('previous-server-config'))
  }

  it('task 无 id 时 previous-server-config 使用 route.params.taskId', async () => {
    const panel = useServerConfigHardwarePanel(
      { task: {}, workspaceId: 'w1' },
      vi.fn(),
      ref(''),
    )
    await panel.fetchPreviousServerConfig()
    const urls = previousConfigUrls()
    expect(urls.length).toBeGreaterThan(0)
    expect(urls.some((u) => u.includes('task_id=task_881388002226499584'))).toBe(true)
  })

  it('task 为 null 时仍用 route.params.taskId 请求，不再因缺少 task 对象而跳过', async () => {
    const panel = useServerConfigHardwarePanel(
      { task: null, workspaceId: 'w1' },
      vi.fn(),
      ref(''),
    )
    await panel.fetchPreviousServerConfig()
    const urls = previousConfigUrls()
    expect(urls.length).toBeGreaterThan(0)
    expect(urls.some((u) => u.includes('task_id=task_881388002226499584'))).toBe(true)
  })

  it('无处可解析 taskId 时不请求 previous-server-config', async () => {
    hoisted.routeMock.params = { tenant: 'TENANT1', workspaceId: 'w1' }
    const panel = useServerConfigHardwarePanel(
      { task: null, workspaceId: 'w1', taskId: '' },
      vi.fn(),
      ref(''),
    )
    await panel.fetchPreviousServerConfig()
    expect(previousConfigUrls().length).toBe(0)
  })
}
