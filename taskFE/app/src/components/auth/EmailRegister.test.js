// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] EmailRegister.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  const { default: EmailRegister } = await import('./EmailRegister.vue')

  // OPT-20260821-025: 字段级 API 错误节点须挂 data-traceId，供 trace-first 排障。
  describe('EmailRegister — 字段级错误节点带 data-traceId', () => {
    it('traceId prop 传入时 #email-error 挂 data-traceId', () => {
      const wrapper = mount(EmailRegister, {
        props: {
          modelValue: { email: '', password: '' },
          errors: { email: '邮箱无效' },
          traceId: 'trace-mail-1',
        },
      })
      const err = wrapper.find('#email-error')
      expect(err.exists()).toBe(true)
      expect(err.attributes('data-traceid')).toBe('trace-mail-1')
      expect(err.text()).toContain('邮箱无效')
    })

    it('traceId 为空时不挂 data-traceId（纯前端校验不伪造）', () => {
      const wrapper = mount(EmailRegister, {
        props: {
          modelValue: { email: '', password: '' },
          errors: { email: '邮箱无效' },
          traceId: '',
        },
      })
      expect(wrapper.find('#email-error').attributes('data-traceid')).toBeUndefined()
    })
  })
}
