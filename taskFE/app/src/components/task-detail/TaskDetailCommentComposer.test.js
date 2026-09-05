// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentComposer.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch,
  }))

  const { default: TaskDetailCommentComposer } = await import('./TaskDetailCommentComposer.vue')

  describe('TaskDetailCommentComposer', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockResolvedValue({
        ok: false,
        json: async () => ({}),
      })
    })

    it('未开启 at-mode 时渲染与任务详情一致的 textarea 与提交按钮', async () => {
      apiFetch.mockImplementation(async (url) => {
        if (String(url).includes('/workspaces/')) {
          return { ok: true, json: async () => ({ container_image_at_mode_enabled: false }) }
        }
        return { ok: false, json: async () => ({}) }
      })
      const wrapper = mount(TaskDetailCommentComposer, {
        props: {
          modelValue: '',
          'onUpdate:modelValue': () => {},
          tenantId: 't1',
          workspaceId: 'ws1',
        },
      })
      await flushPromises()
      const ta = wrapper.get('textarea')
      expect(ta.attributes('placeholder')).toContain('Ctrl+Enter')
      expect(wrapper.get('[data-testid="task-detail-comment-submit"]').text()).toBe('提交评论')
      expect(wrapper.find('[data-testid="comment-composer-repo-identity"]').exists()).toBe(false)
      const root = wrapper.get('.task-detail-comment-composer')
      const html = root.html()
      expect(html.indexOf('comment-execution-dependency-picker')).toBeGreaterThan(-1)
      expect(html.indexOf('comment-execution-dependency-picker')).toBeLessThan(html.indexOf('<textarea'))
      expect(wrapper.find('[data-testid="task-queued-auto-run-toggle"]').exists()).toBe(false)
    })

    it('有 task/tenant/workspace 时在依赖选择器内展示队列 toggle', async () => {
      const wrapper = mount(TaskDetailCommentComposer, {
        props: {
          modelValue: '',
          'onUpdate:modelValue': () => {},
          tenantId: 't1',
          workspaceId: 'ws1',
          task: { id: 'task_1', queued_auto_run: false },
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
      await flushPromises()
      const picker = wrapper.get('[data-testid="comment-execution-dependency-picker"]')
      expect(picker.find('[data-testid="task-queued-auto-run-toggle"]').exists()).toBe(true)
      expect(picker.find('[data-testid="queued-auto-run-join"]').exists()).toBe(true)
    })

    it('开启 at-mode 时复用 CommentImageMentionEditor', async () => {
      apiFetch.mockImplementation(async (url) => {
        if (String(url).includes('/workspaces/')) {
          return { ok: true, json: async () => ({ container_image_at_mode_enabled: true }) }
        }
        if (String(url).includes('/installed-images/')) {
          return {
            ok: true,
            json: async () => [{ id: 'img1', name: 'node' }],
          }
        }
        return { ok: false, json: async () => ({}) }
      })
      const wrapper = mount(TaskDetailCommentComposer, {
        props: {
          modelValue: '',
          'onUpdate:modelValue': () => {},
          tenantId: 't1',
          workspaceId: 'ws1',
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="comment-image-mention-editor"]').exists()).toBe(true)
      expect(wrapper.find('textarea').exists()).toBe(false)
    })

    it('showCancel 时展示取消按钮', async () => {
      const wrapper = mount(TaskDetailCommentComposer, {
        props: {
          modelValue: 'hi',
          'onUpdate:modelValue': () => {},
          showCancel: true,
        },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="task-detail-comment-cancel"]').exists()).toBe(true)
    })
  })
}
