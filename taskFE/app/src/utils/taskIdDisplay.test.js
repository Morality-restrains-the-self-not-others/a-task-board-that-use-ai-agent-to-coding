// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] taskIdDisplay.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { formatTaskDisplayNo, formatTaskIdTitleLabel, taskIdLast6 } = await import('./taskIdDisplay.js')

  describe('taskIdLast6', () => {
    it('空值返回空串', () => {
      expect(taskIdLast6(null)).toBe('')
      expect(taskIdLast6(undefined)).toBe('')
      expect(taskIdLast6('')).toBe('')
    })

    it('不超过 6 位原样返回', () => {
      expect(taskIdLast6('abc')).toBe('abc')
      expect(taskIdLast6(123456)).toBe('123456')
    })

    it('超过 6 位取末尾', () => {
      expect(taskIdLast6('850256677331014872')).toBe('014872')
    })
  })

  describe('formatTaskDisplayNo', () => {
    it('正整数为 #N', () => {
      expect(formatTaskDisplayNo(1)).toBe('#1')
      expect(formatTaskDisplayNo('103')).toBe('#103')
    })

    it('无效序号为空串', () => {
      expect(formatTaskDisplayNo(0)).toBe('')
      expect(formatTaskDisplayNo(null)).toBe('')
      expect(formatTaskDisplayNo('x')).toBe('')
    })
  })

  describe('formatTaskIdTitleLabel', () => {
    it('空 id 且无序号返回空串', () => {
      expect(formatTaskIdTitleLabel('', '标题')).toBe('')
      expect(formatTaskIdTitleLabel(null)).toBe('')
    })

    it('有 workspace_seq 时为 #N + 标题', () => {
      expect(formatTaskIdTitleLabel('task_13646028863068037877', '源任务名称', 12)).toBe(
        '#12 源任务名称',
      )
    })

    it('有序号无标题时仅 #N', () => {
      expect(formatTaskIdTitleLabel('task_13646028863068037877', '', 3)).toBe('#3')
      expect(formatTaskIdTitleLabel('task_13646028863068037877', undefined, 3)).toBe('#3')
    })

    it('无序号不回退技术 ID 后六位', () => {
      expect(formatTaskIdTitleLabel('task_13646028863068037877', '源任务名称')).toBe('源任务名称')
    })
  })
}
