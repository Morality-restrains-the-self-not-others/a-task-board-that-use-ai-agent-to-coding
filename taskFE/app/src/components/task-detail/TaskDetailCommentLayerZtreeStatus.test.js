// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentLayerZtreeStatus.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailCommentLayerZtreeStatus } = await import(
    './TaskDetailCommentLayerZtreeStatus.vue'
  )

  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 't1' } }),
    useRouter: () => ({ push: vi.fn() }),
  }))

  function mountStatus(overrides = {}) {
    return mount(TaskDetailCommentLayerZtreeStatus, {
      props: {
        showLoading: true,
        showReleased: false,
        loadingHint: '引导克隆失败：仓库 Git 授权未齐。',
        loadingIsError: true,
        loadingErrorTraceId: '',
        ...overrides,
      },
    })
  }

  describe('TaskDetailCommentLayerZtreeStatus bootstrap error trace', () => {
    it('挂载 data-traceId 到错误文案节点', () => {
      const wrapper = mountStatus({ loadingErrorTraceId: 'boot-sse-trace-1' })
      const err = wrapper.get('[data-testid="comment-layer-ztree-loading-error"]')
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe(
        'boot-sse-trace-1',
      )
    })

    it('无 trace_id 时不挂 data-traceId', () => {
      const wrapper = mountStatus({ loadingErrorTraceId: '' })
      const err = wrapper.get('[data-testid="comment-layer-ztree-loading-error"]')
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBeUndefined()
    })

    it('非错误加载态不挂 data-traceId', () => {
      const wrapper = mountStatus({
        loadingIsError: false,
        loadingHint: '正在等待可写层…',
        loadingErrorTraceId: 'boot-sse-trace-1',
      })
      expect(wrapper.find('[data-testid="comment-layer-ztree-loading-error"]').exists()).toBe(false)
      const p = wrapper.get('[data-testid="comment-layer-ztree-loading"]').find('p.mt-2')
      expect(p.attributes('data-traceid') || p.attributes('data-traceId')).toBeUndefined()
    })
  })
}
