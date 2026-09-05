// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CommentExecutionDependencyPicker.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { nextTick } = await import('vue')
const { default: CommentExecutionDependencyPicker } = await import('./CommentExecutionDependencyPicker.vue')

describe('CommentExecutionDependencyPicker', () => {
  it('defaults to wait_previous with all-previous scope (empty depends)', async () => {
    const wrapper = mount(CommentExecutionDependencyPicker, {
      props: {
        executionMode: 'wait_previous',
        'onUpdate:executionMode': (v) => wrapper.setProps({ executionMode: v }),
        dependsOnCommentIds: [],
        'onUpdate:dependsOnCommentIds': (v) => wrapper.setProps({ dependsOnCommentIds: v }),
        predecessorOptions: [
          { id: 'c1', authorLabel: 'A', summary: 'hello' },
        ],
      },
    })
    await nextTick()
    expect(wrapper.props('executionMode')).toBe('wait_previous')
    expect(wrapper.props('dependsOnCommentIds')).toEqual([])
    expect(wrapper.find('[data-testid="comment-dep-wait-options"]').exists()).toBe(true)
  })

  it('switches to independent and clears depends', async () => {
    const wrapper = mount(CommentExecutionDependencyPicker, {
      props: {
        executionMode: 'wait_previous',
        'onUpdate:executionMode': (v) => wrapper.setProps({ executionMode: v }),
        dependsOnCommentIds: ['c1'],
        'onUpdate:dependsOnCommentIds': (v) => wrapper.setProps({ dependsOnCommentIds: v }),
        predecessorOptions: [{ id: 'c1', authorLabel: 'A', summary: 'hello' }],
      },
    })
    await wrapper.get('[data-testid="comment-dep-independent"]').setValue(true)
    await nextTick()
    expect(wrapper.props('executionMode')).toBe('independent')
    expect(wrapper.props('dependsOnCommentIds')).toEqual([])
  })

  it('showQueueToggle 未入队：展示加入队列与未锁定的不等待前序', async () => {
    const wrapper = mount(CommentExecutionDependencyPicker, {
      props: {
        showQueueToggle: true,
        task: { id: 't1', queued_auto_run: false },
        tenantId: 'ten1',
        workspaceId: 'ws1',
        executionMode: 'wait_previous',
        'onUpdate:executionMode': (v) => wrapper.setProps({ executionMode: v }),
        dependsOnCommentIds: [],
        'onUpdate:dependsOnCommentIds': (v) => wrapper.setProps({ dependsOnCommentIds: v }),
      },
      global: {
        stubs: {
          'router-link': {
            props: ['to'],
            template: '<a :href="typeof to === \'string\' ? to : \'\'"><slot /></a>',
          },
        },
      },
    })
    await nextTick()
    expect(wrapper.find('[data-testid="task-queued-auto-run-toggle"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="queued-auto-run-join"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="comment-dep-independent"]').attributes('disabled')).toBeUndefined()
    expect(wrapper.find('[data-testid="comment-dep-queue-serial-hint"]').text()).toContain('一条一条执行')
    wrapper.unmount()
  })

  it('queued_auto_run 时锁定 wait_previous 并禁用不等待前序', async () => {
    let mode = 'independent'
    const wrapper = mount(CommentExecutionDependencyPicker, {
      props: {
        showQueueToggle: true,
        task: { id: 't1', queued_auto_run: true, queued_auto_run_status: 'queued' },
        tenantId: 'ten1',
        workspaceId: 'ws1',
        executionMode: mode,
        'onUpdate:executionMode': (v) => { mode = v },
        dependsOnCommentIds: [],
        'onUpdate:dependsOnCommentIds': (v) => {},
      },
      global: {
        stubs: {
          'router-link': {
            props: ['to'],
            template: '<a :href="typeof to === \'string\' ? to : \'\'"><slot /></a>',
          },
        },
      },
    })
    await nextTick()
    expect(wrapper.find('[data-testid="queued-auto-run-leave"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="comment-dep-independent"]').element.disabled).toBe(true)
    expect(mode).toBe('wait_previous')
    expect(wrapper.find('[data-testid="comment-dep-queue-serial-hint"]').text()).toContain('已加入自动执行队列')
    wrapper.unmount()
  })

  it('task 从出队变为入队时把 independent 切回 wait_previous', async () => {
    const wrapper = mount(CommentExecutionDependencyPicker, {
      props: {
        showQueueToggle: true,
        task: { id: 't1', queued_auto_run: false },
        tenantId: 'ten1',
        workspaceId: 'ws1',
        executionMode: 'independent',
        'onUpdate:executionMode': (v) => wrapper.setProps({ executionMode: v }),
        dependsOnCommentIds: [],
        'onUpdate:dependsOnCommentIds': (v) => wrapper.setProps({ dependsOnCommentIds: v }),
      },
      global: {
        stubs: {
          'router-link': {
            props: ['to'],
            template: '<a :href="typeof to === \'string\' ? to : \'\'"><slot /></a>',
          },
        },
      },
    })
    await nextTick()
    expect(wrapper.props('executionMode')).toBe('independent')
    await wrapper.setProps({ task: { id: 't1', queued_auto_run: true } })
    await nextTick()
    expect(wrapper.props('executionMode')).toBe('wait_previous')
    wrapper.unmount()
  })

  it('selected scope emits chosen predecessor ids', async () => {
    const wrapper = mount(CommentExecutionDependencyPicker, {
      props: {
        executionMode: 'wait_previous',
        'onUpdate:executionMode': (v) => wrapper.setProps({ executionMode: v }),
        dependsOnCommentIds: [],
        'onUpdate:dependsOnCommentIds': (v) => wrapper.setProps({ dependsOnCommentIds: v }),
        predecessorOptions: [
          { id: 'c1', authorLabel: 'A', summary: 'one' },
          { id: 'c2', authorLabel: 'B', summary: 'two' },
        ],
      },
    })
    await wrapper.get('[data-testid="comment-dep-scope-selected"]').setValue(true)
    await nextTick()
    const boxes = wrapper.findAll('[data-testid="comment-dep-predecessor-item"]')
    await boxes[0].setValue(true)
    await nextTick()
    expect(wrapper.props('dependsOnCommentIds')).toEqual(['c1'])
  })
})
}
