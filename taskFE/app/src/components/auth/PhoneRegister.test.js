// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] PhoneRegister.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
  const { nextTick } = await import('vue')

  const hoisted = vi.hoisted(() => ({
    fetchAllowedCodes: vi.fn().mockResolvedValue(undefined),
    syncPrefixWithAllowedOptions: vi.fn(),
  }))

  vi.mock('../../composables/auth/useAllowedCountryCodes.js', () => ({
    useAllowedCountryCodes: () => ({
      // 模板 v-for 直接用数组即可；defaultCountryPrefix 以 .value 形式提供（组件仅读一次）
      filteredCountryOptions: [{ code: '+86', name: '中国大陆' }],
      defaultCountryPrefix: { value: '+86' },
      syncPrefixWithAllowedOptions: (...a) => hoisted.syncPrefixWithAllowedOptions(...a),
      loading: { value: false },
      error: { value: '' },
      fetchAllowedCodes: (...a) => hoisted.fetchAllowedCodes(...a),
    }),
  }))

  const { default: PhoneRegister } = await import('./PhoneRegister.vue')

  const mountRegister = () =>
    mount(PhoneRegister, {
      props: {
        modelValue: { phone: '', code: '', password: '' },
        errors: { phone: '', code: '' },
        csrfToken: '',
      },
      attachTo: document.body,
    })

  const setPhone = async (wrapper, national) => {
    await wrapper.find('#register-phone-national').setValue(national)
  }

  const clickSend = async (wrapper) => {
    await wrapper.find('button[type="button"]').trigger('click')
  }

  // 等待 Promise.race 的 resolve 回调链与 Vue 重渲染全部落定
  const flushOutcome = async () => {
    await Promise.resolve()
    await Promise.resolve()
    await nextTick()
  }

  describe('PhoneRegister — 发码成功后才倒计时（OPT-20260811-013）', () => {
    beforeEach(() => {
      vi.useFakeTimers()
      hoisted.fetchAllowedCodes.mockClear()
      hoisted.syncPrefixWithAllowedOptions.mockClear()
    })

    afterEach(() => {
      vi.useRealTimers()
      document.body.innerHTML = ''
    })

    it('成功路径：父组件 resolve ok 后才启动 60s 倒计时', async () => {
      const wrapper = mountRegister()
      await setPhone(wrapper, '13800138000')

      await clickSend(wrapper)

      // 必须发出 send-code 事件，且携带 resolve 回调（父组件据此回报结果）
      const emits = wrapper.emitted('send-code')
      expect(emits).toBeTruthy()
      const [{ phone, resolve }] = emits[0]
      expect(phone).toBe('+8613800138000')
      expect(typeof resolve).toBe('function')

      // resolve 之前：不进入倒计时
      expect(wrapper.text()).not.toContain('秒后重新获取')

      resolve({ ok: true })
      await flushOutcome()

      expect(wrapper.text()).toContain('60秒后重新获取')
      expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    })

    it('失败路径：resolve 失败时按钮上方展示错误并挂 data-traceId，不进入倒计时', async () => {
      const wrapper = mountRegister()
      await setPhone(wrapper, '13800138000')

      await clickSend(wrapper)

      const [{ resolve }] = wrapper.emitted('send-code')[0]
      resolve({
        ok: false,
        message: '短信发送失败：账户余额不足（错误码 isv.AMOUNT_NOT_ENOUGH）',
        traceId: 'trace-9f2a7b',
      })
      await flushOutcome()

      const alert = wrapper.find('[role="alert"]')
      expect(alert.exists()).toBe(true)
      expect(alert.text()).toContain('账户余额不足')
      expect(alert.attributes('data-traceid')).toBe('trace-9f2a7b')
      expect(wrapper.text()).not.toContain('秒后重新获取')
      // 按钮恢复可重试
      expect(wrapper.find('button[type="button"]').text()).toBe('获取验证码')
    })

    it('纯前端手机号校验失败不发 send-code、不伪造 data-traceId', async () => {
      const wrapper = mountRegister()
      await setPhone(wrapper, '123')

      await clickSend(wrapper)

      expect(wrapper.emitted('send-code')).toBeUndefined()
      const alert = wrapper.find('[role="alert"]')
      expect(alert.exists()).toBe(false)
      // phone 字段错误经 update:errors 回报给父组件
      const errorEmits = wrapper.emitted('update:errors')
      expect(errorEmits).toBeTruthy()
      expect(errorEmits.at(-1)[0].phone).toContain('有效')
      expect(wrapper.find('button[type="button"]').text()).toBe('获取验证码')
    })
  })

  describe('PhoneRegister — 字段级错误节点带 data-traceId (OPT-20260821-025)', () => {
    it('traceId prop 传入时 #phone-error/#code-error 挂 data-traceId', () => {
      const wrapper = mount(PhoneRegister, {
        props: {
          modelValue: { phone: '', code: '', password: '' },
          errors: { phone: '手机号无效', code: '验证码错误' },
          csrfToken: '',
          traceId: 'trace-field-abc',
        },
      })
      const phoneErr = wrapper.find('#phone-error')
      expect(phoneErr.exists()).toBe(true)
      expect(phoneErr.attributes('data-traceid')).toBe('trace-field-abc')
      expect(phoneErr.text()).toContain('手机号无效')
      expect(wrapper.find('#code-error').attributes('data-traceid')).toBe('trace-field-abc')
    })

    it('traceId 为空时不挂 data-traceId（纯前端校验不伪造）', () => {
      const wrapper = mount(PhoneRegister, {
        props: {
          modelValue: { phone: '', code: '', password: '' },
          errors: { phone: '手机号无效', code: '' },
          csrfToken: '',
          traceId: '',
        },
      })
      expect(wrapper.find('#phone-error').attributes('data-traceid')).toBeUndefined()
    })
  })
}
