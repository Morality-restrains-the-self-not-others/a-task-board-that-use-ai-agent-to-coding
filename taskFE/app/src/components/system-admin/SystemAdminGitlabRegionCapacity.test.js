// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] SystemAdminGitlabRegionCapacity.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const Capacity = (await import('./SystemAdminGitlabRegionCapacity.vue')).default

  const sharedRegion = {
    slug: 'tencent-sh-1',
    name: '腾讯上海一区',
    bandwidth_shared: true,
    total_bandwidth_mbps: 200,
    remaining_bandwidth_mbps: 80,
    allocated_disk_gb: 10,
    total_disk_gb: 100,
    allocated_traffic_gb: 5,
    total_traffic_gb: 50,
  }

  describe('SystemAdminGitlabRegionCapacity', () => {
    it('共享分区展示总带宽与剩余带宽', () => {
      const wrapper = mount(Capacity, { props: { region: sharedRegion } })
      expect(wrapper.get('[data-testid="gitlab-region-bandwidth-share"]').text()).toBe('带宽共享分区')
      expect(wrapper.get('[data-testid="gitlab-region-bandwidth-total"]').text()).toContain('200')
      expect(wrapper.get('[data-testid="gitlab-region-bandwidth-remaining"]').text()).toBe('80')
    })

    it('独立带宽未配置时为 0', () => {
      const wrapper = mount(Capacity, {
        props: {
          region: {
            slug: 'tencent-shanghai-5',
            bandwidth_shared: false,
            total_bandwidth_mbps: 0,
            remaining_bandwidth_mbps: 0,
            allocated_disk_gb: 0,
            total_disk_gb: 0,
            allocated_traffic_gb: 0,
            total_traffic_gb: 0,
          },
        },
      })
      expect(wrapper.get('[data-testid="gitlab-region-bandwidth-share"]').text()).toBe('独立带宽')
      expect(wrapper.get('[data-testid="gitlab-region-bandwidth-remaining"]').text()).toBe('0')
    })

    it('更新容量提交带宽字段', async () => {
      const wrapper = mount(Capacity, { props: { region: sharedRegion } })
      await wrapper.get('[data-testid="gitlab-region-bandwidth-total-input"]').setValue(300)
      await wrapper.get('[data-testid="gitlab-region-bandwidth-remaining-input"]').setValue(100)
      await wrapper.get('[data-testid="gitlab-region-capacity-save"]').trigger('click')
      expect(wrapper.emitted('save')[0][0]).toMatchObject({
        slug: 'tencent-sh-1',
        total_bandwidth_mbps: 300,
        remaining_bandwidth_mbps: 100,
        bandwidth_shared: true,
      })
    })

    it('剩余带宽大于总量时禁用保存并展示行内提示，不 emit save', async () => {
      const wrapper = mount(Capacity, { props: { region: sharedRegion } })
      await wrapper.get('[data-testid="gitlab-region-bandwidth-total-input"]').setValue(200)
      await wrapper.get('[data-testid="gitlab-region-bandwidth-remaining-input"]').setValue(999)
      const hint = wrapper.get('[data-testid="gitlab-region-bandwidth-invalid-hint"]')
      expect(hint.exists()).toBe(true)
      const btn = wrapper.get('[data-testid="gitlab-region-capacity-save"]')
      expect(btn.attributes('disabled')).toBeDefined()
      await btn.trigger('click')
      expect(wrapper.emitted('save')).toBeUndefined()
    })

    it('剩余带宽回落到总量以下后恢复可保存', async () => {
      const wrapper = mount(Capacity, { props: { region: sharedRegion } })
      await wrapper.get('[data-testid="gitlab-region-bandwidth-total-input"]').setValue(200)
      await wrapper.get('[data-testid="gitlab-region-bandwidth-remaining-input"]').setValue(999)
      expect(wrapper.get('[data-testid="gitlab-region-bandwidth-invalid-hint"]').exists()).toBe(true)
      await wrapper.get('[data-testid="gitlab-region-bandwidth-remaining-input"]').setValue(150)
      expect(wrapper.find('[data-testid="gitlab-region-bandwidth-invalid-hint"]').exists()).toBe(false)
      expect(wrapper.get('[data-testid="gitlab-region-capacity-save"]').attributes('disabled')).toBeUndefined()
    })
  })
}
