// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] createTaskDescriptionCompose.unit.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    applyComposedDescriptionToTask,
    composeCreateTaskDescription,
    emptyCreateTaskStructuredFields,
    normalizeForbidNewDeps,
    parseCreateTaskDescription,
  } = await import('./createTaskDescriptionCompose.js')

  describe('createTaskDescriptionCompose', () => {
    it('T1 compose：仅自由描述', () => {
      expect(composeCreateTaskDescription({ description: '  仅说明  ' })).toBe('仅说明')
    })

    it('T2 compose：部分结构化字段按固定顺序', () => {
      const out = composeCreateTaskDescription({
        description: '补充',
        currentProblem: '报错了',
        taskBackground: '后台',
      })
      expect(out).toBe([
        '补充',
        '',
        '## 任务背景',
        '后台',
        '',
        '## 当前问题是什么',
        '报错了',
      ].join('\n'))
    })

    it('T3 compose：禁止依赖=是', () => {
      const out = composeCreateTaskDescription({ forbidNewDeps: 'yes' })
      expect(out).toContain('## 是否禁止新增依赖\n是')
    })

    it('T4 parse：标准 Markdown 回填', () => {
      const md = [
        '自由段落',
        '',
        '## 任务背景',
        '背景A',
        '',
        '## 是否禁止新增依赖',
        '否',
        '',
        '## 建议验证方式',
        '跑单测',
      ].join('\n')
      const parsed = parseCreateTaskDescription(md)
      expect(parsed.description).toBe('自由段落')
      expect(parsed.taskBackground).toBe('背景A')
      expect(parsed.forbidNewDeps).toBe('no')
      expect(parsed.suggestedVerification).toBe('跑单测')
      expect(parsed.currentProblem).toBe('')
    })

    it('T5 parse：无结构正文全部落入 description', () => {
      const parsed = parseCreateTaskDescription('普通一段话\n第二行')
      expect(parsed.description).toBe('普通一段话\n第二行')
      expect(parsed.taskBackground).toBe('')
      expect(parsed.forbidNewDeps).toBe('')
    })

    it('normalizeForbidNewDeps 识别中英文', () => {
      expect(normalizeForbidNewDeps('是')).toBe('yes')
      expect(normalizeForbidNewDeps('否')).toBe('no')
      expect(normalizeForbidNewDeps('maybe')).toBe('')
    })

    it('applyComposedDescriptionToTask 重试不重复段落', () => {
      const task = {
        description: '补充',
        ...emptyCreateTaskStructuredFields(),
        taskBackground: 'B',
      }
      applyComposedDescriptionToTask(task)
      const first = task.description
      applyComposedDescriptionToTask(task)
      expect(task.description).toBe(first)
      expect(task.description.match(/## 任务背景/g)).toHaveLength(1)
    })
  })
}
