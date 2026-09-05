// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceCreateEditModal.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { default: Modal } = await import('./WorkspaceCreateEditModal.vue')

  describe('WorkspaceCreateEditModal', () => {
    it('勾选是否设为默认会写回 form.is_default', async () => {
      const form = { name: 'Beta', description: '', is_default: false }
      const wrapper = mount(Modal, {
        props: { show: true, editing: true, form, saving: false },
      })
      await wrapper.find('#workspace-is-default').setValue(true)
      await flushPromises()
      expect(form.is_default).toBe(true)
    })

    it('保存按钮在有名称时发出 save', async () => {
      const form = { name: 'Beta', description: '', is_default: true }
      const wrapper = mount(Modal, {
        props: { show: true, editing: false, form, saving: false },
      })
      const saveBtn = wrapper.findAll('button').find((b) => b.text().trim() === '保存')
      await saveBtn.trigger('click')
      expect(wrapper.emitted('save')).toBeTruthy()
    })
  })
}
