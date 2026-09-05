// @vitest-environment jsdom
/**
 * ServerConfigFeatureParamsBlock：智能体资源配置来源选择器
 */
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] ServerConfigFeatureParamsBlock.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: ServerConfigFeatureParamsBlock } = await import('./ServerConfigFeatureParamsBlock.vue')

describe('ServerConfigFeatureParamsBlock', () => {
  it('渲染智能体资源配置选择器（含未选占位）', () => {
    const wrapper = mount(ServerConfigFeatureParamsBlock, {
      props: { featureParamsSource: '' },
    })
    const blockText = wrapper.find('[data-testid="feature-params-block"]').text()
    expect(blockText).toContain('智能体资源配置')
    expect(blockText).not.toContain('环境变量参数')
    expect(wrapper.find('[data-testid="feature-params-source-selector"]').exists()).toBe(true)
    const options = wrapper.findAll('[data-testid="feature-params-source-selector"] option')
    expect(options.some((o) => o.element.value === '')).toBe(true)
    expect(wrapper.find('[data-testid="feature-params-source-required-hint"]').text())
      .toContain('请先选择智能体资源配置')
    expect(wrapper.find('[data-testid="feature-params-personal-config-selector"]').exists()).toBe(false)
  })

  it('已选来源时不展示未选提示', () => {
    const wrapper = mount(ServerConfigFeatureParamsBlock, {
      props: { featureParamsSource: 'company' },
    })
    expect(wrapper.find('[data-testid="feature-params-source-required-hint"]').exists()).toBe(false)
  })

  it('选择个人配置时展示二级下拉与预览按钮', async () => {
    const wrapper = mount(ServerConfigFeatureParamsBlock, {
      props: {
        featureParamsSource: 'personal',
        personalConfigs: [{ id: 'cfg-1', name: '我的配置' }],
        selectedPersonalConfigId: 'cfg-1',
      },
    })
    expect(wrapper.find('[data-testid="feature-params-personal-config-selector"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="feature-params-env-preview-btn"]').exists()).toBe(true)
  })

  it('showPreview=false 时不展示预览按钮', () => {
    const wrapper = mount(ServerConfigFeatureParamsBlock, {
      props: {
        featureParamsSource: 'personal',
        personalConfigs: [{ id: 'cfg-1', name: '我的配置' }],
        selectedPersonalConfigId: 'cfg-1',
        showPreview: false,
      },
    })
    expect(wrapper.find('[data-testid="feature-params-personal-config-selector"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="feature-params-env-preview-btn"]').exists()).toBe(false)
  })

  it('切换来源时发出 update 与 sourceChange', async () => {
    const wrapper = mount(ServerConfigFeatureParamsBlock, {
      props: { featureParamsSource: 'company' },
    })
    const select = wrapper.find('[data-testid="feature-params-source-selector"]')
    await select.setValue('workspace')
    expect(wrapper.emitted('update:featureParamsSource')?.[0]).toEqual(['workspace'])
    expect(wrapper.emitted('sourceChange')).toBeTruthy()
  })

  it('sourcesAvailable=false 时选择器禁用并展示暂无可用提示', () => {
    const wrapper = mount(ServerConfigFeatureParamsBlock, {
      props: { featureParamsSource: '', sourcesAvailable: false, tenantId: '875561774391259136' },
    })
    const select = wrapper.find('[data-testid="feature-params-source-selector"]')
    expect(select.attributes('disabled')).toBeDefined()
    const hint = wrapper.find('[data-testid="feature-params-source-unavailable-hint"]')
    expect(hint.exists()).toBe(true)
    expect(hint.text()).toContain('智能体资源配置')
    expect(hint.text()).not.toContain('功能参数')
    const company = wrapper.find('[data-testid="feature-params-company-settings-link"]')
    const workspace = wrapper.find('[data-testid="feature-params-workspace-settings-link"]')
    const personal = wrapper.find('[data-testid="feature-params-personal-settings-link"]')
    expect(company.element.tagName).toBe('A')
    expect(workspace.element.tagName).toBe('A')
    expect(personal.element.tagName).toBe('A')
    expect(company.attributes('href')).toBe('/tenant/875561774391259136/settings/feature-params/')
    expect(workspace.attributes('href')).toBe('/tenant/875561774391259136/settings/task-panel/')
    expect(personal.attributes('href')).toBe('/profile/feature-params/')
    expect(company.text()).toBe('公司')
    expect(workspace.text()).toBe('工作空间')
    expect(personal.text()).toBe('个人环境变量')
    expect(wrapper.find('[data-testid="feature-params-source-required-hint"]').exists()).toBe(false)
  })

  it('sourcesAvailable 默认 true 时选择器可点', () => {
    const wrapper = mount(ServerConfigFeatureParamsBlock, {
      props: { featureParamsSource: '' },
    })
    const select = wrapper.find('[data-testid="feature-params-source-selector"]')
    expect(select.attributes('disabled')).toBeUndefined()
    expect(wrapper.find('[data-testid="feature-params-source-unavailable-hint"]').exists()).toBe(false)
  })

  it('sourcesAvailable=true 且已选来源时选择器可点', () => {
    const wrapper = mount(ServerConfigFeatureParamsBlock, {
      props: { featureParamsSource: 'company', sourcesAvailable: true },
    })
    expect(wrapper.find('[data-testid="feature-params-source-selector"]').attributes('disabled'))
      .toBeUndefined()
  })

  it('persistError 非空时在镜像区展示可读错误', () => {
    const wrapper = mount(ServerConfigFeatureParamsBlock, {
      props: {
        featureParamsSource: 'company',
        persistError: '仅任务 Owner 可修改功能参数配置',
      },
    })
    const err = wrapper.find('[data-testid="feature-params-persist-error"]')
    expect(err.exists()).toBe(true)
    expect(err.attributes('role')).toBe('alert')
    expect(err.text()).toContain('仅任务 Owner 可修改功能参数配置')
  })

  it('无 persistError 时不展示错误区', () => {
    const wrapper = mount(ServerConfigFeatureParamsBlock, {
      props: { featureParamsSource: 'company', persistError: '' },
    })
    expect(wrapper.find('[data-testid="feature-params-persist-error"]').exists()).toBe(false)
  })
})
}
