// @vitest-environment jsdom
// 创建任务弹窗元字段：「已安装镜像」字段（#task-container-image）已下线，
// 镜像/技能选择收敛到任务描述 @ 弹层（TaskDescriptionSkillField）。
// 此处守卫该字段不再渲染，并验证 feature_params / priority / due_date 仍按字段设置渲染。
if (!process.env.VITEST) {
  console.log('[skip] CreateTaskMetaFields.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  const apiFetchMock = vi.hoisted(() => vi.fn())
  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: apiFetchMock,
    extractErrorMessage: vi.fn(() => ''),
  }))
  vi.mock('../composables/useCreateTaskFeatureParams.js', () => ({
    useCreateTaskFeatureParams: () => ({
      personalConfigs: { value: [] },
      resolvedEnvPreview: { value: '' },
      isEnvPreviewLoading: { value: false },
      envPreviewExpanded: { value: false },
      featureParamsSourcesAvailable: { value: [] },
      featureParamsSourceModel: { value: 'none' },
      selectedPersonalConfigIdModel: { value: '' },
      onFeatureParamsSourceUpdate: () => {},
      onPersonalConfigIdUpdate: () => {},
      onFeatureParamsSourceChange: () => {},
      fetchCreateTaskEnvPreview: () => {},
      syncFeatureParamsFromEditingTask: () => {},
    }),
  }))

  const { default: CreateTaskMetaFields } = await import('./CreateTaskMetaFields.vue')

  const baseProps = {
    editingTask: { id: '', container_image: { id: '' }, priority: 'medium', due_date: '' },
    fieldSettings: { feature_params: false, priority: true, due_date: true },
    currentWorkspace: {},
    tenantId: 't1',
    show: true,
  }

  describe('CreateTaskMetaFields 已安装镜像字段下线守卫', () => {
    it('不再渲染 #task-container-image 输入框与「已安装镜像」标签（即使字段设置开启）', () => {
      apiFetchMock.mockReset()
      const wrapper = mount(CreateTaskMetaFields, {
        props: {
          ...baseProps,
          editingTask: { id: '', container_image: { id: 'img-1' }, priority: 'medium', due_date: '' },
          fieldSettings: { container_image: true, feature_params: false, priority: true, due_date: true },
        },
        global: { stubs: { ServerConfigFeatureParamsBlock: true } },
      })
      expect(wrapper.find('#task-container-image').exists()).toBe(false)
      expect(wrapper.find('label[for="task-container-image"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('已安装镜像')
      wrapper.unmount()
    })

    it('不再渲染镜像加载/错误占位（无 #task-container-image 相关容器）', () => {
      apiFetchMock.mockReset()
      const wrapper = mount(CreateTaskMetaFields, {
        props: {
          ...baseProps,
          editingTask: { id: '', container_image: { id: '' }, priority: 'medium', due_date: '' },
        },
        global: { stubs: { ServerConfigFeatureParamsBlock: true } },
      })
      expect(wrapper.find('input[id="task-container-image"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('无可用已安装镜像')
      wrapper.unmount()
    })

    it('feature_params 按字段设置渲染/隐藏智能体资源配置块', () => {
      apiFetchMock.mockReset()
      const enabled = mount(CreateTaskMetaFields, {
        props: {
          ...baseProps,
          editingTask: { id: '', priority: 'medium', due_date: '' },
          fieldSettings: { feature_params: true, priority: false, due_date: false },
        },
        global: { stubs: { ServerConfigFeatureParamsBlock: true } },
      })
      expect(enabled.findComponent({ name: 'ServerConfigFeatureParamsBlock' }).exists()).toBe(true)
      enabled.unmount()

      const disabled = mount(CreateTaskMetaFields, {
        props: {
          ...baseProps,
          fieldSettings: { feature_params: false, priority: false, due_date: false },
        },
        global: { stubs: { ServerConfigFeatureParamsBlock: true } },
      })
      expect(disabled.findComponent({ name: 'ServerConfigFeatureParamsBlock' }).exists()).toBe(false)
      disabled.unmount()
    })

    it('priority / due_date 仍按字段设置渲染', () => {
      apiFetchMock.mockReset()
      const wrapper = mount(CreateTaskMetaFields, {
        props: {
          ...baseProps,
          editingTask: { id: '', priority: 'medium', due_date: '' },
        },
        global: { stubs: { ServerConfigFeatureParamsBlock: true } },
      })
      expect(wrapper.find('#task-priority').exists()).toBe(true)
      expect(wrapper.find('#task-deadline').exists()).toBe(true)

      const hidden = mount(CreateTaskMetaFields, {
        props: {
          ...baseProps,
          editingTask: { id: '', priority: 'medium', due_date: '' },
          fieldSettings: { priority: false, due_date: false },
        },
        global: { stubs: { ServerConfigFeatureParamsBlock: true } },
      })
      expect(hidden.find('#task-priority').exists()).toBe(false)
      expect(hidden.find('#task-deadline').exists()).toBe(false)
      hidden.unmount()
      wrapper.unmount()
    })
  })
}
