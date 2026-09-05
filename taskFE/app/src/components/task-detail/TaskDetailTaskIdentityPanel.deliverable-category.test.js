// @vitest-environment jsdom
/**
 * TaskDetailTaskIdentityPanel：交付物类别显示与编辑态修改
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailTaskIdentityPanel.deliverable-category.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { createRouter, createMemoryHistory } = await import('vue-router')
  const { reactive } = await import('vue')

  const { default: TaskDetailTaskIdentityPanel } = await import('./TaskDetailTaskIdentityPanel.vue')

  const categoryOptions = [
    { id: 'd1', name: '价值流', order: 1 },
    { id: 'd2', name: '活动', order: 2 },
  ]

  async function mountPanel({ isEditing = false, taskOverrides = {}, editingTask = null, workspaceTodos = [] } = {}) {
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
          deliverable_obj_id: 'd2',
          ...taskOverrides,
        },
        isEditing,
        editingTask,
        forkSourceTaskRoute: '/',
        progressStatusOptions: [],
        deliverableCategoryOptions: categoryOptions,
        workspaceTodos,
        isDeliverableCategoriesLoading: false,
        deliverableCategoryError: '',
      },
      global: {
        plugins: [router],
        stubs: {
          MarkdownContent: true,
          TaskDetailLlmBudgetPanel: true,
          TagMemberInput: true,
          AutoRunStepsPreview: true,
        },
      },
    })
  }

  describe('TaskDetailTaskIdentityPanel deliverable category', () => {
    it('只读态展示当前交付物类别名称', async () => {
      const wrapper = await mountPanel()
      const root = wrapper.find('[data-testid="task-deliverable-category"]')
      expect(root.exists()).toBe(true)
      expect(root.text()).toContain('交付物类别')
      expect(root.text()).toContain('活动')
      expect(root.find('select').exists()).toBe(false)
    })

    it('编辑态展示下拉并可修改 deliverable_obj_id', async () => {
      const editingTask = reactive({
        priority: 1,
        deliverable_obj_id: 'd2',
        parent_task: '',
        owner: '',
        assignees: [],
      })
      const wrapper = await mountPanel({ isEditing: true, editingTask })
      const root = wrapper.find('[data-testid="task-deliverable-category"]')
      const select = root.find('select')
      expect(select.exists()).toBe(true)
      expect(select.element.value).toBe('d2')
      const texts = select.findAll('option').map((o) => o.text())
      expect(texts).toContain('价值流')
      expect(texts).toContain('活动')
      expect(texts).toContain('未分类')
      await select.setValue('d1')
      expect(editingTask.deliverable_obj_id).toBe('d1')
    })

    it('编辑非顶层类别时展示上层交付物下拉且候选仅上一层', async () => {
      const editingTask = reactive({
        priority: 1,
        deliverable_obj_id: 'd2',
        parent_task: '',
        owner: '',
        assignees: [],
      })
      const wrapper = await mountPanel({
        isEditing: true,
        editingTask,
        workspaceTodos: [
          { id: 'p1', title: '价值流A', deliverable_obj_id: 'd1' },
          { id: 'p2', title: '活动B', deliverable_obj_id: 'd2' },
        ],
      })
      const parentField = wrapper.find('[data-testid="create-task-parent-deliverable"]')
      expect(parentField.exists()).toBe(true)
      const optionTexts = parentField.findAll('option').map((o) => o.text())
      expect(optionTexts.some((t) => t.includes('价值流A'))).toBe(true)
      expect(optionTexts.some((t) => t.includes('活动B'))).toBe(false)
    })

    it('编辑顶层类别时不展示上层交付物下拉', async () => {
      const editingTask = reactive({
        priority: 1,
        deliverable_obj_id: 'd1',
        parent_task: '',
        owner: '',
        assignees: [],
      })
      const wrapper = await mountPanel({ isEditing: true, editingTask })
      expect(wrapper.find('[data-testid="create-task-parent-deliverable"]').exists()).toBe(false)
    })
  })
}
