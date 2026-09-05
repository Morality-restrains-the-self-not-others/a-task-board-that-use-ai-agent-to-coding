// @vitest-environment jsdom
/**
 * 评论区 $镜像 后输入 / 的技能下拉：image_skills 数据管线 + 编辑器行为。
 * 回归目标：composer 映射剥离 image_skills 导致 mentionedSkills 恒空、技能下拉永不打开。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentComposer.skillMenu.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')

  const { mockApiFetch } = vi.hoisted(() => ({ mockApiFetch: vi.fn() }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: mockApiFetch,
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ path: '/t', query: {}, hash: '' }),
    useRouter: () => ({ replace: () => {} }),
  }))
  vi.mock('../../utils/sessionUserIdUtils.js', () => ({
    resolveAuthenticatedUserId: async () => 'u1',
  }))
  vi.mock('../../utils/githubAppReturnStorage.js', () => ({
    createGithubAppReturnKey: () => 'a'.repeat(32),
    setGithubAppReturnTarget: () => {},
  }))

  const { default: TaskDetailCommentComposer } = await import('./TaskDetailCommentComposer.vue')

  const INSTALLED_WITH_SKILLS = [
    {
      id: 'img-a',
      name: 'trae-agent',
      version: 'x86_64-latest',
      image_skills: {
        version: 1,
        default_skill: 'general-coding',
        skills: [
          { name: 'general-coding', description: '通用编码', is_default: true },
          { name: 'k8s-debug', description: 'K8s 调试' },
        ],
      },
      image_skills_extract_status: 'ok',
    },
    {
      id: 'img-b',
      name: 'other',
      version: '1',
      image_skills: null,
      image_skills_extract_status: 'pending',
    },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
    mockApiFetch.mockImplementation(async (url) => {
      if (String(url).includes('workspaces')) {
        return {
          ok: true,
          json: async () => ({ container_image_at_mode_enabled: true }),
        }
      }
      if (String(url).includes('installed-images')) {
        return { ok: true, json: async () => INSTALLED_WITH_SKILLS }
      }
      return { ok: true, json: async () => ({}) }
    })
  })

  const composerStubs = {
    CommentExecutionDependencyPicker: true,
    ServerConfigImageSectionHints: true,
    CommentComposerHardwareCard: true,
  }

  describe('评论区 $镜像 技能下拉（image_skills 管线）', () => {
    it('传入编辑器的 installedImages 保留 image_skills 与提取状态（根因回归）', async () => {
      let mentionProps = null
      const wrapper = mount(TaskDetailCommentComposer, {
        props: {
          modelValue: 'hello',
          'onUpdate:modelValue': () => {},
          tenantId: 't1',
          workspaceId: 'w1',
          showDependencyPicker: false,
        },
        global: {
          stubs: {
            CommentImageMentionEditor: {
              template: '<div data-testid="comment-image-mention-editor" />',
              props: ['modelValue', 'installedImages', 'placeholder'],
              emits: ['mention-change', 'keydown', 'update:modelValue'],
              mounted() { mentionProps = this.$props },
            },
            ...composerStubs,
          },
        },
      })

      await flushPromises()
      await nextTick()

      expect(mentionProps).not.toBeNull()
      const first = mentionProps.installedImages.find((x) => x.id === 'img-a')
      expect(first).toBeTruthy()
      expect(first.image_skills).toEqual(INSTALLED_WITH_SKILLS[0].image_skills)
      expect(first.image_skills_extract_status).toBe('ok')
      // 无技能的镜像原样保留状态字段（pending 分支仍可透出）
      const second = mentionProps.installedImages.find((x) => x.id === 'img-b')
      expect(second.image_skills_extract_status).toBe('pending')

      wrapper.unmount()
    })

    it('$镜像 空格确认后输入 / 弹出技能下拉（端到端）', async () => {
      const wrapper = mount(TaskDetailCommentComposer, {
        props: {
          modelValue: '',
          'onUpdate:modelValue': () => {},
          tenantId: 't1',
          workspaceId: 'w1',
          showDependencyPicker: false,
        },
        global: { stubs: composerStubs },
      })

      await flushPromises()
      await nextTick()

      const editor = wrapper.find('[data-testid="comment-content-editor"]')
      expect(editor.exists()).toBe(true)

      // 输入 $trae-agent → 镜像下拉出现
      editor.element.innerHTML = '$trae-agent'
      await editor.trigger('input')
      await nextTick()
      const picker = wrapper.find('[data-testid="comment-image-mention-picker"]')
      expect(picker.exists()).toBe(true)

      // 选中镜像 → 正文生成 mention chip 并补空格
      await picker.findAll('li')[0].trigger('mousedown')
      await nextTick()
      await nextTick()
      expect(editor.element.querySelector('[data-mention-id]')).toBeTruthy()

      // 追加 / → 技能下拉必须出现
      editor.element.appendChild(document.createTextNode('/'))
      await editor.trigger('input')
      await nextTick()
      const skillPicker = wrapper.find('[data-testid="comment-image-skill-picker"]')
      expect(skillPicker.exists()).toBe(true)
      const skillItems = skillPicker.findAll('li').map((li) => li.text())
      expect(skillItems.join('|')).toContain('/general-coding')
      expect(skillItems.join('|')).toContain('/k8s-debug')

      wrapper.unmount()
    })
  })
}
