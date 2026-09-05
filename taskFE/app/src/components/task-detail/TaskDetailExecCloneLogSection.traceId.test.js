// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailExecCloneLogSection.traceId.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: TaskDetailExecCloneLogSection } = await import('./TaskDetailExecCloneLogSection.vue')

describe('TaskDetailExecCloneLogSection data-traceId', () => {
  it('sets data-traceId on clone-log fetch error element when traceId present', () => {
    const wrapper = mount(TaskDetailExecCloneLogSection, {
      props: {
        layerId: 'layer-1',
        cloneLogFetchError: 'Invalid or missing access token',
        cloneLogFetchErrorTraceId: 'tid-clone-401',
        cloneLogText: '',
      },
    })
    const err = wrapper.find('p.text-xs.text-red-600')
    expect(err.exists()).toBe(true)
    expect(err.text()).toBe('Invalid or missing access token')
    // jsdom 会将 HTML 属性名小写化；选择器 [data-traceId] 在浏览器中仍可用
    expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe('tid-clone-401')
    wrapper.unmount()
  })

  it('omits data-traceId when traceId is empty', () => {
    const wrapper = mount(TaskDetailExecCloneLogSection, {
      props: {
        layerId: 'layer-1',
        cloneLogFetchError: 'Invalid or missing access token',
        cloneLogFetchErrorTraceId: '',
        cloneLogText: '',
      },
    })
    const err = wrapper.find('p.text-xs.text-red-600')
    expect(err.exists()).toBe(true)
    expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBeUndefined()
    wrapper.unmount()
  })
})

}
