// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ImageSkillsList.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: ImageSkillsList } = await import('./ImageSkillsList.vue')

  describe('ImageSkillsList', () => {
    it('空列表且无抽取状态时不渲染', () => {
      const wrapper = mount(ImageSkillsList, { props: { image: {} } })
      expect(wrapper.find('[data-testid="image-skills-list"]').exists()).toBe(false)
    })

    it('渲染技能并标记默认', () => {
      const wrapper = mount(ImageSkillsList, {
        props: {
          image: {
            image_skills: {
              version: 1,
              default_skill: 'general-coding',
              skills: [
                { name: 'general-coding', description: 'dev', is_default: true },
                { name: 'k8s-debug' },
              ],
            },
          },
        },
      })
      expect(wrapper.find('[data-testid="image-skills-list"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="image-skill-default"]').text()).toContain('/general-coding')
      expect(wrapper.text()).toContain('/k8s-debug')
      expect(wrapper.text()).toContain('默认')
    })
  })
}
