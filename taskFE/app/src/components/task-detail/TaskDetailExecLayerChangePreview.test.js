// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailExecLayerChangePreview.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: TaskDetailExecLayerChangePreview } = await import('./TaskDetailExecLayerChangePreview.vue')

describe('TaskDetailExecLayerChangePreview current branch', () => {
  it('T4: 仓库根且有 currentBranch 时标题旁显示分支', () => {
    const wrapper = mount(TaskDetailExecLayerChangePreview, {
      props: {
        previewKind: 'git',
        selectedPath: 'repo-a',
        loading: false,
        error: '',
        gitLogText: 'abc 2026-01-01 init',
        isRepoRoot: true,
        currentBranch: 'feature/mock-work',
      },
    })
    expect(wrapper.text()).toContain('提交日志')
    const branchEl = wrapper.find('[data-testid="git-log-current-branch"]')
    expect(branchEl.exists()).toBe(true)
    expect(branchEl.text()).toBe('· feature/mock-work')
  })

  it('T5: 非仓库根不显示分支', () => {
    const wrapper = mount(TaskDetailExecLayerChangePreview, {
      props: {
        previewKind: 'git',
        selectedPath: 'repo-a/src',
        loading: false,
        error: '',
        gitLogText: 'abc 2026-01-01 init',
        isRepoRoot: false,
        currentBranch: '',
      },
    })
    expect(wrapper.text()).toContain('提交日志')
    expect(wrapper.find('[data-testid="git-log-current-branch"]').exists()).toBe(false)
  })

  it('sets data-traceId on preview error when errorTraceId present', () => {
    const wrapper = mount(TaskDetailExecLayerChangePreview, {
      props: {
        previewKind: 'git',
        selectedPath: 'ram-work',
        loading: false,
        error: 'Invalid or missing access token',
        errorTraceId: 'tid-git-log-401',
        gitLogText: '',
      },
    })
    const err = wrapper.find('[data-testid="layer-change-preview-error"]')
    expect(err.exists()).toBe(true)
    expect(err.text()).toBe('Invalid or missing access token')
    expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe('tid-git-log-401')
    wrapper.unmount()
  })

  it('omits data-traceId on preview error when errorTraceId empty', () => {
    const wrapper = mount(TaskDetailExecLayerChangePreview, {
      props: {
        previewKind: 'git',
        selectedPath: 'ram-work',
        loading: false,
        error: 'Invalid or missing access token',
        errorTraceId: '',
        gitLogText: '',
      },
    })
    const err = wrapper.find('[data-testid="layer-change-preview-error"]')
    expect(err.exists()).toBe(true)
    expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBeUndefined()
    wrapper.unmount()
  })

  it('T5: kind=binary 显示文件属性，不渲染文本 pre', () => {
    const wrapper = mount(TaskDetailExecLayerChangePreview, {
      props: {
        previewKind: 'file',
        selectedPath: 'ram-work/jre/lib/modules',
        loading: false,
        error: '',
        payload: {
          kind: 'binary',
          path: 'ram-work/jre/lib/modules',
          basename: 'modules',
          size_bytes: 1024,
          size_human: '1.0 KB',
          mtime_iso: '2026-07-20T00:00:00.000Z',
          ext: '',
        },
      },
    })
    expect(wrapper.text()).toContain('文件属性')
    const propsEl = wrapper.find('[data-testid="layer-change-preview-binary-props"]')
    expect(propsEl.exists()).toBe(true)
    expect(propsEl.text()).toContain('二进制')
    expect(propsEl.text()).toContain('1.0 KB')
    expect(wrapper.find('[data-testid="layer-change-preview-text"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('旧 API content 含 NUL 时按二进制属性展示', () => {
    const wrapper = mount(TaskDetailExecLayerChangePreview, {
      props: {
        previewKind: 'file',
        selectedPath: 'jre/lib/modules',
        loading: false,
        error: '',
        payload: { path: 'jre/lib/modules', content: 'jIMG\0\x01\x02', truncated: false },
      },
    })
    expect(wrapper.text()).toContain('文件属性')
    expect(wrapper.find('[data-testid="layer-change-preview-binary-props"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="layer-change-preview-text"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('T6: 文本 payload 显示内容 pre', () => {
    const wrapper = mount(TaskDetailExecLayerChangePreview, {
      props: {
        previewKind: 'file',
        selectedPath: 'README.md',
        loading: false,
        error: '',
        payload: { kind: 'text', content: '# hello' },
      },
    })
    expect(wrapper.text()).toContain('文件内容预览')
    const pre = wrapper.find('[data-testid="layer-change-preview-text"]')
    expect(pre.exists()).toBe(true)
    expect(pre.text()).toBe('# hello')
    expect(wrapper.find('[data-testid="layer-change-preview-binary-props"]').exists()).toBe(false)
    wrapper.unmount()
  })
})

}
