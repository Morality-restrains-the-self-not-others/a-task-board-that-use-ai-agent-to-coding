// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] imageSkills.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    findActiveSlashSkillQuery,
    highlightSkillTokens,
    imageSkillsFromImage,
    insertSkillToken,
    shouldShowImageSkills,
  } = await import('./imageSkills.js')

  const image = {
    image_skills: {
      version: 1,
      default_skill: 'general-coding',
      skills: [
        { name: 'general-coding', description: 'dev', is_default: true },
        { name: 'k8s-debug', is_default: false },
      ],
    },
  }

  describe('imageSkillsFromImage', () => {
    it('uses first skill as default', () => {
      const list = imageSkillsFromImage(image)
      expect(list.defaultSkill).toBe('general-coding')
      expect(list.skills[0].isDefault).toBe(true)
      expect(list.skills[1].isDefault).toBe(false)
    })

    it('透传服务端派生的稳定技能 ID；缺失时为空串', () => {
      const withId = {
        image_skills: {
          skills: [
            { id: 'sk_abc123', name: 'general-coding' },
            { name: 'k8s-debug' },
          ],
        },
      }
      const list = imageSkillsFromImage(withId)
      expect(list.skills[0].id).toBe('sk_abc123')
      expect(list.skills[1].id).toBe('')
    })
  })

  describe('shouldShowImageSkills', () => {
    it('shows when skills exist', () => {
      expect(shouldShowImageSkills(image)).toBe(true)
      expect(shouldShowImageSkills({})).toBe(false)
    })
  })

  describe('highlightSkillTokens', () => {
    it('highlights matching /skill only', () => {
      const parts = highlightSkillTokens('请用 /k8s-debug 排查 /unknown', ['k8s-debug', 'general-coding'])
      const hits = parts.filter((p) => p.highlight).map((p) => p.text)
      expect(hits).toEqual(['/k8s-debug'])
    })
  })

  describe('findActiveSlashSkillQuery / insertSkillToken', () => {
    it('detects slash query after space', () => {
      expect(findActiveSlashSkillQuery('$img /k8', 8)).toEqual({ start: 5, query: 'k8' })
    })
    it('replaces active query', () => {
      expect(insertSkillToken('$img /k8', 8, 'k8s-debug')).toEqual({
        text: '$img /k8s-debug',
        caret: 15,
      })
    })
  })
}
