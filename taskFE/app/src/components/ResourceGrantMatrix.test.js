// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ResourceGrantMatrix.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  const samplePages = [
    {
      id: 'rg-page-people-access',
      group_key: 'people.access',
      display_name: '访问管理',
      kind: 'page',
      children: [
        { id: 'rg-reg-subject', group_key: 'people.access.subject_list', display_name: '主体列表', kind: 'ui_region' },
        { id: 'rg-reg-save', group_key: 'people.access.save_actions', display_name: '保存操作', kind: 'ui_region' },
      ],
    },
  ]

  describe('ResourceGrantMatrix', () => {
    it('渲染 page + 子 region 勾选行', async () => {
      const Comp = (await import('./ResourceGrantMatrix.vue')).default
      const wrapper = mount(Comp, {
        props: { pages: samplePages, modelValue: {} },
      })
      expect(wrapper.find('[data-testid="resource-grant-matrix"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('访问管理')
      expect(wrapper.text()).toContain('主体列表')
      expect(wrapper.text()).toContain('可访问')
      expect(wrapper.text()).toContain('可编辑执行')
    })

    it('整页勾选发出 page + 全部 region operate', async () => {
      const Comp = (await import('./ResourceGrantMatrix.vue')).default
      const wrapper = mount(Comp, {
        props: { pages: samplePages, modelValue: {} },
      })
      const pageBox = wrapper.find('input[type="checkbox"]')
      await pageBox.setValue(true)
      const emitted = wrapper.emitted('update:modelValue')
      expect(emitted?.length).toBeGreaterThan(0)
      const last = emitted[emitted.length - 1][0]
      expect(last['people.access']).toBe('operate')
      expect(last['people.access.subject_list']).toBe('operate')
      expect(last['people.access.save_actions']).toBe('operate')
    })

    it('region 可访问勾选发出 view，再勾可编辑执行升为 operate', async () => {
      const Comp = (await import('./ResourceGrantMatrix.vue')).default
      const wrapper = mount(Comp, {
        props: { pages: samplePages, modelValue: {} },
      })
      // 模拟 v-model：emit 后把新值写回 prop（父组件持有状态）
      const sync = async () => {
        const last = wrapper.emitted('update:modelValue').at(-1)[0]
        await wrapper.setProps({ modelValue: last })
      }
      // 第一个子 region 的「可访问」复选框
      const regionBoxes = wrapper.findAll('input[type="checkbox"]')
      // index 0 = page 整页；index 1 = 第一个 region 可访问；index 2 = 第一个 region 可编辑执行
      await regionBoxes[1].setValue(true)
      await sync()
      expect(wrapper.emitted('update:modelValue').at(-1)[0]['people.access.subject_list']).toBe('view')

      await regionBoxes[2].setValue(true)
      await sync()
      expect(wrapper.emitted('update:modelValue').at(-1)[0]['people.access.subject_list']).toBe('operate')

      // 取消可编辑执行 → 降级回 view
      await regionBoxes[2].setValue(false)
      await sync()
      expect(wrapper.emitted('update:modelValue').at(-1)[0]['people.access.subject_list']).toBe('view')
    })

    it('disabled 时点击不发出更新', async () => {
      const Comp = (await import('./ResourceGrantMatrix.vue')).default
      const wrapper = mount(Comp, {
        props: { pages: samplePages, modelValue: {}, disabled: true },
      })
      const pageBox = wrapper.find('input[type="checkbox"]')
      await pageBox.setValue(true)
      expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    })

    it('无目录时展示空态', async () => {
      const Comp = (await import('./ResourceGrantMatrix.vue')).default
      const wrapper = mount(Comp, {
        props: { pages: [], modelValue: {} },
      })
      expect(wrapper.text()).toContain('暂无资源组目录')
    })
  })
}
