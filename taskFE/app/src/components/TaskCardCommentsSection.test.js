// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskCardCommentsSection.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { nextTick } = await import('vue')

  const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

  vi.mock('../utils/apiUtils.js', () => ({ apiFetch }))

  const { default: TaskCardCommentsSection } = await import('./TaskCardCommentsSection.vue')
  const { default: TaskDetailCommentComposer } = await import(
    './task-detail/TaskDetailCommentComposer.vue'
  )

  describe('TaskCardCommentsSection', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url) => {
        const path = String(url || '')
        if (path.includes('/api/projects/')) {
          return {
            ok: true,
            json: async () => [
              { id: 'p1', name: 'Proj', git_repos: ['https://github.com/acme/demo.git'] },
            ],
          }
        }
        if (path.includes('/workspaces/')) {
          return { ok: true, json: async () => ({ container_image_at_mode_enabled: true }) }
        }
        if (path.includes('/installed-images/')) {
          return { ok: true, json: async () => [{ id: 'img1', name: 'node' }] }
        }
        if (path.includes('/git-identities/')) {
          return {
            ok: true,
            json: async () => ({
              identities: [{ id: 'gid-1', git_user_name: 'Ann', git_user_email: 'a@b.c' }],
            }),
          }
        }
        if (path.includes('/git-oauth/')) {
          return { ok: true, json: async () => ({ connected: false, connections: [] }) }
        }
        return { ok: true, json: async () => ({}) }
      })
    })

    it('有仓库且 mention 后出现 comment-composer-repo-identity（OPT-20260816-009）', async () => {
      const wrapper = mount(TaskCardCommentsSection, {
        props: {
          task: {
            id: 'task1',
            projects: [{ project_id: 'p1' }],
            comments: [],
          },
          tenantId: 't1',
          workspaceId: 'w1',
        },
      })
      await wrapper.find('[data-testid="task-card-toggle-comments"]').trigger('click')
      await flushPromises()
      const composer = wrapper.findComponent(TaskDetailCommentComposer)
      expect(composer.exists()).toBe(true)
      composer.vm.onMentionChange({ id: 'img1', name: 'node' })
      await nextTick()
      await flushPromises()
      expect(wrapper.find('[data-testid="comment-composer-repo-identity"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('https://github.com/acme/demo.git')
    })

    it('无仓库时 mention 后不渲染 repo-identity', async () => {
      apiFetch.mockImplementation(async (url) => {
        const path = String(url || '')
        if (path.includes('/api/projects/')) {
          return { ok: true, json: async () => [] }
        }
        if (path.includes('/workspaces/')) {
          return { ok: true, json: async () => ({ container_image_at_mode_enabled: true }) }
        }
        return { ok: true, json: async () => ({}) }
      })
      const wrapper = mount(TaskCardCommentsSection, {
        props: {
          task: { id: 'task2', projects: [], comments: [] },
          tenantId: 't1',
          workspaceId: 'w1',
        },
      })
      await wrapper.find('[data-testid="task-card-toggle-comments"]').trigger('click')
      await flushPromises()
      const composer = wrapper.findComponent(TaskDetailCommentComposer)
      composer.vm.onMentionChange({ id: 'img1', name: 'node' })
      await nextTick()
      await flushPromises()
      expect(wrapper.find('[data-testid="comment-composer-repo-identity"]').exists()).toBe(false)
    })
  })
}
