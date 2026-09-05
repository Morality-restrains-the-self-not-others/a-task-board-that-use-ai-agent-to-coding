// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] createTaskImageMention.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    applyImageMentionToDescription,
    applyImageMentionToTaskDescription,
    bindTaskContainerImageSkillId,
    parseImageMentionFromText,
    stripImageMentionFromDescription,
    syncTaskContainerImageFromMention,
  } = await import('./createTaskImageMention.js')

  const IMAGES = [
    { id: 'img-1', name: 'trae-agent', version: 'x86_64-latest' },
    { id: 'img-2', name: 'python-env', version: '3.11' },
  ]
  const IMAGE_WITH_SKILLS = [
    { id: 'img-1', name: 'trae-agent', version: 'x86_64-latest', image_skills: { skills: [{ name: 'dev', description: '开发' }, { name: 'debug' }] } },
  ]

  describe('parseImageMentionFromText', () => {
    it('解析 $镜像 与 /技能', () => {
      expect(parseImageMentionFromText('$trae-agent /dev', IMAGE_WITH_SKILLS)).toEqual({
        id: 'img-1',
        name: 'trae-agent',
        skill: 'dev',
      })
    })

    it('技能需在所选镜像技能目录内（未知技能忽略）', () => {
      expect(parseImageMentionFromText('$trae-agent /unknown', IMAGE_WITH_SKILLS)).toEqual({
        id: 'img-1',
        name: 'trae-agent',
        skill: '',
      })
    })

    it('仅镜像不带技能', () => {
      expect(parseImageMentionFromText('$python-env', IMAGES)).toEqual({
        id: 'img-2',
        name: 'python-env',
        skill: '',
      })
    })

    it('未匹配已安装列表返回 null', () => {
      expect(parseImageMentionFromText('$nope', IMAGES)).toBeNull()
    })
  })

  describe('stripImageMentionFromDescription', () => {
    it('剥离自由段开头 mention 行，保留正文与结构化段', () => {
      const desc = '$trae-agent /dev\n\n这是任务正文\n\n## 任务背景\n内容'
      const stripped = stripImageMentionFromDescription(desc)
      expect(stripped).toBe('这是任务正文\n\n## 任务背景\n内容')
    })

    it('仅 mention 时清空', () => {
      expect(stripImageMentionFromDescription('$trae-agent /dev')).toBe('')
    })

    it('无 mention 时原样返回', () => {
      expect(stripImageMentionFromDescription('正文')).toBe('正文')
    })

    it('不误伤正文中段 $ 符号', () => {
      const desc = '联系 $运维 处理'
      expect(stripImageMentionFromDescription(desc)).toBe('联系 $运维 处理')
    })
  })

  describe('applyImageMentionToDescription', () => {
    it('空文本时只剥离旧 mention', () => {
      expect(applyImageMentionToDescription('$trae-agent /dev\n\n正文', '')).toBe('正文')
    })

    it('置入新 mention（自由段开头、空行分隔）', () => {
      expect(applyImageMentionToDescription('正文', '$python-env')).toBe('$python-env\n\n正文')
    })

    it('先剥离旧再置入新（幂等）', () => {
      const once = applyImageMentionToDescription('$trae-agent /dev\n\n正文', '$trae-agent /debug')
      expect(once).toBe('$trae-agent /debug\n\n正文')
      const twice = applyImageMentionToDescription(once, '$trae-agent /debug')
      expect(twice).toBe('$trae-agent /debug\n\n正文')
    })

    it('保留结构化段落', () => {
      const out = applyImageMentionToDescription(
        '## 任务背景\n内容',
        '$trae-agent /dev',
      )
      expect(out).toBe('$trae-agent /dev\n\n## 任务背景\n内容')
    })
  })

  describe('applyImageMentionToTaskDescription', () => {
    it('从草稿合并进 description', () => {
      const task = {
        description: '正文',
        container_image: { id: 'img-1', mentionText: '$trae-agent /dev' },
      }
      applyImageMentionToTaskDescription(task)
      expect(task.description).toBe('$trae-agent /dev\n\n正文')
    })

    it('mentionText 为空时剥离描述中的旧 mention', () => {
      const task = {
        description: '$trae-agent /dev\n\n正文',
        container_image: { id: null, mentionText: '' },
      }
      applyImageMentionToTaskDescription(task)
      expect(task.description).toBe('正文')
    })

    it('无 container_image 草稿时原样返回', () => {
      const task = { description: '正文' }
      applyImageMentionToTaskDescription(task)
      expect(task.description).toBe('正文')
    })
  })

  describe('syncTaskContainerImageFromMention', () => {
    it('mention 命中 → 写 id / mentionText / skill', () => {
      const task = { description: '', container_image: { id: null } }
      const result = syncTaskContainerImageFromMention(task, '$trae-agent /dev', IMAGE_WITH_SKILLS)
      expect(result.mention).not.toBeNull()
      expect(task.container_image).toEqual({
        id: 'img-1',
        mentionText: '$trae-agent /dev',
        skill: 'dev',
      })
    })

    it('同名多变体：显式 id 优先，$name 首命中不覆盖', () => {
      const dupImages = [
        { id: 'dup-1', name: 'trae-agent', version: 'private_x86_64-latest' },
        { id: 'dup-2', name: 'trae-agent', version: 'x86_64-latest', image_skills: { skills: [{ name: 'general-coding' }] } },
      ]
      const task = { container_image: {} }
      const result = syncTaskContainerImageFromMention(task, '$trae-agent /general-coding', dupImages, 'dup-2')
      expect(result.resolved.id).toBe('dup-2')
      expect(task.container_image.id).toBe('dup-2')
      expect(task.container_image.skill).toBe('general-coding')
    })

    it('未命中 → 不改动 task，返回 mention null', () => {
      const task = { container_image: { id: 'img-1', mentionText: '旧' } }
      const result = syncTaskContainerImageFromMention(task, '普通正文', IMAGES)
      expect(result.mention).toBeNull()
      expect(task.container_image).toEqual({ id: 'img-1', mentionText: '旧' })
    })

    it('无 container_image 草稿时自动初始化对象', () => {
      const task = { description: '' }
      const result = syncTaskContainerImageFromMention(task, '$trae-agent', IMAGES)
      expect(result.mention).not.toBeNull()
      expect(task.container_image.id).toBe('img-1')
    })
  })

  describe('bindTaskContainerImageSkillId', () => {
    const ID_IMAGES = [
      {
        id: 'img-1',
        name: 'trae-agent',
        image_skills: {
          skills: [
            { id: 'sk_abc123', name: 'dev', description: '开发' },
            { id: 'sk_def456', name: 'debug' },
          ],
        },
      },
      { id: 'img-2', name: 'legacy-img', image_skills: { skills: [{ name: 'old' }] } },
    ]

    it('技能名 → 稳定 ID（D1=B 目录反查）', () => {
      const task = { container_image: { id: 'img-1', skill: 'debug' } }
      expect(bindTaskContainerImageSkillId(task, ID_IMAGES)).toBe(true)
      expect(task.container_image.skillId).toBe('sk_def456')
    })

    it('目录无 id（老数据）→ skillId 置空，不阻断', () => {
      const task = { container_image: { id: 'img-2', skill: 'old' } }
      bindTaskContainerImageSkillId(task, ID_IMAGES)
      expect(task.container_image.skillId).toBe('')
    })

    it('技能名不在所选镜像列表 → 置空', () => {
      const task = { container_image: { id: 'img-1', skill: 'unknown' } }
      bindTaskContainerImageSkillId(task, ID_IMAGES)
      expect(task.container_image.skillId).toBe('')
    })

    it('无镜像 id 或无技能名 → 置空', () => {
      const task = { container_image: { id: '', skill: 'dev' } }
      bindTaskContainerImageSkillId(task, ID_IMAGES)
      expect(task.container_image.skillId).toBe('')
      const task2 = { container_image: { id: 'img-1', skill: '' } }
      bindTaskContainerImageSkillId(task2, ID_IMAGES)
      expect(task2.container_image.skillId).toBe('')
    })

    it('无 container_image 草稿 → no-op', () => {
      const task = { description: '' }
      expect(bindTaskContainerImageSkillId(task, ID_IMAGES)).toBe(false)
      expect(task.container_image).toBeUndefined()
    })

    it('非法 task → false', () => {
      expect(bindTaskContainerImageSkillId(null, ID_IMAGES)).toBe(false)
    })
  })
}
