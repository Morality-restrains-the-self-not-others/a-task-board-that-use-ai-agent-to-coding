// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ProjectCardBody.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const ProjectCardBody = (await import('./ProjectCardBody.vue')).default

  const stubProps = {
    tenantPath: '/tenant/1',
    statusBadgeClass: 'bg-green-100',
    formatDate: (v) => String(v || ''),
    formatImageLabel: (v) => String(v || ''),
  }

  describe('ProjectCardBody tags', () => {
    it('renders tag badges when project has tags', () => {
      const wrapper = mount(ProjectCardBody, {
        props: {
          ...stubProps,
          project: {
            id: '1',
            name: 'Demo',
            tags: ['frontend', 'api'],
          },
        },
      })
      expect(wrapper.get('[data-testid="project-card-tags"]').text()).toContain('frontend')
      expect(wrapper.get('[data-testid="project-card-tags"]').text()).toContain('api')
    })

    it('hides tags section when empty', () => {
      const wrapper = mount(ProjectCardBody, {
        props: {
          ...stubProps,
          project: { id: '1', name: 'Demo', tags: [] },
        },
      })
      expect(wrapper.find('[data-testid="project-card-tags"]').exists()).toBe(false)
    })

    it('shows overflow count when more than 3 tags', () => {
      const wrapper = mount(ProjectCardBody, {
        props: {
          ...stubProps,
          project: {
            id: '1',
            name: 'Demo',
            tags: ['a', 'b', 'c', 'd', 'e'],
          },
        },
      })
      const tagsEl = wrapper.get('[data-testid="project-card-tags"]')
      expect(tagsEl.text()).toContain('a')
      expect(tagsEl.text()).toContain('+2')
      expect(tagsEl.text()).not.toContain('e')
    })
  })
}
