// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailLinkedProjectsToolbar.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  describe('TaskDetailLinkedProjectsToolbar — 默认镜像', () => {
    it('U6 标题行右侧展示默认镜像标签', async () => {
      const Comp = (await import('./TaskDetailLinkedProjectsToolbar.vue')).default
      const wrapper = mount(Comp, {
        props: {
          isEditing: false,
          defaultImageLabel: 'trae-agent:x86_64-latest',
          defaultImageId: '878236719185424384',
        },
      })
      const el = wrapper.get('[data-testid="task-default-container-image"]')
      expect(el.text()).toContain('默认镜像')
      expect(el.text()).toContain('trae-agent:x86_64-latest')
      expect(wrapper.text()).toContain('关联项目')
    })

    it('U5 无默认镜像时展示未设置', async () => {
      const Comp = (await import('./TaskDetailLinkedProjectsToolbar.vue')).default
      const wrapper = mount(Comp, { props: { isEditing: false } })
      expect(wrapper.get('[data-testid="task-default-container-image"]').text()).toContain('未设置')
    })

    it('U4 仅有 id 时展示已绑定', async () => {
      const Comp = (await import('./TaskDetailLinkedProjectsToolbar.vue')).default
      const wrapper = mount(Comp, {
        props: { isEditing: false, defaultImageId: '877435294134071296' },
      })
      expect(wrapper.get('[data-testid="task-default-container-image"]').text()).toContain('已绑定')
    })
  })
}
