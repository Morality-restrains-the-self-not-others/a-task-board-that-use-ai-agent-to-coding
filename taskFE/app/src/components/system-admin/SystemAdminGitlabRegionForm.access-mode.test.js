// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGitlabRegionForm.access-mode.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const Form = (await import('./SystemAdminGitlabRegionForm.vue')).default

  describe('SystemAdminGitlabRegionForm access_mode', () => {
    it('defaults to release and emits development when selected', async () => {
      const wrapper = mount(Form)
      const select = wrapper.get('select[aria-label="区域模式"]')
      expect(select.element.value).toBe('release')
      await select.setValue('development')
      await wrapper.get('button').trigger('click')
      const payload = wrapper.emitted('submit')[0][0]
      expect(payload.access_mode).toBe('development')
    })
  })
}
