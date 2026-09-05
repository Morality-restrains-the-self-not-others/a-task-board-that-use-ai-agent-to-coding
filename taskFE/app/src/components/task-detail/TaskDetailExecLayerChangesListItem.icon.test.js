// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] TaskDetailExecLayerChangesListItem.icon.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailExecLayerChangesListItem } = await import('./TaskDetailExecLayerChangesListItem.vue')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

describe('TaskDetailExecLayerChangesListItem file type icon', () => {
  it('T7: 文件名前有 file-type-icon', () => {
    const wrapper = mount(TaskDetailExecLayerChangesListItem, {
      props: {
        change: { path: 'ram-work/jre/lib/modules', kind: 'modified' },
        layerId: 'L1',
        tenantId: 't1',
        workspaceId: 'w1',
        taskId: 'task1',
      },
    })
    const icon = wrapper.find('[data-testid="file-type-icon"]')
    expect(icon.exists()).toBe(true)
    expect(icon.attributes('data-file-type')).toBe('binary')
    wrapper.unmount()
  })
})
}
