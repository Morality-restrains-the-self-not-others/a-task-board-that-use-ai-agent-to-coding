// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDescriptionSkillField.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  const { default: TaskDescriptionSkillField } = await import('./TaskDescriptionSkillField.vue')

  const IMAGES = [
    { id: 'img-1', name: 'trae-agent', version: 'x86_64-latest' },
    { id: 'img-2', name: 'python-env', version: '3.11' },
    {
      id: 'img-3',
      name: 'agent-dev',
      version: 'latest',
      image_skills: { skills: [{ name: 'dev', description: '开发' }, { name: 'debug' }, { name: 'test' }] },
    },
  ]

  // 同名多变体（生产环境 trae-agent 两个版本，private 无技能）
  const DUP_IMAGES = [
    { id: 'dup-1', name: 'trae-agent', version: 'private_x86_64-latest' },
    {
      id: 'dup-2',
      name: 'trae-agent',
      version: 'x86_64-latest',
      image_skills: { skills: [{ name: 'general-coding', description: '通用编码' }, { name: 'k8s-debug' }] },
    },
  ]

  const mountField = (props = {}) =>
    mount(TaskDescriptionSkillField, {
      props: { installedImages: IMAGES, image: null, modelValue: '', ...props },
    })

  const area = (wrapper) => wrapper.find('textarea')
  const setText = async (wrapper, value) => {
    await area(wrapper).setValue(value)
  }
  const lastMention = (wrapper) => {
    const calls = wrapper.emitted('mention-change')
    return calls && calls.length ? calls[calls.length - 1][0] : null
  }

  describe('TaskDescriptionSkillField 描述内 $镜像 选择', () => {
    it('输入 $ 后弹出镜像列表（带版本标签），正文不匹配项不出现', async () => {
      const wrapper = mountField()
      await setText(wrapper, '$trae')
      expect(wrapper.find('[data-testid="task-description-image-mention-picker"]').exists()).toBe(true)
      const labels = wrapper.findAll('[data-testid="task-description-image-mention-picker"] li').map((l) => l.text())
      expect(labels).toContain('trae-agent:x86_64-latest')
      expect(labels).not.toContain('python-env:3.11')
      wrapper.unmount()
    })

    it('镜像选项同时展示版本标签、更新时间与说明（无元信息时不渲染第二行）', async () => {
      const wrapper = mountField({
        installedImages: [
          { id: 'img-meta', name: 'trae-agent-skill', version: 'x86_64-latest', description: '技能运行环境', updated_at: '2026-08-20T10:11:12Z' },
          { id: 'img-plain', name: 'plain-img', version: 'latest' },
          { id: 'img-fallback', name: 'fallback-img', version: 'v1', installed_at: '2026-08-01T02:03:00Z' },
        ],
      })
      await setText(wrapper, '$')
      const lis = wrapper.findAll('[data-testid="task-description-image-mention-picker"] li')
      const metaOf = (name) => lis.find((l) => l.text().includes(name)).text()
      expect(metaOf('trae-agent-skill')).toContain('trae-agent-skill:x86_64-latest')
      expect(metaOf('trae-agent-skill')).toContain('更新时间')
      expect(metaOf('trae-agent-skill')).toContain('技能运行环境')
      // updated_at 缺失时回退 installed_at 渲染更新时间
      expect(metaOf('fallback-img')).toContain('更新时间')
      // 无描述/无时间 → 不渲染第二行元信息
      expect(metaOf('plain-img')).not.toContain('更新时间')
      wrapper.unmount()
    })

    it('镜像选项展示镜像组图标（有 icon_url 渲染 img，缺失回退占位图）', async () => {
      const wrapper = mountField({
        installedImages: [
          { id: 'img-icon', name: 'icon-image', version: 'latest', icon_url: '/api/ai-provider/public-image-groups/7/icon?h=abc' },
          { id: 'img-noicon', name: 'noicon-image', version: 'latest' },
        ],
      })
      await setText(wrapper, '$')
      const picker = wrapper.find('[data-testid="task-description-image-mention-picker"]')
      expect(picker.exists()).toBe(true)
      const img = picker.find('img[data-testid="task-description-image-mention-icon"]')
      expect(img.exists()).toBe(true)
      expect(img.attributes('src')).toContain('/api/ai-provider/public-image-groups/7/icon?h=abc')
      // 无 icon_url 的镜像回退占位图，不渲染 img
      const noIconLi = picker.findAll('li').find((l) => l.text().includes('noicon-image'))
      expect(noIconLi).toBeTruthy()
      expect(noIconLi.find('img[data-testid="task-description-image-mention-icon"]').exists()).toBe(false)
      expect(noIconLi.find('[data-testid="task-description-image-mention-icon-placeholder"]').exists()).toBe(true)
      wrapper.unmount()
    })

    it('普通正文输入不弹出镜像列表，mention-change 带空 id', async () => {
      const wrapper = mountField()
      await setText(wrapper, '帮我检查下部署问题')
      expect(wrapper.find('[data-testid="task-description-image-mention-picker"]').exists()).toBe(false)
      expect(lastMention(wrapper)).toEqual({ text: '帮我检查下部署问题', id: '' })
      wrapper.unmount()
    })

    it('Enter 选中镜像 → 文本写入 $name，mention-change 携带显式 id', async () => {
      const wrapper = mountField()
      await setText(wrapper, '$agent-d')
      await area(wrapper).trigger('keydown', { key: 'Enter' })
      await wrapper.vm.$nextTick()
      expect(area(wrapper).element.value).toBe('$agent-dev ')
      expect(lastMention(wrapper)).toEqual({ text: '$agent-dev ', id: 'img-3' })
      wrapper.unmount()
    })

    it('选中镜像后输入 / 弹出技能列表，Enter 选中写入技能 token', async () => {
      const wrapper = mountField()
      await setText(wrapper, '$agent-d')
      await area(wrapper).trigger('keydown', { key: 'Enter' })
      await wrapper.vm.$nextTick()
      await setText(wrapper, '$agent-dev /de')
      expect(wrapper.find('[data-testid="task-description-skill-picker"]').exists()).toBe(true)
      const skills = wrapper.findAll('[data-testid="task-description-skill-picker"] li').map((l) => l.text())
      expect(skills.some((s) => s.includes('/dev'))).toBe(true)
      expect(skills.some((s) => s.includes('/debug'))).toBe(true)

      await area(wrapper).trigger('keydown', { key: 'ArrowDown' })
      await area(wrapper).trigger('keydown', { key: 'Enter' })
      await wrapper.vm.$nextTick()
      expect(area(wrapper).element.value).toBe('$agent-dev /debug')
      expect(lastMention(wrapper)).toEqual({ text: '$agent-dev /debug', id: 'img-3' })
      wrapper.unmount()
    })

    it('删除 mention 后 mention-change 带空 id（供父级解绑）', async () => {
      const wrapper = mountField({ modelValue: '$agent-dev /dev' })
      await setText(wrapper, '只是正文，没有镜像了')
      expect(lastMention(wrapper)).toEqual({ text: '只是正文，没有镜像了', id: '' })
      wrapper.unmount()
    })
  })

  describe('TaskDescriptionSkillField 同名多变体', () => {
    it('已绑定 image（dup-2）时 mention-change 携带绑定镜像 id，而非 $name 首命中', async () => {
      const wrapper = mountField({
        installedImages: DUP_IMAGES,
        image: DUP_IMAGES[1], // x86_64-latest 变体，已有技能
      })
      await setText(wrapper, '$trae-agent /general-coding')
      expect(lastMention(wrapper)).toEqual({ text: '$trae-agent /general-coding', id: 'dup-2' })
      wrapper.unmount()
    })

    it('未绑定镜像时按 $name 首命中回退', async () => {
      const wrapper = mountField({ installedImages: DUP_IMAGES, image: null })
      await setText(wrapper, '$trae-agent')
      expect(lastMention(wrapper)).toEqual({ text: '$trae-agent', id: 'dup-1' })
      wrapper.unmount()
    })
  })

  describe('TaskDescriptionSkillField 技能 chips', () => {
    it('image 有技能时渲染 $镜像名 /技能名 提示，空描述点击写入完整 mention 并带镜像 id', async () => {
      const wrapper = mountField({ image: IMAGES[2] })
      const chips = wrapper.findAll('[data-testid="task-description-skill-chips"] button')
      expect(chips.length).toBe(3)
      // 提示展示完整 mention 语法：$镜像名 /技能名；默认技能保留「默认」标记
      expect(chips[0].text()).toContain('$agent-dev /dev')
      expect(chips[0].text()).toContain('默认')
      expect(chips[1].text()).toBe('$agent-dev /debug')
      expect(chips[2].text()).toBe('$agent-dev /test')
      await chips[0].trigger('click')
      await wrapper.vm.$nextTick()
      // 空描述 + 已绑定镜像（如项目默认镜像）：写入完整 `$镜像 /技能`，禁止只插 /技能 导致解绑
      expect(area(wrapper).element.value).toBe('$agent-dev /dev')
      expect(lastMention(wrapper)).toEqual({ text: '$agent-dev /dev', id: 'img-3' })
      wrapper.unmount()
    })

    it('描述已有补充说明时点击 chip 把 mention 置入自由段开头并保留正文', async () => {
      const wrapper = mountField({ image: IMAGES[2], modelValue: '帮我修部署' })
      const chips = wrapper.findAll('[data-testid="task-description-skill-chips"] button')
      await chips[1].trigger('click')
      await wrapper.vm.$nextTick()
      expect(area(wrapper).element.value).toBe('$agent-dev /debug\n\n帮我修部署')
      expect(lastMention(wrapper)).toEqual({ text: '$agent-dev /debug\n\n帮我修部署', id: 'img-3' })
      wrapper.unmount()
    })

    it('已有其它技能 mention 时点击另一 chip 只替换技能、不丢镜像 id', async () => {
      const wrapper = mountField({ image: IMAGES[2], modelValue: '$agent-dev /dev' })
      const chips = wrapper.findAll('[data-testid="task-description-skill-chips"] button')
      await chips[1].trigger('click')
      await wrapper.vm.$nextTick()
      expect(area(wrapper).element.value).toBe('$agent-dev /debug')
      expect(lastMention(wrapper)).toEqual({ text: '$agent-dev /debug', id: 'img-3' })
      wrapper.unmount()
    })

    it('同名多变体已绑定 dup-2 时点击 chip emit 该变体 id 而非 $name 首命中', async () => {
      const wrapper = mountField({
        installedImages: DUP_IMAGES,
        image: DUP_IMAGES[1],
        modelValue: '',
      })
      const chips = wrapper.findAll('[data-testid="task-description-skill-chips"] button')
      expect(chips[0].text()).toContain('$trae-agent /general-coding')
      await chips[0].trigger('click')
      await wrapper.vm.$nextTick()
      expect(area(wrapper).element.value).toBe('$trae-agent /general-coding')
      expect(lastMention(wrapper)).toEqual({ text: '$trae-agent /general-coding', id: 'dup-2' })
      wrapper.unmount()
    })

    it('镜像名带前导 $ 时标签剥离前缀，不出现 $$', async () => {
      const wrapper = mountField({
        image: {
          id: 'img-x',
          name: '$agent-dev',
          version: 'latest',
          image_skills: { skills: [{ name: 'dev' }] },
        },
      })
      const chip = wrapper.find('[data-testid="task-description-skill-chips"] button')
      expect(chip.text()).toBe('$agent-dev /dev 默认')
      wrapper.unmount()
    })
  })
}
