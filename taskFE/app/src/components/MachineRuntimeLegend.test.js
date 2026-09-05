// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] MachineRuntimeLegend.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  const { default: MachineRuntimeLegend } = await import('./MachineRuntimeLegend.vue')
  const {
    RUNTIME_MACHINE_TIP,
    RUNTIME_CONTAINER_TIP,
  } = await import('../utils/workPanelRuntimeIndicators.js')

  describe('MachineRuntimeLegend', () => {
    it('展示空心=已启动机器 / 实心=已启动容器，并带独立 title', () => {
      const wrapper = mount(MachineRuntimeLegend, {
        attrs: { class: 'mr-1.5' },
      })
      const root = wrapper.get('[data-testid="machine-runtime-legend"]')
      expect(root.classes()).toContain('mr-1.5')
      expect(root.text()).toContain('空心=已启动机器')
      expect(root.text()).toContain('实心=已启动容器')
      expect(root.attributes('aria-label')).toContain('空心环')
      const titles = root.findAll('[title]').map((n) => n.attributes('title'))
      expect(titles).toEqual([RUNTIME_MACHINE_TIP, RUNTIME_CONTAINER_TIP])
    })
  })
}
