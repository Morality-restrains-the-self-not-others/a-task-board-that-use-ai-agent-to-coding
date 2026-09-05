// @vitest-environment jsdom
// OPT-20260806-066：个人资料页邮箱绑定面板（厂商门户前置）。
// 无邮箱 → 绑定表单；有邮箱 → 当前绑定 + 更换表单；验证码发送与绑定提交走
// /api/accounts/users/send_verification_code/ 与 /bind_email/，成功后 emit。
if (!process.env.VITEST) {
  console.log('[skip] UserProfileEmailBindingPanel.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const apiFetchMock = vi.hoisted(() => vi.fn())
  vi.mock('../utils/apiUtils', () => ({ apiFetch: apiFetchMock }))

  const { default: UserProfileEmailBindingPanel } = await import('./UserProfileEmailBindingPanel.vue')

  beforeEach(() => {
    apiFetchMock.mockReset()
  })

  const mountPanel = (props = {}) =>
    mount(UserProfileEmailBindingPanel, { props: { hasEmail: false, email: '', ...props } })

  describe('UserProfileEmailBindingPanel 渲染', () => {
    it('未绑定邮箱：显示绑定表单（无「当前绑定」）', () => {
      const wrapper = mountPanel()
      expect(wrapper.text()).toContain('身份绑定（邮箱）')
      expect(wrapper.text()).toContain('当前未绑定邮箱')
      expect(wrapper.text()).not.toContain('当前绑定：')
    })

    it('已绑定邮箱：显示当前邮箱与「更换邮箱」表单', () => {
      const wrapper = mountPanel({ hasEmail: true, email: 'vendor@example.com' })
      expect(wrapper.text()).toContain('当前绑定：vendor@example.com')
      expect(wrapper.text()).toContain('更换邮箱')
    })
  })

  describe('UserProfileEmailBindingPanel 绑定流程', () => {
    it('发送验证码：调用 send_verification_code（邮箱小写归一）', async () => {
      apiFetchMock.mockResolvedValue({ ok: true, json: async () => ({ message: 'ok' }) })
      const wrapper = mountPanel()
      await wrapper.find('input[type="email"]').setValue('Vendor@Example.com')
      await wrapper.findAll('button').find((b) => b.text().includes('发送验证码')).trigger('click')
      await flushPromises()
      expect(apiFetchMock).toHaveBeenCalledWith(
        '/api/accounts/users/send_verification_code/',
        expect.objectContaining({ method: 'POST', body: JSON.stringify({ email: 'vendor@example.com' }) }),
      )
    })

    it('绑定成功：调用 bind_email 并 emit profile-updated / message', async () => {
      apiFetchMock.mockResolvedValue({ ok: true, json: async () => ({ bound: true, email: 'vendor@example.com' }) })
      const wrapper = mountPanel()
      await wrapper.find('input[type="email"]').setValue('vendor@example.com')
      await wrapper.find('input[autocomplete="one-time-code"]').setValue('123456')
      await wrapper.findAll('button').find((b) => b.text().includes('验证并绑定')).trigger('click')
      await flushPromises()
      expect(apiFetchMock).toHaveBeenCalledWith(
        '/api/accounts/users/bind_email/',
        expect.objectContaining({ method: 'POST', body: JSON.stringify({ email: 'vendor@example.com', code: '123456' }) }),
      )
      expect(wrapper.emitted('profile-updated')).toHaveLength(1)
      // 提交时先清空 message，成功后 emit 成功文案 → 取最后一次
      expect(wrapper.emitted('message')?.at(-1)?.[0]).toContain('绑定邮箱')
    })

    it('被占用（409）：显示后端 detail 提示，不 emit', async () => {
      apiFetchMock.mockResolvedValue({ ok: false, json: async () => ({ detail: '该邮箱已绑定其他账号' }) })
      const wrapper = mountPanel()
      await wrapper.find('input[type="email"]').setValue('taken@example.com')
      await wrapper.find('input[autocomplete="one-time-code"]').setValue('123456')
      await wrapper.findAll('button').find((b) => b.text().includes('验证并绑定')).trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('该邮箱已绑定其他账号')
      expect(wrapper.emitted('profile-updated')).toBeUndefined()
    })

    it('合成邮箱（@sso.invalid）前端拦截：不发起请求', async () => {
      const wrapper = mountPanel()
      await wrapper.find('input[type="email"]').setValue('sso-123@sso.invalid')
      await wrapper.findAll('button').find((b) => b.text().includes('发送验证码')).trigger('click')
      await flushPromises()
      expect(apiFetchMock).not.toHaveBeenCalled()
      expect(wrapper.text()).toContain('请输入有效邮箱地址')
    })

    it('网络错误（Failed to fetch）：内联文案中文化而非裸露英文（OPT-20260811-073）', async () => {
      apiFetchMock.mockRejectedValue(new Error('Failed to fetch'))
      const wrapper = mountPanel()
      await wrapper.find('input[type="email"]').setValue('vendor@example.com')
      await wrapper.findAll('button').find((b) => b.text().includes('发送验证码')).trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('网络错误，请稍后重试')
      expect(wrapper.text()).not.toContain('Failed to fetch')
    })

    it('发送失败：内联错误挂载 data-traceId（响应体 trace_id）', async () => {
      const headers = new Headers({ 'X-Trace-Id': 'trace-email-bind-001' })
      apiFetchMock.mockResolvedValue({
        ok: false,
        headers,
        json: async () => ({
          detail: '邮件发送失败，请稍后重试',
          trace_id: 'trace-email-bind-001',
        }),
      })
      const wrapper = mountPanel()
      await wrapper.find('input[type="email"]').setValue('vendor@example.com')
      await wrapper.findAll('button').find((b) => b.text().includes('发送验证码')).trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('邮件发送失败，请稍后重试')
      expect(wrapper.text()).not.toContain('验证码已发送，请查收邮箱')
      const errEl = wrapper.find('p.text-danger')
      expect(errEl.exists()).toBe(true)
      expect(errEl.attributes('data-traceid') || errEl.attributes('data-traceId')).toBe(
        'trace-email-bind-001',
      )
    })
  })
}
