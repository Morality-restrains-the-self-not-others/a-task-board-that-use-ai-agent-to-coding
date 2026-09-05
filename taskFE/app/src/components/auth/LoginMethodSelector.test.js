// @vitest-environment jsdom
// LoginMethodSelector 组件级回归（OPT-20260824-055）：登录方式 Tab 渲染集合受
// allowPhoneLogin 门控，切换必须同时发 update:modelValue + method-changed 双事件，
// 防止「手机号/密码」验证码登录 Tab 在登录方式变更时复活。
if (!process.env.VITEST) {
  console.log('[skip] LoginMethodSelector.test.js requires vitest runtime')
} else {
const { mount } = await import('@vue/test-utils')
const { describe, expect, it, afterEach } = await import('vitest')
const { nextTick } = await import('vue')

const { default: LoginMethodSelector } = await import('./LoginMethodSelector.vue')

const mountSelector = (props = {}) =>
  mount(LoginMethodSelector, {
    props: {
      modelValue: 'emailPassword',
      allowPhoneLogin: false,
      ...props,
    },
    attachTo: document.body,
  })

const tabTexts = (wrapper) => wrapper.findAll('button').map((b) => b.text().trim())

const activeTab = (wrapper) =>
  wrapper.findAll('button').find((b) => b.classes().includes('bg-primary'))

const clickTab = async (wrapper, label) => {
  const btn = wrapper.findAll('button').find((b) => b.text().includes(label))
  expect(btn, `按钮「${label}」应存在`).toBeTruthy()
  await btn.trigger('click')
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('LoginMethodSelector — Tab 渲染集合与切换事件（OPT-20260824-055）', () => {
  it('allowPhoneLogin=false 时只渲染 邮箱/密码 与 访问令牌，无 手机号/密码', () => {
    const wrapper = mountSelector()
    const texts = tabTexts(wrapper)
    expect(texts).toContain('邮箱/密码')
    expect(texts).toContain('访问令牌')
    expect(texts).not.toContain('手机号/密码')
  })

  it('allowPhoneLogin=true 时渲染 邮箱/密码、手机号/密码、访问令牌 三个 Tab', () => {
    const wrapper = mountSelector({ allowPhoneLogin: true })
    expect(tabTexts(wrapper)).toEqual(['邮箱/密码', '手机号/密码', '访问令牌'])
  })

  it('默认 modelValue=emailPassword 时高亮 邮箱/密码（bg-primary 类）', () => {
    const wrapper = mountSelector()
    const active = activeTab(wrapper)
    expect(active).toBeTruthy()
    expect(active.text()).toContain('邮箱/密码')
  })

  it('点击 访问令牌 → update:modelValue 与 method-changed 均为 accessToken，高亮切换', async () => {
    const wrapper = mountSelector()
    await clickTab(wrapper, '访问令牌')
    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')[0]).toEqual(['accessToken'])
    expect(wrapper.emitted('method-changed')[0]).toEqual(['accessToken'])
    expect(activeTab(wrapper).text()).toContain('访问令牌')
  })

  it('allowPhoneLogin=true 时点击 手机号/密码 → 双事件 phonePassword', async () => {
    const wrapper = mountSelector({ allowPhoneLogin: true })
    await clickTab(wrapper, '手机号/密码')
    expect(wrapper.emitted('update:modelValue')[0]).toEqual(['phonePassword'])
    expect(wrapper.emitted('method-changed')[0]).toEqual(['phonePassword'])
  })

  it('modelValue prop 变化时经 watch 同步高亮（父级受控路径）', async () => {
    const wrapper = mountSelector({ allowPhoneLogin: true, modelValue: 'accessToken' })
    expect(activeTab(wrapper).text()).toContain('访问令牌')

    await wrapper.setProps({ modelValue: 'emailPassword' })
    await nextTick()
    expect(activeTab(wrapper).text()).toContain('邮箱/密码')
  })
})
}
