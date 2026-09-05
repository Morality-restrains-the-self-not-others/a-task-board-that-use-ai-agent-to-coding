// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] EnvVarTableEditor.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  describe('EnvVarTableEditor', () => {
    it('点击添加变量后应出现空草稿行，且不因空 key 被 v-model 回写冲掉', async () => {
      const component = await import('./EnvVarTableEditor.vue')
      const wrapper = mount(component.default, {
        props: {
          modelValue: [],
          'onUpdate:modelValue': (v) => wrapper.setProps({ modelValue: v }),
        },
      })
      await flushPromises()

      expect(wrapper.findAll('input[placeholder="MY_VAR"]')).toHaveLength(0)

      const addBtn = wrapper.findAll('button').find((b) => b.text().includes('添加变量'))
      expect(addBtn).toBeTruthy()
      await addBtn.trigger('click')
      await flushPromises()

      const keyInputs = wrapper.findAll('input[placeholder="MY_VAR"]')
      expect(keyInputs.length).toBe(1)
      // 空草稿不应立刻写入父级（避免空 key 被后端拒绝 / 被 filter 后回写消失）
      expect(wrapper.emitted('update:modelValue') || []).toHaveLength(0)

      await keyInputs[0].setValue('DEBUG_AGENT')
      await keyInputs[0].trigger('input')
      await flushPromises()

      const emitted = wrapper.emitted('update:modelValue')
      expect(emitted).toBeTruthy()
      expect(emitted[emitted.length - 1][0]).toEqual([
        { key: 'DEBUG_AGENT', value: '', description: '' },
      ])
      expect(wrapper.findAll('input[placeholder="MY_VAR"]')).toHaveLength(1)
    })

    it('已有变量时再添加，应保留已有行并追加空草稿', async () => {
      const component = await import('./EnvVarTableEditor.vue')
      const wrapper = mount(component.default, {
        props: {
          modelValue: [{ key: 'LOG_LEVEL', value: 'info', description: '' }],
          'onUpdate:modelValue': (v) => wrapper.setProps({ modelValue: v }),
        },
      })
      await flushPromises()

      const addBtn = wrapper.findAll('button').find((b) => b.text().includes('添加变量'))
      expect(addBtn).toBeTruthy()
      await addBtn.trigger('click')
      await flushPromises()

      expect(wrapper.findAll('input[placeholder="MY_VAR"]')).toHaveLength(2)
      expect(wrapper.find('input[placeholder="MY_VAR"]').element.value).toBe('LOG_LEVEL')
    })
  })
}
