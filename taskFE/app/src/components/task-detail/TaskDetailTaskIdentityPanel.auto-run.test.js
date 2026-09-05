// @vitest-environment jsdom
/**
 * TaskDetailTaskIdentityPanel：是否自动运行只读展示
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailTaskIdentityPanel.auto-run.test.js requires vitest runtime')
  process.exit(0)
}
const { describe, it, expect, vi } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { createRouter, createMemoryHistory } = await import('vue-router')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(async () => ({
      ok: true,
      json: async () => ({ current_deliverable_objs: [] }),
    })),
  }))

  const { default: TaskDetailTaskIdentityPanel } = await import('./TaskDetailTaskIdentityPanel.vue')

  async function mountPanel(taskOverrides = {}) {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }],
    })
    await router.push('/')
    await router.isReady()
    return mount(TaskDetailTaskIdentityPanel, {
      props: {
        task: {
          id: 'task_1',
          title: 't',
          description: '',
          priority: 1,
          created_at: '2026-07-13T00:00:00Z',
          auto_run: false,
          ...taskOverrides,
        },
        forkSourceTaskRoute: '/',
        progressStatusOptions: [],
      },
      global: {
        plugins: [router],
        stubs: {
          MarkdownContent: true,
          TaskDetailLlmBudgetPanel: true,
          TagMemberInput: true,
        },
      },
    })
  }

  describe('TaskDetailTaskIdentityPanel auto_run readonly', () => {
    it('auto_run=true 时只读展示「是」', async () => {
      const wrapper = await mountPanel({ auto_run: true })
      const el = wrapper.find('[data-testid="task-auto-run-readonly"]')
      expect(el.exists()).toBe(true)
      expect(el.text()).toContain('是否自动运行')
      expect(el.text()).toContain('是')
      expect(el.find('input').exists()).toBe(false)
      expect(el.find('select').exists()).toBe(false)
    })

    it('auto_run=false 时只读展示「否」', async () => {
      const wrapper = await mountPanel({ auto_run: false })
      const el = wrapper.find('[data-testid="task-auto-run-readonly"]')
      expect(el.text()).toContain('否')
      expect(el.find('input[type="checkbox"]').exists()).toBe(false)
    })

    it('auto_run=true 时自动运行说明默认收起', async () => {
      const wrapper = await mountPanel({ auto_run: true })
      const steps = wrapper.find('[data-testid="task-detail-auto-run-steps"]')
      expect(steps.exists()).toBe(true)
      const toggle = steps.find('[data-testid="auto-run-steps-toggle"]')
      expect(toggle.exists()).toBe(true)
      expect(toggle.text()).toBe('查看自动运行说明')
      expect(steps.find('[data-testid="auto-run-steps-body"]').exists()).toBe(false)
    })
  })
