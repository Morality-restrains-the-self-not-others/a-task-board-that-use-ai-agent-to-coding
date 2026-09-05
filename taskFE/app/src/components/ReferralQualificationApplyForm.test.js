// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ReferralQualificationApplyForm.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { default: Form } = await import('./ReferralQualificationApplyForm.vue')

  const VALID_INTRO = '我是平台活跃用户，日常使用任务与云主机，希望通过推荐帮助同事上手本平台。'

  describe('ReferralQualificationApplyForm', () => {
    beforeEach(() => {
      vi.stubGlobal('crypto', { randomUUID: () => 'ik-test-1' })
    })

    it('disables submit until intro, legal name and identity bind are valid', async () => {
      const apply = vi.fn()
      const wrapper = mount(Form, { props: { apply } })
      const btn = wrapper.get('[data-testid="referral-apply-submit"]')
      expect(btn.attributes('disabled')).toBeDefined()
      await wrapper.get('[data-testid="referral-personal-intro"]').setValue('太短了')
      expect(btn.attributes('disabled')).toBeDefined()
      await wrapper.get('[data-testid="referral-personal-intro"]').setValue(VALID_INTRO)
      expect(btn.attributes('disabled')).toBeDefined()
      expect(wrapper.text()).toContain('分成名额有限')
      expect(wrapper.get('[data-testid="referral-legal-name-warning"]').text()).toContain('填错')
      expect(wrapper.get('[data-testid="referral-legal-name-warning"]').text()).toContain('分账失败')
      await wrapper.get('[data-testid="referral-legal-name"]').setValue('张三')
      expect(btn.attributes('disabled')).toBeDefined()
      expect(wrapper.get('[data-testid="referral-identity-bind-notice"]').text()).toContain('账号标识')
      expect(wrapper.get('[data-testid="referral-identity-bind-notice"]').text()).toContain('用户身份信息')
      expect(wrapper.get('[data-testid="referral-identity-bind-notice"]').text()).toContain('微信支付')
      await wrapper.get('[data-testid="referral-identity-bind-consent"]').setValue(true)
      expect(btn.attributes('disabled')).toBeUndefined()
    })

    it('submits personal intro with legal name, identity bind consent and idempotency key', async () => {
      const apply = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(Form, { props: { apply, submitLabel: '申请推荐资格' } })
      await wrapper.get('[data-testid="referral-personal-intro"]').setValue(VALID_INTRO)
      await wrapper.get('[data-testid="referral-legal-name"]').setValue('张三')
      await wrapper.get('[data-testid="referral-identity-bind-consent"]').setValue(true)
      await wrapper.get('[data-testid="referral-apply-submit"]').trigger('click')
      await flushPromises()
      expect(apply).toHaveBeenCalledTimes(1)
      expect(apply.mock.calls[0][0]).toEqual({
        personalIntro: VALID_INTRO,
        legalName: '张三',
        identityBindConsent: true,
        idempotencyKey: 'ik-test-1',
      })
    })
  })
}
