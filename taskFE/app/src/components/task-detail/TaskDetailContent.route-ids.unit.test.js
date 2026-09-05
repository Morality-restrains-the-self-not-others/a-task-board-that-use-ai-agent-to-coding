// @vitest-environment jsdom
/**
 * TaskDetailContent.logic 须经 resolveTaskRouteIds：work-panel 弹窗仅有 tenant 路由时，
 * 用 props 补齐 workspace/taskId 再下传给 TaskDetailView。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailContent.route-ids.unit.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    routeMock: {
      params: { tenant: '850256677331562496' },
      query: {},
    },
    received: { tenantId: '', workspaceId: '', taskId: '' },
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
    useRouter: () => ({ push: vi.fn() }),
  }))

  vi.mock('../../views/TaskDetail.vue', () => ({
    default: {
      name: 'TaskDetailViewStub',
      props: ['task', 'tenantId', 'workspaceId', 'taskId'],
      template: '<div data-testid="task-detail-view-stub" />',
      setup(props) {
        hoistedMocks.received.tenantId = String(props.tenantId || '')
        hoistedMocks.received.workspaceId = String(props.workspaceId || '')
        hoistedMocks.received.taskId = String(props.taskId || '')
      },
    },
  }))

  const { default: TaskDetailContent } = await import('../TaskDetailContent.logic.vue')

  describe('TaskDetailContent.logic resolved route ids', () => {
    it('work-panel 形态：route 仅 tenant 时用 props 下传完整 IDs', () => {
      hoistedMocks.routeMock.params = { tenant: '850256677331562496' }
      mount(TaskDetailContent, {
        props: {
          tenantId: '850256677331562496',
          workspaceId: '861623708318031872',
          taskId: 'task_99',
          task: { id: 'task_99', title: 't' },
        },
      })
      expect(hoistedMocks.received).toEqual({
        tenantId: '850256677331562496',
        workspaceId: '861623708318031872',
        taskId: 'task_99',
      })
    })

    it('无 props 时从完整 task-detail 路由回退', () => {
      hoistedMocks.routeMock.params = {
        tenant: 't1',
        workspace: 'w1',
        taskId: 'task_a',
      }
      mount(TaskDetailContent, {
        props: {
          task: { id: 'task_a' },
        },
      })
      expect(hoistedMocks.received).toEqual({
        tenantId: 't1',
        workspaceId: 'w1',
        taskId: 'task_a',
      })
    })
  })
}
