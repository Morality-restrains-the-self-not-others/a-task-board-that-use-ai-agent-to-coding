// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentExecutionDetails.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: TaskDetailCommentExecutionDetails } = await import('./TaskDetailCommentExecutionDetails.vue')

describe('TaskDetailCommentExecutionDetails', () => {
  it('shows wait_previous badge by default', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: { commentId: 'C1', dependencyMode: 'wait_previous', isActive: false },
    })
    expect(wrapper.get('[data-testid="comment-execution-dependency-badge"]').text()).toContain('串行')
    expect(wrapper.get('[data-testid="comment-execution-inactive-hint"]').exists()).toBe(true)
  })

  it('wait_previous without effective predecessor shows 串行（无前序）', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'wait_previous',
        hasEffectivePredecessors: false,
        isActive: false,
      },
    })
    const badge = wrapper.get('[data-testid="comment-execution-dependency-badge"]').text()
    expect(badge).toContain('串行（无前序）')
    expect(badge).not.toContain('等待前序完成')
  })

  it('wait_previous with effective predecessor shows 串行（等待前序完成）', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C2',
        dependencyMode: 'wait_previous',
        hasEffectivePredecessors: true,
        isActive: false,
      },
    })
    const badge = wrapper.get('[data-testid="comment-execution-dependency-badge"]').text()
    expect(badge).toContain('串行（等待前序完成）')
    expect(badge).not.toContain('无前序')
  })

  it('active comment shows current badge and slot content', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: { commentId: 'C1', dependencyMode: 'independent', isActive: true, defaultOpen: true },
      slots: { default: '<div data-testid="slot-body">panel</div>' },
    })
    expect(wrapper.get('[data-testid="comment-execution-active-badge"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="comment-execution-dependency-badge"]').text()).toContain('可并行')
    expect(wrapper.get('[data-testid="slot-body"]').text()).toBe('panel')
    expect(wrapper.find('[data-testid="comment-execution-inactive-hint"]').exists()).toBe(false)
  })

  it('posted comments hide mode toggle controls by default', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C9',
        dependencyMode: 'wait_previous',
        isActive: true,
        defaultOpen: true,
      },
    })
    expect(wrapper.find('[data-testid="comment-execution-mode-controls"]').exists()).toBe(false)
  })

  it('emits change-dependency-mode only when canEditMode is explicitly true', async () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C9',
        dependencyMode: 'wait_previous',
        isActive: true,
        canEditMode: true,
        defaultOpen: true,
      },
    })
    await wrapper.get('[data-testid="comment-execution-mode-independent"]').trigger('click')
    const ev = wrapper.emitted('change-dependency-mode')
    expect(ev).toBeTruthy()
    expect(ev[0][0]).toEqual({ commentId: 'C9', executionMode: 'independent' })
  })

  it('queued task never emits independent PATCH from execution details (OPT-20260827-013)', async () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C9',
        dependencyMode: 'wait_previous',
        isActive: true,
        canEditMode: true,
        queuedAutoRun: true,
        defaultOpen: true,
      },
    })
    const btn = wrapper.get('[data-testid="comment-execution-mode-independent"]')
    expect(btn.attributes('disabled')).toBeDefined()
    await btn.trigger('click')
    expect(wrapper.emitted('change-dependency-mode')).toBeUndefined()
    expect(wrapper.emitted('update:dependency-mode')).toBeUndefined()
    expect(wrapper.get('[data-testid="comment-execution-details"]').attributes('data-dependency-mode')).toBe('wait_previous')
  })

  it('queued task shows queue serial hint aligned with composer copy (OPT-20260827-013)', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C9',
        dependencyMode: 'wait_previous',
        isActive: true,
        canEditMode: true,
        queuedAutoRun: true,
        defaultOpen: true,
      },
    })
    const hint = wrapper.get('[data-testid="comment-execution-queue-serial-hint"]')
    expect(hint.text()).toContain('已加入自动执行队列')
    expect(hint.text()).toContain('按提交顺序一条一条执行')
  })

  it('queued task renders serial badge even if comment stored independent (OPT-20260827-013)', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C9',
        dependencyMode: 'independent',
        isActive: true,
        queuedAutoRun: true,
        defaultOpen: true,
      },
    })
    const badge = wrapper.get('[data-testid="comment-execution-dependency-badge"]')
    expect(badge.text()).toContain('串行')
    expect(badge.text()).not.toContain('可并行')
    expect(wrapper.get('[data-testid="comment-execution-details"]').attributes('data-dependency-mode')).toBe('wait_previous')
  })

  it('non-queued task keeps independent badge and emits as before', async () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C9',
        dependencyMode: 'wait_previous',
        isActive: true,
        canEditMode: true,
        queuedAutoRun: false,
        defaultOpen: true,
      },
    })
    expect(wrapper.get('[data-testid="comment-execution-dependency-badge"]').text()).toContain('串行')
    const btn = wrapper.get('[data-testid="comment-execution-mode-independent"]')
    expect(btn.attributes('disabled')).toBeUndefined()
    await btn.trigger('click')
    expect(wrapper.emitted('change-dependency-mode')).toBeTruthy()
    expect(wrapper.emitted('change-dependency-mode')[0][0]).toEqual({ commentId: 'C9', executionMode: 'independent' })
  })

  it('shows binding status badge', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'wait_previous',
        isActive: false,
        bindingStatus: 'waiting_previous',
      },
    })
    expect(wrapper.get('[data-testid="comment-execution-binding-status"]').text()).toContain('等待前序')
  })

  it('shows 服务器已释放 when binding is running but runtime is Released', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'wait_previous',
        isActive: false,
        bindingStatus: 'running',
        serverRuntimeStatus: 'Released',
      },
    })
    const badge = wrapper.get('[data-testid="comment-execution-binding-status"]').text()
    expect(badge).toBe('服务器已释放')
    expect(badge).not.toContain('容器 运行中')
  })

  it('collapses details when server becomes released', async () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'wait_previous',
        isActive: true,
        defaultOpen: true,
        bindingStatus: 'running',
      },
    })
    const details = wrapper.get('[data-testid="comment-execution-details"]')
    expect(details.element.open).toBe(true)
    await wrapper.setProps({ serverRuntimeStatus: 'Released' })
    expect(details.element.open).toBe(false)
    expect(wrapper.get('[data-testid="comment-execution-binding-status"]').text()).toBe('服务器已释放')
  })

  it('does not auto-open details when already released', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'wait_previous',
        isActive: true,
        defaultOpen: true,
        bindingStatus: 'released',
      },
    })
    expect(wrapper.get('[data-testid="comment-execution-details"]').element.open).toBe(false)
  })

  it('keeps 容器 运行中 when runtime is Running', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        bindingStatus: 'running',
        serverRuntimeStatus: 'Running',
      },
    })
    expect(wrapper.get('[data-testid="comment-execution-binding-status"]').text()).toBe('容器 运行中')
  })

  it('shows terminate control when waiting_previous', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C-wait',
        dependencyMode: 'wait_previous',
        bindingStatus: 'waiting_previous',
      },
    })
    expect(wrapper.get('[data-testid="comment-execution-cancel-waiting"]').text()).toContain('终止')
  })

  it('hides terminate control when not waiting_previous', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C-run',
        dependencyMode: 'wait_previous',
        bindingStatus: 'running',
      },
    })
    expect(wrapper.find('[data-testid="comment-execution-cancel-waiting"]').exists()).toBe(false)
  })

  it('puts predecessors in a tab instead of occupying space above the tablist', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C3',
        dependencyMode: 'wait_previous',
        isActive: false,
        bindingStatus: 'waiting_previous',
        defaultOpen: true,
        serverRuntimeStatusTab: true,
        predecessors: [
          {
            id: 'C1',
            summary: '先跑测试套件',
            status: 'running',
            statusLabel: '运行中',
            finished: false,
            missing: false,
          },
        ],
      },
    })
    expect(wrapper.get('[data-testid="comment-execution-binding-status"]').text()).toContain('· 1')
    const tabs = wrapper.get('[data-testid="comment-execution-tablist"]')
    expect(tabs.get('[data-testid="comment-execution-tab-predecessors"]').text()).toContain('前序评论')
    expect(tabs.get('[data-testid="comment-execution-tab-predecessors"]').text()).toContain('1')
    expect(wrapper.find('[data-testid="comment-execution-predecessor-list"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="comment-execution-panel-details"]').exists()).toBe(true)
  })

  it('shows predecessor list after clicking the predecessors tab', async () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C3',
        dependencyMode: 'wait_previous',
        isActive: false,
        bindingStatus: 'waiting_previous',
        defaultOpen: true,
        serverRuntimeStatusTab: true,
        predecessors: [
          {
            id: 'C1',
            summary: '先跑测试套件',
            status: 'running',
            statusLabel: '运行中',
            finished: false,
            missing: false,
          },
        ],
      },
    })
    await wrapper.get('[data-testid="comment-execution-tab-predecessors"]').trigger('click')
    expect(wrapper.find('[data-testid="comment-execution-panel-details"]').exists()).toBe(false)
    const list = wrapper.get('[data-testid="comment-execution-predecessor-list"]')
    expect(list.text()).toContain('先跑测试套件')
    expect(list.text()).toContain('运行中')
  })

  it('hides predecessor list when not waiting and no predecessors', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'independent',
        isActive: true,
        bindingStatus: 'running',
        defaultOpen: true,
        predecessors: [],
      },
    })
    expect(wrapper.find('[data-testid="comment-execution-predecessor-list"]').exists()).toBe(false)
  })

  it('opens details when waiting-previous badge is clicked', async () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C3',
        dependencyMode: 'wait_previous',
        isActive: false,
        bindingStatus: 'waiting_previous',
        defaultOpen: false,
        predecessors: [
          {
            id: 'C1',
            summary: '先跑测试套件',
            status: 'pending',
            statusLabel: '待调度',
            finished: false,
            missing: false,
          },
        ],
      },
    })
    const details = wrapper.get('[data-testid="comment-execution-details"]')
    expect(details.element.open).toBe(false)
    await wrapper.get('[data-testid="comment-execution-binding-status"]').trigger('click')
    expect(details.element.open).toBe(true)
    expect(wrapper.get('[data-testid="comment-execution-tab-predecessors"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="comment-execution-predecessor-list"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="comment-execution-panel-details"]').exists()).toBe(false)
  })

  it('shows per-comment container name on summary', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt99',
        taskId: 'task42',
        dependencyMode: 'independent',
        isActive: true,
        bindingStatus: 'running',
        containerName: 'task_task42_cmt99',
        cscId: 'csc_abc',
        defaultOpen: true,
      },
    })
    expect(wrapper.get('[data-testid="comment-execution-container-name"]').text()).toBe('task_task42_cmt99')
    expect(wrapper.get('[data-testid="comment-execution-container-name-copy"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="comment-execution-container-name-full"]').text()).toBe('task_task42_cmt99')
    expect(wrapper.get('[data-testid="comment-execution-csc-id"]').text()).toBe('csc_abc')
  })

  it('normalizes legacy double task_ prefix from API prop', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt_15666877397783348868',
        taskId: 'task_15666874162351520866',
        dependencyMode: 'independent',
        isActive: true,
        bindingStatus: 'running',
        containerName: 'task_task_15666874162351520866_cmt_15666877397783348868',
        defaultOpen: true,
      },
    })
    expect(wrapper.get('[data-testid="comment-execution-container-name"]').text()).toBe(
      'task_15666874162351520866_cmt_15666877397783348868',
    )
    expect(wrapper.get('[data-testid="comment-execution-container-name-full"]').text()).toBe(
      'task_15666874162351520866_cmt_15666877397783348868',
    )
  })

  it('shows start TraceId inside comment-execution-container-meta when provided', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt99',
        taskId: 'task42',
        dependencyMode: 'independent',
        isActive: true,
        bindingStatus: 'running',
        containerName: 'task_task42_cmt99',
        cscId: 'csc_abc',
        startTraceId: 'web-start-trace-001',
        defaultOpen: true,
      },
    })
    const meta = wrapper.get('[data-testid="comment-execution-container-meta"]')
    const row = meta.get('[data-testid="comment-execution-start-trace-id"]')
    expect(row.text()).toContain('启动 TraceId')
    expect(row.get('[data-testid="comment-execution-start-trace-id-value"]').text()).toBe('web-start-trace-001')
    expect(row.text()).toMatch(/启动 TraceId[：:]/)
    expect(row.get('[data-testid="comment-execution-start-trace-id-copy"]').exists()).toBe(true)
    expect(
      row.attributes('data-traceid') || row.attributes('data-traceId'),
    ).toBe('web-start-trace-001')
  })

  it('omits start TraceId row when startTraceId empty', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt99',
        taskId: 'task42',
        dependencyMode: 'independent',
        isActive: true,
        containerName: 'task_task42_cmt99',
        startTraceId: '',
        defaultOpen: true,
      },
    })
    const meta = wrapper.get('[data-testid="comment-execution-container-meta"]')
    expect(meta.find('[data-testid="comment-execution-start-trace-id"]').exists()).toBe(false)
  })

  it('derives container name as task_{taskId}_{commentId} when prop empty', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cA',
        taskId: 'tB',
        dependencyMode: 'wait_previous',
        isActive: false,
      },
    })
    expect(wrapper.get('[data-testid="comment-execution-container-name"]').text()).toBe('task_tB_cA')
  })

  it('independent starting without CSC explains dedicated allocation', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C2',
        dependencyMode: 'independent',
        isActive: false,
        bindingStatus: 'starting',
        ownsSharedContainer: false,
      },
    })
    const hint = wrapper.get('[data-testid="comment-execution-inactive-hint"]').text()
    expect(hint).toContain('不等待前序')
    expect(hint).toContain('独立 CSC')
    expect(hint).not.toContain('当前执行')
  })

  it('inactive comment with own CSC does not redirect to a singleton current-execution panel', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C2',
        dependencyMode: 'independent',
        isActive: false,
        bindingStatus: 'running',
        ownsSharedContainer: true,
        cscId: 'csc_own',
        defaultOpen: true,
      },
    })
    const hint = wrapper.find('[data-testid="comment-execution-inactive-hint"]')
    if (hint.exists()) {
      expect(hint.text()).not.toContain('当前执行')
      expect(hint.text()).not.toContain('完整连接面板')
      expect(hint.text()).not.toContain('完整容器面板')
    }
  })

  it('enables runtime tabs on inactive comments (panel is not a singleton)', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C-inactive',
        dependencyMode: 'independent',
        isActive: false,
        defaultOpen: true,
        serverRuntimeStatusTab: true,
      },
      slots: {
        'server-runtime-status': '<div data-testid="slot-runtime">runtime-C-inactive</div>',
      },
    })
    expect(wrapper.get('[data-testid="comment-execution-details"]').attributes('data-comment-id')).toBe('C-inactive')
    expect(wrapper.get('[data-testid="comment-execution-tablist"]').exists()).toBe(true)
  })

  it('hides tab bar when serverRuntimeStatusTab is disabled (backward compatible)', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'wait_previous',
        isActive: true,
        defaultOpen: true,
      },
      slots: { default: '<div data-testid="slot-body">panel</div>' },
    })
    expect(wrapper.find('[data-testid="comment-execution-tablist"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="comment-execution-panel-details"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="slot-body"]').text()).toBe('panel')
  })

  it('shows details panel by default when tab mode enabled', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'wait_previous',
        isActive: true,
        defaultOpen: true,
        serverRuntimeStatusTab: true,
      },
      slots: {
        default: '<div data-testid="slot-body">panel</div>',
        'server-runtime-status': '<div data-testid="slot-runtime">runtime panel</div>',
      },
    })
    const tabs = wrapper.get('[data-testid="comment-execution-tablist"]')
    expect(tabs.get('[data-testid="comment-execution-tab-details"]').text()).toContain('执行细节')
    expect(tabs.get('[data-testid="comment-execution-tab-server-runtime"]').text()).toContain('服务器运行状态')
    expect(wrapper.get('[data-testid="comment-execution-panel-details"]').get('[data-testid="slot-body"]').text()).toBe('panel')
    expect(wrapper.find('[data-testid="comment-execution-panel-server-runtime"]').exists()).toBe(false)
  })

  it('switches to server runtime panel on tab click and back', async () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'C1',
        dependencyMode: 'wait_previous',
        isActive: true,
        defaultOpen: true,
        serverRuntimeStatusTab: true,
      },
      slots: {
        default: '<div data-testid="slot-body">panel</div>',
        'server-runtime-status': '<div data-testid="slot-runtime">runtime panel</div>',
      },
    })
    await wrapper.get('[data-testid="comment-execution-tab-server-runtime"]').trigger('click')
    expect(wrapper.find('[data-testid="comment-execution-panel-details"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="comment-execution-panel-server-runtime"]').get('[data-testid="slot-runtime"]').text()).toBe('runtime panel')
    await wrapper.get('[data-testid="comment-execution-tab-details"]').trigger('click')
    expect(wrapper.get('[data-testid="comment-execution-panel-details"]').get('[data-testid="slot-body"]').text()).toBe('panel')
    expect(wrapper.find('[data-testid="comment-execution-panel-server-runtime"]').exists()).toBe(false)
  })

  function mountTabWithContainerMeta() {
    return mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt99',
        taskId: 'task42',
        dependencyMode: 'independent',
        isActive: true,
        bindingStatus: 'running',
        containerName: 'task_task42_cmt99',
        cscId: 'csc_abc',
        startTraceId: 'web-start-trace-001',
        defaultOpen: true,
        serverRuntimeStatusTab: true,
      },
      slots: {
        default: '<div data-testid="slot-body">panel</div>',
        'server-runtime-status': '<div data-testid="slot-runtime">runtime panel</div>',
      },
    })
  }

  function expectContainerMeta(wrapper) {
    const meta = wrapper.get('[data-testid="comment-execution-container-meta"]')
    expect(meta.get('[data-testid="comment-execution-container-name-full"]').text()).toBe('task_task42_cmt99')
    expect(meta.get('[data-testid="comment-execution-csc-id"]').text()).toBe('csc_abc')
    const row = meta.get('[data-testid="comment-execution-start-trace-id"]')
    expect(row.text()).toContain('启动 TraceId')
    expect(row.get('[data-testid="comment-execution-start-trace-id-value"]').text()).toBe('web-start-trace-001')
    expect(
      row.attributes('data-traceid') || row.attributes('data-traceId'),
    ).toBe('web-start-trace-001')
  }

  it('keeps container meta on details tab when tab mode enabled', () => {
    const wrapper = mountTabWithContainerMeta()
    expect(wrapper.get('[data-testid="comment-execution-panel-details"]').exists()).toBe(true)
    expectContainerMeta(wrapper)
  })

  it('shows container name, CSC and start TraceId on server runtime tab', async () => {
    const wrapper = mountTabWithContainerMeta()
    await wrapper.get('[data-testid="comment-execution-tab-server-runtime"]').trigger('click')
    expect(wrapper.get('[data-testid="comment-execution-panel-server-runtime"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="comment-execution-panel-server-runtime"]').get('[data-testid="slot-runtime"]').text()).toBe('runtime panel')
    expectContainerMeta(wrapper)
  })

  it('keeps container meta after switching back to details tab', async () => {
    const wrapper = mountTabWithContainerMeta()
    await wrapper.get('[data-testid="comment-execution-tab-server-runtime"]').trigger('click')
    await wrapper.get('[data-testid="comment-execution-tab-details"]').trigger('click')
    expect(wrapper.get('[data-testid="comment-execution-panel-details"]').exists()).toBe(true)
    expectContainerMeta(wrapper)
  })

  it('shows clone progress bar under start TraceId', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt99',
        taskId: 'task42',
        dependencyMode: 'independent',
        isActive: true,
        bindingStatus: 'running',
        containerName: 'task_task42_cmt99',
        cscId: 'csc_abc',
        startTraceId: 'web-start-trace-001',
        defaultOpen: true,
        cloneProgressRows: [
          { key: 'alpha', label: 'alpha', progress: 66, message: '【项目克隆】(1/1) alpha … 66%', failed: false },
        ],
      },
    })
    const meta = wrapper.get('[data-testid="comment-execution-container-meta"]')
    const trace = meta.get('[data-testid="comment-execution-start-trace-id"]')
    const bar = meta.get('[data-testid="comment-execution-clone-progress"]')
    expect(trace.get('[data-testid="comment-execution-start-trace-id-value"]').text()).toBe('web-start-trace-001')
    expect(bar.get('[data-testid="comment-execution-clone-progress-overall"]').text()).toBe('66%')
    expect(bar.text()).toContain('alpha')
    const html = meta.html()
    expect(html.indexOf('comment-execution-start-trace-id')).toBeLessThan(
      html.indexOf('comment-execution-clone-progress'),
    )
  })

  it('omits clone progress bar when cloneProgressRows empty', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt99',
        taskId: 'task42',
        dependencyMode: 'independent',
        isActive: true,
        containerName: 'task_task42_cmt99',
        startTraceId: 'web-start-trace-001',
        defaultOpen: true,
        cloneProgressRows: [],
      },
    })
    const meta = wrapper.get('[data-testid="comment-execution-container-meta"]')
    expect(meta.find('[data-testid="comment-execution-clone-progress"]').exists()).toBe(false)
  })

  it('keeps clone progress on server runtime tab', async () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt99',
        taskId: 'task42',
        dependencyMode: 'independent',
        isActive: true,
        bindingStatus: 'running',
        containerName: 'task_task42_cmt99',
        cscId: 'csc_abc',
        startTraceId: 'web-start-trace-001',
        defaultOpen: true,
        serverRuntimeStatusTab: true,
        cloneProgressRows: [
          { key: 'alpha', label: 'alpha', progress: 40, message: 'cloning', failed: false },
        ],
      },
      slots: {
        'server-runtime-status': '<div data-testid="slot-runtime">runtime panel</div>',
      },
    })
    await wrapper.get('[data-testid="comment-execution-tab-server-runtime"]').trigger('click')
    expect(wrapper.get('[data-testid="comment-execution-panel-server-runtime"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="comment-execution-clone-progress"]').exists()).toBe(true)
  })

  it('injects commentId into repo-reclone emit', async () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt99',
        taskId: 'task42',
        dependencyMode: 'independent',
        isActive: true,
        defaultOpen: true,
        cloneProgressRows: [
          {
            key: 'https://git.example/alpha.git',
            label: 'alpha',
            progress: 0,
            message: '【项目克隆】(1/1) 失败 alpha: git exit 128',
            failed: true,
            repoUrl: 'https://git.example/alpha.git',
          },
        ],
      },
    })
    await wrapper.get('[data-testid="comment-execution-clone-progress-retry"]').trigger('click')
    expect(wrapper.emitted('repo-reclone')?.[0]?.[0]).toEqual({
      repoUrl: 'https://git.example/alpha.git',
      commentId: 'cmt99',
      comment_id: 'cmt99',
    })
  })

  it('shows Git identity chip from comment repo_identities on the summary', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt_878603172422119424',
        dependencyMode: 'independent',
        isActive: true,
        repoIdentities: [
          {
            repo_url: 'https://gitlab.example/somanyad.git',
            git_identity_id: 'gi_877397592462356480',
          },
        ],
        gitIdentityOptions: [
          {
            id: 'gi_877397592462356480',
            label: '默认身份',
            git_user_name: 'zhenghe',
            git_user_email: 'zhenghe@example.com',
            is_default: true,
          },
        ],
      },
    })
    const chip = wrapper.get('[data-testid="comment-execution-git-identity"]')
    expect(chip.text()).toBe('Git 身份 · 默认身份')
    expect(chip.attributes('title')).toBe('zhenghe <zhenghe@example.com> · 默认身份')
    expect(wrapper.get('[data-testid="comment-execution-details-summary"]').text()).toContain('默认身份')
  })

    it('hides Git identity chip when the comment has no repo_identities', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt_auto',
        dependencyMode: 'wait_previous',
        repoIdentities: [],
        gitIdentityOptions: [{ id: 'gi_1', label: '默认身份' }],
      },
    })
    expect(wrapper.find('[data-testid="comment-execution-git-identity"]').exists()).toBe(false)
  })

  it('shows Git OAuth bound chip on the summary next to Git identity', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt_878902053739458560',
        dependencyMode: 'wait_previous',
        isActive: true,
        repoIdentities: [
          {
            repo_url: 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad.git',
            git_identity_id: 'gi_1',
          },
        ],
        gitIdentityOptions: [{ id: 'gi_1', label: 'system-auto' }],
        oauthReadiness: {
          loading: false,
          hasOAuthRepos: true,
          allBound: true,
          unboundRepoUrls: [],
        },
      },
    })
    const summary = wrapper.get('[data-testid="comment-execution-details-summary"]')
    expect(summary.text()).toContain('Git 身份 · system-auto')
    expect(summary.text()).toContain('Git OAuth · 已绑定')
    expect(wrapper.get('[data-testid="comment-execution-git-oauth"]').attributes('data-kind')).toBe('bound')
    expect(wrapper.find('[data-testid="comment-execution-git-oauth-bind"]').exists()).toBe(false)
  })

  it('shows Git OAuth 网络不可达 chip when probe reports GitLab unreachable', () => {
    const repoUrl = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad.git'
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt_oauth_down',
        dependencyMode: 'wait_previous',
        isActive: true,
        repoIdentities: [{ repo_url: repoUrl, git_identity_id: 'gi_1' }],
        gitIdentityOptions: [{ id: 'gi_1', label: 'system-auto' }],
        oauthReadiness: {
          loading: false,
          hasOAuthRepos: true,
          allBound: true,
          unboundRepoUrls: [],
          unreachableRepoUrls: [repoUrl],
        },
      },
    })
    const chip = wrapper.get('[data-testid="comment-execution-git-oauth"]')
    expect(chip.text()).toBe('Git OAuth · 网络不可达')
    expect(chip.attributes('data-kind')).toBe('unreachable')
    expect(wrapper.find('[data-testid="comment-execution-git-oauth-bind"]').exists()).toBe(false)
  })

  it('shows Git OAuth 无写权限 chip with re-authorize href when last push was permission denied', () => {
    const repoUrl = 'https://github.com/ruandao/helloworld.git'
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt_push_perm',
        dependencyMode: 'wait_previous',
        isActive: true,
        repoIdentities: [{ repo_url: repoUrl, git_identity_id: 'gi_1' }],
        gitIdentityOptions: [{ id: 'gi_1', label: 'system-auto' }],
        oauthReadiness: {
          loading: false,
          hasOAuthRepos: true,
          allBound: true,
          unboundRepoUrls: [],
        },
        lastPushError: 'remote: Permission to ruandao/helloworld.git denied to alice.',
      },
    })
    const chip = wrapper.get('[data-testid="comment-execution-git-oauth"]')
    expect(chip.text()).toBe('Git OAuth · 无写权限')
    expect(chip.attributes('data-kind')).toBe('bound_no_write')
    const bind = wrapper.get('[data-testid="comment-execution-git-oauth-bind"]')
    expect(bind.element.tagName).toBe('A')
    expect(bind.text()).toContain('换账号授权')
    expect(bind.attributes('href')).toContain('github-start-from-gateway')
  })

  it('shows Git OAuth unbound chip with a real bind href on the summary', () => {
    const repoUrl = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad.git'
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt_oauth',
        repoIdentities: [{ repo_url: repoUrl, git_identity_id: 'gi_1' }],
        oauthReadiness: {
          loading: false,
          hasOAuthRepos: true,
          allBound: false,
          unboundRepoUrls: [repoUrl],
        },
      },
    })
    const chip = wrapper.get('[data-testid="comment-execution-git-oauth"]')
    expect(chip.text()).toBe('Git OAuth · 未绑定')
    expect(chip.attributes('data-kind')).toBe('unbound')
    const bind = wrapper.get('[data-testid="comment-execution-git-oauth-bind"]')
    expect(bind.element.tagName).toBe('A')
    expect(bind.attributes('href')).toContain('/api/git-oauth/gitlab-start-from-gateway/')
    expect(bind.attributes('href')).toContain(encodeURIComponent(repoUrl))
    expect(bind.text()).toContain('去绑定')
  })

  it('hides Git OAuth chip when the comment has no oauth-capable repos', () => {
    const wrapper = mount(TaskDetailCommentExecutionDetails, {
      props: {
        commentId: 'cmt_ssh',
        repoIdentities: [{ repo_url: 'git@internal.example:group/repo.git', git_identity_id: 'gi_1' }],
        oauthReadiness: { loading: false, hasOAuthRepos: false, allBound: true, unboundRepoUrls: [] },
      },
    })
    expect(wrapper.find('[data-testid="comment-execution-git-oauth"]').exists()).toBe(false)
  })
})
}
