// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceMachineSummary.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  const { nextTick } = await import('vue')
  const { default: WorkspaceMachineSummary } = await import('./WorkspaceMachineSummary.vue')
  const {
    RUNTIME_MACHINE_TIP,
    RUNTIME_CONTAINER_TIP,
  } = await import('../utils/workPanelRuntimeIndicators.js')

  const summaryProps = {
    machineSummary: {
      startedCount: 2,
      idleCount: 1,
      startingCount: 0,
      idleRecycleMinutes: 30,
    },
  }
  const infoBtn = (wrapper) => wrapper.get('[data-alias="workspace-machine-summary-info"]')
  const tooltip = (wrapper) => wrapper.get('[data-testid="workspace-machine-summary-tooltip"]')
  const hasTooltip = (wrapper) => wrapper.find('[data-testid="workspace-machine-summary-tooltip"]').exists()

  describe('WorkspaceMachineSummary 紧凑摘要', () => {
    it('主视图仅展示徽标与数字，图例与回收策略在详情提示中', async () => {
      const wrapper = mount(WorkspaceMachineSummary, {
        props: {
          machineSummary: {
            startedCount: 2,
            idleCount: 1,
            startingCount: 0,
            idleRecycleMinutes: 30,
          },
        },
      })
      const root = wrapper.get('[data-alias="workspace-machine-summary"]')
      expect(root.text()).not.toContain('机器节点：')
      expect(root.text()).not.toContain('空心=已启动机器')
      expect(root.text()).not.toContain('实心=已启动容器')
      expect(root.text()).not.toContain('闲置回收')

      expect(wrapper.get('[data-alias="machine-filter-started"]').text()).toContain('2')
      expect(wrapper.get('[data-alias="machine-filter-idle"]').text()).toContain('1')

      const infoBtn = wrapper.get('[data-alias="workspace-machine-summary-info"]')
      expect(infoBtn.attributes('aria-label')).toContain('机器节点')

      await infoBtn.trigger('mouseenter')
      const tooltip = wrapper.get('[data-testid="workspace-machine-summary-tooltip"]')
      expect(tooltip.text()).toContain('空心=已启动机器')
      expect(tooltip.text()).toContain('实心=已启动容器')
      expect(tooltip.text()).toContain('闲置回收：30 分钟')
      expect(tooltip.text()).toContain(RUNTIME_MACHINE_TIP)
      expect(tooltip.text()).toContain(RUNTIME_CONTAINER_TIP)
    })

    it('启动中数量大于 0 时在主视图展示徽标数字，详情提示含说明', async () => {
      const wrapper = mount(WorkspaceMachineSummary, {
        props: {
          machineSummary: {
            startedCount: 1,
            idleCount: 0,
            startingCount: 3,
            idleRecycleMinutes: 5,
          },
        },
      })
      expect(wrapper.get('[data-alias="machine-summary-starting"]').text()).toContain('3')

      await wrapper.get('[data-alias="workspace-machine-summary-info"]').trigger('mouseenter')
      expect(wrapper.get('[data-testid="workspace-machine-summary-tooltip"]').text()).toContain('启动中 3')
    })

    it('无汇总时展示占位符，详情提示仍含图例', async () => {
      const wrapper = mount(WorkspaceMachineSummary, {
        props: { machineSummary: null },
      })
      expect(wrapper.get('[data-alias="workspace-machine-summary"]').text()).toContain('—')

      await wrapper.get('[data-alias="workspace-machine-summary-info"]').trigger('mouseenter')
      expect(wrapper.get('[data-testid="workspace-machine-summary-tooltip"]').text()).toContain('空心=已启动机器')
    })

    it('触屏无 hover 场景：点击 ! 打开详情提示，再次点击关闭', async () => {
      const wrapper = mount(WorkspaceMachineSummary, { props: summaryProps })
      expect(hasTooltip(wrapper)).toBe(false)

      // 模拟触屏 tap：pointerdown(touch) 抑制后续合成 mouseenter/focus，click 决定开关
      await infoBtn(wrapper).trigger('pointerdown', { pointerType: 'touch' })
      await infoBtn(wrapper).trigger('click')
      expect(hasTooltip(wrapper)).toBe(true)
      expect(tooltip(wrapper).text()).toContain('闲置回收：30 分钟')

      await infoBtn(wrapper).trigger('click')
      expect(hasTooltip(wrapper)).toBe(false)
    })

    it('点击外部关闭已展开的详情提示', async () => {
      const wrapper = mount(WorkspaceMachineSummary, { props: summaryProps })
      await infoBtn(wrapper).trigger('click')
      expect(hasTooltip(wrapper)).toBe(true)

      document.dispatchEvent(new Event('pointerdown', { bubbles: true }))
      await nextTick()
      expect(hasTooltip(wrapper)).toBe(false)
    })

    it('Esc 关闭已展开的详情提示', async () => {
      const wrapper = mount(WorkspaceMachineSummary, { props: summaryProps })
      await infoBtn(wrapper).trigger('click')
      expect(hasTooltip(wrapper)).toBe(true)

      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
      await nextTick()
      expect(hasTooltip(wrapper)).toBe(false)
    })

    it('click 锁定 hover 展开后鼠标移出不关闭，再次点击关闭', async () => {
      const wrapper = mount(WorkspaceMachineSummary, { props: summaryProps })
      await infoBtn(wrapper).trigger('mouseenter')
      expect(hasTooltip(wrapper)).toBe(true)

      // 点击把 hover 提示锁定为 click 模式，mouseleave 不关闭
      await infoBtn(wrapper).trigger('click')
      await infoBtn(wrapper).trigger('mouseleave')
      expect(hasTooltip(wrapper)).toBe(true)

      await infoBtn(wrapper).trigger('click')
      expect(hasTooltip(wrapper)).toBe(false)
    })

    it('点击过滤按钮向父组件抛出 machine-filter', async () => {
      const wrapper = mount(WorkspaceMachineSummary, {
        props: {
          machineSummary: {
            startedCount: 1,
            idleCount: 2,
            startingCount: 0,
            idleRecycleMinutes: 0,
          },
          machineRuntimeFilter: null,
        },
      })
      await wrapper.get('[data-alias="machine-filter-started"]').trigger('click')
      expect(wrapper.emitted('machine-filter')).toEqual([['started']])
      await wrapper.get('[data-alias="machine-filter-idle"]').trigger('click')
      expect(wrapper.emitted('machine-filter')).toEqual([['started'], ['idle']])
    })
  })
}
