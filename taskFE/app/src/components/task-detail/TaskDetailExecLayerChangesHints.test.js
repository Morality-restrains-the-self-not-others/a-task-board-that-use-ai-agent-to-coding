// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] TaskDetailExecLayerChangesHints.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailExecLayerChangesHints } = await import('./TaskDetailExecLayerChangesHints.vue')

describe('TaskDetailExecLayerChangesHints', () => {
  it('shows scroll load-more hint when hasMore', () => {
    const wrapper = mount(TaskDetailExecLayerChangesHints, {
      props: {
        hasChanges: true,
        hasMore: true,
        truncated: true,
      },
    })
    const el = wrapper.find('[data-testid="task-detail-layer-changes-truncated-hint"]')
    expect(el.exists()).toBe(true)
    expect(el.text()).toContain('向下滚动可加载更多')
  })

  it('shows scan-cap hint when truncated and no more pages', () => {
    const wrapper = mount(TaskDetailExecLayerChangesHints, {
      props: {
        hasChanges: true,
        hasMore: false,
        truncated: true,
      },
    })
    const el = wrapper.find('[data-testid="task-detail-layer-changes-truncated-hint"]')
    expect(el.exists()).toBe(true)
    expect(el.text()).toContain('目录扫描已达上限')
  })

  it('挂载 data-traceId when truncatedTraceId provided', () => {
    const wrapper = mount(TaskDetailExecLayerChangesHints, {
      props: {
        hasChanges: true,
        hasMore: false,
        truncated: true,
        truncatedTraceId: 'tid-scan-cap-001',
      },
    })
    const el = wrapper.find('[data-testid="task-detail-layer-changes-truncated-hint"]')
    // jsdom 会将 HTML 属性名小写化；选择器 [data-traceId] 在浏览器中仍可用
    expect(el.attributes('data-traceid') || el.attributes('data-traceId')).toBe('tid-scan-cap-001')
  })

  it('omits data-traceId when truncatedTraceId empty', () => {
    const wrapper = mount(TaskDetailExecLayerChangesHints, {
      props: {
        hasChanges: true,
        hasMore: false,
        truncated: true,
        truncatedTraceId: '',
      },
    })
    const el = wrapper.find('[data-testid="task-detail-layer-changes-truncated-hint"]')
    expect(el.attributes('data-traceid') || el.attributes('data-traceId')).toBeUndefined()
  })
})
}
