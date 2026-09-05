// @vitest-environment jsdom
// 列表徽章必须跟 authorization.is_active 走：启用显示「已启用」，否则「已禁用」。
if (!process.env.VITEST) {
  console.log('[skip] CloudPlatformAuthorizationRow.is-active.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: Row } = await import('./CloudPlatformAuthorizationRow.vue')

  function mountRow(isActive) {
    return mount(Row, {
      props: {
        authorization: {
          id: 'cpa-1',
          platform_type: 'aliyun',
          authorization_type: 'access_key',
          secret_id: 'AK****',
          remark: 'row-badge',
          created_at: '2026-08-28T00:00:00Z',
          is_active: isActive,
        },
      },
    })
  }

  describe('CloudPlatformAuthorizationRow 启用徽章', () => {
    it('is_active true 显示已启用', () => {
      const wrapper = mountRow(true)
      expect(wrapper.text()).toContain('已启用')
      expect(wrapper.text()).not.toContain('已禁用')
    })

    it('is_active false 显示已禁用', () => {
      const wrapper = mountRow(false)
      expect(wrapper.text()).toContain('已禁用')
      expect(wrapper.text()).not.toContain('已启用')
    })
  })
}
