// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetail.runtimeStatusPanelBridge.test.js requires vitest runtime')
} else {
/**
 * OPT-20260809-026: 执行细节「服务器运行状态」Tab 桥接回归测试
 *
 * 背景：ServerConfig.logic.vue 通过 defineExpose 暴露 serverRuntimeStatusPanel（computed ref）。
 * Vue 3 的模板 ref 访问 exposed 属性时经 getExposeProxy → proxyRefs 包装，ref 会被自动解包：
 * 父组件拿到的已经是对象本身，而非 ref。旧实现误读 panel.value（undefined）导致
 * typeof panel.value !== 'object' 恒真 → Tab 永远隐藏。
 *
 * 本测试验证：
 * 1. 暴露的 computed ref 经模板 ref 访问自动解包（旧代码误判的根因）
 * 2. 修复后的判定逻辑（对象直接返回）在解包语义下成立
 */
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { computed, defineComponent, nextTick, ref } = await import('vue')

/**
 * 模拟 ServerConfig.logic.vue：暴露 computed ref。
 * 注意：setup() 返回值与 defineExpose 的暴露对象一样，都会被 proxyRefs 包装——
 * 模板 ref 访问时 ref 自动解包（与真实链路 getExposeProxy 语义一致）。
 */
const ChildWithExposedPanel = defineComponent({
  setup() {
    const serverRuntimeStatusPanel = computed(() => ({
      serverRuntimeStatusDisplayText: '服务器运行中',
      showRuntimeActionButtons: true,
    }))
    return { serverRuntimeStatusPanel }
  },
  template: '<div data-testid="child" />',
})

/** 模拟 TaskDetail.vue 修复前的桥接判定（误读 .value） */
function brokenPickPanel(sc) {
  const panel = sc?.serverRuntimeStatusPanel
  if (!panel || typeof panel.value !== 'object' || !panel.value) return null
  return panel.value
}

/** 模拟 TaskDetail.vue 修复后的桥接判定（解包对象直接返回） */
function fixedPickPanel(sc) {
  const panel = sc?.serverRuntimeStatusPanel
  return panel && typeof panel === 'object' ? panel : null
}

const Parent = defineComponent({
  components: { ChildWithExposedPanel },
  setup() {
    const childRef = ref(null)
    return { childRef, pick: fixedPickPanel }
  },
  template: '<div><ChildWithExposedPanel ref="childRef" data-testid="parent" /><span data-testid="picked">{{ pick(childRef) && pick(childRef).serverRuntimeStatusDisplayText }}</span></div>',
})

describe('TaskDetail runtime status panel bridge (OPT-20260809-026)', () => {
  it('defineExpose 暴露的 computed ref 经模板 ref 访问被自动解包（非 ref 对象）', () => {
    const wrapper = mount(Parent)
    const sc = wrapper.vm.childRef
    const exposed = sc.serverRuntimeStatusPanel
    // 关键语义：exposed 是对象本身，不是 ref（无 .value）
    expect(exposed).toEqual({ serverRuntimeStatusDisplayText: '服务器运行中', showRuntimeActionButtons: true })
    expect(exposed.value).toBeUndefined()
  })

  it('旧判定误读 .value 会永远返回 null（bug 复现）', () => {
    const wrapper = mount(Parent)
    const sc = wrapper.vm.childRef
    expect(brokenPickPanel(sc)).toBeNull()
  })

  it('修复后判定直接返回解包对象', async () => {
    const wrapper = mount(Parent)
    const sc = wrapper.vm.childRef
    expect(fixedPickPanel(sc)).toEqual({
      serverRuntimeStatusDisplayText: '服务器运行中',
      showRuntimeActionButtons: true,
    })
    // 模板 ref 在父 mounted 阶段赋值，触发下一轮渲染 → 等待 nextTick
    await nextTick()
    // 模板链路：picked 渲染出显示文本
    expect(wrapper.get('[data-testid="picked"]').text()).toBe('服务器运行中')
  })

  it('serverConfigRef 未就绪时返回 null（向后兼容）', () => {
    expect(fixedPickPanel(null)).toBeNull()
    expect(fixedPickPanel(undefined)).toBeNull()
  })
})
}
