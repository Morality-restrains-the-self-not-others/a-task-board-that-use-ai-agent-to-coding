// @vitest-environment jsdom
// 集成测试：任务描述 textarea 的 $镜像/技能选择 → container_image 绑定（单一入口，
// 「已安装镜像」独立字段已下线）。精确复现用户操作序列，每步断言草稿状态，
// 定位「技能选中后 mentionText 被截断值覆盖」的竞态。
if (!process.env.VITEST) {
  console.log('[skip] CreateTaskMentionSync.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { reactive } = await import('vue')

  const { default: CreateTaskBasicFields } = await import('./CreateTaskBasicFields.vue')
  const { defaultCreateTaskFieldSettings } = await import('../utils/createTaskFieldSettings.js')
  const { applyImageMentionToTaskDescription } = await import('../utils/createTaskImageMention.js')

  // 生产形态：同名多变体（private 无技能 / x86_64 有技能），x86_64 是默认选中的目标
  const DUP_IMAGES = [
    { id: 'dup-1', name: 'trae-agent', version: 'private_x86_64-latest' },
    {
      id: 'dup-2',
      name: 'trae-agent',
      version: 'x86_64-latest',
      image_skills: {
        version: 1,
        default_skill: 'general-coding',
        skills: [
          { name: 'general-coding', description: 'x', is_default: true },
          { name: 'k8s-debug', description: 'y', is_default: false },
        ],
      },
    },
  ]

  const mountHost = () => {
    const task = reactive({
      title: '',
      description: '',
      progressColumn: { id: null },
      task_type: { id: null },
      container_image: { id: null },
    })
    const fieldSettings = defaultCreateTaskFieldSettings()
    const wrapper = mount(
      {
        components: { CreateTaskBasicFields },
        props: { images: { type: Array, default: () => [] } },
        template: `
          <div>
            <CreateTaskBasicFields
              :editing-task="task"
              :task-statuses="[]"
              :task-types="[]"
              :field-settings="fieldSettings"
              :installed-images="images"
            />
          </div>
        `,
        setup(props) {
          return { task, fieldSettings, images: props.images }
        },
      },
      { props: { images: DUP_IMAGES } },
    )
    return { wrapper, task }
  }

  const area = (wrapper) => wrapper.find('#task-description')
  const imgPicker = (wrapper) => wrapper.find('[data-testid="task-description-image-mention-picker"]')
  const skillPicker = (wrapper) => wrapper.find('[data-testid="task-description-skill-picker"]')

  const settle = async (wrapper, rounds = 3) => {
    // watch flush + defineModel prop 下行需多轮微任务
    for (let i = 0; i < rounds; i += 1) {
      await wrapper.vm.$nextTick()
    }
  }

  // jsdom 下 vue-test-utils setValue 的 input 事件 handler 顺序与真实浏览器不同
  // （v-model handler 可能晚于 onInput 执行，onInput 读到旧值不刷新弹层）；
  // 补一次 input 触发使状态与真实输入一致（真实浏览器弹层行为已实测正常）。
  const typeText = async (wrapper, value) => {
    await area(wrapper).setValue(value)
    await area(wrapper).trigger('input')
    await settle(wrapper)
  }

  describe('创建任务描述 $镜像/技能 → 绑定同步（集成，单一入口）', () => {
    it('完整流程：选镜像 → 输技能 → 选技能 → mentionText 保持完整', async () => {
      const { wrapper, task } = mountHost()
      await settle(wrapper, 5)
      expect(area(wrapper).exists()).toBe(true)

      // 1. 输入 $trae → 镜像弹层
      await typeText(wrapper, '$trae')
      expect(imgPicker(wrapper).exists()).toBe(true)
      const labels = imgPicker(wrapper).findAll('li').map((l) => l.text())
      expect(labels.join('|')).toContain('trae-agent:x86_64-latest')

      // 2. ArrowDown 高亮第二个（x86_64）→ Enter 选中
      await area(wrapper).trigger('keydown', { key: 'ArrowDown' })
      await area(wrapper).trigger('keydown', { key: 'Enter' })
      await settle(wrapper, 5)
      expect(area(wrapper).element.value).toBe('$trae-agent ')
      expect(task.container_image.id).toBe('dup-2')

      // 3. 输入不完整技能 /general-c → 技能弹层（中间态 mentionText 跟随文本截断）
      await typeText(wrapper, '$trae-agent /general-c')
      await settle(wrapper, 5)
      expect(skillPicker(wrapper).exists()).toBe(true)
      expect(task.container_image.mentionText).toBe('$trae-agent /general-c')

      // 4. Enter 选中技能 general-coding（回归：selectSkill 后 emit 必须携带完整文本，
      //    不得因 defineModel prop 下行滞后读到旧值而把 mentionText 覆盖回截断值）
      await area(wrapper).trigger('keydown', { key: 'Enter' })
      await settle(wrapper, 8)
      const taValue = area(wrapper).element.value
      const mentionText = task.container_image.mentionText
      const skill = task.container_image.skill

      // 期望：全部完整
      expect(taValue).toBe('$trae-agent /general-coding')
      expect(mentionText).toBe('$trae-agent /general-coding')
      expect(skill).toBe('general-coding')

      // 5. 模拟 handleSubmit 的合并
      applyImageMentionToTaskDescription(task, DUP_IMAGES)
      expect(task.description).toBe('$trae-agent /general-coding')
      wrapper.unmount()
    })

    it('已绑定默认镜像且描述为空时点击技能 chip 写入完整 mention 且不解绑', async () => {
      const { wrapper, task } = mountHost()
      task.container_image.id = 'dup-2'
      await settle(wrapper, 5)
      const chips = wrapper.findAll('[data-testid="task-description-skill-chips"] button')
      expect(chips.length).toBeGreaterThan(0)
      expect(chips[0].text()).toContain('$trae-agent /general-coding')
      await chips[0].trigger('click')
      await settle(wrapper, 8)
      expect(area(wrapper).element.value).toBe('$trae-agent /general-coding')
      expect(task.container_image.id).toBe('dup-2')
      expect(task.container_image.skill).toBe('general-coding')
      expect(task.container_image.mentionText).toBe('$trae-agent /general-coding')
      wrapper.unmount()
    })
  })
}
