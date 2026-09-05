// @vitest-environment jsdom
// LoginPhonePasswordFields 回归：模板 @input/@change/@click 依赖 script setup 中
// `const emit = defineEmits(...)` 的返回值。此前只有裸 defineEmits([...]) 调用，
// 编译后 _ctx.emit 为 undefined → 每次输入抛 "TypeError: i.emit is not a function"
// → 父组件 phoneNationalPassword ref 永不更新 → isPhoneValidForPassword 恒 false
// → 登录按钮永远禁用「请填写正确手机号」（线上 https://www.daydaymoney.com/auth/login/ 复现）。
if (!process.env.VITEST) {
  console.log('[skip] LoginPhonePasswordFields.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')

  const { default: LoginPhonePasswordFields } = await import('../../components/auth/LoginPhonePasswordFields.vue')

  const baseProps = {
    loginPhonePrefix: '+86',
    phoneNationalPassword: '',
    showPhonePassword: false,
    countryDialOptions: [{ code: '+86', name: '中国' }],
    countryLoading: false,
    countryErrorHint: '',
    nationalPlaceholder: '1×××××××××',
    maxNationalLen: 11,
  }

  describe('LoginPhonePasswordFields 事件透传（emit 必须可调用）', () => {
    it('手机号输入触发 national-pwd-input 事件并携带目标值', async () => {
      const wrapper = mount(LoginPhonePasswordFields, { props: baseProps })
      const input = wrapper.find('input#login-phone-national-pwd')
      await input.setValue('13800138000')
      const events = wrapper.emitted('national-pwd-input')
      expect(events).toBeTruthy()
      expect(events[0][0].target.value).toBe('13800138000')
    })

    it('手机号粘贴触发 national-pwd-paste 事件', async () => {
      const wrapper = mount(LoginPhonePasswordFields, { props: baseProps })
      const input = wrapper.find('input#login-phone-national-pwd')
      await input.trigger('paste', { clipboardData: { getData: () => '+8613800138000' } })
      expect(wrapper.emitted('national-pwd-paste')).toBeTruthy()
    })

    it('区号切换触发 update:loginPhonePrefix 事件', async () => {
      const wrapper = mount(LoginPhonePasswordFields, {
        props: {
          ...baseProps,
          countryDialOptions: [
            { code: '+86', name: '中国' },
            { code: '+852', name: '中国香港' },
          ],
        },
      })
      const select = wrapper.find('select[aria-label="国家或地区代码"]')
      // option 的 value 绑定为 c.code（'+852'），显示文本为「+852 中国香港」
      await select.setValue('+852')
      const events = wrapper.emitted('update:loginPhonePrefix')
      expect(events).toBeTruthy()
      expect(events[0][0]).toBe('+852')
    })

    it('密码显示按钮触发 update:showPhonePassword 事件', async () => {
      const wrapper = mount(LoginPhonePasswordFields, { props: baseProps })
      await wrapper.find('button[type="button"]').trigger('click')
      const events = wrapper.emitted('update:showPhonePassword')
      expect(events).toBeTruthy()
      expect(events[0][0]).toBe(true)
    })
  })
}
