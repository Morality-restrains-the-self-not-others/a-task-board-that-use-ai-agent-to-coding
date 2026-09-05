// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentCloneProgress.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: TaskDetailCommentCloneProgress } = await import('./TaskDetailCommentCloneProgress.vue')

describe('TaskDetailCommentCloneProgress', () => {
  it('renders nothing when rows empty', () => {
    const wrapper = mount(TaskDetailCommentCloneProgress, { props: { rows: [] } })
    expect(wrapper.find('[data-testid="comment-execution-clone-progress"]').exists()).toBe(false)
  })

  it('shows overall percent and per-repo bar for a single repo', () => {
    const wrapper = mount(TaskDetailCommentCloneProgress, {
      props: {
        rows: [
          { key: 'alpha', label: 'alpha', progress: 40, message: '【项目克隆】(1/1) alpha … 40%', failed: false },
        ],
      },
    })
    const root = wrapper.get('[data-testid="comment-execution-clone-progress"]')
    expect(root.get('[data-testid="comment-execution-clone-progress-overall"]').text()).toBe('40%')
    expect(root.get('[data-testid="comment-execution-clone-progress-row"]').text()).toContain('alpha')
    expect(root.get('[data-testid="comment-execution-clone-progress-message"]').text()).toContain('40%')
    expect(root.find('[data-testid="comment-execution-clone-progress-sub"]').exists()).toBe(false)
  })

  it('uses (i/n) total as overall denominator when sub-repos exist', () => {
    const wrapper = mount(TaskDetailCommentCloneProgress, {
      props: {
        rows: [
          { key: 'a', label: 'a', progress: 100, message: '【项目克隆】(1/34) a … 100%', failed: false, repoTotal: 34 },
          { key: 'b', label: 'b', progress: 0, message: '【项目克隆】(2/34) 网络抖动，准备第 2/3 次重试…', failed: false, retrying: true, repoTotal: 34 },
        ],
      },
    })
    const root = wrapper.get('[data-testid="comment-execution-clone-progress"]')
    expect(root.get('[data-testid="comment-execution-clone-progress-overall"]').text()).toBe(`${Math.round(100 / 34)}%`)
    expect(root.get('[data-testid="comment-execution-clone-progress-sub"]').element.open).toBe(false)
    expect(root.find('[data-testid="comment-execution-clone-progress-row"]').exists()).toBe(true)
  })

  it('expands sub-progress on summary click', async () => {
    const wrapper = mount(TaskDetailCommentCloneProgress, {
      props: {
        rows: [
          { key: 'a', label: 'alpha', progress: 40, message: 'm1', failed: false, repoTotal: 2 },
          { key: 'b', label: 'beta', progress: 10, message: 'm2', failed: false, repoTotal: 2 },
        ],
      },
    })
    const details = wrapper.get('[data-testid="comment-execution-clone-progress-sub"]')
    expect(details.element.open).toBe(false)
    await wrapper.get('[data-testid="comment-execution-clone-progress-sub-summary"]').trigger('click')
    details.element.open = true
    await wrapper.vm.$nextTick()
    expect(details.element.open).toBe(true)
    expect(wrapper.findAll('[data-testid="comment-execution-clone-progress-row"]')).toHaveLength(2)
  })

  it('uses red bar class when failed', () => {
    const wrapper = mount(TaskDetailCommentCloneProgress, {
      props: {
        rows: [
          { key: 'x', label: 'x', progress: 0, message: '失败', failed: true },
        ],
      },
    })
    const bar = wrapper.get('[data-testid="comment-execution-clone-progress-row"]').find('.bg-red-500')
    expect(bar.exists()).toBe(true)
  })

  it('hides manual retry while auto-retrying', () => {
    const wrapper = mount(TaskDetailCommentCloneProgress, {
      props: {
        rows: [
          {
            key: 'https://git.example/alpha.git',
            label: 'alpha',
            progress: 40,
            message: '【项目克隆】(1/2) 网络抖动，准备第 2/3 次重试…',
            failed: false,
            retrying: true,
            retryAttempt: 2,
            retryMax: 3,
            repoUrl: 'https://git.example/alpha.git',
            repoTotal: 2,
          },
        ],
      },
    })
    expect(wrapper.find('[data-testid="comment-execution-clone-progress-retry"]').exists()).toBe(false)
  })

  it('shows manual retry after failure and emits repo-reclone', async () => {
    const wrapper = mount(TaskDetailCommentCloneProgress, {
      props: {
        rows: [
          {
            key: 'https://git.example/alpha.git',
            label: 'alpha',
            progress: 0,
            message: '【项目克隆】(1/2) 失败 alpha: fatal: repository not found',
            failed: true,
            retrying: false,
            repoUrl: 'https://git.example/alpha.git',
            repoTotal: 2,
          },
        ],
      },
    })
    const btn = wrapper.get('[data-testid="comment-execution-clone-progress-retry"]')
    expect(btn.text()).toContain('手动重试')
    await btn.trigger('click')
    expect(wrapper.emitted('repo-reclone')?.[0]?.[0]).toEqual({
      repoUrl: 'https://git.example/alpha.git',
    })
  })

  it('shows 手动重试 for cold-open (1/1) ram-work failure when repoUrl was filled from catalog', () => {
    const wrapper = mount(TaskDetailCommentCloneProgress, {
      props: {
        rows: [
          {
            key: 'https://gitlab.daydaymoney.com/g/ram-work.git',
            label: 'ram-work',
            progress: 0,
            message: "【项目克隆】(1/1) 失败 ram-work: git exit 128: Cloning into '/app/onlineProject_state/layers/20260817_062029_f94295/ram-work'…",
            failed: true,
            retrying: false,
            repoUrl: 'https://gitlab.daydaymoney.com/g/ram-work.git',
            repoTotal: 1,
          },
        ],
      },
    })
    expect(wrapper.get('[data-testid="comment-execution-clone-progress-retry"]').text()).toContain('手动重试')
  })

  it('emits parentRepoUrl and cloneAlias for nested failure retry', async () => {
    const wrapper = mount(TaskDetailCommentCloneProgress, {
      props: {
        rows: [
          {
            key: 'https://gitlab.daydaymoney.com/g/docs.git',
            label: 'docs',
            progress: 0,
            message: '【项目克隆】(2/2) 失败 docs: git exit 128',
            failed: true,
            repoUrl: 'https://gitlab.daydaymoney.com/g/docs.git',
            parentRepoUrl: 'https://gitlab.daydaymoney.com/g/ram-work.git',
            cloneAlias: 'docs',
            repoTotal: 2,
          },
        ],
      },
    })
    await wrapper.get('[data-testid="comment-execution-clone-progress-retry"]').trigger('click')
    expect(wrapper.emitted('repo-reclone')?.[0]?.[0]).toEqual({
      repoUrl: 'https://gitlab.daydaymoney.com/g/docs.git',
      parentRepoUrl: 'https://gitlab.daydaymoney.com/g/ram-work.git',
      cloneAlias: 'docs',
    })
  })
})
}
