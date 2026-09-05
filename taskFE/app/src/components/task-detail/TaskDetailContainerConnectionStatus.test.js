// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] TaskDetailContainerConnectionStatus.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailContainerConnectionStatus } = await import('./TaskDetailContainerConnectionStatus.vue')

describe('TaskDetailContainerConnectionStatus', () => {
  it('renders 容器连接状态 when connected', () => {
    const wrapper = mount(TaskDetailContainerConnectionStatus, {
      props: {
        containerHeartbeatStatus: 'connected',
        containerHeartbeatLastSuccess: new Date('2026-07-22T10:45:16Z'),
        containerHeartbeatAttempts: 6,
        containerHeartbeatError: '',
        containerHeartbeatSeqInfo: {
          containerSeq: 6,
          containerAck: 5,
          saasSeq: 6,
          saasAck: 6,
          uplinkOk: true,
          downlinkOk: true,
          probeOk: true,
          bidirectionalOk: true,
        },
        containerHeartbeatLogLines: [],
      },
    })
    expect(wrapper.get('[data-testid="container-connection-status"]').text()).toContain('容器连接状态')
    expect(wrapper.text()).toContain('双向已连接')
    expect(wrapper.text()).toContain('双向确认')
  })

  it('hides when idle and no logs unless forceVisible', () => {
    const idle = mount(TaskDetailContainerConnectionStatus, {
      props: {
        containerHeartbeatStatus: 'idle',
        containerHeartbeatAttempts: 0,
        containerHeartbeatSeqInfo: {},
        containerHeartbeatLogLines: [],
      },
    })
    expect(idle.find('[data-testid="container-connection-status"]').exists()).toBe(false)

    const forced = mount(TaskDetailContainerConnectionStatus, {
      props: {
        containerHeartbeatStatus: 'idle',
        containerHeartbeatAttempts: 0,
        containerHeartbeatSeqInfo: {},
        containerHeartbeatLogLines: [],
        forceVisible: true,
      },
    })
    expect(forced.find('[data-testid="container-connection-status"]').exists()).toBe(true)
  })

  it('shows 等待容器登记 when connecting with registration hint', () => {
    const wrapper = mount(TaskDetailContainerConnectionStatus, {
      props: {
        containerHeartbeatStatus: 'connecting',
        containerHeartbeatAttempts: 0,
        containerHeartbeatError: '服务器已启动，等待容器镜像初始化并登记业务地址…',
        containerHeartbeatSeqInfo: {},
        containerHeartbeatLogLines: [],
        forceVisible: true,
      },
    })
    expect(wrapper.text()).toContain('等待容器登记')
    expect(wrapper.text()).toContain('登记业务地址')
  })
})
}
